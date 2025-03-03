// Supervisor 是这个模拟器中的一个抽象角色，可以读取 txs，生成分区信息，
//并处理历史数据。它还可以将区块信息发送给领导者，以便领导者可以将其转发给其他节点。

package supervisor

import (
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"blockEmulator/supervisor/committee"
	"blockEmulator/supervisor/measure"
	"blockEmulator/supervisor/signal"
	"blockEmulator/supervisor/supervisor_log"
	"bufio"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type Supervisor struct { //Supervisor结构包含客户端的各种信息
	//基本信息
	//为了Supervisor可以实现全局以及消息通信，需要知道分片区块链网络中各个节点的具体地址系统配置信息，同时作为节点，需要设置相应的TCP端口以及日志信息
	IPaddr       string                       //主管的 IP 地址
	ChainConfig  *params.ChainConfig          //链配置
	Ip_nodeTable map[uint64]map[uint64]string //区块链模拟中节点的 IP 地址映射，IPmap_nodeTable它是一个两级映射，

	// tcp控制
	listenStop bool         //是否停止监听
	tcpLn      net.Listener //tcp监听器
	tcpLock    sync.Mutex   //tcp锁
	//记录器模块
	sl *supervisor_log.SupervisorLog //主管日志

	//控制元件
	Ss *signal.StopSignal //负责全局网络的节点的终止信息分布

	//委员会模块
	comMod committee.CommitteeModule //负责设置区块链系统使用的意见协议以及跨分片意见机制，如PBFT、Relay、BrokerChain、CLPA

	//测量模块
	testMeasureMods []measure.MeasureModule //负责区块测试链的各类性能，如TPS、延迟、跨分片交易率等

	//在此处添加更多结构或类
}

func (d *Supervisor) NewSupervisor(ip string, pcc *params.ChainConfig, committeeMethod string, mearsureModNames ...string) {
	//NewSupervisor方法用于创建和配置客户端，参数分别代表节点总数、分片总数和委员会方法
	d.IPaddr = ip                           //将主管的 IP 地址设置为ip
	d.ChainConfig = pcc                     //将链配置设置为pcc
	d.Ip_nodeTable = params.IPmap_nodeTable //将区块链模拟中节点的 IP 地址映射设置为params.IPmap_nodeTable

	d.sl = supervisor_log.NewSupervisorLog() //创建一个新的主管日志

	d.Ss = signal.NewStopSignal(2 * int(pcc.ShardNums)) //创建一个新的停止信号

	switch committeeMethod {
	case "CLPA_Broker":
		d.comMod = committee.NewCLPACommitteeMod_Broker(d.Ip_nodeTable, d.Ss, d.sl, params.FileInput, params.TotalDataSize, params.BatchSize, 80)
	case "CLPA":
		d.comMod = committee.NewCLPACommitteeModule(d.Ip_nodeTable, d.Ss, d.sl, params.FileInput, params.TotalDataSize, params.BatchSize, 80)
	case "Broker":
		d.comMod = committee.NewBrokerCommitteeMod(d.Ip_nodeTable, d.Ss, d.sl, params.FileInput, params.TotalDataSize, params.BatchSize)
	default:
		d.comMod = committee.NewRelayCommitteeModule(d.Ip_nodeTable, d.Ss, d.sl, params.FileInput, params.TotalDataSize, params.BatchSize) //创建一个新的委员会模块
	}

	d.testMeasureMods = make([]measure.MeasureModule, 0) //创建一个新的测量模块
	for _, mModName := range mearsureModNames {          //遍历mearsureModNames
		switch mModName {
		case "TPS_Relay":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestModule_avgTPS_Relay()) //将新的测量模块添加到d.testMeasureMods中
		case "TPS_Broker":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestModule_avgTPS_Broker())
		case "TCL_Relay":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestModule_TCL_Relay())
		case "TCL_Broker":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestModule_TCL_Broker())
		case "CrossTxRate_Relay":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestCrossTxRate_Relay())
		case "CrossTxRate_Broker":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestCrossTxRate_Broker())
		case "TxNumberCount_Relay":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestTxNumCount_Relay())
		case "TxNumberCount_Broker":
			d.testMeasureMods = append(d.testMeasureMods, measure.NewTestTxNumCount_Broker())
		default:
		}
	}
}

// Supervisor收到Leader发来的区块信息，通过处理消息来衡量性能。
func (d *Supervisor) handleBlockInfos(content []byte) {
	//handleBlockInfos方法用于处理区块信息，参数content是一个字节切片，表示区块信息
	bim := new(message.BlockInfoMsg)    //创建一个新的BlockInfoMsg结构，BlockInfoMsg结构包含区块信息消息的各种信息
	err := json.Unmarshal(content, bim) //将content解码为bim
	if err != nil {
		log.Panic()
	}
	// StopSignal check
	if bim.BlockBodyLength == 0 { //如果区块体长度为0，则表明该块体没有交易
		d.Ss.StopGap_Inc() //增加停止间隔
	} else {
		d.Ss.StopGap_Reset() //重置停止间隔
	}

	d.comMod.AdjustByBlockInfos(bim) //根据区块信息调整委员会模块

	// measure update
	for _, measureMod := range d.testMeasureMods { //遍历d.testMeasureMods
		measureMod.UpdateMeasureRecord(bim) //更新测量记录
	}
	// add codes here ...
}

// 从数据文件中读取交易。当数据量足够的时候，Supervisor 将进行重新分区并将partitionMSG 和txs 发送给领导者。
func (d *Supervisor) SupervisorTxHandling() { //SupervisorTxHandling方法用于处理交易
	d.comMod.TxHandling() //委员会模块处理交易

	// TxHandling is end
	for !d.Ss.GapEnough() { //等待所有交易被处理，并检查是否满足某个间隙条件 (d.Ss.GapEnough())。如果不满足间隙条件，它将等待一秒 (time.Sleep(time.Second))，然后再次检查。一旦满足间隙条件，就会进行下一步
		time.Sleep(time.Second) //休眠1秒
	}
	// 向所有节点发送停止消息
	stopmsg := message.MergeMessage(message.CStop, []byte("this is a stop message~")) // 通过将消息类型 (message.CStop) 与包含停止消息描述的字节片合并来准备停止消息 (stopmsg)，然后将其发送到所有节点
	d.sl.Slog.Println("Supervisor: now sending cstop message to all nodes")           //打印日志
	for sid := uint64(0); sid < d.ChainConfig.ShardNums; sid++ {                      //遍历分片
		for nid := uint64(0); nid < d.ChainConfig.Nodes_perShard; nid++ { //遍历节点
			networks.TcpDial(stopmsg, d.Ip_nodeTable[sid][nid]) //通过TCP连接发送stopmsg消息
		}
	}
	d.sl.Slog.Println("Supervisor: now Closing") //打印日志
	d.listenStop = true                          //设置listenStop为true，表明客户端应该停止侦听或处理进一步的消息
	d.CloseSupervisor()                          //关闭客户端
}

//上面的函数用于管理事务的处理，等待满足特定条件，向所有节点发送停止消息，并为 Supervisor 启动关闭过程。

// 处理传入的消息。
func (d *Supervisor) handleMessage(msg []byte) { //handleMessage方法用于处理消息，msg表示传入消息的字节片
	msgType, content := message.SplitMessage(msg) //将消息拆分为消息类型和内容
	switch msgType {                              //根据消息类型进行不同处理
	case message.CBlockInfo: //如果消息类型为CBlockInfo，则调用d.handleBlockInfos(content)，这用于处理块信息消息
		d.handleBlockInfos(content)
		// add codes for more functionality
	default: //否则，调用d.comMod.HandleOtherMessage(msg)，以使用委员会模块处理消息。然后，遍历d.testMeasureMods，调用mm.HandleExtraMessage(msg)以处理额外消息，这是处理非块信息的不同类型消息的通用机制。
		d.comMod.HandleOtherMessage(msg)
		for _, mm := range d.testMeasureMods { //遍历d.testMeasureMods
			mm.HandleExtraMessage(msg) //处理额外消息
		}
	}
}

func (d *Supervisor) handleClientRequest(con net.Conn) { //handleClientRequest方法用于处理客户端请求，con表示客户端连接
	defer con.Close()                    //延迟关闭连接
	clientReader := bufio.NewReader(con) //创建一个新的缓冲读取器
	for {
		clientRequest, err := clientReader.ReadBytes('\n') //读取客户端请求
		switch err {
		case nil: //如果没有错误，则调用d.handleMessage(clientRequest)以处理消息
			d.tcpLock.Lock()
			d.handleMessage(clientRequest)
			d.tcpLock.Unlock()
		case io.EOF: //如果发生EOF错误，则打印日志，然后返回
			log.Println("client closed the connection by terminating the process")
			return
		default: //否则，打印错误日志，然后返回
			log.Printf("error: %v\n", err)
			return
		}
	}
}

func (d *Supervisor) TcpListen() { //TcpListen方法用于监听TCP连接
	ln, err := net.Listen("tcp", d.IPaddr)
	if err != nil {
		log.Panic(err)
	}
	d.tcpLn = ln
	for {
		conn, err := d.tcpLn.Accept()
		if err != nil {
			return
		}
		go d.handleClientRequest(conn)
	}
}

// tcp listen for Supervisor
func (d *Supervisor) OldTcpListen() { //OldTcpListen方法用于监听TCP连接
	ipaddr, err := net.ResolveTCPAddr("tcp", d.IPaddr)
	if err != nil {
		log.Panic(err)
	}
	ln, err := net.ListenTCP("tcp", ipaddr)
	d.tcpLn = ln
	if err != nil {
		log.Panic(err)
	}
	d.sl.Slog.Printf("Supervisor begins listening：%s\n", d.IPaddr)

	for {
		conn, err := d.tcpLn.Accept()
		if err != nil {
			if d.listenStop {
				return
			}
			log.Panic(err)
		}
		b, err := io.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		d.handleMessage(b)
		conn.(*net.TCPConn).SetLinger(0)
		defer conn.Close()
	}
}

// 关闭Supervisor，并将数据记录在.csv文件中
func (d *Supervisor) CloseSupervisor() { //CloseSupervisor方法用于关闭客户端
	d.sl.Slog.Println("Closing...")
	for _, measureMod := range d.testMeasureMods {
		d.sl.Slog.Println(measureMod.OutputMetricName())
		d.sl.Slog.Println(measureMod.OutputRecord())
		println()
	}

	d.sl.Slog.Println("Trying to input .csv")
	// write to .csv file
	dirpath := params.DataWrite_path + "supervisor_measureOutput/"
	err := os.MkdirAll(dirpath, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}
	for _, measureMod := range d.testMeasureMods {
		targetPath := dirpath + measureMod.OutputMetricName() + ".csv"
		f, err := os.Open(targetPath)
		resultPerEpoch, totResult := measureMod.OutputRecord()
		resultStr := make([]string, 0)
		for _, result := range resultPerEpoch {
			resultStr = append(resultStr, strconv.FormatFloat(result, 'f', 8, 64))
		}
		resultStr = append(resultStr, strconv.FormatFloat(totResult, 'f', 8, 64))
		if err != nil && os.IsNotExist(err) {
			file, er := os.Create(targetPath)
			if er != nil {
				panic(er)
			}
			defer file.Close()

			w := csv.NewWriter(file)
			title := []string{measureMod.OutputMetricName()}
			w.Write(title)
			w.Flush()
			w.Write(resultStr)
			w.Flush()
		} else {
			file, err := os.OpenFile(targetPath, os.O_APPEND|os.O_RDWR, 0666)

			if err != nil {
				log.Panic(err)
			}
			defer file.Close()
			writer := csv.NewWriter(file)
			err = writer.Write(resultStr)
			if err != nil {
				log.Panic()
			}
			writer.Flush()
		}
		f.Close()
		d.sl.Slog.Println(measureMod.OutputRecord())
	}
	networks.CloseAllConnInPool()
	d.tcpLn.Close()
}
