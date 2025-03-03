package test

import (
	"blockEmulator/query"
	"fmt"
)

func TestQueryBlocks() { //此测试函数从具有指定分片 ID (2) 和节点 ID (0) 的区块链查询块列表，并打印返回的块数。
	blocks := query.QueryBlocks(2, 0) //查询区块高度为 2 的区块
	fmt.Println(len(blocks))          //打印区块的数量
}

func TestQueryBlock() { //此测试函数查询具有分片 ID (0)、节点 ID (1) 和块编号 (22) 的特定块，然后打印块主体的长度。它似乎检索块内的交易。
	block := query.QueryBlock(0, 1, 22)
	fmt.Println(len(block.Body))
}

func TestQueryNewestBlock() { //该测试函数查询特定分片ID（0）和节点ID（0）的最新块，并打印最新块的块号。
	blocks := query.QueryNewestBlock(0, 0)
	fmt.Println(blocks.Header.Number)
}

func TestQueryBlockTxs() { //此测试函数查询具有分片 ID (0)、节点 ID (0) 和区块编号 (3) 的特定区块内的交易，然后打印该区块中的交易数量。
	txs := query.QueryBlockTxs(0, 0, 3) //查询区块高度为 0 的区块中的交易
	fmt.Println(len(txs))               //打印交易的数量
}

func TestQueryAccountState() { //此测试函数查询特定分片 ID (0) 和节点 ID (0) 内特定账户地址（“32be343b94f860124dc4fee278fdcbd38c102d88”）的账户状态。然后它会打印该帐户的余额。
	accountState := query.QueryAccountState(0, 0, "32be343b94f860124dc4fee278fdcbd38c102d88")
	fmt.Println(accountState.Balance)
}

//这些测试函数用于测试和验证与区块链数据检索相关的查询包或方法的功能。它们有助于确保区块链和查询机制按预期工作。
