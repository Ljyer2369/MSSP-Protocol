// 新共识的附加模块
package pbft_all

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

// pbftHandleModule 接口的简单实现 ...
// 仅适用于区块请求并使用交易中继
type RawRelayPbftExtraHandleMod struct { //RawRelayPbftExtraHandleMod结构包含用于PBFT共识的各种配置参数
	pbftNode *PbftConsensusNode // 指向 pbft 数据的指针，PbftConsensusNode结构包含用于PBFT共识的各种配置参数
}

// 提出不同类型的请求
func (rphm *RawRelayPbftExtraHandleMod) HandleinPropose() (bool, *message.Request) { //HandleinPropose方法用于提出不同类型的请求
	// new blocks
	block := rphm.pbftNode.CurChain.GenerateBlock() //生成区块
	r := &message.Request{                          //创建一个新的请求
		RequestType: message.BlockRequest, //请求类型
		ReqTime:     time.Now(),           //请求时间
	}
	r.Msg.Content = block.Encode() //将区块编码后的内容设置为请求的内容

	return true, r
}

// preprepare中的diy操作
func (rphm *RawRelayPbftExtraHandleMod) HandleinPrePrepare(ppmsg *message.PrePrepare) bool { //HandleinPrePrepare方法用于处理预准备消息
	if rphm.pbftNode.CurChain.IsValidBlock(core.DecodeB(ppmsg.RequestMsg.Msg.Content)) != nil { //如果区块不合法
		rphm.pbftNode.pl.Plog.Printf("S%dN%d : not a valid block\n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID)
		return false
	}
	rphm.pbftNode.pl.Plog.Printf("S%dN%d : the pre-prepare message is correct, putting it into the RequestPool. \n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID) //打印日志
	rphm.pbftNode.requestPool[string(ppmsg.Digest)] = ppmsg.RequestMsg                                                                                            //将预准备消息放入请求池
	//合并为准备消息
	return true
}

// 在prepare中的操作，以及在pbft + tx中继中，这个函数不需要做任何事情。
func (rphm *RawRelayPbftExtraHandleMod) HandleinPrepare(pmsg *message.Prepare) bool { //HandleinPrepare方法用于处理准备消息
	fmt.Println("No operations are performed in Extra handle mod")
	return true
}

// commit 中的操作。在这里，我们将区块添加到区块链中，并将交易中继到其他分片。
func (rphm *RawRelayPbftExtraHandleMod) HandleinCommit(cmsg *message.Commit) bool { //HandleinCommit方法用于处理commit消息
	r := rphm.pbftNode.requestPool[string(cmsg.Digest)] //从请求池中获取请求
	// 请求类型 ...
	block := core.DecodeB(r.Msg.Content)                                                                                                                                                                   //解码区块
	rphm.pbftNode.pl.Plog.Printf("S%dN%d : adding the block %d...now height = %d \n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID, block.Header.Number, rphm.pbftNode.CurChain.CurrentBlock.Header.Number) //打印日志
	rphm.pbftNode.CurChain.AddBlock(block)                                                                                                                                                                 //将区块添加到区块链中
	rphm.pbftNode.pl.Plog.Printf("S%dN%d : added the block %d... \n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID, block.Header.Number)                                                                    //打印日志
	rphm.pbftNode.CurChain.PrintBlockChain()                                                                                                                                                               //打印区块链

	// 现在尝试将 txs 中继到其他分片（如果当前节点是主节点（大概是分片的领导者或协调者））
	if rphm.pbftNode.NodeID == rphm.pbftNode.view { //如果是主节点
		rphm.pbftNode.pl.Plog.Printf("S%dN%d : main node is trying to send relay txs at height = %d \n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID, block.Header.Number) //打印日志，它记录在块的高度发送中继交易的尝试。
		// 生成中继池并收集执行的txs
		//它初始化事务中继的数据结构
		txExcuted := make([]*core.Transaction, 0)                                      //创建一个新的交易切片
		rphm.pbftNode.CurChain.Txpool.RelayPool = make(map[uint64][]*core.Transaction) //创建一个新的交易池
		relay1Txs := make([]*core.Transaction, 0)                                      //创建一个新的交易切片
		for _, tx := range block.Body {                                                //遍历区块中的交易，对于区块中的每笔交易
			rsid := rphm.pbftNode.CurChain.Get_PartitionMap(tx.Recipient) //使用 Get_PartitionMap 确定接收者的分片
			if rsid != rphm.pbftNode.ShardID {                            //如果接收方与发送方不在同一分片中，则该交易将被标记为中继并添加到中继池中。
				ntx := tx                                           //创建一个新的交易
				ntx.Relayed = true                                  //将交易的中继标志设置为true
				rphm.pbftNode.CurChain.Txpool.AddRelayTx(ntx, rsid) //将交易添加到交易池中
				relay1Txs = append(relay1Txs, tx)                   //将交易添加到交易切片中
			} else {
				txExcuted = append(txExcuted, tx) //将交易添加到交易切片中
			}
		}
		// 发送中继交易
		//对于除当前分片之外的每个分片，它使用单独的 Goroutine 发送包含中继事务 (message.Relay) 的消息。
		//发送中继交易后，中继池将被清除
		for sid := uint64(0); sid < rphm.pbftNode.pbftChainConfig.ShardNums; sid++ {
			if sid == rphm.pbftNode.ShardID {
				continue
			}
			relay := message.Relay{
				Txs:           rphm.pbftNode.CurChain.Txpool.RelayPool[sid],
				SenderShardID: rphm.pbftNode.ShardID,
				SenderSeq:     rphm.pbftNode.sequenceID,
			}
			rByte, err := json.Marshal(relay)
			if err != nil {
				log.Panic()
			}
			msg_send := message.MergeMessage(message.CRelay, rByte)
			go networks.TcpDial(msg_send, rphm.pbftNode.ip_nodeTable[sid][0]) //通过TCP连接发送消息
			rphm.pbftNode.pl.Plog.Printf("S%dN%d : sended relay txs to %d\n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID, sid)
		}
		rphm.pbftNode.CurChain.Txpool.ClearRelayPool() //清除中继池
		// 将该块中执行的tx发送给监听器
		// 添加更多消息来测量更多指标
		//有关已执行事务和中继事务的信息被收集并发送给侦听器，用于监视或分析目的。
		bim := message.BlockInfoMsg{
			BlockBodyLength: len(block.Body),
			ExcutedTxs:      txExcuted,
			Epoch:           0,
			Relay1Txs:       relay1Txs,
			Relay1TxNum:     uint64(len(relay1Txs)),
			SenderShardID:   rphm.pbftNode.ShardID,
			ProposeTime:     r.ReqTime,
			CommitTime:      time.Now(),
		}
		bByte, err := json.Marshal(bim)
		if err != nil {
			log.Panic()
		}
		msg_send := message.MergeMessage(message.CBlockInfo, bByte)
		go networks.TcpDial(msg_send, rphm.pbftNode.ip_nodeTable[params.DeciderShard][0])
		rphm.pbftNode.pl.Plog.Printf("S%dN%d : sended excuted txs\n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID)
		rphm.pbftNode.CurChain.Txpool.GetLocked()
		rphm.pbftNode.writeCSVline([]string{strconv.Itoa(len(rphm.pbftNode.CurChain.Txpool.TxQueue)), strconv.Itoa(len(txExcuted)), strconv.Itoa(int(bim.Relay1TxNum))})
		rphm.pbftNode.CurChain.Txpool.GetUnlocked()
	}
	return true
}

func (rphm *RawRelayPbftExtraHandleMod) HandleReqestforOldSeq(*message.RequestOldMessage) bool { //HandleReqestforOldSeq方法用于处理旧序列的请求
	fmt.Println("No operations are performed in Extra handle mod") //
	return true
}

// the operation for sequential requests
func (rphm *RawRelayPbftExtraHandleMod) HandleforSequentialRequest(som *message.SendOldMessage) bool { //HandleforSequentialRequest方法用于处理顺序请求
	if int(som.SeqEndHeight-som.SeqStartHeight+1) != len(som.OldRequest) { //它检查 OldRequest 切片中的元素数量是否等于 SeqEndHeight 和 SeqStartHeight 之间的差值加一。如果它们不相等，则会记录一条消息，指示 SendOldMessage 不够。
		rphm.pbftNode.pl.Plog.Printf("S%dN%d : the SendOldMessage message is not enough\n", rphm.pbftNode.ShardID, rphm.pbftNode.NodeID)
	} else { // 如果 OldRequest 中的元素数量与预期的顺序请求数量匹配，将区块添加到节点 pbft 区块链中
		for height := som.SeqStartHeight; height <= som.SeqEndHeight; height++ { //遍历区块高度
			r := som.OldRequest[height-som.SeqStartHeight] //对于范围内的每个高度，它使用当前高度和 SeqStartHeight 之间的差作为索引，从 OldRequest 切片中检索请求 r。
			if r.RequestType == message.BlockRequest {     //如果请求类型为BlockRequest，则将区块添加到节点pbft区块链中
				b := core.DecodeB(r.Msg.Content)   //解码区块
				rphm.pbftNode.CurChain.AddBlock(b) //使用 AddBlock 方法将解码后的块添加到当前区块链 (rphm.pbftNode.CurChain)。
			}
		}
		rphm.pbftNode.sequenceID = som.SeqEndHeight + 1
		rphm.pbftNode.CurChain.PrintBlockChain()
	}
	return true
}
