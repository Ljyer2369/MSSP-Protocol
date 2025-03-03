package pbft_all

import (
	"blockEmulator/message"
	"encoding/json"
	"log"
)

// 该模块在区块链中使用，使用交易中继机制
// “Raw”表示pbft仅达成区块共识
type RawRelayOutsideModule struct { //RawRelayOutsideModule结构包含用于PBFT共识外部的各种配置参数
	pbftNode *PbftConsensusNode
}

// msgType可以在message中定义
func (rrom *RawRelayOutsideModule) HandleMessageOutsidePBFT(msgType message.MessageType, content []byte) bool { //HandleMessageOutsidePBFT方法用于处理PBFT共识外部的消息
	switch msgType { //使用一条switch语句来处理不同类型的消息
	case message.CRelay: //如果消息类型为CRelay
		rrom.handleRelay(content)
	case message.CInject: //如果消息类型为CInject
		rrom.handleInjectTx(content)
	default:
	}
	return true
}

// 接收中继交易，用于跨分片交易
func (rrom *RawRelayOutsideModule) handleRelay(content []byte) { //handleRelay方法用于处理中继消息
	relay := new(message.Relay)           //使用new()函数创建一个message.Relay结构的指针
	err := json.Unmarshal(content, relay) //使用json.Unmarshal()函数将消息内容解码为message.Relay结构
	if err != nil {
		log.Panic(err)
	}
	rrom.pbftNode.pl.Plog.Printf("S%dN%d : has received relay txs from shard %d, the senderSeq is %d\n", rrom.pbftNode.ShardID, rrom.pbftNode.NodeID, relay.SenderShardID, relay.SenderSeq) //打印日志，指示该函数已收到中继事务。日志消息包含有关分片和发送者序列的信息
	rrom.pbftNode.CurChain.Txpool.AddTxs2Pool(relay.Txs)                                                                                                                                    //将交易添加到交易池中
	rrom.pbftNode.seqMapLock.Lock()                                                                                                                                                         //使用互斥锁
	rrom.pbftNode.seqIDMap[relay.SenderShardID] = relay.SenderSeq                                                                                                                           //将发送方的分片ID和序列ID添加到seqIDMap中
	rrom.pbftNode.seqMapLock.Unlock()                                                                                                                                                       //解锁
	rrom.pbftNode.pl.Plog.Printf("S%dN%d : has handled relay txs msg\n", rrom.pbftNode.ShardID, rrom.pbftNode.NodeID)                                                                       //打印日志，指示中继事务已被处理
}

//该函数负责接收中继的交易，将其添加到交易池中，并管理与发送者分片相关的序列信息。

func (rrom *RawRelayOutsideModule) handleInjectTx(content []byte) { //handleInjectTx方法用于处理注入交易消息，用于将外部生成的交易添加到区块链或分布式系统内的交易池中
	it := new(message.InjectTxs)       //使用new()函数创建一个message.InjectTxs结构的指针
	err := json.Unmarshal(content, it) //使用json.Unmarshal()函数将消息内容解码为message.InjectTxs结构
	if err != nil {
		log.Panic(err)
	}
	rrom.pbftNode.CurChain.Txpool.AddTxs2Pool(it.Txs)                                                                                           //将交易添加到交易池中
	rrom.pbftNode.pl.Plog.Printf("S%dN%d : has handled injected txs msg, txs: %d \n", rrom.pbftNode.ShardID, rrom.pbftNode.NodeID, len(it.Txs)) //打印日志，指示注入的交易已被处理，日志消息包括有关分片和处理的事务数量的信息
}

//该函数负责处理外部生成的交易并将其添加到交易池中。这是区块链系统中的一种常见机制，允许外部实体提交新交易以包含在区块链中。该函数对注入的交易执行必要的反序列化，并将它们添加到池中以供后续验证并包含在区块链中
