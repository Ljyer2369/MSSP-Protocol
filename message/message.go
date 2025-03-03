package message

import (
	"blockEmulator/core"
	"blockEmulator/shard"
	"time"
)

var prefixMSGtypeLen = 30 //消息类型的长度

type MessageType string //消息类型
type RequestType string //请求类型

const (
	CPrePrepare        MessageType = "preprepare"        //表示预准备消息
	CPrepare           MessageType = "prepare"           //表示准备消息
	CCommit            MessageType = "commit"            //表示提交消息
	CRequestOldrequest MessageType = "requestOldrequest" //表示请求旧请求消息
	CSendOldrequest    MessageType = "sendOldrequest"    //表示发送旧请求消息
	CStop              MessageType = "stop"              //表示停止消息

	CRelay  MessageType = "relay"  //表示中继消息
	CInject MessageType = "inject" //表示注入消息

	CBlockInfo MessageType = "BlockInfo"  //表示区块信息消息
	CSeqIDinfo MessageType = "SequenceID" //表示序列ID信息消息
)

var (
	BlockRequest RequestType = "Block" //表示块请求
	// add more types
	// ...
)

type RawMessage struct { //RawMessage结构包含原始消息的各种信息
	Content []byte //包括原始消息、交易和区块（大多数情况）的内容
}

type Request struct { //Request结构包含请求消息的各种信息
	RequestType RequestType //请求类型
	Msg         RawMessage  //原始消息
	ReqTime     time.Time   //请求时间
}

type PrePrepare struct { //PrePrepare结构包含预准备消息的各种信息
	RequestMsg *Request //指向请求消息的指针
	Digest     []byte   //该请求的摘要，这是唯一的标识符
	SeqID      uint64   //序列ID
}

type Prepare struct { //Prepare结构包含准备消息的各种信息
	Digest     []byte      //该请求的摘要，这是唯一的标识符
	SeqID      uint64      //序列ID
	SenderNode *shard.Node //发送此消息的节点
}

type Commit struct { //Commit结构包含提交消息的各种信息
	Digest     []byte      //该请求的摘要，这是唯一的标识符
	SeqID      uint64      //序列ID
	SenderNode *shard.Node //发送此消息的节点
}

type Reply struct { //Reply结构包含响应消息的各种信息
	MessageID  uint64      //消息ID
	SenderNode *shard.Node //发送此消息的节点
	Result     bool        //响应的结果
}

type RequestOldMessage struct { //RequestOldMessage结构包含请求旧消息的各种信息
	SeqStartHeight uint64      //序列开始高度
	SeqEndHeight   uint64      //序列结束高度
	ServerNode     *shard.Node //将此请求发送到服务器节点
	SenderNode     *shard.Node //发送此消息的节点
}

type SendOldMessage struct { //SendOldMessage结构包含发送旧消息的各种信息
	SeqStartHeight uint64      //序列开始高度
	SeqEndHeight   uint64      //序列结束高度
	OldRequest     []*Request  //旧请求
	SenderNode     *shard.Node //发送此消息的节点
}

type InjectTxs struct { //InjectTxs结构包含注入的交易的各种信息
	Txs       []*core.Transaction //指向交易的指针
	ToShardID uint64              //目标分片ID
}

type BlockInfoMsg struct { //BlockInfoMsg结构包含区块信息消息的各种信息
	BlockBodyLength int                 //区块体长度
	ExcutedTxs      []*core.Transaction //已完全执行的交易
	Epoch           int                 //当前时期

	ProposeTime   time.Time //记录该区块的提出时间（txs）
	CommitTime    time.Time //记录该块的提交时间（txs）
	SenderShardID uint64    //发送此消息的分片ID

	//用于交易中继
	Relay1TxNum uint64              //跨分片交易数量
	Relay1Txs   []*core.Transaction //链上首次跨分片交易

	// for broker
	Broker1TxNum uint64              // the number of broker 1
	Broker1Txs   []*core.Transaction // cross transactions at first time by broker
	Broker2TxNum uint64              // the number of broker 2
	Broker2Txs   []*core.Transaction // cross transactions at second time by broker
}

type SeqIDinfo struct { //SeqIDinfo结构包含序列ID信息消息的各种信息
	SenderShardID uint64 //发送此消息的分片ID
	SenderSeq     uint64 //发送此消息的序列ID
}

func MergeMessage(msgType MessageType, content []byte) []byte {
	//MergeMessage()函数用于合并消息类型和内容。它需要两个参数： msgType（类型为message.MessageType）：这是一个枚举类型，表示消息类型。 content（类型为[]字节）：这是一个字节切片，表示消息内容。
	b := make([]byte, prefixMSGtypeLen)
	for i, v := range []byte(msgType) {
		b[i] = v
	}
	merge := append(b, content...)
	return merge //返回合并的消息
}

func SplitMessage(message []byte) (MessageType, []byte) {
	//SplitMessage()函数用于拆分消息类型和内容。它需要一个参数： message（类型为[]字节）：这是一个字节切片，表示原始消息。它返回两个值： msgType（类型为message.MessageType）：这是一个枚举类型，表示消息类型。 content（类型为[]字节）：这是一个字节切片，表示消息内容。
	msgTypeBytes := message[:prefixMSGtypeLen]
	msgType_pruned := make([]byte, 0)
	for _, v := range msgTypeBytes {
		if v != byte(0) {
			msgType_pruned = append(msgType_pruned, v)
		}
	}
	msgType := string(msgType_pruned)
	content := message[prefixMSGtypeLen:]
	return MessageType(msgType), content //返回消息类型和内容
}
