//txpool的定义和一些操作

package core

import (
	"blockEmulator/utils"
	"sync"
	"time"
)

type TxPool struct { //TxPool结构包含交易池的各种信息
	TxQueue   []*Transaction            //交易队列
	RelayPool map[uint64][]*Transaction //中继池，专为分片区块链设计，来自 Monride
	lock      sync.Mutex                //锁
	// The pending list is ignored
}

func NewTxPool() *TxPool { //NewTxPool函数创建并返回一个交易池
	return &TxPool{
		TxQueue:   make([]*Transaction, 0),         //它被初始化为指向 Transaction 的空指针切片，长度为0.用于保存事务队列。
		RelayPool: make(map[uint64][]*Transaction), //它被初始化为一个空的map，其中包含 uint64 键和指向 Transaction 的指针片段作为值 ，用于保存中继池。
	}
}

// 将交易添加到池中（仅考虑队列）
func (txpool *TxPool) AddTx2Pool(tx *Transaction) {
	txpool.lock.Lock()         //锁定交易池
	defer txpool.lock.Unlock() //设置延迟函数调用，以便在函数返回后解锁交易池
	if tx.Time.IsZero() {      //
		tx.Time = time.Now()
	} //如果Time交易字段是零，表示之前尚未设置，将该字段设置为当前时间time.Now()。该时间戳可用于记录交易何时被添加到池中。
	txpool.TxQueue = append(txpool.TxQueue, tx) //将交易添加到交易队列中
}

//通过使用锁，该方法确保事务以安全且同步的方式添加到池中。如果事务还没有时间戳，它还会为事务添加时间戳，然后将其附加到池的队列中。

// 将交易列表添加到池中
func (txpool *TxPool) AddTxs2Pool(txs []*Transaction) { //
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	for _, tx := range txs {
		if tx.Time.IsZero() {
			tx.Time = time.Now()
		}
		txpool.TxQueue = append(txpool.TxQueue, tx)
	}
}

// 将交易添加到池头
func (txpool *TxPool) AddTxs2Pool_Head(tx []*Transaction) {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	txpool.TxQueue = append(tx, txpool.TxQueue...)
}

// 打包提案的交易
func (txpool *TxPool) PackTxs(max_txs uint64) []*Transaction { //PackTxs()函数用于打包交易。它需要一个参数： max_txs（类型为uint64）：这是一个无符号整数，表示要打包的最大交易数。它返回一个指向 Transaction 的指针切片，表示打包的交易。
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	txNum := max_txs
	if uint64(len(txpool.TxQueue)) < txNum {
		txNum = uint64(len(txpool.TxQueue))
	}
	txs_Packed := txpool.TxQueue[:txNum]
	txpool.TxQueue = txpool.TxQueue[txNum:]
	return txs_Packed
}

// 中继交易
func (txpool *TxPool) AddRelayTx(tx *Transaction, shardID uint64) { //AddRelayTx()函数用于将交易添加到中继池。它需要两个参数： tx（类型为*Transaction）：这是一个指向 Transaction 的指针，表示要添加到中继池的交易。 shardID（类型为uint64）：这是一个无符号整数，表示交易的分片ID。它没有返回值。
	txpool.lock.Lock() //锁定交易池
	defer txpool.lock.Unlock()
	_, ok := txpool.RelayPool[shardID] //检查中继池中是否存在给定分片ID的条目
	if !ok {                           //如果不存在，则创建一个新的条目
		txpool.RelayPool[shardID] = make([]*Transaction, 0)
	}
	txpool.RelayPool[shardID] = append(txpool.RelayPool[shardID], tx) //将提供的交易 (tx) 附加到与指定 shardID 关联的中继池。
}

// txpool get locked
func (txpool *TxPool) GetLocked() { //GetLocked()函数用于锁定交易池
	txpool.lock.Lock()
}

// txpool get unlocked
func (txpool *TxPool) GetUnlocked() { //GetUnlocked()函数用于解锁交易池
	txpool.lock.Unlock()
}

// get the length of tx queue
func (txpool *TxPool) GetTxQueueLen() int { //GetTxQueueLen()函数用于获取交易队列的长度
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	return len(txpool.TxQueue)
}

// get the length of ClearRelayPool
func (txpool *TxPool) ClearRelayPool() { //ClearRelayPool()函数用于清除中继池
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	txpool.RelayPool = nil
}

// abort ! 从中继池打包中继交易
func (txpool *TxPool) PackRelayTxs(shardID, minRelaySize, maxRelaySize uint64) ([]*Transaction, bool) {
	//PackRelayTxs()函数用于打包中继池中的交易。它需要三个参数： shardID（类型为uint64）：这是一个无符号整数，表示要打包的交易的分片ID。 minRelaySize（类型为uint64）：这是一个无符号整数，表示要打包的最小交易数。 maxRelaySize（类型为uint64）：这是一个无符号整数，表示要打包的最大交易数。它返回两个值： []*Transaction：这是一个指向 Transaction 的指针切片，表示打包的交易。 bool：这是一个布尔值，表示是否成功打包交易。
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	if _, ok := txpool.RelayPool[shardID]; !ok {
		return nil, false
	} //检查中继池中是否存在给定分片ID的条目
	if len(txpool.RelayPool[shardID]) < int(minRelaySize) {
		return nil, false
	} //检查中继池中是否存在足够的交易
	txNum := maxRelaySize                               //如果存在足够的交易，则将交易数设置为 maxRelaySize
	if uint64(len(txpool.RelayPool[shardID])) < txNum { //如果中继池中的交易数小于 maxRelaySize，则将交易数设置为中继池中的交易数
		txNum = uint64(len(txpool.RelayPool[shardID]))
	}
	relayTxPacked := txpool.RelayPool[shardID][:txNum]            //将中继池中的交易打包
	txpool.RelayPool[shardID] = txpool.RelayPool[shardID][txNum:] //从中继池中删除打包的交易
	return relayTxPacked, true
}

// abort ! 重新分片时转移交易
func (txpool *TxPool) TransferTxs(addr utils.Address) []*Transaction {
	//TransferTxs()函数用于转移交易。它需要一个参数： addr（类型为utils.Address）：这是一个地址，表示要转移交易的地址。它返回一个指向 Transaction 的指针切片，表示转移的交易。
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	txTransfered := make([]*Transaction, 0) //创建一个指向 Transaction 的指针切片，用于保存转移的交易
	newTxQueue := make([]*Transaction, 0)   //创建一个指向 Transaction 的指针切片，用于保存交易队列
	for _, tx := range txpool.TxQueue {     //遍历交易队列
		if tx.Sender == addr { //如果交易的发送者是给定地址，则将其添加到 txTransfered
			txTransfered = append(txTransfered, tx)
		} else { //否则将其添加到 newTxQueue
			newTxQueue = append(newTxQueue, tx)
		}
	}
	newRelayPool := make(map[uint64][]*Transaction)    //创建一个map，用于保存中继池
	for shardID, shardPool := range txpool.RelayPool { //遍历中继池
		for _, tx := range shardPool {
			if tx.Sender == addr { //如果交易的发送者是给定地址，则将其添加到 txTransfered
				txTransfered = append(txTransfered, tx)
			} else { //否则将其添加到 newRelayPool
				if _, ok := newRelayPool[shardID]; !ok { //检查中继池中是否存在给定分片ID的条目
					newRelayPool[shardID] = make([]*Transaction, 0) //如果不存在，则创建一个新的条目
				}
				newRelayPool[shardID] = append(newRelayPool[shardID], tx) //
			}
		}
	}
	txpool.TxQueue = newTxQueue     //将 newTxQueue 赋值给交易队列
	txpool.RelayPool = newRelayPool //将 newRelayPool 赋值给中继池
	return txTransfered             //返回需要转移的交易
}
