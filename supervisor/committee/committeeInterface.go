package committee

import "blockEmulator/message"

type CommitteeModule interface { //CommitteeModule接口定义了委员会模块的接口
	AdjustByBlockInfos(*message.BlockInfoMsg) //AdjustByBlockInfos()函数用于根据区块信息调整委员会
	TxHandling()                              //TxHandling()函数用于处理交易
	HandleOtherMessage([]byte)                //HandleOtherMessage()函数用于处理其他消息
}
