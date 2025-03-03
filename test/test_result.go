package test

import (
	"blockEmulator/chain"
	"blockEmulator/core"
	"blockEmulator/params"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

//对区块链网络的状态执行测试，以在处理一组交易后验证帐户余额。

func data2tx(data []string, nonce uint64) (*core.Transaction, bool) {
	if data[6] == "0" && data[7] == "0" && len(data[3]) > 16 && len(data[4]) > 16 && data[3] != data[4] {
		val, ok := new(big.Int).SetString(data[8], 10)
		if !ok {
			log.Panic("new int failed\n")
		}
		tx := core.NewTransaction(data[3][2:], data[4][2:], val, nonce)
		return tx, true
	}
	return &core.Transaction{}, false
}

//data2tx 函数，它接受一个字符串切片 data 和一个整数 nonce 作为参数，并返回一个指向 core.Transaction 结构的指针和一个布尔值。
//这个函数用于将字符串切片 data 转换为 core.Transaction 结构。如果转换成功，它将返回一个指向 core.Transaction 结构的指针和一个布尔值 true.
//否则，它将返回一个空的 core.Transaction 结构和一个布尔值 false。

func Ttestresult(ShardNums int) {
	accountBalance := make(map[string]*big.Int) //accountBalance 变量，它是一个映射，用于保存账户地址和账户余额之间的映射关系。
	acCorrect := make(map[string]bool)          //acCorrect 变量，它是一个映射，用于保存账户地址和布尔值之间的映射关系。这些布尔值表示账户余额是否正确。

	txfile, err := os.Open(params.FileInput) //打开包含交易数据的 CSV 文件
	if err != nil {
		log.Panic(err)
	}
	defer txfile.Close()
	reader := csv.NewReader(txfile) //创建了一个 CSV 读取器，以从打开的 CSV 文件中读取数据
	nowDataNum := 0                 //nowDataNum 变量，它是一个整数，用于记录读取的数据条数。在这里，我们将其初始化为 0
	for {                           //通过一个 for 循环，你读取了 CSV 文件中的数据，并将其添加到 accountBalance 变量中
		data, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		if nowDataNum == params.TotalDataSize { //如果读取到了文件末尾，或者读取的数据条数达到了 200000 条，就退出循环
			break
		}
		if tx, ok := data2tx(data, uint64(nowDataNum)); ok { //调用了 data2tx 函数，将读取的数据转换为 core.Transaction 结构
			nowDataNum++                                 //增加 nowDataNum 计数器以跟踪已处理记录的数量
			if _, ok := accountBalance[tx.Sender]; !ok { //如果发送方账户地址不存在，就创建一个新的账户地址，并将其余额设置为 params.Init_Balance
				accountBalance[tx.Sender] = new(big.Int)
				accountBalance[tx.Sender].Add(accountBalance[tx.Sender], params.Init_Balance)
			}

			if _, ok := accountBalance[tx.Recipient]; !ok { //如果接收方账户地址不存在，就创建一个新的账户地址，并将其余额设置为 params.Init_Balance
				accountBalance[tx.Recipient] = new(big.Int)
				accountBalance[tx.Recipient].Add(accountBalance[tx.Recipient], params.Init_Balance)
			}
			if accountBalance[tx.Sender].Cmp(tx.Value) != -1 { //验证发送者是否有足够的余额来执行交易，如果是的话，它会相应地更新帐户余额。
				accountBalance[tx.Sender].Sub(accountBalance[tx.Sender], tx.Value)
				accountBalance[tx.Recipient].Add(accountBalance[tx.Recipient], tx.Value)
			}
		}
	}
	fmt.Println(len(accountBalance))       //处理完所有交易后，该函数会在 accountBalance 映射中打印唯一帐户的数量。
	for sid := 0; sid < ShardNums; sid++ { //迭代每个分片（由 sid 标识），为该分片创建区块链，并使用区块链获取 accountBalance 中每个账户的账户状态。
		fp := "./record/ldb/s" + strconv.FormatUint(uint64(sid), 10) + "/n0" //fp变量，它是一个字符串，表示区块链数据库文件的路径。在这里，我们将其设置为 ./record/ldb/s0/N0，这意味着我们将在 record/ldb/s0/N0 目录中创建一个名为 accountState 的数据库文件
		db, err := rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false) //db变量，它是一个指向 rawdb.Database 结构的指针。该结构用于表示区块链数据库，它是一个键值对数据库，用于存储区块链的状态。在这里，我们使用 rawdb.NewLevelDBDatabase 函数创建了一个 LevelDB 数据库，该函数接受一个文件路径作为参数，该文件路径用于存储数据库文件。我们还将数据库的缓存大小设置为 0，将文件的打开模式设置为 1，将数据库的名称设置为 accountState，将数据库的只读标志设置为 false
		if err != nil {
			log.Panic(err)
		}
		pcc := &params.ChainConfig{
			ChainID:        uint64(sid),
			NodeID:         0,
			ShardID:        uint64(sid),
			Nodes_perShard: uint64(params.NodesInShard),
			ShardNums:      uint64(ShardNums),
			BlockSize:      uint64(params.MaxBlockSize_global),
			BlockInterval:  uint64(params.Block_Interval),
			InjectSpeed:    uint64(params.InjectSpeed),
		}
		CurChain, _ := chain.NewBlockChain(pcc, db) //创建了一个区块链对象 CurChain，并传入前面创建的 params.ChainConfig 结构和数据库 db
		for key, val := range accountBalance {      //对于每个帐户，它将区块链中的帐户余额 (v[0].Balance) 与 accountBalance 映射中的预期余额 (val) 进行比较。如果余额匹配，则该帐户会在 acCorrect 映射中标记为正确。
			v := CurChain.FetchAccounts([]string{key}) //调用 FetchAccounts 方法，从区块链中检索了帐户余额
			if val.Cmp(v[0].Balance) == 0 {
				acCorrect[key] = true
			}
		}
		CurChain.CloseBlockChain() //关闭了区块链数据库
	}
	fmt.Println(len(acCorrect))
	if len(acCorrect) == len(accountBalance) { //最后，该函数会检查 acCorrect 映射中的帐户数量是否与 accountBalance 映射中的帐户数量相同。如果是的话，它会打印一条消息，表示测试通过。表明区块链的帐户状态与预期结果匹配。否则，它会打印错误帐户的数量以及有关机制中潜在错误的消息。
		fmt.Println("test pass")
	} else {
		fmt.Println(len(accountBalance)-len(acCorrect), "accounts errs, they may be brokers~;")
		fmt.Println("if the number of err accounts is too large, the mechanism has bugs")
	}
}

//该代码似乎是在处理交易并将其与预期结果进行比较后测试区块链中帐户余额的一致性。如果区块链中的所有账户余额与预期余额相符，则认为测试成功。
