package committee

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"blockEmulator/supervisor/signal"
	"blockEmulator/supervisor/supervisor_log"
	"blockEmulator/utils"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"os"
	"time"
)

type RelayCommitteeModule struct { //RelayCommitteeModule结构包含中继委员会模块的各种信息
	csvPath      string                        //csv文件路径，指定从中读取或写入数据或记录的文件。CSV 文件通常用于数据存储和交换。
	dataTotalNum int                           //数据总数，指示中继委员会模块正在管理或处理的数据总量
	nowDataNum   int                           //当前数据量，它可能会跟踪已处理或当前正在考虑的数据记录的数量。
	batchDataNum int                           //批次中的数据记录数，指示中继委员会模块在每个批次中处理的数据记录数。
	IpNodeTable  map[uint64]map[uint64]string  //区块链模拟中节点的 IP 地址映射，IpNodeTable它是一个两级映射，其中外部映射具有 uint64 类型的键，可以表示分片 ID，内部映射也具有 uint64 类型的键并映射到字符串值，表示 IP 地址。此映射允许模块根据节点的分片和节点 ID 确定节点的 IP 地址。
	sl           *supervisor_log.SupervisorLog //主管日志
	Ss           *signal.StopSignal            //负责全局网络的节点的终止信息分送，用于表示某些进程或操作的终止
}

func NewRelayCommitteeModule(Ip_nodeTable map[uint64]map[uint64]string, Ss *signal.StopSignal, slog *supervisor_log.SupervisorLog, csvFilePath string, dataNum, batchNum int) *RelayCommitteeModule {
	//NewRelayCommitteeModule方法用于创建和配置中继委员会模块，参数分别代表节点总数、分片总数、委员会方法、委员会模块的日志、csv文件路径、数据总数、批次中的数据记录数
	return &RelayCommitteeModule{
		csvPath:      csvFilePath,
		dataTotalNum: dataNum,
		batchDataNum: batchNum,
		nowDataNum:   0,
		IpNodeTable:  Ip_nodeTable,
		Ss:           Ss,
		sl:           slog,
	}
}

// transfrom, data to transaction
// 检查是否是合法的txs消息。如果是，则读取txs并将其放入txlist中
func data2tx(data []string, nonce uint64) (*core.Transaction, bool) {
	//检查给定的数据集是否代表有效的交易消息，如果是，则将数据转换为 core.Transaction 对象。
	// data：这是一个字符串片段，包含各种信息，可能代表交易消息。
	//nonce：这是一个无符号 64 位整数，表示随机数值。它在创建交易时使用。
	if data[6] == "0" && data[7] == "0" && len(data[3]) > 16 && len(data[4]) > 16 && data[3] != data[4] { //检查交易是否有效，
		// 它检查 data[6] 和 data[7] 是否都等于“0”，这可能表明使消息有效的某些条件。
		//它检查 data[3] 和 data[4] 的长度是否大于 16，这可能是验证标准的一部分。
		//它检查 data[3] 是否不等于 data[4]，这可能表明发件人和收件人地址不同。
		val, ok := new(big.Int).SetString(data[8], 10) //将 data[8] 转换为 big.Int 对象。
		if !ok {                                       //检查转换是否成功。
			log.Panic("new int failed\n")
		}
		tx := core.NewTransaction(data[3][2:], data[4][2:], val, nonce) //通过调用 core.NewTransaction(data[3][2:], data[4][2:], val, nonce) 创建一个 core.Transaction 对象 (tx)。 data[3][2:]和data[4][2:]可能表示带有“0x”前缀的十六进制地址，val表示交易值。nonce为交易提供随机数值。
		return tx, true
	}
	return &core.Transaction{}, false
	//如果数据表示有效交易，则该函数将返回 tx 对象以及 true。否则，它返回一个空的 core.Transaction 对象和 false 来指示该数据不是有效的交易。
}

//上面的函数负责解析和验证交易数据，并在数据有效时创建交易对象。它通常在区块链系统中用于处理传入的交易消息。

func (rthm *RelayCommitteeModule) HandleOtherMessage([]byte) {} //HandleOtherMessage()函数用于处理其他消息

func (rthm *RelayCommitteeModule) txSending(txlist []*core.Transaction) { //txSending()函数用于发送交易，txlist是表示需要发送的交易的 core.Transaction 对象的切片
	//将一批交易发送到各个分片中，根据发送者的分片ID将交易进行分类和批量发送。
	// the txs will be sent
	sendToShard := make(map[uint64][]*core.Transaction) //是一个映射，其中键是分片的ID，值是一个 core.Transaction 对象的切片。它用于组织和存储将要发送到特定分片的交易。在一开始，这个映射是一个空映射。

	for idx := 0; idx <= len(txlist); idx++ { //遍历 txlist 中的交易。txlist 包含了要发送的交易的列表
		if idx > 0 && (idx%params.InjectSpeed == 0 || idx == len(txlist)) {
			//首先检查是否已经处理了一定数量的交易，或者是否已经处理完了所有交易：
			//idx > 0 表示已经处理了一定数量的交易。
			//(idx % params.InjectSpeed == 0 || idx == len(txlist)) 表示已经处理了 params.InjectSpeed 定义的特定数量的交易，或者已经处理完了所有交易。
			// 如果上述条件满足，就会将交易发送到各个分片
			for sid := uint64(0); sid < uint64(params.ShardNum); sid++ { //循环遍历各个分片，其中 sid 表示分片的ID
				it := message.InjectTxs{ //创建一个新的 InjectTxs 对象 (it)，其中包含要发送到分片的交易列表 (Txs) 和目标分片 ID (ToShardID)
					Txs:       sendToShard[sid],
					ToShardID: sid,
				}
				itByte, err := json.Marshal(it) //通过调用 json.Marshal(it) 将 it 对象转换为字节切片 (itByte)
				if err != nil {
					log.Panic(err)
				}
				send_msg := message.MergeMessage(message.CInject, itByte) //创建一个消息 send_msg，将消息类型设为 message.CInject，并将 itByte 数据合并进来
				go networks.TcpDial(send_msg, rthm.IpNodeTable[sid][0])   //通过调用 networks.TcpDial(send_msg, rthm.IpNodeTable[sid][0]) 将消息发送到目标分片中的第一个节点 (rthm.IpNodeTable[sid][0])。
			}
			sendToShard = make(map[uint64][]*core.Transaction) //发送完交易后，重置 sendToShard 映射，以便在下一批次中使用。
			time.Sleep(time.Second)                            //在每批交易之间添加一秒的延迟，以控制发送交易的速率
		}
		if idx == len(txlist) { //如果已经处理完所有交易，则退出循环
			break
		}
		tx := txlist[idx] //从 txlist 中获取下一笔交易 tx
		sendersid := uint64(utils.Addr2Shard(tx.Sender))
		//对于 txlist 中的每笔交易，它通过使用 utils.Addr2Shard(tx.Sender) 将发送者的地址转换为分片 ID 来计算发送者的分片 ID。发送者的分片 ID (sendersid) 用于确定交易的目标分片。
		sendToShard[sendersid] = append(sendToShard[sendersid], tx)
		//将这笔交易添加到 sendToShard 映射中，根据 sendersid 所对应的分片，将该交易追加到相应的列表中
	}
}

//上面的函数负责组织交易并将其发送到各自的分片。它是跨分片交易系统的重要组成部分，其中交易需要在区块链网络中的多个分片上分布和执行。

// 读取交易，交易数量为 -batchDataNum
func (rthm *RelayCommitteeModule) TxHandling() { //TxHandling()函数用于处理交易
	txfile, err := os.Open(rthm.csvPath) //通过 os.Open(rthm.csvPath) 打开指定路径 rthm.csvPath 的 CSV 文件，并将返回的文件句柄存储在 txfile 变量中
	if err != nil {
		log.Panic(err)
	}
	defer txfile.Close()
	reader := csv.NewReader(txfile)        //使用 csv.NewReader(txfile) 创建一个 CSV 读取器 reader，用于逐行读取 CSV 文件中的数据
	txlist := make([]*core.Transaction, 0) // 创建一个空的交易列表 txlist，用于存储当前处理的一批交易。

	for { //循环读取 CSV 文件中的数据
		data, err := reader.Read() //通过调用 reader.Read() 读取 CSV 文件中的一行数据，并将其存储在 data 变量中
		if err == io.EOF {         //如果读取到文件末尾，则退出循环
			break
		}
		if err != nil { //如果读取过程中发生错误，则打印错误日志并退出循环
			log.Panic(err)
		}
		if tx, ok := data2tx(data, uint64(rthm.nowDataNum)); ok { //检查读取的数据是否有效：调用 data2tx(data, uint64(rthm.nowDataNum)) 函数将读取的数据转换为交易对象，并检查交易是否有效。如果有效，将其添加到 txlist 中，同时将 rthm.nowDataNum 递增。
			txlist = append(txlist, tx)
			rthm.nowDataNum++
		}

		// 检查是否需要重新分片
		if len(txlist) == int(rthm.batchDataNum) || rthm.nowDataNum == rthm.dataTotalNum { //如果 txlist 中的交易数量达到 rthm.batchDataNum（定义的批量数据数）或者 rthm.nowDataNum 达到了 rthm.dataTotalNum（总数据数），则说明需要重新分片
			rthm.txSending(txlist) //调用 rthm.txSending(txlist) 函数，将 txlist 中的交易发送到各个分片中
			// 重置与交易发送相关的变量
			txlist = make([]*core.Transaction, 0) //重置 txlist，以便在下一批次中使用
			rthm.Ss.StopGap_Reset()               //使用 rthm.Ss.StopGap_Reset() 重置与交易发送间隔相关的变量。
		}

		if rthm.nowDataNum == rthm.dataTotalNum { //如果 rthm.nowDataNum 达到了 rthm.dataTotalNum，则说明已经处理完所有交易，此时退出循环
			break
		}
	}
}

// no operation here
func (rthm *RelayCommitteeModule) AdjustByBlockInfos(b *message.BlockInfoMsg) { //AdjustByBlockInfos()函数用于根据区块信息调整委员会模块，b表示区块信息消息
	rthm.sl.Slog.Printf("received from shard %d in epoch %d.\n", b.SenderShardID, b.Epoch) //打印日志
}
