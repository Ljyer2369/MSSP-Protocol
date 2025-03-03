// definition of node and shard

package shard

import (
	"fmt"
)

type Node struct { //Node结构包含用于区块链仿真或模拟的节点的各种配置参数
	NodeID  uint64 //NodeID：该变量似乎代表节点的 ID
	ShardID uint64 //ShardID：该变量似乎代表分片的 ID
	IPaddr  string //IPaddr：该变量似乎代表节点的 IP 地址
}

func (n *Node) PrintNode() { //函数PrintNode负责打印节点的各种配置参数
	v := []interface{}{ //构造一个空接口的切片，其中保存Node的NodeID、ShardID和IPaddr值
		n.NodeID,
		n.ShardID,
		n.IPaddr,
	}
	fmt.Printf("%v\n", v)
}
