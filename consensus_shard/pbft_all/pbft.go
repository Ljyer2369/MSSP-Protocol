// The pbft consensus process

package pbft_all

import (
	"blockEmulator/chain"
	"blockEmulator/consensus_shard/pbft_all/dataSupport"
	"blockEmulator/consensus_shard/pbft_all/pbft_log"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"blockEmulator/shard"
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type PbftConsensusNode struct { //PbftConsensusNode结构包含用于PBFT共识的各种配置参数
	// pbft的本地配置
	RunningNode *shard.Node // 运行PBFT共识的节点信息
	ShardID     uint64      // 表示分片（或PBFT）的ID，表示一个分片内只有一个PBFT共识
	NodeID      uint64      // 表示PBFT（分片）内节点的ID

	// 区块链的数据结构
	CurChain *chain.BlockChain // 对分片中所有节点维护的区块链的引用
	db       ethdb.Database    // 用于保存 Merkle Patricia Trie (MPT) 的数据库

	// pbft的全局配置
	pbftChainConfig *params.ChainConfig          //pbft 中的链配置
	ip_nodeTable    map[uint64]map[uint64]string //表示特定节点的ip地址映射，其中键是uint64值，并且每个键都映射到另一个映射。内部映射又使用uint64键来映射到string值
	node_nums       uint64                       //该PBFT实例中的节点数量，记为N
	malicious_nums  uint64                       //恶意节点的数量（f），其中3f + 1 = N
	view            uint64                       //表示当前视图的ID

	//pbft中的控制消息和消息检查实用程序
	sequenceID        uint64                          //表示PBFT的消息序列ID
	stop              bool                            //表示共识停止的布尔标志
	pStop             chan uint64                     //表示共识停止的通道
	requestPool       map[string]*message.Request     //表示请求池，其中键是字符串，值是指向Request结构的指针
	cntPrepareConfirm map[string]map[*shard.Node]bool //用于统计准备确认消息的数量，其中键是字符串，值是指向shard.Node结构的指针的映射
	cntCommitConfirm  map[string]map[*shard.Node]bool //用于统计提交确认消息的数量，其中键是字符串，值是指向shard.Node结构的指针的映射
	isCommitBordcast  map[string]bool                 //表示是否已经广播提交消息，其中键是字符串
	isReply           map[string]bool                 //指示消息是否是回复的映射。其中键是字符串，值是布尔值
	height2Digest     map[uint64]string               //表示消息高度到消息摘要的映射，其中键是uint64值，值是字符串

	//关于 pbft 的锁
	sequenceLock sync.Mutex //锁定序列ID
	lock         sync.Mutex //锁定共识
	askForLock   sync.Mutex //锁定请求
	stopLock     sync.Mutex //锁定停止

	//其他Shards的seqID，用于同步
	seqIDMap   map[uint64]uint64 //用于与其他分片同步序列ID的映射。
	seqMapLock sync.Mutex        //锁定seqIDMap

	// pbft 日志
	pl *pbft_log.PbftLog //用于记录日志。它是一个记录器，允许您将消息记录到各种输出源。
	// tcp 控制
	tcpln       net.Listener //用于监听TCP连接的TCP侦听器
	tcpPoolLock sync.Mutex   //用于管理TCP连接池的互斥锁

	//处理pbft中的消息
	ihm PbftInsideExtraHandleMod //用于处理pbft内部消息的接口

	// 处理pbft外部的消息
	ohm PbftOutsideHandleMod //用于处理pbft外部消息的接口
}

// 为节点生成 pbft 共识
func NewPbftNode(shardID, nodeID uint64, pcc *params.ChainConfig, messageHandleType string) *PbftConsensusNode {
	//NewPbftNode()函数用于创建一个新的PbftConsensusNode结构。它需要四个参数： shardID（类型为uint64）：这是表示分片 ID 的参数。 nodeID（类型uint64）：这是表示节点 ID 的参数。 pcc（类型为*params.ChainConfig）：这是一个指向params.ChainConfig结构的指针。该结构包含用于区块链仿真或模拟的各种配置参数。 messageHandleType（类型为字符串）：这是一个字符串，表示如何处理pbft内部消息。
	p := new(PbftConsensusNode) //使用new关键字创建一个PbftConsensusNode结构的新实例。它返回一个指向新创建的结构的指针。
	//初始化PbftConsensusNode实例的各个字段
	p.ip_nodeTable = params.IPmap_nodeTable //该变量似乎代表节点的 IP 地址映射，IPmap_nodeTable它是一个映射，其中键是uint64值，并且每个键都映射到另一个映射。内部映射又使用uint64键来映射到string值
	p.node_nums = pcc.Nodes_perShard        //该PBFT实例中的节点数量
	p.ShardID = shardID
	p.NodeID = nodeID
	p.pbftChainConfig = pcc                                                                          //该变量代表PBFT中的链配置
	fp := "./record/ldb/s" + strconv.FormatUint(shardID, 10) + "/n" + strconv.FormatUint(nodeID, 10) //构建一个文件路径，用于保存 Merkle Patricia Trie (MPT) 的数据库。该文件路径是根据shardID和nodeID参数构建的，其中似乎包括ShardID和NodeID。这将为ShardID和NodeID的每个组合创建一个唯一的文件路径。
	var err error
	p.db, err = rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false) //使用rawdb.NewLevelDBDatabase()函数创建一个新的LevelDBDatabase。它需要五个参数： fp（类型为字符串）：该变量似乎代表文件路径。该文件路径是根据shardID和nodeID参数构建的，其中似乎包括ShardID和NodeID。这将为ShardID和NodeID的每个组合创建一个唯一的文件路径。 0：该变量似乎代表缓存大小。 1：该变量似乎代表缓存增量。 "accountState"：该变量似乎代表数据库名称。 false：该变量似乎代表是否只读。
	if err != nil {
		log.Panic(err)
	}
	p.CurChain, err = chain.NewBlockChain(pcc, p.db) //使用chain.NewBlockChain()函数创建一个新的BlockChain。它需要两个参数： pcc（类型为*params.ChainConfig）：这是一个指向params.ChainConfig结构的指针。该结构包含用于区块链仿真或模拟的各种配置参数。 p.db（类型为ethdb.Database）：该变量似乎代表数据库。它是一个接口，允许您将键映射到值。
	if err != nil {
		log.Panic("cannot new a blockchain")
	}

	p.RunningNode = &shard.Node{ //创建一个指向shard.Node结构的指针并使用特定值对其进行初始化
		NodeID:  nodeID,
		ShardID: shardID,
		IPaddr:  p.ip_nodeTable[shardID][nodeID],
	}

	p.stop = false //将stop字段设置为false
	p.sequenceID = p.CurChain.CurrentBlock.Header.Number + 1
	p.pStop = make(chan uint64)
	p.requestPool = make(map[string]*message.Request)
	p.cntPrepareConfirm = make(map[string]map[*shard.Node]bool)
	p.cntCommitConfirm = make(map[string]map[*shard.Node]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	p.height2Digest = make(map[uint64]string)
	p.malicious_nums = (p.node_nums - 1) / 3
	p.view = 0

	p.seqIDMap = make(map[uint64]uint64)

	p.pl = pbft_log.NewPbftLog(shardID, nodeID)

	//选择如何处理 pbft 中或 pbft 之外的消息
	switch string(messageHandleType) { //
	case "CLPA_Broker":
		ncdm := dataSupport.NewCLPADataSupport()
		p.ihm = &CLPAPbftInsideExtraHandleMod_forBroker{
			pbftNode: p,
			cdm:      ncdm,
		}
		p.ohm = &CLPABrokerOutsideModule{
			pbftNode: p,
			cdm:      ncdm,
		}
	case "CLPA":
		ncdm := dataSupport.NewCLPADataSupport() //使用dataSupport.NewCLPADataSupport()函数创建一个新的CLPADataSupport。它不需要参数。它返回一个指向CLPADataSupport结构的指针。
		p.ihm = &CLPAPbftInsideExtraHandleMod{
			pbftNode: p,
			cdm:      ncdm,
		}
		p.ohm = &CLPARelayOutsideModule{
			pbftNode: p,
			cdm:      ncdm,
		}
	case "Broker":
		p.ihm = &RawBrokerPbftExtraHandleMod{
			pbftNode: p,
		}
		p.ohm = &RawBrokerOutsideModule{
			pbftNode: p,
		}
	default:
		p.ihm = &RawRelayPbftExtraHandleMod{ //使用&运算符创建一个指向RawRelayPbftExtraHandleMod结构的指针。它需要一个参数： p（类型为*PbftConsensusNode）：这是一个指向PbftConsensusNode结构的指针。
			pbftNode: p,
		}
		p.ohm = &RawRelayOutsideModule{
			pbftNode: p,
		}
	}

	return p
}

// 处理原始消息，将其发送到相应的接口
func (p *PbftConsensusNode) handleMessage(msg []byte) { //handleMessage()函数用于处理原始消息。它需要一个参数： msg（类型为[]字节）：这是一个字节切片，表示原始消息。
	msgType, content := message.SplitMessage(msg) //使用message.SplitMessage()函数将原始消息拆分为消息类型和内容。它需要一个参数： msg（类型为[]字节）：这是一个字节切片，表示原始消息。它返回两个值： msgType（类型为message.MessageType）：这是一个枚举类型，表示消息类型。 content（类型为[]字节）：这是一个字节切片，表示消息内容。
	//使用一条switch语句来处理不同类型的消息
	switch msgType {
	//pbft 内部消息类型
	case message.CPrePrepare:
		p.handlePrePrepare(content)
	case message.CPrepare:
		p.handlePrepare(content)
	case message.CCommit:
		p.handleCommit(content)
	case message.CRequestOldrequest:
		p.handleRequestOldSeq(content)
	case message.CSendOldrequest:
		p.handleSendOldSeq(content)
	case message.CStop:
		p.WaitToStop()

	//处理来自外部的消息
	default:
		p.ohm.HandleMessageOutsidePBFT(msgType, content) //对于未知或未处理类型的消息，它使用 p.ohm.HandleMessageOutsidePBFT(msgType, content) 将它们转发到外部处理程序。这使得来自 PBFT 共识算法外部的消息能够得到相应的处理。
	}
}

//上面函数充当消息调度程序，根据传入消息的类型将传入消息路由到适当的处理程序。是PBFT共识节点消息处理逻辑的核心部分。

func (p *PbftConsensusNode) handleClientRequest(con net.Conn) { //handleClientRequest()函数用于处理客户端请求。它需要一个参数： con（类型为net.Conn）：这是一个net.Conn接口，表示TCP连接。
	defer con.Close()
	clientReader := bufio.NewReader(con)
	for {
		clientRequest, err := clientReader.ReadBytes('\n')
		if p.getStopSignal() {
			return
		}
		switch err {
		case nil:
			p.tcpPoolLock.Lock()
			p.handleMessage(clientRequest)
			p.tcpPoolLock.Unlock()
		case io.EOF:
			log.Println("client closed the connection by terminating the process")
			return
		default:
			log.Printf("error: %v\n", err)
			return
		}
	}
}

func (p *PbftConsensusNode) TcpListen() { //TcpListen()函数用于监听TCP连接。它需要一个参数： p（类型为*PbftConsensusNode）：这是一个指向PbftConsensusNode结构的指针。
	ln, err := net.Listen("tcp", p.RunningNode.IPaddr)
	p.tcpln = ln
	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := p.tcpln.Accept()
		if err != nil {
			return
		}
		go p.handleClientRequest(conn)
	}
}

// listen to the request
func (p *PbftConsensusNode) OldTcpListen() { //OldTcpListen()函数用于监听TCP连接。它需要一个参数： p（类型为*PbftConsensusNode）：这是一个指向PbftConsensusNode结构的指针。
	ipaddr, err := net.ResolveTCPAddr("tcp", p.RunningNode.IPaddr)
	if err != nil {
		log.Panic(err)
	}
	ln, err := net.ListenTCP("tcp", ipaddr)
	p.tcpln = ln
	if err != nil {
		log.Panic(err)
	}
	p.pl.Plog.Printf("S%dN%d begins listening：%s\n", p.ShardID, p.NodeID, p.RunningNode.IPaddr)

	for {
		if p.getStopSignal() {
			p.closePbft()
			return
		}
		conn, err := p.tcpln.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := io.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		p.handleMessage(b)
		conn.(*net.TCPConn).SetLinger(0)
		defer conn.Close()
	}
}

// 当收到停止消息时，关闭共识
func (p *PbftConsensusNode) WaitToStop() { //WaitToStop()函数用于等待共识停止。它需要一个参数： p（类型为*PbftConsensusNode）：这是一个指向PbftConsensusNode结构的指针。
	p.pl.Plog.Println("handling stop message")
	p.stopLock.Lock()
	p.stop = true
	p.stopLock.Unlock()
	if p.NodeID == p.view {
		p.pStop <- 1
	}
	networks.CloseAllConnInPool()
	p.tcpln.Close()
	p.closePbft()
	p.pl.Plog.Println("handled stop message")
}

func (p *PbftConsensusNode) getStopSignal() bool { //getStopSignal()函数用于获取共识停止信号。它需要一个参数： p（类型为*PbftConsensusNode）：这是一个指向PbftConsensusNode结构的指针。
	p.stopLock.Lock()
	defer p.stopLock.Unlock()
	return p.stop
}

// close the pbft
func (p *PbftConsensusNode) closePbft() { //closePbft()函数用于关闭PBFT共识。它需要一个参数： p（类型为*PbftConsensusNode）：这是一个指向PbftConsensusNode结构的指针。
	p.CurChain.CloseBlockChain()
}
