package build

//三个函数分别用于创建和配置主管节点、创建和配置新的PBFT节点以及初始化和配置params.ChainConfig结构
import (
	"blockEmulator/consensus_shard/pbft_all"
	"blockEmulator/params"
	"blockEmulator/supervisor"
	"strconv"
	"time"
)

func initConfig(nid, nnm, sid, snm uint64) *params.ChainConfig { //函数initConfig负责初始化和配置params.ChainConfig结构，该结构可能包含用于区块链仿真或模拟的各种配置参数
	//它需要四个参数：节点ID、节点总数、分片ID和分片总数
	params.ShardNum = int(snm)         //将params里面的ShardNum变量设置为整数值snm。ShardNum代表区块链网络中的分片总数
	for i := uint64(0); i < snm; i++ { //负责初始化区块链模拟中节点的 IP 地址映射
		if _, ok := params.IPmap_nodeTable[i]; !ok { //
			params.IPmap_nodeTable[i] = make(map[uint64]string) //将params.IPmap_nodeTable[i]变量设置为map[uint64]string类型的值。
		}
		for j := uint64(0); j < nnm; j++ {
			params.IPmap_nodeTable[i][j] = "127.0.0.1:" + strconv.Itoa(28800+int(i)*100+int(j))
			//params.IPmap_nodeTable[i][j]：这部分代码使用两个键 i 和 j 访问多级映射 IPmap_nodeTable。
			//“127.0.0.1:”是一个字符串，表示前缀为“127.0.0.1:”的 IP 地址。字符串的这一部分是恒定的。
			//strconv.Itoa(28800+int(i)*100+int(j)) 用于将整数表达式转换为字符串。表达式28800+int(i)*100+int(j)根据i和j的值计算端口号，然后将其转换为字符串。计算涉及将 28800 添加到 i*100 和 j。
			//最终结果是一个类似“127.0.0.1:PORT_NUMBER”的字符串，其中PORT_NUMBER是根据i和j的值计算得出的。
		}
	}
	params.IPmap_nodeTable[params.DeciderShard] = make(map[uint64]string)  //将params.IPmap_nodeTable[params.DeciderShard]变量设置为map[uint64]string类型的值。这表明params.DeciderShard分片中的节点将被分配到map[uint64]string类型的变量中
	params.IPmap_nodeTable[params.DeciderShard][0] = params.SupervisorAddr //将主管分片的第一个节点的地址设置为params.SupervisorAddr变量的值。这表明决策分片中的第一个节点是主管节点
	params.NodesInShard = int(nnm)                                         //将每个分片中的节点总数设置为nnm
	params.ShardNum = int(snm)                                             //将分片总数设置为snm

	pcc := &params.ChainConfig{ //创建一个指向params.ChainConfig结构的指针并使用特定值对其进行初始化
		ChainID:        sid,
		NodeID:         nid,
		ShardID:        sid,
		Nodes_perShard: uint64(params.NodesInShard),
		ShardNums:      snm,
		BlockSize:      uint64(params.MaxBlockSize_global),
		BlockInterval:  uint64(params.Block_Interval),
		InjectSpeed:    uint64(params.InjectSpeed),
	}
	return pcc
}

func BuildSupervisor(nnm, snm, mod uint64) { //函数BuildSupervisor负责创建和配置主管节点，参数分别代表节点总数、分片总数和委员会方法
	var measureMod []string
	if mod == 0 || mod == 2 {
		measureMod = params.MeasureBrokerMod
	} else {
		measureMod = params.MeasureRelayMod
	}

	lsn := new(supervisor.Supervisor)                                                                                    //创建一个指向supervisor.Supervisor结构的指针
	lsn.NewSupervisor(params.SupervisorAddr, initConfig(123, nnm, 123, snm), params.CommitteeMethod[mod], measureMod...) //初始化主管节点
	time.Sleep(10000 * time.Millisecond)
	go lsn.SupervisorTxHandling()
	lsn.TcpListen()
}

func BuildNewPbftNode(nid, nnm, sid, snm, mod uint64) { //函数BuildNewPbftNode负责创建和配置新的PBFT节点，参数分别代表节点ID、节点总数、分片ID、分片总数和委员会方法
	worker := pbft_all.NewPbftNode(sid, nid, initConfig(nid, nnm, sid, snm), params.CommitteeMethod[mod])
	if nid == 0 {
		go worker.Propose()
		worker.TcpListen()
	} else {
		worker.TcpListen()
	}
}
