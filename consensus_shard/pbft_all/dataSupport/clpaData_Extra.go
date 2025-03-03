package dataSupport

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"sync"
)

type Data_supportCLPA struct { //Data_supportCLPA结构包含用于CLPA机制的各种数据结构
	ModifiedMap             []map[string]uint64                   //记录决策者修改后的地图
	AccountTransferRound    uint64                                //表示 accountTransfer 执行的次数
	ReceivedNewAccountState map[string]*core.AccountState         // 来自其他分片的新 accountState
	ReceivedNewTx           []*core.Transaction                   //来自其他分片池的新交易
	AccountStateTx          map[uint64]*message.AccountStateAndTx //accountState 和交易、池的映射
	PartitionOn             bool                                  //判断nextEpoch是否分区

	PartitionReady map[uint64]bool //判断所有分片是否都完成了所有txs
	P_ReadyLock    sync.Mutex      //锁定准备就绪

	ReadySeq     map[uint64]uint64 //当分片准备好时记录 seqid
	ReadySeqLock sync.Mutex        // lock for seqMap

	CollectOver bool       //判断是否所有tx都被收集
	CollectLock sync.Mutex // lock for collect
}

func NewCLPADataSupport() *Data_supportCLPA { //NewCLPADataSupport函数用于创建和配置 CLPA 委员会模块，参数分别代表节点总数、分片总数、委员会方法、委员会模块的日志、csv文件路径、数据总数、批次中的数据记录数、CLPA算法的频率
	return &Data_supportCLPA{
		ModifiedMap:             make([]map[string]uint64, 0),
		AccountTransferRound:    0,
		ReceivedNewAccountState: make(map[string]*core.AccountState),
		ReceivedNewTx:           make([]*core.Transaction, 0),
		AccountStateTx:          make(map[uint64]*message.AccountStateAndTx),
		PartitionOn:             false,
		PartitionReady:          make(map[uint64]bool),
		CollectOver:             false,
		ReadySeq:                make(map[uint64]uint64),
	}
}
