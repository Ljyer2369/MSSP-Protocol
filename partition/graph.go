// 图的相关操作，该数据结构用于描述区块链网络中的交易集合
package partition

// 图中的结点，即区块链网络中参与交易的账户
type Vertex struct {
	Addr string // 账户地址
	// 其他属性待补充
	//Addr它有一个用于存储帐户地址的字段。
	//它可能还有其他属性，尽管它们被标记为“待补充”（待补充），表明可以添加其他属性。
	Location string //地理位置信息
}

// 描述当前区块链交易集合的图，代表区块链中整个交易图
type Graph struct {
	VertexSet map[Vertex]bool     // 节点集合，其实是 set
	EdgeSet   map[Vertex][]Vertex // 存储顶点之间的邻接信息的映射。表示记录节点与节点间是否存在交易，它是作为邻接列表实现的
	//map[Vertex][]Vertex是一个具有 Vertex 类型的键和 []Vertex 类型的值的映射，[]Vertex是 Vertex 类型的切片。
	//[]Vertex表示map中每个key关联的值是Vertex的切片。 Go 中的切片是动态大小的、有序的元素集合，在这种情况下，映射中的每个键都可以与多个顶点值关联。
	// 因此，该数据结构用于表示 Vertex 对象之间的关系，允许每个 Vertex 与多个其他 Vertex 实例关联。
	//它通常用于实现图的邻接列表，其中给定的顶点具有与其连接的其他顶点的列表。这对于表示各种类型网络中的关系非常有用，包括区块链网络，其中 Vertex 可能表示帐户或节点，而 Vertex 的切片表示它们之间的连接或关系。
	// lock      sync.RWMutex       // 锁，但是每个储存节点各自存储一份图，不需要此
	// 根据地理分片需要添加其他字段
	GeographicalConstraint float64 // 地理邻近度或约束的某种度量
}

// 创建节点，允许创建一个新的Vertex并使用地址对其进行初始化
func (v *Vertex) ConstructVertex(s string) {
	v.Addr = s
}

// 增加图中的点。此方法将一个顶点添加到图的 VertexSet（节点集合） 中。如果 VertexSet 为零，则首先对其进行初始化。
func (g *Graph) AddVertex(v Vertex) {
	if g.VertexSet == nil {
		g.VertexSet = make(map[Vertex]bool)
	}
	g.VertexSet[v] = true
}

// 增加图中的边。此方法在两个顶点之间添加一条边（事务）。如果图中不存在任一顶点，则首先添加它们
func (g *Graph) AddEdge(u, v Vertex) {
	// 如果没有点，则增加边，权恒定为 1
	if _, ok := g.VertexSet[u]; !ok {
		g.AddVertex(u)
	}
	if _, ok := g.VertexSet[v]; !ok {
		g.AddVertex(v)
	}
	if g.EdgeSet == nil {
		g.EdgeSet = make(map[Vertex][]Vertex)
	}
	// 无向图，使用双向边。假设该图表示无向图，因此当从 u 到 v 添加一条边时，也会从 v 到 u 添加一条对应的边。
	g.EdgeSet[u] = append(g.EdgeSet[u], v) //将顶点v添加到顶点u的邻接列表中
	g.EdgeSet[v] = append(g.EdgeSet[v], u) //将顶点u添加到顶点v的邻接列表中
}

// 复制图。此方法允许您创建图形的副本。它将源图的顶点和边复制到目标图
func (dst *Graph) CopyGraph(src Graph) {
	dst.VertexSet = make(map[Vertex]bool)
	for v := range src.VertexSet {
		dst.VertexSet[v] = true
	}
	if src.EdgeSet != nil {
		dst.EdgeSet = make(map[Vertex][]Vertex)
		for v := range src.VertexSet {
			dst.EdgeSet[v] = make([]Vertex, len(src.EdgeSet[v]))
			copy(dst.EdgeSet[v], src.EdgeSet[v])
		}
	}
}

// 输出图。此方法将图表的内容打印到控制台。它列出了每个顶点（帐户）及其相邻顶点，代表帐户之间的交易。
func (g Graph) PrintGraph() {
	for v := range g.VertexSet {
		print(v.Addr, " ")
		print("edge:")
		for _, u := range g.EdgeSet[v] {
			print(" ", u.Addr, "\t")
		}
		println()
	}
	println()
}
