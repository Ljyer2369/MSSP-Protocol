// 当leader收到重新分区消息时发生账户转移。
// 领导者将要转账的账户信息发送给其他领导者，并进行处理。

package pbft_all

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"encoding/json"
	"log"
	"time"
)

// 这个message在提议阶段使用，因此将由 InsidePBFT_Module 调用
func (cphm *CLPAPbftInsideExtraHandleMod) sendPartitionReady() {
	//它在提案阶段使用，并且可能由 InsidePBFT_Module 调用。这个方法涉及到一个共识协议，PBFT（实用拜占庭容错），它用于通知其他分片或节点当前分片已准备好进行分区。
	cphm.cdm.P_ReadyLock.Lock()                           //锁定互斥体（cphm.cdm.P_ReadyLock）以保护对 PartitionReady 数据结构的访问，该结构似乎是跟踪每个分片是否准备好分区的映射或数组。
	cphm.cdm.PartitionReady[cphm.pbftNode.ShardID] = true //将当前分片的准备状态设置为 true
	cphm.cdm.P_ReadyLock.Unlock()                         //解锁互斥体（cphm.cdm.P_ReadyLock）

	pr := message.PartitionReady{ //创建一个新的 PartitionReady 结构，该结构包含有关当前分片的信息，以及当前分片的序列ID。
		FromShard: cphm.pbftNode.ShardID,    //当前分片的ID
		NowSeqID:  cphm.pbftNode.sequenceID, //当前分片的序列ID
	}
	pByte, err := json.Marshal(pr) //使用 json.Marshal() 函数将 PartitionReady 结构编码为 JSON 格式
	if err != nil {
		log.Panic()
	}
	send_msg := message.MergeMessage(message.CPartitionReady, pByte)          //通过将 CPartitionReady 消息类型与封送的 pr 消息合并来构造要发送的消息 (send_msg)
	for sid := 0; sid < int(cphm.pbftNode.pbftChainConfig.ShardNums); sid++ { //迭代所有分片（由 sid 表示），并使用名为 Networks.TcpDial 的函数或库将 send_msg 发送到除当前分片之外的其他分片。此步骤通知其他分片当前分片已准备好进行分区。
		if sid != int(pr.FromShard) {
			networks.TcpDial(send_msg, cphm.pbftNode.ip_nodeTable[uint64(sid)][0]) //通过TCP连接发送消息
		}
	}
	cphm.pbftNode.pl.Plog.Print("Ready for partition\n") //打印日志，指示当前分片已准备好进行分区
}

//该函数向其他分片发送消息，通知它们当前分片的准备情况，这是共识协议中的重要一步。

// 获取所有分片是否准备就绪，将由 InsidePBFT_Module 调用
func (cphm *CLPAPbftInsideExtraHandleMod) getPartitionReady() bool { //getPartitionReady方法用于获取所有分片是否准备就绪
	cphm.cdm.P_ReadyLock.Lock()
	defer cphm.cdm.P_ReadyLock.Unlock()
	cphm.pbftNode.seqMapLock.Lock()
	defer cphm.pbftNode.seqMapLock.Unlock()
	cphm.cdm.ReadySeqLock.Lock()
	defer cphm.cdm.ReadySeqLock.Unlock()

	flag := true //flag变量用于跟踪当前分片是否准备好分区
	for sid, val := range cphm.pbftNode.seqIDMap {
		if rval, ok := cphm.cdm.ReadySeq[sid]; !ok || (rval-1 != val) {
			flag = false
		}
	}
	return len(cphm.cdm.PartitionReady) == int(cphm.pbftNode.pbftChainConfig.ShardNums) && flag //如果所有分片都准备好分区，则返回 true，否则返回 false
}

// 将交易和 accountState 发送给其他领导者
func (cphm *CLPAPbftInsideExtraHandleMod) sendAccounts_and_Txs() { //sendAccounts_and_Txs方法用于将交易和账户状态发送给其他领导者
	// 生成账户转账和 txs 消息
	accountToFetch := make([]string, 0)
	lastMapid := len(cphm.cdm.ModifiedMap) - 1
	for key, val := range cphm.cdm.ModifiedMap[lastMapid] {
		if val != cphm.pbftNode.ShardID && cphm.pbftNode.CurChain.Get_PartitionMap(key) == cphm.pbftNode.ShardID {
			accountToFetch = append(accountToFetch, key)
		}
	}
	asFetched := cphm.pbftNode.CurChain.FetchAccounts(accountToFetch)
	//将账户发送到其他分片
	cphm.pbftNode.CurChain.Txpool.GetLocked()
	cphm.pbftNode.pl.Plog.Println("The size of tx pool is: ", len(cphm.pbftNode.CurChain.Txpool.TxQueue)) //打印日志，指示当前分片的交易池中的交易数量
	for i := uint64(0); i < cphm.pbftNode.pbftChainConfig.ShardNums; i++ {                                //迭代所有分片（由 i 表示）
		if i == cphm.pbftNode.ShardID {
			continue
		}
		addrSend := make([]string, 0) //addrSend变量用于跟踪要发送到分片 i 的地址
		addrSet := make(map[string]bool)
		asSend := make([]*core.AccountState, 0)
		for idx, addr := range accountToFetch { //迭代所有要发送到分片 i 的地址（由 addr 表示）
			if cphm.cdm.ModifiedMap[lastMapid][addr] == i {
				addrSend = append(addrSend, addr)
				addrSet[addr] = true
				asSend = append(asSend, asFetched[idx])
			}
		}
		//向其中获取交易，获取交易后，将其从池中删除
		txSend := make([]*core.Transaction, 0)
		firstPtr := 0
		for secondPtr := 0; secondPtr < len(cphm.pbftNode.CurChain.Txpool.TxQueue); secondPtr++ {
			ptx := cphm.pbftNode.CurChain.Txpool.TxQueue[secondPtr]
			//如果这是一个正常的交易或者重新分片之前的ctx1 && addr是对应的
			_, ok1 := addrSet[ptx.Sender]
			condition1 := ok1 && !ptx.Relayed
			// if this tx is ctx2
			_, ok2 := addrSet[ptx.Recipient]
			condition2 := ok2 && ptx.Relayed
			if condition1 || condition2 {
				txSend = append(txSend, ptx)
			} else {
				cphm.pbftNode.CurChain.Txpool.TxQueue[firstPtr] = ptx
				firstPtr++
			}
		}
		cphm.pbftNode.CurChain.Txpool.TxQueue = cphm.pbftNode.CurChain.Txpool.TxQueue[:firstPtr]

		cphm.pbftNode.pl.Plog.Printf("The txSend to shard %d is generated \n", i)
		ast := message.AccountStateAndTx{ //创建一个新的 AccountStateAndTx 结构，该结构包含有关当前分片的信息，以及当前分片的序列ID。
			Addrs:        addrSend,
			AccountState: asSend,
			FromShard:    cphm.pbftNode.ShardID,
			Txs:          txSend,
		}
		aByte, err := json.Marshal(ast)
		if err != nil {
			log.Panic()
		}
		send_msg := message.MergeMessage(message.AccountState_and_TX, aByte)
		networks.TcpDial(send_msg, cphm.pbftNode.ip_nodeTable[i][0])
		cphm.pbftNode.pl.Plog.Printf("The message to shard %d is sent\n", i)
	}
	cphm.pbftNode.pl.Plog.Println("after sending, The size of tx pool is: ", len(cphm.pbftNode.CurChain.Txpool.TxQueue))
	cphm.pbftNode.CurChain.Txpool.GetUnlocked()
}

// 获取收集信息 （fetch collect infos）
func (cphm *CLPAPbftInsideExtraHandleMod) getCollectOver() bool {
	cphm.cdm.CollectLock.Lock()
	defer cphm.cdm.CollectLock.Unlock()
	return cphm.cdm.CollectOver
}

// 提出一条分区消息（propose a partition message）
func (cphm *CLPAPbftInsideExtraHandleMod) proposePartition() (bool, *message.Request) { //proposePartition方法用于提出一条分区消息
	cphm.pbftNode.pl.Plog.Printf("S%dN%d : begin partition proposing\n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID) //打印日志，指示当前分片已准备好分区
	//将池中的所有数据添加到集合中
	for _, at := range cphm.cdm.AccountStateTx { //迭代所有的 accountStateTx
		for i, addr := range at.Addrs { //迭代所有的地址
			cphm.cdm.ReceivedNewAccountState[addr] = at.AccountState[i] //将地址和账户状态添加到 cphm.cdm.ReceivedNewAccountState 中
		}
		cphm.cdm.ReceivedNewTx = append(cphm.cdm.ReceivedNewTx, at.Txs...) //将交易添加到 cphm.cdm.ReceivedNewTx 中
	}
	// 提议，将所有交易发送到分片中的其他节点（propose, send all txs to other nodes in shard）
	cphm.pbftNode.pl.Plog.Println("The number of ReceivedNewTx: ", len(cphm.cdm.ReceivedNewTx))
	for _, tx := range cphm.cdm.ReceivedNewTx { //迭代所有的交易
		if !tx.Relayed && cphm.cdm.ModifiedMap[cphm.cdm.AccountTransferRound][tx.Sender] != cphm.pbftNode.ShardID {
			log.Panic("error tx")
		}
		if tx.Relayed && cphm.cdm.ModifiedMap[cphm.cdm.AccountTransferRound][tx.Recipient] != cphm.pbftNode.ShardID {
			log.Panic("error tx")
		}
	}
	cphm.pbftNode.CurChain.Txpool.AddTxs2Pool(cphm.cdm.ReceivedNewTx)//将交易添加到交易池中
	cphm.pbftNode.pl.Plog.Println("The size of txpool: ", len(cphm.pbftNode.CurChain.Txpool.TxQueue))//打印日志，指示当前分片的交易池中的交易数量

	atmaddr := make([]string, 0)
	atmAs := make([]*core.AccountState, 0)
	for key, val := range cphm.cdm.ReceivedNewAccountState { //迭代所有的 cphm.cdm.ReceivedNewAccountState
		atmaddr = append(atmaddr, key)
		atmAs = append(atmAs, val)
	}
	atm := message.AccountTransferMsg{//创建一个新的 AccountTransferMsg 结构，该结构包含有关当前分片的信息，以及当前分片的序列ID。
		ModifiedMap:  cphm.cdm.ModifiedMap[cphm.cdm.AccountTransferRound],
		Addrs:        atmaddr,
		AccountState: atmAs,
		ATid:         uint64(len(cphm.cdm.ModifiedMap)),
	}
	atmbyte := atm.Encode()
	r := &message.Request{//创建一个新的 Request 结构，该结构包含有关当前分片的信息，以及当前分片的序列ID。
		RequestType: message.PartitionReq,
		Msg: message.RawMessage{
			Content: atmbyte,
		},
		ReqTime: time.Now(),
	}
	return true, r
}

// 分片中的所有节点都会进行账户传输，以同步状态树
func (cphm *CLPAPbftInsideExtraHandleMod) accountTransfer_do(atm *message.AccountTransferMsg) { //accountTransfer_do方法用于在分片中的所有节点都进行账户传输，以同步状态树
	// 更改分区图（change the partition Map）
	cnt := 0
	for key, val := range atm.ModifiedMap {
		cnt++
		cphm.pbftNode.CurChain.Update_PartitionMap(key, val)
	}
	cphm.pbftNode.pl.Plog.Printf("%d key-vals are updated\n", cnt)
	// 将帐户添加到状态树中
	cphm.pbftNode.pl.Plog.Printf("%d addrs to add\n", len(atm.Addrs))
	cphm.pbftNode.pl.Plog.Printf("%d accountstates to add\n", len(atm.AccountState))
	cphm.pbftNode.CurChain.AddAccounts(atm.Addrs, atm.AccountState)

	if uint64(len(cphm.cdm.ModifiedMap)) != atm.ATid {
		cphm.cdm.ModifiedMap = append(cphm.cdm.ModifiedMap, atm.ModifiedMap)
	}
	cphm.cdm.AccountTransferRound = atm.ATid
	cphm.cdm.AccountStateTx = make(map[uint64]*message.AccountStateAndTx)
	cphm.cdm.ReceivedNewAccountState = make(map[string]*core.AccountState)
	cphm.cdm.ReceivedNewTx = make([]*core.Transaction, 0)
	cphm.cdm.PartitionOn = false

	cphm.cdm.CollectLock.Lock()
	cphm.cdm.CollectOver = false
	cphm.cdm.CollectLock.Unlock()

	cphm.cdm.P_ReadyLock.Lock()
	cphm.cdm.PartitionReady = make(map[uint64]bool)
	cphm.cdm.P_ReadyLock.Unlock()

	cphm.pbftNode.CurChain.PrintBlockChain()
}
