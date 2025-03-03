package test

import (
	"blockEmulator/partition"
	"encoding/csv"
	"io"
	"log"
	"os"
	"testing"
)

// CLPA测试函数初始化一个CLPAState对象，处理包含交易数据的 CSV 文件，并打印结果。
func TestCLPA(t *testing.T) {
	k := new(partition.CLPAState) //创建了一个 CLPAState 对象 k
	k.Init_CLPAState(0.5, 100, 4) //初始化一个 CLPAState 对象，设置了权重惩罚、最大迭代次数和分片数目

	txfile, err := os.Open("../0to999999_BlockTransaction.csv") //打开包含交易数据的 CSV 文件
	if err != nil {
		log.Panic(err)
	}

	defer txfile.Close()
	reader := csv.NewReader(txfile) //创建了一个 CSV 读取器，以从打开的 CSV 文件中读取数据
	datanum := 0                    //datanum 变量，它是一个整数，用于记录读取的数据条数。在这里，我们将其初始化为 0
	reader.Read()                   //调用了 Read 方法，以跳过 CSV 文件的第一行，该行包含了标题信息
	for {                           //通过一个 for 循环，你读取了 CSV 文件中的数据，并将其添加到 CLPAState 对象 k 中
		data, err := reader.Read()              //调用了 Read 方法，以读取 CSV 文件中的一行数据。
		if err == io.EOF || datanum == 200000 { //如果读取到了文件末尾，或者读取的数据条数达到了 200000 条，就退出循环
			break
		}
		if err != nil { //如果在读取数据时出现了错误，就打印错误信息并退出程序
			log.Panic(err)
		}
		if data[6] == "0" && data[7] == "0" && len(data[3]) > 16 && len(data[4]) > 16 && data[3] != data[4] { //如果交易的状态为 0，就将交易的发送方和接收方添加到 CLPAState 对象 k 中
			s := partition.Vertex{ //交易的发送方
				Addr: data[3][2:], //交易发送方的地址
			}
			r := partition.Vertex{ //交易的接收方
				Addr: data[4][2:],
			}
			k.AddEdge(s, r) //将这两个表示交易发送方和接收方的节点添加到 k 对象中，并在它们之间创建一条边。这表示在 CLPAState 对象 k 中，存在一条边，连接着发送方和接收方
			datanum++       //增加 datanum 计数器以跟踪已处理记录的数量
		}
	}

	k.CLPA_Partition() //调用了 CLPA_Partition 方法，以执行 CLPA 算法

	print(k.CrossShardEdgeNum) //它打印 k.CrossShardEdgeNum 的值，该值表示跨分片交易的数量
}

/*运行结果
=== RUN   TestCLPA //测试函数 TestCLPA 即将运行
153974  //表示处理的数据行数
0 has vertexs: 1823
1 has vertexs: 5122
2 has vertexs: 3071
3 has vertexs: 3090
//上面四行表示每个分片（标记为0、1、2、3）的顶点数量。这些分片是根据 CLPA 算法创建的
65635 //表示分片之间的交叉边数量
65635--- PASS: TestCLPA (7.95s) //这是测试函数运行的总结。PASS 表示测试成功，而 7.95s 表示测试运行的时间
PASS

//测试函数 TestCLPA 运行了成功，输出了一些统计数据和结果，包括每个分片的顶点数量以及交叉边的数量。
//这些结果表明 CLPA 分区算法已成功创建了分区图，并计算出相应的分片信息。
*/
