package partition

import (
	"blockEmulator/utils"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"strconv"
)

// CLPA算法状态，state of constraint label propagation algorithm
type CLPAState struct { //CLPAState结构包含CLPA算法的各种信息
	NetGraph          Graph          // 需运行CLPA算法的图
	PartitionMap      map[Vertex]int // 记录分片信息的 map，某个节点属于哪个分片
	Edges2Shard       []int          // Shard 相邻接的边数，对应论文中的 total weight of edges associated with label k
	VertexsNumInShard []int          // 分片（Shard） 内节点的数目
	WeightPenalty     float64        // 权重惩罚，对应论文中的 beta
	MinEdges2Shard    int            // 最少的 Shard 邻接边数，最小的 total weight of edges associated with label k
	MaxIterations     int            // 最大迭代次数，constraint，对应论文中的\tau
	CrossShardEdgeNum int            // 跨分片边的总数
	ShardNum          int            // 分片数目
	GraphHash         []byte         // 图的哈希值
}

//CLPA 算法应用到分片区块链场景下时，顶点（Vertex）指的是账户（account），边（edge）指的是交易（transaction），
//重新调整图的过程其实就是重新调整账户在不同分片的过程

func (graph *CLPAState) Hash() []byte { //Hash方法返回图的哈希值
	hash := sha256.Sum256(graph.Encode())
	return hash[:]
}

func (graph *CLPAState) Encode() []byte { //Encode方法返回图的编码
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff) //使用gob包的编码器
	err := enc.Encode(graph)     //将图编码为字节流
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// 加入节点，需要将它默认归到一个分片中
func (cs *CLPAState) AddVertex(v Vertex) { //AddVertex方法将节点加入图中
	cs.NetGraph.AddVertex(v)                //调用Graph的AddVertex方法，将一个顶点添加到图的 VertexSet（节点集合） 中
	if val, ok := cs.PartitionMap[v]; !ok { //如果节点v不在分片中，则将其默认归到一个分片中
		cs.PartitionMap[v] = utils.Addr2Shard(v.Addr) //将节点v的地址转换为分片ID
	} else {
		cs.PartitionMap[v] = val //如果节点v已经在分片中，则不做处理
	}
	cs.VertexsNumInShard[cs.PartitionMap[v]] += 1 // 此处可以批处理完之后再修改 VertexsNumInShard 参数
	// 当然也可以不处理，因为 CLPA 算法运行前会更新最新的参数
}

// 加入边，需要将它的端点（如果不存在）默认归到一个分片中
func (cs *CLPAState) AddEdge(u, v Vertex) { //AddEdge方法在两个顶点之间添加一条边（事务）
	// 如果没有点，则增加边，权恒定为 1
	if _, ok := cs.NetGraph.VertexSet[u]; !ok { //如果节点u不在图中，则将其加入图中
		cs.AddVertex(u)
	}
	if _, ok := cs.NetGraph.VertexSet[v]; !ok {
		cs.AddVertex(v)
	}
	cs.NetGraph.AddEdge(u, v) //调用Graph的AddEdge方法，在两个顶点之间添加一条边（事务）
	// 可以批处理完之后再修改 Edges2Shard 等参数
	// 当然也可以不处理，因为 CLPA 算法运行前会更新最新的参数
}

// 复制CLPA状态
func (dst *CLPAState) CopyCLPA(src CLPAState) { //CopyCLPA方法允许您创建CLPA状态的副本。它将源CLPA状态的顶点和边复制到目标CLPA状态
	dst.NetGraph.CopyGraph(src.NetGraph)    //调用Graph的CopyGraph方法，复制图
	dst.PartitionMap = make(map[Vertex]int) //创建一个map，用于记录分片信息
	for v := range src.PartitionMap {       //遍历源CLPA状态的分片信息
		dst.PartitionMap[v] = src.PartitionMap[v] //将源CLPA状态的分片信息复制到目标CLPA状态
	}
	dst.Edges2Shard = make([]int, src.ShardNum)   //创建一个切片，用于记录分片相邻接的边数
	copy(dst.Edges2Shard, src.Edges2Shard)        //将源CLPA状态的分片相邻接的边数复制到目标CLPA状态
	dst.VertexsNumInShard = src.VertexsNumInShard //记录分片内节点的数目
	dst.WeightPenalty = src.WeightPenalty         //记录权重惩罚
	dst.MinEdges2Shard = src.MinEdges2Shard       //记录最少的分片邻接边数
	dst.MaxIterations = src.MaxIterations         //记录最大迭代次数
	dst.ShardNum = src.ShardNum                   //记录分片数目
}

// 输出CLPA
func (cs *CLPAState) PrintCLPA() {
	cs.NetGraph.PrintGraph()
	println(cs.MinEdges2Shard)
	for v, item := range cs.PartitionMap {
		print(v.Addr, " ", item, "\t")
	}
	for _, item := range cs.Edges2Shard {
		print(item, " ")
	}
	println()
}

// 根据当前划分，计算 Wk，即 Edges2Shard
func (cs *CLPAState) ComputeEdges2Shard() { //ComputeEdges2Shard方法用于计算Wk，即Edges2Shard
	cs.Edges2Shard = make([]int, cs.ShardNum)
	interEdge := make([]int, cs.ShardNum)
	cs.MinEdges2Shard = 0x7fffffff // INT_MAX

	for idx := 0; idx < cs.ShardNum; idx++ {
		cs.Edges2Shard[idx] = 0
		interEdge[idx] = 0
	}

	for v, lst := range cs.NetGraph.EdgeSet {
		// 获取节点 v 所属的shard
		vShard := cs.PartitionMap[v]
		for _, u := range lst {
			// 同上，获取节点 u 所属的shard
			uShard := cs.PartitionMap[u]
			if vShard != uShard {
				// 判断节点 v, u 不属于同一分片，则对应的 Edges2Shard 加一
				// 仅计算入度，这样不会重复计算
				cs.Edges2Shard[uShard] += 1
			} else {
				interEdge[uShard]++
			}
		}
	}

	cs.CrossShardEdgeNum = 0
	for _, val := range cs.Edges2Shard {
		cs.CrossShardEdgeNum += val
	}
	cs.CrossShardEdgeNum /= 2

	for idx := 0; idx < cs.ShardNum; idx++ {
		cs.Edges2Shard[idx] += interEdge[idx] / 2
	}
	// 修改 MinEdges2Shard, CrossShardEdgeNum
	for _, val := range cs.Edges2Shard {
		if cs.MinEdges2Shard > val {
			cs.MinEdges2Shard = val
		}
	}
}

// 在账户所属分片变动时，重新计算各个参数，faster
func (cs *CLPAState) changeShardRecompute(v Vertex, old int) {
	new := cs.PartitionMap[v]
	for _, u := range cs.NetGraph.EdgeSet[v] {
		neighborShard := cs.PartitionMap[u]
		if neighborShard != new && neighborShard != old {
			cs.Edges2Shard[new]++
			cs.Edges2Shard[old]--
		} else if neighborShard == new {
			cs.Edges2Shard[old]--
			cs.CrossShardEdgeNum--
		} else {
			cs.Edges2Shard[new]++
			cs.CrossShardEdgeNum++
		}
	}
	cs.MinEdges2Shard = 0x7ffffffff
	// 修改 MinEdges2Shard, CrossShardEdgeNum
	for _, val := range cs.Edges2Shard {
		if cs.MinEdges2Shard > val {
			cs.MinEdges2Shard = val
		}
	}
}

// 设置参数
func (cs *CLPAState) Init_CLPAState(wp float64, mIter, sn int) {
	cs.WeightPenalty = wp                           // 权重惩罚
	cs.MaxIterations = mIter                        // 最大迭代次数
	cs.ShardNum = sn                                // 分片数目
	cs.VertexsNumInShard = make([]int, cs.ShardNum) // 分片内节点数目
	cs.PartitionMap = make(map[Vertex]int)          // 节点所属分片
}

// 初始化划分，使用节点地址的尾数划分，应该保证初始化的时候不会出现空分片
func (cs *CLPAState) Init_Partition() { //Init_Partition方法用于初始化划分
	// 设置划分默认参数
	cs.VertexsNumInShard = make([]int, cs.ShardNum) //创建一个切片，用于记录分片内节点的数目
	cs.PartitionMap = make(map[Vertex]int)          //创建一个map，用于记录分片信息
	for v := range cs.NetGraph.VertexSet {          //遍历图中的节点
		var va = v.Addr[len(v.Addr)-8:]          //获取节点v的地址的尾数
		num, err := strconv.ParseInt(va, 16, 64) //将节点v的地址的尾数转换为整数
		if err != nil {
			log.Panic()
		}
		cs.PartitionMap[v] = int(num) % cs.ShardNum   //将节点v的地址的尾数对分片数目取余，得到节点v所属分片
		cs.VertexsNumInShard[cs.PartitionMap[v]] += 1 //将节点v所属分片的节点数目加一
	}
	cs.ComputeEdges2Shard() //调用该函数来计算分片之间的边，确定有多少条边连接不同分片中的顶点。结果存储在cs.CrossShardEdgeNum中，它表示连接不同分片的边数。
}

// 不会出现空分片的初始化划分
func (cs *CLPAState) Stable_Init_Partition() error { //Stable_Init_Partition方法用于初始化划分，不会出现空分片
	// 设置划分默认参数
	if cs.ShardNum > len(cs.NetGraph.VertexSet) { //如果分片数目大于图中的节点数目，则返回错误
		return errors.New("too many shards, number of shards should be less than nodes. ")
	}
	cs.VertexsNumInShard = make([]int, cs.ShardNum) //创建一个切片，用于记录分片内节点的数目
	cs.PartitionMap = make(map[Vertex]int)          //创建一个map，用于记录分片信息
	cnt := 0
	for v := range cs.NetGraph.VertexSet { //遍历图中的节点
		cs.PartitionMap[v] = int(cnt) % cs.ShardNum   //将节点v的地址的尾数对分片数目取余，得到节点v所属分片
		cs.VertexsNumInShard[cs.PartitionMap[v]] += 1 //将节点v所属分片的节点数目加一
		cnt++                                         //cnt用于记录已经分配的分片数目
	}
	cs.ComputeEdges2Shard() // 删掉会更快一点，但是这样方便输出（毕竟只执行一次Init，也快不了多少）
	return nil
}

// 计算 将节点 v 放入 uShard 所产生的 score
func (cs *CLPAState) getShard_score(v Vertex, uShard int) float64 { //getShard_score方法用于计算将节点v放入uShard所产生的score
	var score float64
	// 节点 v 的出度
	v_outdegree := len(cs.NetGraph.EdgeSet[v])
	// uShard 与节点 v 相连的边数
	Edgesto_uShard := 0
	for _, item := range cs.NetGraph.EdgeSet[v] {
		if cs.PartitionMap[item] == uShard {
			Edgesto_uShard += 1
		}
	}
	score = float64(Edgesto_uShard) / float64(v_outdegree) * (1 - cs.WeightPenalty*float64(cs.Edges2Shard[uShard])/float64(cs.MinEdges2Shard))
	return score
}

// CLPA 划分算法
func (cs *CLPAState) CLPA_Partition() (map[string]uint64, int) { //实现基于图的网络的 CLPA（约束标签传播算法）分区算法
	cs.ComputeEdges2Shard()                             //调用该函数来计算分片之间的边，确定有多少条边连接不同分片中的顶点。结果存储在cs.CrossShardEdgeNum中，它表示连接不同分片的边数。
	fmt.Println(cs.CrossShardEdgeNum)                   //打印连接不同分片的边数
	res := make(map[string]uint64)                      //创建一个map，用于记录节点所属分片
	updateTreshold := make(map[string]int)              //创建一个map，用于记录节点更新的次数
	for iter := 0; iter < cs.MaxIterations; iter += 1 { //进入一个控制 CLPA 算法迭代次数的循环。该循环最多运行 cs.MaxIterations 次。
		for v := range cs.NetGraph.VertexSet { //遍历图中的节点
			if updateTreshold[v.Addr] >= 50 { //如果节点更新的次数超过50次，则跳过该节点
				continue
			}
			neighborShardScore := make(map[int]float64)                         //创建一个map，用于记录邻居分片的分数
			max_score := -9999.0                                                //创建一个变量，用于记录最大分数
			vNowShard, max_scoreShard := cs.PartitionMap[v], cs.PartitionMap[v] //创建两个变量，分别用于记录节点当前所属分片和最大分数的分片
			for _, u := range cs.NetGraph.EdgeSet[v] {                          //遍历节点v的邻居
				uShard := cs.PartitionMap[u] //获取节点u所属分片.
				// 对于属于 uShard 的邻居，仅需计算一次
				if _, computed := neighborShardScore[uShard]; !computed { //如果节点u所属分片的分数没有被计算过，则计算节点u所属分片的分数
					neighborShardScore[uShard] = cs.getShard_score(v, uShard) //调用getShard_score方法，计算将节点v放入uShard所产生的score
					if max_score < neighborShardScore[uShard] {               //如果最大分数小于节点u所属分片的分数，则更新最大分数和最大分数的分片
						max_score = neighborShardScore[uShard]
						max_scoreShard = uShard
					}
				}
			}
			if vNowShard != max_scoreShard && cs.VertexsNumInShard[vNowShard] > 1 { //如果节点v当前所属分片不等于最大分数的分片且节点v当前所属分片的节点数目大于1
				cs.PartitionMap[v] = max_scoreShard
				res[v.Addr] = uint64(max_scoreShard)
				updateTreshold[v.Addr]++
				// 重新计算 VertexsNumInShard
				cs.VertexsNumInShard[vNowShard] -= 1
				cs.VertexsNumInShard[max_scoreShard] += 1
				// 重新计算Wk
				cs.changeShardRecompute(v, vNowShard)
			}
		}
	}
	for sid, n := range cs.VertexsNumInShard { //遍历分片内节点的数目
		fmt.Printf("%d has vertexs: %d\n", sid, n) //打印分片内节点的数目
	}

	cs.ComputeEdges2Shard() //调用该函数来计算分片之间的边，确定有多少条边连接不同分片中的顶点。结果存储在cs.CrossShardEdgeNum中，它表示连接不同分片的边数。
	fmt.Println(cs.CrossShardEdgeNum)
	return res, cs.CrossShardEdgeNum //返回节点所属分片和连接不同分片的边数
}

func (cs *CLPAState) EraseEdges() { //EraseEdges方法用于擦除边
	cs.NetGraph.EdgeSet = make(map[Vertex][]Vertex)
}
