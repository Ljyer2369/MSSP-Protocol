package test

import (
	"blockEmulator/consensus_shard/pbft_all"
	"blockEmulator/params"
	"blockEmulator/supervisor"
	"strconv"
	"time"
)

// TEST case
//nid, _ := strconv.ParseUint(os.Args[1], 10, 64)
//nnm, _ := strconv.ParseUint(os.Args[2], 10, 64)
//sid, _ := strconv.ParseUint(os.Args[3], 10, 64)
//snm, _ := strconv.ParseUint(os.Args[4], 10, 64)
//test.TestPBFT(nid, nnm, sid, snm)

func TestPBFT(nid, nnm, sid, snm uint64) { //
	params.ShardNum = int(snm)         //分片数量
	for i := uint64(0); i < snm; i++ { //通过一个循环迭代，为每个分片（snm 指定的数量）创建网络连接的地址
		if _, ok := params.IPmap_nodeTable[i]; !ok { //IPmap_nodeTable：保存了每个分片中的节点的 IP 地址,检查 params.IPmap_nodeTable 中是否已经有了一个与分片 i 相关的映射,如果映射不存在，创建一个空映射。
			params.IPmap_nodeTable[i] = make(map[uint64]string)
		}
		for j := uint64(0); j < nnm; j++ { //通过另一个循环迭代，为每个分片中的节点（nnm 指定的数量）创建网络连接的地址
			params.IPmap_nodeTable[i][j] = "127.0.0.1:" + strconv.Itoa(8800+int(i)*100+int(j)) //使用 IP 地址 "127.0.0.1" 和端口号，创建一个节点的地址，将其添加到 params.IPmap_nodeTable 中
		}
	}
	params.IPmap_nodeTable[params.DeciderShard] = make(map[uint64]string) //为决策分片（params.DeciderShard 指定的分片）创建一个空映射。
	params.IPmap_nodeTable[params.DeciderShard][0] = "127.0.0.1:18800"    //为决策分片中的节点（节点ID为0）分配一个特定的地址。

	pcc := &params.ChainConfig{ //创建了一个 params.ChainConfig 结构，用于保存区块链的配置参数。这些参数包括链的ID、节点ID、分片ID、分片的数量、块大小、块间隔和注入速度等。
		ChainID:        sid,
		NodeID:         nid,
		ShardID:        sid,
		Nodes_perShard: uint64(params.NodesInShard),
		ShardNums:      snm,
		BlockSize:      uint64(params.MaxBlockSize_global),
		BlockInterval:  uint64(params.Block_Interval),
		InjectSpeed:    uint64(params.InjectSpeed),
	}

	if nid == 12345678 { //如果节点 ID 为 12345678，那么这个节点将作为监督节点运行
		lsn := new(supervisor.Supervisor)                                                                                            //创建了一个 supervisor.Supervisor 结构，用于监督区块链网络中的所有节点。这个结构包含了一些用于监督的方法，例如 SupervisorTxHandling() 和 TcpListen()。
		lsn.NewSupervisor("127.0.0.1:18800", pcc, "Relay", "TPS_Relay", "Latency_Relay", "CrossTxRate_Relay", "TxNumberCount_Relay") //使用 NewSupervisor() 方法，为监督节点指定了一些参数，包括节点的地址、区块链的配置参数、委员会方法 ID 和一些用于监督的指标。
		time.Sleep(10000 * time.Millisecond)                                                                                         //睡眠 10 秒钟，以便其他节点能够启动。
		go lsn.SupervisorTxHandling()                                                                                                //调用 SupervisorTxHandling() 方法，监督节点将开始监督区块链网络中的所有节点。
		lsn.TcpListen()                                                                                                              //调用 TcpListen() 方法，监督节点将开始监听网络连接。
		return                                                                                                                       //监督节点的工作完成后，程序将退出。
	}

	worker := pbft_all.NewPbftNode(sid, nid, pcc, "Relay") //创建一个 PBFT 节点实例 worker，传入节点ID、分片ID和配置信息。
	time.Sleep(5 * time.Second)                            //睡眠 5 秒钟，以便其他节点能够启动。
	if nid == 0 {                                          //如果节点 ID 为 0，那么这个节点将作为提议节点运行
		go worker.Propose() //调用 Propose() 方法，提议节点将开始提议区块。
		worker.TcpListen()  //调用 TcpListen() 方法，提议节点将开始监听网络连接。
	} else {
		worker.TcpListen() //如果节点ID不等于0，则只执行 worker.TcpListen()，即仅启动 PBFT 节点的 TCP 监听。
	}
}

//这个测试函数用于模拟 PBFT 网络中的不同节点行为，包括监视器、提案节点以及其他普通节点。根据节点ID的不同，执行不同的操作，以测试 PBFT 协议的各个部分的行为。
