package message

import "blockEmulator/core"

// 如果使用事务中继，该消息也用于发送序列ID
type Relay struct { //Relay结构包含中继消息的各种信息
	Txs           []*core.Transaction //指向交易的指针
	SenderShardID uint64              //发送此消息的分片ID
	SenderSeq     uint64              //发送此消息的序列ID
}
