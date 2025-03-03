package query

import (
	"blockEmulator/core"
	"blockEmulator/params"
	"blockEmulator/storage"
	"github.com/boltdb/bolt"
)

func initStorage(ShardID, NodeID uint64) *storage.Storage { //initStorage函数用于初始化存储对象，该对象用于管理区块链的数据
	pcc := &params.ChainConfig{ //创建链配置：使用 params.ChainConfig 结构创建链配置 pcc，该结构包含了区块链的基本信息，如分片总数、节点总数、创世区块等。
		ChainID: ShardID,
		NodeID:  NodeID,
		ShardID: ShardID,
	}
	return storage.NewStorage(pcc) //创建存储：使用 storage.NewStorage 函数创建存储，该函数需要一个参数：pcc（类型为 *params.ChainConfig）：链配置。
}

func QueryBlocks(ShardID, NodeID uint64) []*core.Block { //QueryBlocks函数用于查询区块链中的所有区块
	db := initStorage(ShardID, NodeID).DataBase //初始化存储 (db)：通过调用 initStorage 函数初始化一个存储对象，该对象用于管理区块链的数据。函数的参数包括分片ID (ShardID) 和节点ID (NodeID)，以确定要查询的特定分片和节点的数据。
	defer db.Close()
	blocks := make([]*core.Block, 0)          //创建区块切片：创建一个区块切片 blocks，用于存储查询到的所有区块。
	err1 := db.View(func(tx *bolt.Tx) error { //执行数据库查询：通过 db.View 函数对数据库进行只读事务的查询。在这个只读事务中，查询区块数据。
		bbucket := tx.Bucket([]byte("block"))               //获取区块存储桶：通过 tx.Bucket([]byte("block")) 获取区块的bucket，该bucket用于存储区块数据。
		if err := bbucket.ForEach(func(k, v []byte) error { //遍历区块数据：对区块数据进行遍历，通过查询区块的 bucket，并使用 bbucket.ForEach 遍历其中的键值对。对于每个键值对，执行以下操作：
			res := core.DecodeB(v)       //解码区块：使用 core.DecodeB(v) 函数将查询到的区块数据 v 解码为 core.Block 结构，将解码后的区块存储在 res 变量中。
			blocks = append(blocks, res) //将解码后的区块添加到 blocks 切片中。
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err1 != nil {
		err1.Error()
	}
	return blocks //返回存储了查询到的所有区块数据的 blocks 切片
}

func QueryBlock(ShardID, NodeID, Number uint64) *core.Block {
	//QueryBlock函数用于查询区块链中的特定区块，通过提供区块的编号 (Number) 参数，可以获取特定区块的详细信息。
	db := initStorage(ShardID, NodeID).DataBase //初始化存储 (db)：通过调用 initStorage 函数初始化一个存储对象，该对象用于管理区块链的数据。函数的参数包括分片ID (ShardID) 和节点ID (NodeID)，以确定要查询的特定分片和节点的数据
	defer db.Close()
	block := new(core.Block) //创建一个新的 core.Block 对象，用于存储查询到的特定区块
	err1 := db.View(func(tx *bolt.Tx) error {
		bbucket := tx.Bucket([]byte("block"))
		if err := bbucket.ForEach(func(k, v []byte) error {
			res := core.DecodeB(v)
			if res.Header.Number == Number { //检查区块编号：检查解码后的区块的区块编号 (res.Header.Number) 是否与目标区块编号 (Number) 匹配。
				block = res //如果匹配，将查询到的区块赋值给 block 对象
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err1 != nil {
		err1.Error()
	}
	return block
}

func QueryNewestBlock(ShardID, NodeID uint64) *core.Block { //QueryNewestBlock函数用于查询区块链中的最新区块
	storage := initStorage(ShardID, NodeID)
	defer storage.DataBase.Close()
	hash, _ := storage.GetNewestBlockHash() //获取最新区块的哈希值：使用 storage.GetNewestBlockHash 函数获取最新区块的哈希值，并将其存储在 hash 中。
	block, _ := storage.GetBlock(hash)      //获取最新区块：使用 storage.GetBlock(hash) 获取包含最新区块信息的块对象，存储在 block 中。
	return block
}

func QueryBlockTxs(ShardID, NodeID, Number uint64) []*core.Transaction { //QueryBlockTxs函数用于查询区块中的交易
	db := initStorage(ShardID, NodeID).DataBase
	defer db.Close()
	block := new(core.Block) //创建一个新的 core.Block 对象，用于存储查询到的特定区块。
	err1 := db.View(func(tx *bolt.Tx) error {
		bbucket := tx.Bucket([]byte("block"))
		if err := bbucket.ForEach(func(k, v []byte) error {
			res := core.DecodeB(v)
			if res.Header.Number == Number { //检查解码后的区块的区块编号 (res.Header.Number) 是否与目标区块编号 (Number) 匹配
				block = res
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err1 != nil {
		err1.Error()
	}
	return block.Body //返回区块中的交易：返回存储了查询到的特定区块中的交易数据的 block.Body，它是一个 []*core.Transaction 类型的切片，包含了区块中的所有交易。
}
