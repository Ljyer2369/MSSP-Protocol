package query

import (
	"blockEmulator/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"
	"log"
	"strconv"
)

func QueryAccountState(ShardID, NodeID uint64, address string) *core.AccountState { //QueryAccountState函数用于查询账户状态
	fp := "./record/ldb/s" + strconv.FormatUint(ShardID, 10) + "/n" + strconv.FormatUint(NodeID, 10) //构建 LevelDB 数据库路径 (fp)：根据输入的 ShardID 和 NodeID 构建 LevelDB 数据库的文件路径 fp，该路径包括分片ID和节点ID的信息。
	db, _ := rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false)                               //打开 LevelDB 数据库：使用 rawdb.NewLevelDBDatabase 函数打开 LevelDB 数据库，并将返回的数据库实例存储在 db 中。
	triedb := trie.NewDatabaseWithConfig(db, &trie.Config{                                           //创建 trie 数据库：使用 trie.NewDatabaseWithConfig 函数创建 trie 数据库 triedb，该函数需要两个参数：db（类型为 rawdb.Database）：这是一个 LevelDB 数据库实例，该数据库使用 LevelDB 作为后端存储。config（类型为 *trie.Config）：这是一个 trie 配置实例，它包含 trie 数据库的配置信息。
		Cache:     0,    //缓存大小
		Preimages: true, //是否存储预图像
	})
	defer db.Close()                                                                    //确保 LevelDB 数据库在函数退出时被关闭
	storage := initStorage(ShardID, NodeID)                                             //初始化存储对象，该对象用于管理区块链的数据：使用 initStorage 函数初始化存储，该函数需要两个参数：ShardID（类型为 uint64）：分片ID。NodeID（类型为 uint64）：节点ID。
	curHash, err := storage.GetNewestBlockHash()                                        //获取最新区块的哈希值：使用 storage.GetNewestBlockHash 函数获取最新区块的哈希值，并将其存储在 curHash 中。
	curb, err := storage.GetBlock(curHash)                                              //获取当前区块：使用 storage.GetBlock(curHash) 获取包含最新区块信息的块对象，存储在 curb 中
	st, err := trie.New(trie.TrieID(common.BytesToHash(curb.Header.StateRoot)), triedb) //创建 Trie 树：使用区块中的状态根哈希构建 Trie 树，通过 trie.New 函数创建 Trie 树实例，并将根哈希以及 Trie 数据库传递给该函数。得到的 Trie 实例存储在 st 中
	if err != nil {
		log.Panic()
	}
	asenc, _ := st.Get([]byte(address)) //查询账户状态：通过 Trie 树 st 查询给定地址 address 对应的账户状态。这是通过调用 st.Get([]byte(address)) 完成的，其中 address 被转换成字芀数组作为键来检索对应的账户状态
	var state_a *core.AccountState
	state_a = core.DecodeAS(asenc) //解码账户状态：使用 core.DecodeAS(asenc) 函数将查询到的账户状态的字芀数据 asenc 解码为 core.AccountState 结构，将解码后的账户状态存储在 state_a 变量中。
	return state_a
}
