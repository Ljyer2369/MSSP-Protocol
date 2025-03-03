package main

import (
	"blockEmulator/build"

	"github.com/spf13/pflag"
)

/*"blockEmulator/build"：导入与您正在使用的区块链模拟器或分布式系统相关的包。
"github.com/spf13/pflag"：导入pflag包，用于解析命令行参数。*/

var (
	shardNum int
	nodeNum  int
	shardID  int
	nodeID   int
	modID    int
	isClient bool
	isGen    bool
)

/*定义全局变量：
shardNum、nodeNum、shardID、nodeID、modID：这些变量保存系统的各种配置参数，例如分片数量、节点数量、分片和节点 ID 以及选择委员会方法 ID。
isClient和isGen：这些布尔变量指示节点是否是客户端或者是否要生成某种批处理文件。*/

func main() {
	pflag.IntVarP(&shardNum, "shardNum", "S", 2, "indicate that how many shards are deployed")
	/*使用了 Go 语言中的 pflag 包的功能来定义一个整数类型的命令行参数，并将其与一个全局变量 shardNum 关联起来。&shardNum：这是一个指向 shardNum 变量的指针。这表示命令行参数解析后的值将被存储在 shardNum 变量中。
	"shardNum"：这是命令行参数的名称，表示用户在命令行中使用 --shardNum 这个选项来设置 shardNum 变量的值。
	"S"：这是 shardNum 参数的短选项，表示用户也可以使用 -S 来设置 shardNum 变量的值，这是一种简化输入的方式。
	2：这是 shardNum 参数的默认值。如果用户没有在命令行中提供 --shardNum 或 -S 选项，那么 shardNum 变量将被设置为默认值 2。
	"indicate that how many shards are deployed"：这是参数的描述，用于解释参数的作用。这个描述将会在帮助信息中显示，以帮助用户了解如何使用这个参数。
	总之，这行代码的作用是定义一个命令行参数 shardNum，允许用户通过命令行选项 --shardNum 或 -S 来设置程序中的 shardNum 变量的值。如果用户没有提供命令行选项，那么 shardNum 变量将被设置为默认值 2。*/
	pflag.IntVarP(&nodeNum, "nodeNum", "N", 4, "indicate how many nodes of each shard are deployed")
	pflag.IntVarP(&shardID, "shardID", "s", 0, "id of the shard to which this node belongs, for example, 0")
	pflag.IntVarP(&nodeID, "nodeID", "n", 0, "id of this node, for example, 0")
	pflag.IntVarP(&modID, "modID", "m", 3, "choice Committee Method,for example, 0, [CLPA_Broker,CLPA,Broker,Relay] ")
	pflag.BoolVarP(&isClient, "client", "c", false, "whether this node is a client")
	pflag.BoolVarP(&isGen, "gen", "g", false, "generation bat")
	pflag.Parse()

	if isGen { //是否生成批处理文件
		build.GenerateBatFile(nodeNum, shardNum, modID) //传入参数：节点数量、分片数量、委员会方法 ID
		return
	}
	if isClient { //是否是客户端
		build.BuildSupervisor(uint64(nodeNum), uint64(shardNum), uint64(modID)) //传入参数：节点数量、分片数量、委员会方法 ID
	} else {
		build.BuildNewPbftNode(uint64(nodeID), uint64(nodeNum), uint64(shardID), uint64(shardNum), uint64(modID)) //传入参数：节点 ID、节点数量、分片 ID、分片数量、委员会方法 ID
	}
}
