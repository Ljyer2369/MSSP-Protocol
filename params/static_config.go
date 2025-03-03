package params

import "math/big"

type ChainConfig struct { //ChainConfig结构包含用于区块链仿真或模拟的各种配置参数
	ChainID        uint64 //ChainID：该变量似乎代表区块链网络的 ID
	NodeID         uint64 //NodeID：该变量似乎代表节点的 ID
	ShardID        uint64 //ShardID：该变量似乎代表分片的 ID
	Nodes_perShard uint64 //Nodes_perShard：该变量似乎代表每个分片中的节点总数
	ShardNums      uint64 //ShardNums：该变量似乎代表区块链网络中的分片总数
	BlockSize      uint64 //BlockSize：该变量似乎代表区块的大小
	BlockInterval  uint64 //BlockInterval：该变量似乎代表生成新块的时间间隔
	InjectSpeed    uint64 //InjectSpeed：该变量似乎代表交易注入区块链网络的速度

	// used in transaction relaying, useless in brokerchain mechanism
	MaxRelayBlockSize uint64 //MaxRelayBlockSize：该变量似乎代表中继交易的最大块大小
}

var (
	DeciderShard    = uint64(0xffffffff)                                                          //该变量被分配了最大可能uint64值（0xffffffff），它可以用作代码中的哨兵或特殊值来指示特定分片，可能是决策分片。
	Init_Balance, _ = new(big.Int).SetString("100000000000000000000000000000000000000000000", 10) //该变量可能代表分配给区块链模拟中的账户或参与者的初始余额或代币金额
	IPmap_nodeTable = make(map[uint64]map[uint64]string)                                          //IPmap_nodeTable：该变量代表区块链模拟中节点的 IP 地址映射，IPmap_nodeTable它是一个映射，其中键是uint64值，
	// 并且每个键都映射到另一个映射。内部映射又使用uint64键来映射到string值
	//外部映射允许您通过某些唯一标识符（例如节点的 ID）对节点进行索引，内部映射用于存储与这些标识符对应的节点的 IP 地址。
	CommitteeMethod  = []string{"CLPA_Broker", "CLPA", "Broker", "Relay"}                                 //该变量似乎代表委员会方法
	MeasureBrokerMod = []string{"TPS_Broker", "TCL_Broker", "CrossTxRate_Broker", "TxNumberCount_Broker"} //包含特定于“代理”机制的各种测量方法。这些方法似乎是用于测量区块链模拟中的各种度量的方法
	MeasureRelayMod  = []string{"TPS_Relay", "TCL_Relay", "CrossTxRate_Relay", "TxNumberCount_Relay"}     //包含特定于“Relay”机制的各种测量方法
)
