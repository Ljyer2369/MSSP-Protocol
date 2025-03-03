package pbft_all

import "blockEmulator/message"

type PbftInsideExtraHandleMod interface { //PbftInsideExtraHandleMod接口用于处理PBFT共识内部的额外处理
	HandleinPropose() (bool, *message.Request)               //HandleinPropose方法用于提出不同类型的请求
	HandleinPrePrepare(*message.PrePrepare) bool             //HandleinPrePrepare方法用于处理预准备消息
	HandleinPrepare(*message.Prepare) bool                   //HandleinPrepare方法用于处理准备消息
	HandleinCommit(*message.Commit) bool                     //HandleinCommit方法用于处理commit消息
	HandleReqestforOldSeq(*message.RequestOldMessage) bool   //HandleReqestforOldSeq方法用于处理旧序列的请求
	HandleforSequentialRequest(*message.SendOldMessage) bool //HandleforSequentialRequest方法用于处理顺序请求
}

type PbftOutsideHandleMod interface { //PbftOutsideHandleMod接口用于处理PBFT共识外部的额外处理
	HandleMessageOutsidePBFT(message.MessageType, []byte) bool
}
