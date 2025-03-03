package params

var (
	Block_Interval      = 5000   // generate new block interval
	MaxBlockSize_global = 2000   // the block contains the maximum number of transactions
	InjectSpeed         = 2000   // the transaction inject speed
	TotalDataSize       = 100000 // the total number of txs
	BatchSize           = 16000  // supervisor read a batch of txs then send them, it should be larger than inject speed
	BrokerNum           = 10
	NodesInShard        = 4
	ShardNum            = 4
	DataWrite_path      = "./result/"                                                                                    // measurement data result output path
	LogWrite_path       = "./log"                                                                                        // log output path
	SupervisorAddr      = "127.0.0.1:18800"                                                                              //supervisor ip address
	FileInput           = `D:\\GolandProjects\\2000000to2999999_BlockTransaction\\2000000to2999999_BlockTransaction.csv` //the raw BlockTransaction data path
)

/*这些是 Go 代码中定义的全局变量。它们似乎是区块链仿真或模拟中使用的配置参数和常量。
Block_Interval：该变量设置为 5000，表示在区块链模拟中生成新块的时间间隔（以某个时间单位为单位）。
MaxBlockSize_global：该变量设置为 2000，表示模拟中一个块可以包含的最大交易数量。
InjectSpeed：该变量设置为 2000，代表交易注入区块链网络的速度。
TotalDataSize：此变量设置为 100000，似乎表示模拟将处理的事务总数或数据大小。
BatchSize：该变量设置为 16000，可能表示主管节点读取和发送的一批交易的大小。它应该大于注入速度，表明它控制一次处理和发送的交易数量。
BrokerNum：此变量设置为 10，可能代表模拟中代理或中介组件的数量。
NodesInShard：此变量设置为 4，表示区块链网络中每个分片内的节点数。
ShardNum：此变量设置为 4，可能代表区块链网络中的分片总数。
DataWrite_path：该变量设置为“./result/”，表示测量数据结果的输出路径。
LogWrite_path：该变量设置为“./log”，表示日志文件所在的位置。
SupervisorAddr：该变量设置为“127.0.0.1:18800”，似乎代表模拟中管理节点的 IP 地址和端口。
FileInput：此变量设置为特定文件路径 ( ../2000000to2999999_Block_Info.csv)，可能代表模拟的某些原始数据输入的路径。
这些变量为您的区块链模拟提供必要的配置和参数，允许您控制模拟的各个方面，例如块生成间隔、事务批量大小、网络拓扑和数据路径。它们通常在整个代码中使用来配置模拟的行为。*/
