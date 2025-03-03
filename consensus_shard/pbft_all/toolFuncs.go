package pbft_all

import (
	"blockEmulator/message"
	"blockEmulator/params"
	"blockEmulator/shard"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"strconv"
)

// 设置2d地图，仅适用于pbft地图，如果第一个参数为true，则设置cntPrepareConfirm地图，
// 否则，将设置 cntCommitConfirm 映射
func (p *PbftConsensusNode) set2DMap(isPrePareConfirm bool, key string, val *shard.Node) { //set2DMap方法用于设置2d地图，仅适用于pbft地图
	if isPrePareConfirm {
		if _, ok := p.cntPrepareConfirm[key]; !ok {
			p.cntPrepareConfirm[key] = make(map[*shard.Node]bool)
		}
		p.cntPrepareConfirm[key][val] = true
	} else {
		if _, ok := p.cntCommitConfirm[key]; !ok {
			p.cntCommitConfirm[key] = make(map[*shard.Node]bool)
		}
		p.cntCommitConfirm[key][val] = true
	}
}

// 获取分片中的邻居节点
func (p *PbftConsensusNode) getNeighborNodes() []string { //getNeighborNodes方法用于获取分片中的邻居节点
	receiverNodes := make([]string, 0)             //创建一个string类型的切片
	for _, ip := range p.ip_nodeTable[p.ShardID] { //遍历ip_nodeTable中的节点
		receiverNodes = append(receiverNodes, ip) //将节点添加到receiverNodes中
	}
	return receiverNodes //返回receiverNodes
}

func (p *PbftConsensusNode) writeCSVline(str []string) { //writeCSVline方法用于将数据写入CSV文件
	dirpath := params.DataWrite_path + "pbft_" + strconv.Itoa(int(p.pbftChainConfig.ShardNums)) //设置路径
	err := os.MkdirAll(dirpath, os.ModePerm)                                                    //创建目录
	if err != nil {
		log.Panic(err)
	}

	targetPath := dirpath + "/Shard" + strconv.Itoa(int(p.ShardID)) + strconv.Itoa(int(p.pbftChainConfig.ShardNums)) + ".csv" //设置目标路径
	f, err := os.Open(targetPath)
	if err != nil && os.IsNotExist(err) {
		file, er := os.Create(targetPath)
		if er != nil {
			panic(er)
		}
		defer file.Close()

		w := csv.NewWriter(file)
		title := []string{"txpool size", "tx", "ctx"}
		w.Write(title)
		w.Flush()
		w.Write(str)
		w.Flush()
	} else {
		file, err := os.OpenFile(targetPath, os.O_APPEND|os.O_RDWR, 0666)

		if err != nil {
			log.Panic(err)
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		err = writer.Write(str)
		if err != nil {
			log.Panic()
		}
		writer.Flush()
	}

	f.Close()
}

// 获取请求的摘要
func getDigest(r *message.Request) []byte {
	b, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	hash := sha256.Sum256(b)
	return hash[:]
}
