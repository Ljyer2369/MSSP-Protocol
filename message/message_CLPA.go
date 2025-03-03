package message

import (
	"blockEmulator/core"
	"bytes"
	"encoding/gob"
	"log"
)

var (
	AccountState_and_TX MessageType = "AccountState&txs"
	PartitionReq        RequestType = "PartitionReq"
	CPartitionMsg       MessageType = "PartitionModifiedMap"
	CPartitionReady     MessageType = "ready for partition"
)

type PartitionModifiedMap struct {
	PartitionModified map[string]uint64
}

type AccountTransferMsg struct { //AccountTransferMsg结构包含账户转移消息的各种信息
	ModifiedMap  map[string]uint64
	Addrs        []string
	AccountState []*core.AccountState
	ATid         uint64
}

type PartitionReady struct { //PartitionReady结构包含准备分区消息的各种信息
	FromShard uint64 //来自的分片
	NowSeqID  uint64 //现在的序列ID
}

// this message used in inter-shard, it will be sent between leaders.
type AccountStateAndTx struct { //AccountStateAndTx结构包含账户状态和交易的各种信息
	Addrs        []string
	AccountState []*core.AccountState
	Txs          []*core.Transaction
	FromShard    uint64
}

func (atm *AccountTransferMsg) Encode() []byte { //Encode()函数用于编码账户转移消息
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(atm)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func DecodeAccountTransferMsg(content []byte) *AccountTransferMsg { //DecodeAccountTransferMsg()函数用于解码账户转移消息
	var atm AccountTransferMsg

	decoder := gob.NewDecoder(bytes.NewReader(content))
	err := decoder.Decode(&atm)
	if err != nil {
		log.Panic(err)
	}

	return &atm
}
