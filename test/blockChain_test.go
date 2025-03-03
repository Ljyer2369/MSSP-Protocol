package test

import (
	"blockEmulator/chain"
	"blockEmulator/core"
	"blockEmulator/params"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestBlockChain(t *testing.T) {
	accounts := []string{"000000000001", "00000000002", "00000000003", "00000000004", "00000000005", "00000000006"} //字符串切片 accounts，其中包含了一些账户的地址。这些地址在接下来的测试中将用作账户标识
	as := make([]*core.AccountState, 0)                                                                             //空切片 as，其中每个元素都是指向 core.AccountState 结构的指针。这些结构将用于表示账户的状态，包括余额等信息
	for idx := range accounts {                                                                                     //通过一个循环，为每个账户地址创建了一个 core.AccountState 结构，并为每个账户的余额赋予不同的整数值。这些账户状态结构将在后续步骤中添加到区块链。                                                                               //
		as = append(as, &core.AccountState{
			Balance: big.NewInt(int64(idx)),
		})
	}
	for sid := 0; sid < 1; sid++ {
		//通过外部循环（sid 从 0 开始，最大为 0），你创建了一个区块链数据库 db，并将其与特定的数据目录相关联。这部分代码是为了模拟一个分片，因此 params.ShardNum 被设置为 1。
		fp := "./record/ldb/s0/N0"                                           //fp变量，它是一个字符串，表示区块链数据库文件的路径。在这里，我们将其设置为 ./record/ldb/s0/N0，这意味着我们将在 record/ldb/s0/N0 目录中创建一个名为 accountState 的数据库文件
		db, err := rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false) //db变量，它是一个指向 rawdb.Database 结构的指针。该结构用于表示区块链数据库，它是一个键值对数据库，用于存储区块链的状态。在这里，我们使用 rawdb.NewLevelDBDatabase 函数创建了一个 LevelDB 数据库，该函数接受一个文件路径作为参数，该文件路径用于存储数据库文件。我们还将数据库的缓存大小设置为 0，将文件的打开模式设置为 1，将数据库的名称设置为 accountState，将数据库的只读标志设置为 false
		if err != nil {
			log.Panic(err)
		}
		params.ShardNum = 1         //表示区块链网络中的分片数量。在这里，我们将其设置为 1，这意味着我们将创建一个只包含一个分片的区块链网络
		pcc := &params.ChainConfig{ //包含了一些区块链的配置参数。这些参数包括链的ID、节点ID、分片ID、分片的数量、块大小、块间隔和注入速度等
			ChainID:        uint64(sid),
			NodeID:         0,
			ShardID:        uint64(sid),
			Nodes_perShard: uint64(1),
			ShardNums:      4,
			BlockSize:      uint64(params.MaxBlockSize_global),
			BlockInterval:  uint64(params.Block_Interval),
			InjectSpeed:    uint64(params.InjectSpeed),
		}
		CurChain, _ := chain.NewBlockChain(pcc, db) //创建了一个区块链对象 CurChain，并传入前面创建的 params.ChainConfig 结构和数据库 db
		CurChain.PrintBlockChain()                  //调用 PrintBlockChain 方法，打印了区块链的初始状态。这将显示一个空的区块链，因为还没有添加任何区块
		CurChain.AddAccounts(accounts, as)          //使用 CurChain.AddAccounts 方法，你将前面创建的账户地址和账户状态添加到区块链中。这会在区块链中创建初始的账户状态
		CurChain.PrintBlockChain()                  //打印了更新后的区块链状态，显示了添加的账户信息。

		astates := CurChain.FetchAccounts(accounts) //从区块链中检索了刚刚添加的账户状态，并打印了它们的余额。
		for _, state := range astates {
			fmt.Println(state.Balance)
		}
		CurChain.CloseBlockChain() //关闭了区块链数据库
	}
}

//这个测试函数用于验证区块链的核心功能，包括创建账户、添加账户到区块链、检索账户状态和关闭区块链

/*输出结果：
Generating a new blockchain &{0xc00018e240} //表示正在生成一个新的区块链对象，同时显示了该对象的地址
Get newest block hash err //错误消息，表明在获取最新区块的哈希时出现了错误。通常，这可能是因为在测试的初始阶段还没有创建区块，因此无法获取最新区块的哈希。
The newest block is updated //表示最新的区块已经更新。这通常在创建区块链后立即发生。
Block is added //表示一个新的区块已经被添加到区块链中
New genisis block //显示了一个新的创世块的信息，包括该块的哈希值和时间戳。
[0 [39 20 163 237 217 53 105 45 106 125 69 249 133 117 28 117 71 219 241 4 161 98 237 223 76 249 32 159 233 102 108 250] [86 232 31 23 27 204 85 166 255 131 69 230 146 192 248 110 91 72 224 27 153 108 173 192 1 98 47 181 227 99 180 33] 0001-01-01 00:00:00 +0000 UTC 0xc000000000]
//上面一行显示了区块链中的一个区块的信息，包括块号、块哈希、父块哈希和时间戳等。
The len of accounts is 6, now adding the accounts //表示账户的数量为6，并且将要添加这些账户到区块链。
The newest block is updated //再次表示最新的区块已经更新
Block is added //一个新的区块已经被添加到区块链中
[1 [84 95 57 89 40 15 203 246 223 240 113 184 247 242 136 66 150 77 156 1 25 45 219 74 71 192 127 223 141 128 26 173] [145 66 16 182 61 69 232 42 200 120 127 51 99 105 116 221 67 253 57 199 53 129 225 131 60 162 164 188 227 135 15 185] 0001-01-01 00:00:00 +0000 UTC 0xc000000000]
//显示了另一个区块的信息，包括块号、块哈希、父块哈希和时间戳等。
0
1
2
3
4
5
//一系列数字（0、1、2、3、4、5）表示账户的余额。这可能是通过 FetchAccounts 方法检索的账户余额
--- PASS: TestBlockChain (0.35s) //这是测试结果的总结，表示测试函数 TestBlockChain 已通过。
PASS

*/
