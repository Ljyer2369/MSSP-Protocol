package pbft_all

import (
	"blockEmulator/consensus_shard/pbft_all/dataSupport"
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

type CLPAPbftInsideExtraHandleMod_forBroker struct { //CLPAPbftInsideExtraHandleMod_forBroker结构包含用于PBFT共识的各种配置参数
	cdm      *dataSupport.Data_supportCLPA //cdm是一个指向Data_supportCLPA结构的指针，其中包含用于PBFT共识的各种配置参数
	pbftNode *PbftConsensusNode            //pbftNode是一个指向PbftConsensusNode结构的指针，其中包含用于PBFT共识的各种配置参数
}

// 提出不同类型的请求
func (cphm *CLPAPbftInsideExtraHandleMod_forBroker) HandleinPropose() (bool, *message.Request) { //HandleinPropose方法用于提出不同类型的请求
	if cphm.cdm.PartitionOn { //如果当前分片已经分区，则执行以下操作
		cphm.sendPartitionReady()
		for !cphm.getPartitionReady() {
			time.Sleep(time.Second)
		}
		// send accounts and txs
		cphm.sendAccounts_and_Txs()
		// propose a partition
		for !cphm.getCollectOver() {
			time.Sleep(time.Second)
		}
		return cphm.proposePartition()
	}

	// ELSE: propose a block
	block := cphm.pbftNode.CurChain.GenerateBlock()
	r := &message.Request{
		RequestType: message.BlockRequest,
		ReqTime:     time.Now(),
	}
	r.Msg.Content = block.Encode()
	return true, r

}

// the diy operation in preprepare
func (cphm *CLPAPbftInsideExtraHandleMod_forBroker) HandleinPrePrepare(ppmsg *message.PrePrepare) bool { //HandleinPrePrepare方法用于在preprepare中执行diy操作
	// judge whether it is a partitionRequest or not
	isPartitionReq := ppmsg.RequestMsg.RequestType == message.PartitionReq

	if isPartitionReq {
		// after some checking
		cphm.pbftNode.pl.Plog.Printf("S%dN%d : a partition block\n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID)
	} else {
		// the request is a block
		if cphm.pbftNode.CurChain.IsValidBlock(core.DecodeB(ppmsg.RequestMsg.Msg.Content)) != nil {
			cphm.pbftNode.pl.Plog.Printf("S%dN%d : not a valid block\n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID)
			return false
		}
	}
	cphm.pbftNode.pl.Plog.Printf("S%dN%d : the pre-prepare message is correct, putting it into the RequestPool. \n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID)
	cphm.pbftNode.requestPool[string(ppmsg.Digest)] = ppmsg.RequestMsg
	// merge to be a prepare message
	return true
}

// 在prepare中的操作，以及在pbft + tx中继中，这个函数不需要做任何事情.
func (cphm *CLPAPbftInsideExtraHandleMod_forBroker) HandleinPrepare(pmsg *message.Prepare) bool {
	fmt.Println("No operations are performed in Extra handle mod")
	return true
}

// commit 中的操作。如果是分区请求，则执行分区操作；如果是块请求，则执行块操作
func (cphm *CLPAPbftInsideExtraHandleMod_forBroker) HandleinCommit(cmsg *message.Commit) bool {
	r := cphm.pbftNode.requestPool[string(cmsg.Digest)]
	// requestType ...
	if r.RequestType == message.PartitionReq {
		// if a partition Requst ...
		atm := message.DecodeAccountTransferMsg(r.Msg.Content)
		cphm.accountTransfer_do(atm) //调用accountTransfer_do方法，执行账户转移操作
		return true
	}
	// if a block request ...
	block := core.DecodeB(r.Msg.Content)
	cphm.pbftNode.pl.Plog.Printf("S%dN%d : adding the block %d...now height = %d \n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID, block.Header.Number, cphm.pbftNode.CurChain.CurrentBlock.Header.Number)
	cphm.pbftNode.CurChain.AddBlock(block)
	cphm.pbftNode.pl.Plog.Printf("S%dN%d : added the block %d... \n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID, block.Header.Number)
	cphm.pbftNode.CurChain.PrintBlockChain()

	// 现在尝试将 txs 中继到其他分片（对于主节点）
	if cphm.pbftNode.NodeID == cphm.pbftNode.view {
		cphm.pbftNode.pl.Plog.Printf("S%dN%d : main node is trying to send broker confirm txs at height = %d \n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID, block.Header.Number)
		// generate brokertxs and collect txs excuted
		txExcuted := make([]*core.Transaction, 0)
		broker1Txs := make([]*core.Transaction, 0)
		broker2Txs := make([]*core.Transaction, 0)

		// generate block infos
		for _, tx := range block.Body {
			isBroker1Tx := tx.Sender == tx.OriginalSender
			isBroker2Tx := tx.Recipient == tx.FinalRecipient

			senderIsInshard := cphm.pbftNode.CurChain.Get_PartitionMap(tx.Sender) == cphm.pbftNode.ShardID
			recipientIsInshard := cphm.pbftNode.CurChain.Get_PartitionMap(tx.Recipient) == cphm.pbftNode.ShardID
			if isBroker1Tx && !senderIsInshard {
				log.Panic("Err tx1")
			}
			if isBroker2Tx && !recipientIsInshard {
				log.Panic("Err tx2")
			}
			if tx.RawTxHash == nil {
				if tx.HasBroker {
					if tx.SenderIsBroker && !recipientIsInshard {
						log.Panic("err tx 1 - recipient")
					}
					if !tx.SenderIsBroker && !senderIsInshard {
						log.Panic("err tx 1 - sender")
					}
				} else {
					if !senderIsInshard || !recipientIsInshard {
						log.Panic("err tx - without broker")
					}
				}
			}

			if isBroker2Tx {
				broker2Txs = append(broker2Txs, tx)
			} else if isBroker1Tx {
				broker1Txs = append(broker1Txs, tx)
			} else {
				txExcuted = append(txExcuted, tx)
			}
		}
		// send seqID
		for sid := uint64(0); sid < cphm.pbftNode.pbftChainConfig.ShardNums; sid++ {
			if sid == cphm.pbftNode.ShardID {
				continue
			}
			sii := message.SeqIDinfo{
				SenderShardID: cphm.pbftNode.ShardID,
				SenderSeq:     cphm.pbftNode.sequenceID,
			}
			sByte, err := json.Marshal(sii)
			if err != nil {
				log.Panic()
			}
			msg_send := message.MergeMessage(message.CSeqIDinfo, sByte)
			networks.TcpDial(msg_send, cphm.pbftNode.ip_nodeTable[sid][0])
			cphm.pbftNode.pl.Plog.Printf("S%dN%d : sended sequence ids to %d\n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID, sid)
		}
		// send txs excuted in this block to the listener
		// add more message to measure more metrics
		bim := message.BlockInfoMsg{
			BlockBodyLength: len(block.Body),
			ExcutedTxs:      txExcuted,
			Broker1TxNum:    uint64(len(broker1Txs)),
			Broker1Txs:      broker1Txs,
			Broker2TxNum:    uint64(len(broker2Txs)),
			Broker2Txs:      broker2Txs,
			Epoch:           int(cphm.cdm.AccountTransferRound),
			SenderShardID:   cphm.pbftNode.ShardID,
			ProposeTime:     r.ReqTime,
			CommitTime:      time.Now(),
		}
		bByte, err := json.Marshal(bim)
		if err != nil {
			log.Panic()
		}
		msg_send := message.MergeMessage(message.CBlockInfo, bByte)
		networks.TcpDial(msg_send, cphm.pbftNode.ip_nodeTable[params.DeciderShard][0])
		cphm.pbftNode.pl.Plog.Printf("S%dN%d : sended excuted txs\n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID)
		cphm.pbftNode.CurChain.Txpool.GetLocked()
		cphm.pbftNode.writeCSVline([]string{strconv.Itoa(len(cphm.pbftNode.CurChain.Txpool.TxQueue)), strconv.Itoa(len(txExcuted)), strconv.Itoa(int(bim.Relay1TxNum))})
		cphm.pbftNode.CurChain.Txpool.GetUnlocked()
	}
	return true
}

func (cphm *CLPAPbftInsideExtraHandleMod_forBroker) HandleReqestforOldSeq(*message.RequestOldMessage) bool {
	fmt.Println("No operations are performed in Extra handle mod")
	return true
}

// 顺序请求的操作
func (cphm *CLPAPbftInsideExtraHandleMod_forBroker) HandleforSequentialRequest(som *message.SendOldMessage) bool {
	if int(som.SeqEndHeight-som.SeqStartHeight+1) != len(som.OldRequest) { //如果顺序请求的结束高度减去开始高度加1不等于顺序请求的长度，则打印错误信息
		cphm.pbftNode.pl.Plog.Printf("S%dN%d : the SendOldMessage message is not enough\n", cphm.pbftNode.ShardID, cphm.pbftNode.NodeID)
	} else { // add the block into the node pbft blockchain
		for height := som.SeqStartHeight; height <= som.SeqEndHeight; height++ {
			r := som.OldRequest[height-som.SeqStartHeight]
			if r.RequestType == message.BlockRequest {
				b := core.DecodeB(r.Msg.Content)
				cphm.pbftNode.CurChain.AddBlock(b)
			} else {
				atm := message.DecodeAccountTransferMsg(r.Msg.Content)
				cphm.accountTransfer_do(atm)
			}
		}
		cphm.pbftNode.sequenceID = som.SeqEndHeight + 1
		cphm.pbftNode.CurChain.PrintBlockChain()
	}
	return true
}
