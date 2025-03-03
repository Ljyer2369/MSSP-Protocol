// storage is a key-value database and its interfaces indeed
// 块的信息将被保存在storage中，storage是一个键值数据库，其接口实际上是bolt数据库的接口
// 在Go（Golang）中，“存储”通常指的是键值数据库或数据存储系统。要在 Go 中使用键值数据库，您通常使用一个接口来定义可以对数据库执行的基本操作
// 什么是键值数据库？
// 键值数据库（Key-Value Database）是一种数据库类型，它以简单而高效的方式存储数据。在键值数据库中，数据以键值对（key-value pair）的形式进行存储和检索。
// 每个键都是唯一的，而且与一个值相关联。这种数据库类型通常用于需要快速存储和检索数据的场景，特别是在分布式系统、缓存、会话存储和存储配置等应用中非常有用。
// LevelDB：Google开发的轻量级键值数据库，用于本地存储。
package storage

import (
	"blockEmulator/core"
	"blockEmulator/params"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/boltdb/bolt"
)

type Storage struct { //Storage结构包含用于区块链存储的各种配置参数
	dbFilePath            string   // 这是一个字符串，表示数据库文件的路径，它指定了区块链数据库在文件系统上的存储位置。
	blockBucket           string   // 代表 Bolt 数据库中存储桶的名称。在 Bolt 中，存储桶是一种键值存储，允许您分层组织数据。
	blockHeaderBucket     string   // Bolt 数据库中另一个存储桶的名称。它可以专门用于存储块头数据。
	newestBlockHashBucket string   // 代表 Bolt 数据库中另一个存储桶的名称。该存储桶可用于存储区块链中最新（最新）块的哈希值。
	DataBase              *bolt.DB //该字段是指向 Bolt 数据库的指针。它用于保存对区块链系统将用于存储的实际 Bolt 数据库实例的引用。
}

//存储桶是一种数据结构，用于在Bolt数据库中组织和存储数据。以下是有关Bolt数据库中存储桶的一些重要信息：
//存储桶是命名空间：存储桶充当数据的命名空间，允许你将相关数据按照一定的逻辑分组在一起。你可以给每个存储桶分配一个唯一的名称，以便将相关数据项组织在一起，便于检索和管理。
//存储桶支持嵌套：Bolt数据库允许你在存储桶内部创建嵌套的存储桶，从而创建多层次的数据结构。这种嵌套结构非常有用，可以帮助你更好地组织和查询数据。
//存储桶是键值对的容器：存储桶内部包含键值对，你可以使用键来检索相关的值。每个键都是唯一的，但在不同的存储桶之间可以使用相同的键。这意味着你可以在不同的存储桶中使用相同的键，而它们不会相互冲突。

// new a storage, build a bolt datase
func NewStorage(cc *params.ChainConfig) *Storage { //NewStorage()函数用于创建一个新的Storage结构。它需要一个参数，即指向params.ChainConfig结构的指针。该结构包含用于区块链存储的各种配置参数
	_, errStat := os.Stat("./record") //检查文件系统上是否存在名为record的目录。
	if os.IsNotExist(errStat) {       //如果record目录不存在，则创建该目录
		errMkdir := os.Mkdir("./record", os.ModePerm)
		if errMkdir != nil { //如果创建目录失败，则打印错误消息并退出程序
			log.Panic(errMkdir)
		}
	} else if errStat != nil { //如果检查目录状态失败，则打印错误消息并退出程序
		log.Panic(errStat)
	}

	s := &Storage{ //创建一个指向Storage结构的指针并使用特定值对其进行初始化
		dbFilePath:            "./record/" + strconv.FormatUint(cc.ShardID, 10) + "_" + strconv.FormatUint(cc.NodeID, 10) + "_database", //数据库文件路径是根据cc（ChainConfig）参数中的值构建的，其中似乎包括ShardID和NodeID。这将为 ShardID 和 NodeID 的每个组合创建一个唯一的数据库文件路径。数据库文件路径构建在“record”目录中。
		blockBucket:           "block",                                                                                                  //该变量似乎代表 Bolt 数据库中存储桶的名称。在 Bolt 中，存储桶是一种键值存储，允许您分层组织数据。
		blockHeaderBucket:     "blockHeader",
		newestBlockHashBucket: "newestBlockHash",
	}

	db, err := bolt.Open(s.dbFilePath, 0600, nil) //它使用 Bolt.Open 打开 Bolt 数据库文件。 0600是文件模式，nil是可选选项。如果在打开数据库文件期间出现错误，它会出现紧急情况并记录错误。
	if err != nil {
		log.Panic(err)
	}

	// create buckets
	db.Update(func(tx *bolt.Tx) error {
		// 在 db.Update 调用中，它使用 CreateBucketIfNotExists 在 Bolt 数据库中创建必要的存储桶（键值存储）。这些桶是“block”、“blockHeader”和“newestBlockHash”。如果在创建存储桶期间发生任何错误，它会出现恐慌并记录错误
		_, err := tx.CreateBucketIfNotExists([]byte(s.blockBucket))
		if err != nil {
			log.Panic("create blocksBucket failed")
		}

		_, err = tx.CreateBucketIfNotExists([]byte(s.blockHeaderBucket))
		if err != nil {
			log.Panic("create blockHeaderBucket failed")
		}

		_, err = tx.CreateBucketIfNotExists([]byte(s.newestBlockHashBucket))
		if err != nil {
			log.Panic("create newestBlockHashBucket failed")
		}

		return nil
	})
	s.DataBase = db //将指向 Bolt 数据库的指针保存到 Storage 结构中的 DataBase 字段中
	return s        //返回指向 Storage 结构的指针
}

//上面的函数有效地设置和初始化区块链系统的存储组件，创建必要的数据库文件、存储桶和相关结构来管理区块链数据。

// update the newest block in the database
func (s *Storage) UpdateNewestBlock(newestbhash []byte) { //UpdateNewestBlock()函数用于更新数据库中的最新块。它需要一个参数，即最新块的哈希值。该函数将最新块的哈希值写入数据库中的存储桶“newestBlockHash”中。
	err := s.DataBase.Update(func(tx *bolt.Tx) error { //它使用 Bolt.Update 打开 Bolt 数据库事务。如果在打开事务期间出现错误，它会出现紧急情况并记录错误。
		nbhBucket := tx.Bucket([]byte(s.newestBlockHashBucket)) //它使用 tx.Bucket 获取存储桶“newestBlockHash”的引用。如果在获取存储桶期间出现错误，它会出现紧急情况并记录错误。
		// the bucket has the only key "OnlyNewestBlock"
		err := nbhBucket.Put([]byte("OnlyNewestBlock"), newestbhash) //该函数将最新块的哈希值写入数据库中的存储桶“newestBlockHash”中。
		if err != nil {
			log.Panic()
		}
		return nil
	})
	if err != nil {
		log.Panic()
	}
	fmt.Println("The newest block is updated")
}

// add a blockheader into the database
func (s *Storage) AddBlockHeader(blockhash []byte, bh *core.BlockHeader) { //AddBlockHeader()函数用于将块头添加到数据库中。它需要两个参数，即块哈希和块头。该函数将块头写入数据库中的存储桶“blockHeader”中。
	err := s.DataBase.Update(func(tx *bolt.Tx) error {
		bhbucket := tx.Bucket([]byte(s.blockHeaderBucket))
		err := bhbucket.Put(blockhash, bh.Encode())
		if err != nil {
			log.Panic()
		}
		return nil
	})
	if err != nil {
		log.Panic()
	}
}

// add a block into the database
func (s *Storage) AddBlock(b *core.Block) { //AddBlock()函数用于将块添加到数据库中。它需要一个参数，即块。该函数将块写入数据库中的存储桶“block”中。
	err := s.DataBase.Update(func(tx *bolt.Tx) error {
		bbucket := tx.Bucket([]byte(s.blockBucket))
		err := bbucket.Put(b.Hash, b.Encode())
		if err != nil {
			log.Panic()
		}
		return nil
	})
	if err != nil {
		log.Panic()
	}
	s.AddBlockHeader(b.Hash, b.Header)
	s.UpdateNewestBlock(b.Hash)
	fmt.Println("Block is added")
}

// read a blockheader from the database
func (s *Storage) GetBlockHeader(bhash []byte) (*core.BlockHeader, error) { //GetBlockHeader()函数用于从数据库中读取块头。它需要一个参数，即块哈希。该函数从数据库中的存储桶“blockHeader”中读取块头。
	var res *core.BlockHeader
	err := s.DataBase.View(func(tx *bolt.Tx) error {
		bhbucket := tx.Bucket([]byte(s.blockHeaderBucket))
		bh_encoded := bhbucket.Get(bhash)
		if bh_encoded == nil {
			return errors.New("the block is not existed")
		} //如果块头不存在，则返回错误消息
		res = core.DecodeBH(bh_encoded) //如果块头存在，则使用core.DecodeBH()函数对其进行解码
		return nil
	})
	return res, err
}

// read a block from the database
func (s *Storage) GetBlock(bhash []byte) (*core.Block, error) {
	var res *core.Block
	err := s.DataBase.View(func(tx *bolt.Tx) error {
		bbucket := tx.Bucket([]byte(s.blockBucket))
		b_encoded := bbucket.Get(bhash)
		if b_encoded == nil {
			return errors.New("the block is not existed")
		}
		res = core.DecodeB(b_encoded)
		return nil
	})
	return res, err
}

// read the Newest block hash
func (s *Storage) GetNewestBlockHash() ([]byte, error) { //GetNewestBlockHash()函数用于从数据库中读取最新块的哈希值。它不需要参数。该函数从数据库中的存储桶“newestBlockHash”中读取最新块的哈希值。
	var nhb []byte
	err := s.DataBase.View(func(tx *bolt.Tx) error {
		bhbucket := tx.Bucket([]byte(s.newestBlockHashBucket))
		// the bucket has the only key "OnlyNewestBlock"
		nhb = bhbucket.Get([]byte("OnlyNewestBlock"))
		if nhb == nil {
			return errors.New("cannot find the newest block hash")
		}
		return nil
	})
	return nhb, err
}
