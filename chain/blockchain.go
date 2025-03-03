// Here the blockchain structrue is defined
// each node in this system will maintain a blockchain object.

package chain

import (
	"blockEmulator/core"
	"blockEmulator/params"
	"blockEmulator/storage"
	"blockEmulator/utils"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

type BlockChain struct { //该结构体用于表示和管理与区块链相关的各种组件和数据
	db ethdb.Database // 该字段代表一个LevelDB数据库，LevelDB是一种流行的键值存储系统，它用于存储与状态树相关的区块链数据。
	// 区块链的状态树是一个重要的数据结构，用于跟踪和记录区块链网络中的账户、合约和其它状态信息。其中根节点代表了整个区块链的初始状态，而每个子节点代表一个不同的账户或智能合约。一旦状态树中的状态被确定，就不可更改。新的状态将在后续区块中记录，而不是修改已经存在的状态。
	triedb       *trie.Database      // 该字段是指向 trie.Database 的指针。它用于管理和存储与区块链状态树相关的数据。区块链系统中经常使用Tries来有效地存储和检索与帐户和合约相关的数据。
	ChainConfig  *params.ChainConfig // 该字段是指向 params.ChainConfig 结构的指针，它有助于识别和配置区块链的各种参数和设置。它可用于指定区块链的特征，如共识规则、网络 ID 等
	CurrentBlock *core.Block         // 该字段保存对区块链中顶部块的引用。当前区块通常是区块链中的最新区块，此引用有助于跟踪区块链的状态。
	Storage      *storage.Storage    // 用于存储区块链的区块。它使用BoltDB（一种嵌入式键值数据库）来高效存储区块链数据。
	Txpool       *core.TxPool        // 交易池，交易池是待处理交易在包含到块中之前存储的地方。它是一个优先级队列，其中包含待处理交易。交易池的大小是有限的，因此它可以存储有限数量的交易。
	PartitionMap map[string]uint64   //这是一个包含由某种算法定义的分区图的映射。它用于协助区块链中的帐户分区。该map用于存储分区图。
	pmlock       sync.RWMutex        // 该字段是一个互斥锁，用于访问时进行读写锁定。它确保对分区图的访问同步，以防止并发修改导致问题。
}

//LevelDB：LevelDB通常用于本地数据存储，特别是在需要轻量级嵌入式数据库的情况下。它不限于与 Go 一起使用，并且有多种语言的实现。
//BoltDB：BoltDB 专为 Go 应用程序中的嵌入式使用而设计。它通常用于在 Go 程序中提供简单、高效和事务性的键值存储。BoltDB与Go生态的结合更加紧密。

func GetTxTreeRoot(txs []*core.Transaction) []byte { //该函数用于获取交易树的根。它接受一个交易数组作为参数，并返回一个字节数组，该字节数组代表交易树的根。
	// use a memory trie database to do this, instead of disk database
	triedb := trie.NewDatabase(rawdb.NewMemoryDatabase())
	transactionTree := trie.NewEmpty(triedb)
	for _, tx := range txs {
		transactionTree.Update(tx.TxHash, tx.Encode())
	}
	return transactionTree.Hash().Bytes()
}

// Write Partition Map
func (bc *BlockChain) Update_PartitionMap(key string, val uint64) { //该函数用于更新分区图。它接受一个字符串和一个uint64值作为参数，并将其添加到分区图中。
	bc.pmlock.Lock()
	defer bc.pmlock.Unlock()
	bc.PartitionMap[key] = val
}

// Get parition (if not exist, return default)
func (bc *BlockChain) Get_PartitionMap(key string) uint64 { //该函数用于获取分区图中的值。它接受一个字符串作为参数，并返回一个uint64值。
	bc.pmlock.RLock()
	defer bc.pmlock.RUnlock()
	if _, ok := bc.PartitionMap[key]; !ok {
		return uint64(utils.Addr2Shard(key))
	}
	return bc.PartitionMap[key]
}

// Send a transaction to the pool (need to decide which pool should be sended)
func (bc *BlockChain) SendTx2Pool(txs []*core.Transaction) { //该函数用于将交易发送到交易池。它接受一个交易数组作为参数，并将其添加到交易池中。
	bc.Txpool.AddTxs2Pool(txs)
}

// handle transactions and modify the status trie
func (bc *BlockChain) GetUpdateStatusTrie(txs []*core.Transaction) common.Hash { //该函数用于处理交易并修改状态树。它接受一个交易数组作为参数，并返回一个common.Hash值。
	fmt.Printf("The len of txs is %d\n", len(txs))
	// 空块（txs 长度为 0）条件
	if len(txs) == 0 {
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	}
	// build trie from the triedb (in disk)
	st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
	if err != nil {
		log.Panic(err)
	}
	cnt := 0
	//处理交易，这里忽略签名检查
	for i, tx := range txs { //遍历交易数组
		// fmt.Printf("tx %d: %s, %s\n", i, tx.Sender, tx.Recipient)
		// senderIn := false
		if !tx.Relayed && (bc.Get_PartitionMap(tx.Sender) == bc.ChainConfig.ShardID || tx.HasBroker) { //如果交易未中继且发送者在本分片中，则执行以下操作
			// senderIn = true
			// fmt.Printf("the sender %s is in this shard %d, \n", tx.Sender, bc.ChainConfig.ShardID)
			// modify local accountstate
			s_state_enc, _ := st.Get([]byte(tx.Sender)) //从状态树中获取发送者的状态
			var s_state *core.AccountState              //创建一个指向 AccountState 的指针
			if s_state_enc == nil {                     //如果状态不存在，则创建一个新的状态
				// fmt.Println("missing account SENDER, now adding account")
				ib := new(big.Int)
				ib.Add(ib, params.Init_Balance) //将初始余额添加到新状态中
				s_state = &core.AccountState{
					Nonce:   uint64(i),
					Balance: ib,
				}
			} else {
				s_state = core.DecodeAS(s_state_enc) //如果状态存在，则使用core.DecodeAS()函数对其进行解码
			}
			s_balance := s_state.Balance       //获取发送者的余额
			if s_balance.Cmp(tx.Value) == -1 { //如果余额小于交易金额，则打印错误消息并继续
				fmt.Printf("the balance is less than the transfer amount\n")
				continue
			}
			s_state.Deduct(tx.Value)                       //否则，减少发送者的余额
			st.Update([]byte(tx.Sender), s_state.Encode()) //更新状态树
			cnt++
		}
		// recipientIn := false
		if bc.Get_PartitionMap(tx.Recipient) == bc.ChainConfig.ShardID || tx.HasBroker { //如果接收者在本分片中，则执行以下操作
			// fmt.Printf("the recipient %s is in this shard %d, \n", tx.Recipient, bc.ChainConfig.ShardID)
			// recipientIn = true
			// modify local state
			r_state_enc, _ := st.Get([]byte(tx.Recipient)) //从状态树中获取接收者的状态
			var r_state *core.AccountState
			if r_state_enc == nil {
				// fmt.Println("missing account RECIPIENT, now adding account")
				ib := new(big.Int)
				ib.Add(ib, params.Init_Balance)
				r_state = &core.AccountState{
					Nonce:   uint64(i),
					Balance: ib,
				}
			} else {
				r_state = core.DecodeAS(r_state_enc)
			}
			r_state.Deposit(tx.Value)
			st.Update([]byte(tx.Recipient), r_state.Encode())
			cnt++
		}

		// if senderIn && !recipientIn {
		// 	// change this part to the pbft stage
		// 	fmt.Printf("this transaciton is cross-shard txs, will be sent to relaypool later\n")
		// }
	}
	// commit the memory trie to the database in the disk
	if cnt == 0 {
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	}
	rt, ns := st.Commit(false)
	err = bc.triedb.Update(trie.NewWithNodeSet(ns))
	if err != nil {
		log.Panic()
	}
	err = bc.triedb.Commit(rt, false)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("modified account number is ", cnt)
	return rt
}

// generate (mine) a block, this function return a block
func (bc *BlockChain) GenerateBlock() *core.Block { //该函数用于生成（挖掘）一个块。它返回一个块。
	// pack the transactions from the txpool
	txs := bc.Txpool.PackTxs(bc.ChainConfig.BlockSize) //从交易池中获取交易
	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash,
		Number:          bc.CurrentBlock.Header.Number + 1,
		Time:            time.Now(),
	}
	// handle transactions to build root
	rt := bc.GetUpdateStatusTrie(txs) //处理交易以构建状态树

	bh.StateRoot = rt.Bytes()
	bh.TxRoot = GetTxTreeRoot(txs)
	b := core.NewBlock(bh, txs)
	b.Header.Miner = 0
	b.Hash = b.Header.Hash()
	return b
}

// 新的 genisis 块（创世区块），该函数只会针对区块链对象调用一次
func (bc *BlockChain) NewGenisisBlock() *core.Block { //该函数用于创建一个新的创世区块。它返回一个块。
	body := make([]*core.Transaction, 0)
	bh := &core.BlockHeader{
		Number: 0,
	}
	// build a new trie database by db
	triedb := trie.NewDatabaseWithConfig(bc.db, &trie.Config{
		Cache:     0,
		Preimages: true,
	})
	bc.triedb = triedb
	statusTrie := trie.NewEmpty(triedb)
	bh.StateRoot = statusTrie.Hash().Bytes()
	bh.TxRoot = GetTxTreeRoot(body)
	b := core.NewBlock(bh, body)
	b.Hash = b.Header.Hash()
	return b
}

// add the genisis block in a blockchain
func (bc *BlockChain) AddGenisisBlock(gb *core.Block) { //该函数用于添加创世区块到区块链。它接受一个块作为参数，并将其添加到区块链中。
	bc.Storage.AddBlock(gb)
	newestHash, err := bc.Storage.GetNewestBlockHash()
	if err != nil {
		log.Panic()
	}
	curb, err := bc.Storage.GetBlock(newestHash)
	if err != nil {
		log.Panic()
	}
	bc.CurrentBlock = curb
}

// add a block
func (bc *BlockChain) AddBlock(b *core.Block) { //该函数用于添加一个块到区块链。它接受一个块作为参数，并将其添加到区块链中。
	if b.Header.Number != bc.CurrentBlock.Header.Number+1 {
		fmt.Println("the block height is not correct")
		return
	}
	// 如果该区块被节点挖出，则无需再次处理交易
	if b.Header.Miner != bc.ChainConfig.NodeID {
		rt := bc.GetUpdateStatusTrie(b.Body)
		fmt.Println(bc.CurrentBlock.Header.Number+1, "the root = ", rt.Bytes())
	}
	bc.CurrentBlock = b
	bc.Storage.AddBlock(b)
}

// 新的区块链。
// ChainConfig是预先定义的，用于标识区块链； db 是磁盘中的状态 trie 数据库
func NewBlockChain(cc *params.ChainConfig, db ethdb.Database) (*BlockChain, error) { //该函数用于创建一个新的区块链。它接受一个 params.ChainConfig 结构和一个 ethdb.Database 数据库作为参数，并返回一个 BlockChain 结构和一个错误。
	fmt.Println("Generating a new blockchain", db)
	bc := &BlockChain{
		db:           db,
		ChainConfig:  cc,
		Txpool:       core.NewTxPool(),
		Storage:      storage.NewStorage(cc),
		PartitionMap: make(map[string]uint64),
	}
	curHash, err := bc.Storage.GetNewestBlockHash()
	if err != nil {
		fmt.Println("Get newest block hash err")
		// if the Storage bolt database cannot find the newest blockhash,
		// it means the blockchain should be built in height = 0
		if err.Error() == "cannot find the newest block hash" {
			genisisBlock := bc.NewGenisisBlock()
			bc.AddGenisisBlock(genisisBlock)
			fmt.Println("New genisis block")
			return bc, nil
		}
		log.Panic()
	}

	// there is a blockchain in the storage
	fmt.Println("Existing blockchain found")
	curb, err := bc.Storage.GetBlock(curHash)
	if err != nil {
		log.Panic()
	}

	bc.CurrentBlock = curb
	triedb := trie.NewDatabaseWithConfig(db, &trie.Config{
		Cache:     0,
		Preimages: true,
	})
	bc.triedb = triedb
	// check the existence of the trie database
	_, err = trie.New(trie.TrieID(common.BytesToHash(curb.Header.StateRoot)), triedb)
	if err != nil {
		log.Panic()
	}
	fmt.Println("The status trie can be built")
	fmt.Println("Generated a new blockchain successfully")
	return bc, nil
}

// 检查此区块链配置中的块是否有效
func (bc *BlockChain) IsValidBlock(b *core.Block) error { //该函数用于检查此区块链配置中的块是否有效。它接受一个块作为参数。
	if string(b.Header.ParentBlockHash) != string(bc.CurrentBlock.Hash) { //如果父块哈希不等于当前块哈希，则打印错误消息并返回错误
		fmt.Println("the parentblock hash is not equal to the current block hash")
		return errors.New("the parentblock hash is not equal to the current block hash")
	} else if string(GetTxTreeRoot(b.Body)) != string(b.Header.TxRoot) {
		fmt.Println("the transaction root is wrong")
		return errors.New("the transaction root is wrong")
	}
	return nil
}

// add accounts
func (bc *BlockChain) AddAccounts(ac []string, as []*core.AccountState) { //该函数用于添加帐户。它接受一个字符串数组和一个 AccountState 数组作为参数，并将其添加到区块链中。
	fmt.Printf("The len of accounts is %d, now adding the accounts\n", len(ac))

	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash,
		Number:          bc.CurrentBlock.Header.Number + 1,
		Time:            time.Time{},
	}
	// handle transactions to build root
	rt := common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	if len(ac) != 0 {
		st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
		if err != nil {
			log.Panic(err)
		}
		for i, addr := range ac {
			if bc.Get_PartitionMap(addr) == bc.ChainConfig.ShardID {
				ib := new(big.Int)
				ib.Add(ib, as[i].Balance)
				new_state := &core.AccountState{
					Balance: ib,
					Nonce:   as[i].Nonce,
				}
				st.Update([]byte(addr), new_state.Encode())
			}
		}
		rrt, ns := st.Commit(false)
		err = bc.triedb.Update(trie.NewWithNodeSet(ns))
		if err != nil {
			log.Panic(err)
		}
		err = bc.triedb.Commit(rt, false)
		if err != nil {
			log.Panic(err)
		}
		rt = rrt
	}

	emptyTxs := make([]*core.Transaction, 0)
	bh.StateRoot = rt.Bytes()
	bh.TxRoot = GetTxTreeRoot(emptyTxs)
	b := core.NewBlock(bh, emptyTxs)
	b.Header.Miner = 0
	b.Hash = b.Header.Hash()

	bc.CurrentBlock = b
	bc.Storage.AddBlock(b)
}

// fetch accounts
func (bc *BlockChain) FetchAccounts(addrs []string) []*core.AccountState { //该函数用于获取帐户。它接受一个字符串数组作为参数，并返回一个 AccountState 数组。
	res := make([]*core.AccountState, 0)
	st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
	if err != nil {
		log.Panic(err)
	}
	for _, addr := range addrs {
		asenc, _ := st.Get([]byte(addr))
		var state_a *core.AccountState
		if asenc == nil {
			ib := new(big.Int)
			ib.Add(ib, params.Init_Balance)
			state_a = &core.AccountState{
				Nonce:   uint64(0),
				Balance: ib,
			}
		} else {
			state_a = core.DecodeAS(asenc)
		}
		res = append(res, state_a)
	}
	return res
}

// close a blockChain, close the database inferfaces
func (bc *BlockChain) CloseBlockChain() { //该函数用于关闭区块链。它关闭与区块链相关的所有数据库接口。
	bc.Storage.DataBase.Close()
	bc.triedb.CommitPreimages()
}

// print the details of a blockchain
func (bc *BlockChain) PrintBlockChain() string { //该函数用于打印区块链的详细信息。它返回一个字符串。
	vals := []interface{}{
		bc.CurrentBlock.Header.Number,
		bc.CurrentBlock.Hash,
		bc.CurrentBlock.Header.StateRoot,
		bc.CurrentBlock.Header.Time,
		bc.triedb,
		// len(bc.Txpool.RelayPool[1]),
	}
	res := fmt.Sprintf("%v\n", vals)
	fmt.Println(res)
	return res
}
