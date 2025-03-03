package pbft_log

import (
	"blockEmulator/params"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type PbftLog struct { //PbftLog结构包含区块链日志的各种配置参数
	Plog *log.Logger //用于记录日志。它是一个记录器，允许您将消息记录到各种输出源。
}

func NewPbftLog(sid, nid uint64) *PbftLog { //构造函数，用于创建和初始化PbftLog结构体的实例。它需要两个参数： sid（类型为uint64）：这是表示分片 ID 的参数。 nid（类型uint64）：这是表示节点 ID 的参数。
	pfx := fmt.Sprintf("S%dN%d: ", sid, nid) //使用sid和nid参数构建一个前缀字符串。这个前缀字符串将在日志消息中使用，以帮助您区分不同的节点和分片。
	writer1 := os.Stdout                     //创建一个指向标准输出的指针。这是一个用于记录日志的输出源。

	dirpath := params.LogWrite_path + "/S" + strconv.Itoa(int(sid)) //构建一个目录路径，用于存储日志文件。该目录路径是根据sid参数构建的，其中似乎包括ShardID。这将为每个ShardID创建一个唯一的目录路径。
	err := os.MkdirAll(dirpath, os.ModePerm)                        //使用os.MkdirAll创建目录。如果在创建目录期间出现错误，它会出现紧急情况并记录错误。
	if err != nil {
		log.Panic(err)
	}
	writer2, err := os.OpenFile(dirpath+"/N"+strconv.Itoa(int(nid))+".log", os.O_WRONLY|os.O_CREATE, 0755) //创建一个指向日志文件的指针。该日志文件是根据sid和nid参数构建的，其中似乎包括ShardID和NodeID。这将为ShardID和NodeID的每个组合创建一个唯一的日志文件。
	if err != nil {
		log.Panic(err)
	}
	pl := log.New(io.MultiWriter(writer1, writer2), pfx, log.Lshortfile|log.Ldate|log.Ltime) //使用log.New创建一个记录器。它使用writer1和writer2作为输出源，使用pfx作为前缀字符串，并使用log.Lshortfile、log.Ldate和log.Ltime作为标志。这些标志将在日志消息中使用，以帮助您区分不同的节点和分片。
	fmt.Println()

	return &PbftLog{ //最后，构造函数返回 PbftLog 结构的新实例，其中 Plog 字段使用新创建的记录器进行初始化。
		Plog: pl,
	}
}

//此代码为 PBFT（实用拜占庭容错）共识算法设置一个记录器，允许将日志消息写入控制台以及特定于给定分片和节点 ID 的日志文件。
//它使用 log.New 函数创建一个记录器。它使用 io.MultiWriter 作为输出源，该输出源将消息写入标准输出和日志文件。它使用 pfx 作为前缀字符串，并使用 log.Lshortfile、log.Ldate 和 log.Ltime 作为标志。这些标志将在日志消息中使用，以帮助您区分不同的节点和分片。
