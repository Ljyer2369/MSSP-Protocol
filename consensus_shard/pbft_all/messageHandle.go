package pbft_all

import (
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/shard"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

//

// 该函数仅由主节点调用，如果请求正确，则主节点将块发送回消息发送者。
func (p *PbftConsensusNode) Propose() { //Propose()函数用于提出新的块。它不需要参数。它不返回任何值。
	if p.view != p.NodeID {
		return
	} //如果节点不是主节点，则返回。在PBFT中，只有主节点负责提议新区块
	for { //无限for循环。该循环用于不断提出新块
		select { //使用一条select语句来处理停止信号。如果在通道上收到信号p.pStop，该函数将打印一条停止消息并返回，从而有效地停止提议过程。
		case <-p.pStop:
			p.pl.Plog.Printf("S%dN%d stop...\n", p.ShardID, p.NodeID)
			return
		default:
		}
		time.Sleep(time.Duration(int64(p.pbftChainConfig.BlockInterval)) * time.Millisecond) //使用time.Sleep函数使主节点休眠一段时间。这段时间是由BlockInterval参数指定的。此睡眠间隔控制提出新块的速率。

		p.sequenceLock.Lock()                                                                               //使用p.sequenceLock锁定共识序列。这是一个互斥锁，用于确保它具有对序列的独占访问权，在提出新块时不会发生竞争。
		p.pl.Plog.Printf("S%dN%d get sequenceLock locked, now trying to propose...\n", p.ShardID, p.NodeID) //打印一条日志消息，指示节点已锁定序列。
		// propose
		//实现接口来生成提案
		_, r := p.ihm.HandleinPropose() //使用HandleinPropose函数生成提案。它返回一个布尔值和一个指向message.Request结构的指针。布尔值指示是否生成了提案。如果没有生成提案，则该函数将返回false。如果生成了提案，则该函数将返回true，并且指向新块的指针将存储在r变量中。

		digest := getDigest(r)                                                              //使用getDigest函数计算提案的摘要。它需要一个参数： r（类型为*message.Request）：这是一个指向message.Request结构的指针。它返回一个指向摘要的指针。
		p.requestPool[string(digest)] = r                                                   //将提案存储在请求池中。它使用摘要作为键，使用提案作为值。这样，可以使用摘要来检索提案。
		p.pl.Plog.Printf("S%dN%d put the request into the pool ...\n", p.ShardID, p.NodeID) //打印一条日志消息，指示节点已将提案存储在请求池中。

		ppmsg := message.PrePrepare{ //使用message.PrePrepare结构创建一个新的PrePrepare消息。它包含请求消息、摘要和序列ID。
			RequestMsg: r,
			Digest:     digest,
			SeqID:      p.sequenceID,
		}
		p.height2Digest[p.sequenceID] = string(digest) //将摘要存储在高度到摘要的映射中。这样，可以使用高度来检索摘要。
		//编组和广播
		ppbyte, err := json.Marshal(ppmsg) //使用json.Marshal函数将PrePrepare消息编组为字节存储在ppbyte。它返回一个字节切片和一个错误。如果在编组过程中发生错误，则该函数将出现紧急情况并记录错误。
		if err != nil {
			log.Panic()
		}
		msg_send := message.MergeMessage(message.CPrePrepare, ppbyte)            //使用message.MergeMessage函数将PrePrepare消息与消息类型合并为字节。它返回一个字节切片。这个字节切片将用于广播PrePrepare消息。
		networks.Broadcast(p.RunningNode.IPaddr, p.getNeighborNodes(), msg_send) //使用networks.Broadcast函数广播PrePrepare消息。它被发送到从 p.getNeighborNodes() 获得的相邻节点的 IP 地址。它需要三个参数： p.RunningNode.IPaddr：这是一个字符串，表示当前节点的IP地址。 p.getNeighborNodes()：这是一个指向shard.Node结构的指针的切片。它包含当前节点的所有邻居节点。 msg_send：这是一个字节切片，包含要广播的消息。
	}
}

func (p *PbftConsensusNode) handlePrePrepare(content []byte) { //handlePrePrepare函数用于处理PrePrepare消息。它需要一个参数： content（类型为[]字节）：这是一个字节切片，包含PrePrepare消息。
	p.RunningNode.PrintNode()
	fmt.Println("received the PrePrepare ...")
	// decode the message
	ppmsg := new(message.PrePrepare)      //使用message.PrePrepare结构创建一个新的PrePrepare消息。
	err := json.Unmarshal(content, ppmsg) //使用json.Unmarshal函数将PrePrepare消息解组为ppmsg。它需要两个参数： content（类型为[]字节）：这是一个字节切片，包含PrePrepare消息。 ppmsg（类型为*message.PrePrepare）：这是一个指向message.PrePrepare结构的指针，用于存储解组的消息。
	if err != nil {
		log.Panic(err)
	}
	flag := false                                                                      //创建一个布尔变量flag，用于指示是否应该广播Prepare消息。
	if digest := getDigest(ppmsg.RequestMsg); string(digest) != string(ppmsg.Digest) { //使用getDigest函数计算请求消息的摘要。如果摘要与PrePrepare消息中的摘要不匹配，则打印一条日志消息，指示节点拒绝准备。
		p.pl.Plog.Printf("S%dN%d : the digest is not consistent, so refuse to prepare. \n", p.ShardID, p.NodeID)
	} else if p.sequenceID < ppmsg.SeqID {
		p.requestPool[string(getDigest(ppmsg.RequestMsg))] = ppmsg.RequestMsg
		p.height2Digest[ppmsg.SeqID] = string(getDigest(ppmsg.RequestMsg))
		p.pl.Plog.Printf("S%dN%d : the Sequence id is not consistent, so refuse to prepare. \n", p.ShardID, p.NodeID)
	} else {
		// do your operation in this interface
		flag = p.ihm.HandleinPrePrepare(ppmsg)
		p.requestPool[string(getDigest(ppmsg.RequestMsg))] = ppmsg.RequestMsg
		p.height2Digest[ppmsg.SeqID] = string(getDigest(ppmsg.RequestMsg))
	}
	// if the message is true, broadcast the prepare message
	if flag {
		pre := message.Prepare{
			Digest:     ppmsg.Digest,
			SeqID:      ppmsg.SeqID,
			SenderNode: p.RunningNode,
		}
		prepareByte, err := json.Marshal(pre)
		if err != nil {
			log.Panic()
		}
		// broadcast
		msg_send := message.MergeMessage(message.CPrepare, prepareByte)
		networks.Broadcast(p.RunningNode.IPaddr, p.getNeighborNodes(), msg_send)
		p.pl.Plog.Printf("S%dN%d : has broadcast the prepare message \n", p.ShardID, p.NodeID)
	}
}

func (p *PbftConsensusNode) handlePrepare(content []byte) { //handlePrepare函数用于处理Prepare消息。它需要一个参数： content（类型为[]字节）：这是一个字节切片，包含Prepare消息。
	p.pl.Plog.Printf("S%dN%d : received the Prepare ...\n", p.ShardID, p.NodeID)
	// decode the message
	pmsg := new(message.Prepare)
	err := json.Unmarshal(content, pmsg)
	if err != nil {
		log.Panic(err)
	}

	if _, ok := p.requestPool[string(pmsg.Digest)]; !ok {
		p.pl.Plog.Printf("S%dN%d : doesn't have the digest in the requst pool, refuse to commit\n", p.ShardID, p.NodeID)
	} else if p.sequenceID < pmsg.SeqID {
		p.pl.Plog.Printf("S%dN%d : inconsistent sequence ID, refuse to commit\n", p.ShardID, p.NodeID)
	} else {
		// if needed more operations, implement interfaces
		p.ihm.HandleinPrepare(pmsg)

		p.set2DMap(true, string(pmsg.Digest), pmsg.SenderNode)
		cnt := 0
		for range p.cntPrepareConfirm[string(pmsg.Digest)] {
			cnt++
		}
		// the main node will not send the prepare message
		specifiedcnt := int(2 * p.malicious_nums)
		if p.NodeID != p.view {
			specifiedcnt -= 1
		}

		// if the node has received 2f messages (itself included), and it haven't committed, then it commit
		p.lock.Lock()
		defer p.lock.Unlock()
		if cnt >= specifiedcnt && !p.isCommitBordcast[string(pmsg.Digest)] {
			p.pl.Plog.Printf("S%dN%d : is going to commit\n", p.ShardID, p.NodeID)
			// generate commit and broadcast
			c := message.Commit{
				Digest:     pmsg.Digest,
				SeqID:      pmsg.SeqID,
				SenderNode: p.RunningNode,
			}
			commitByte, err := json.Marshal(c)
			if err != nil {
				log.Panic()
			}
			msg_send := message.MergeMessage(message.CCommit, commitByte)
			networks.Broadcast(p.RunningNode.IPaddr, p.getNeighborNodes(), msg_send)
			p.isCommitBordcast[string(pmsg.Digest)] = true
			p.pl.Plog.Printf("S%dN%d : commit is broadcast\n", p.ShardID, p.NodeID)
		}
	}
}

func (p *PbftConsensusNode) handleCommit(content []byte) { //handleCommit函数用于处理Commit消息。它需要一个参数： content（类型为[]字节）：这是一个字节切片，包含Commit消息。
	// decode the message
	cmsg := new(message.Commit)
	err := json.Unmarshal(content, cmsg)
	if err != nil {
		log.Panic(err)
	}
	p.pl.Plog.Printf("S%dN%d received the Commit from ...%d\n", p.ShardID, p.NodeID, cmsg.SenderNode.NodeID)
	p.set2DMap(false, string(cmsg.Digest), cmsg.SenderNode)
	cnt := 0
	for range p.cntCommitConfirm[string(cmsg.Digest)] {
		cnt++
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	// the main node will not send the prepare message
	required_cnt := int(2 * p.malicious_nums)
	if cnt >= required_cnt && !p.isReply[string(cmsg.Digest)] {
		p.pl.Plog.Printf("S%dN%d : has received 2f + 1 commits ... \n", p.ShardID, p.NodeID)
		// if this node is left behind, so it need to requst blocks
		if _, ok := p.requestPool[string(cmsg.Digest)]; !ok {
			p.isReply[string(cmsg.Digest)] = true
			p.askForLock.Lock()
			// request the block
			sn := &shard.Node{
				NodeID:  p.view,
				ShardID: p.ShardID,
				IPaddr:  p.ip_nodeTable[p.ShardID][p.view],
			}
			orequest := message.RequestOldMessage{
				SeqStartHeight: p.sequenceID + 1,
				SeqEndHeight:   cmsg.SeqID,
				ServerNode:     sn,
				SenderNode:     p.RunningNode,
			}
			bromyte, err := json.Marshal(orequest)
			if err != nil {
				log.Panic()
			}

			p.pl.Plog.Printf("S%dN%d : is now requesting message (seq %d to %d) ... \n", p.ShardID, p.NodeID, orequest.SeqStartHeight, orequest.SeqEndHeight)
			msg_send := message.MergeMessage(message.CRequestOldrequest, bromyte)
			networks.TcpDial(msg_send, orequest.ServerNode.IPaddr)
		} else {
			// implement interface
			p.ihm.HandleinCommit(cmsg)
			p.isReply[string(cmsg.Digest)] = true
			p.pl.Plog.Printf("S%dN%d: this round of pbft %d is end \n", p.ShardID, p.NodeID, p.sequenceID)
			p.sequenceID += 1
		}

		// if this node is a main node, then unlock the sequencelock
		if p.NodeID == p.view {
			p.sequenceLock.Unlock()
			p.pl.Plog.Printf("S%dN%d get sequenceLock unlocked...\n", p.ShardID, p.NodeID)
		}
	}
}

// 该函数仅由主节点调用
// 如果请求正确，主节点会将块发送回消息发送者。
// 现在这个函数可以发送块和分区
// RequestOldMessage 可能用于协议或应用中的某些需求，以请求旧消息数据，通常由节点之间进行通信以满足某些需要。
// 消息的具体内容和用途通常取决于协议或应用的设计。根据上面提供的代码片段，这些消息通常由主节点用于向其他节点请求旧消息。
func (p *PbftConsensusNode) handleRequestOldSeq(content []byte) { //handleRequestOldSeq函数用于处理RequestOldMessage消息。它需要一个参数： content（类型为[]字节）：这是一个字节切片，包含RequestOldMessage消息。
	//1.检查当前节点是否为主节点
	if p.view != p.NodeID { //如果节点不是主节点，则清空消息内容 content 并返回。在PBFT中，只有主节点负责提议新区块
		content = make([]byte, 0)
		return
	}

	//2.解析消息内容
	rom := new(message.RequestOldMessage)
	err := json.Unmarshal(content, rom) //使用json.Unmarshal函数将RequestOldMessage消息解组为rom。它需要两个参数： content（类型为[]字节）：这是一个字节切片，包含RequestOldMessage消息。 rom（类型为*message.RequestOldMessage）：这是一个指向message.RequestOldMessage结构的指针，用于存储解组的消息。
	if err != nil {
		log.Panic()
	}

	//3.记录消息接收情况
	p.pl.Plog.Printf("S%dN%d : received the old message requst from ...", p.ShardID, p.NodeID) //打印一条日志消息，包括分片ID、节点ID和发送消息的节点信息，指示节点已收到来自rom.SenderNode的旧消息请求。
	rom.SenderNode.PrintNode()                                                                 //打印rom.SenderNode的信息

	//4.处理旧消息请求
	oldR := make([]*message.Request, 0)                                      //创建一个新的message.Request结构的切片oldR。它将用于存储旧消息。
	for height := rom.SeqStartHeight; height <= rom.SeqEndHeight; height++ { //使用for循环遍历rom.SeqStartHeight到rom.SeqEndHeight之间的所有高度。在每次迭代中，将当前高度的消息添加到oldR中。
		if _, ok := p.height2Digest[height]; !ok { //对于每个高度，检查 p.height2Digest 中是否存在与该高度对应的摘要，如果不存在，则记录错误日志并中断循环
			p.pl.Plog.Printf("S%dN%d : has no this digest to this height %d\n", p.ShardID, p.NodeID, height)
			break
		}
		if r, ok := p.requestPool[p.height2Digest[height]]; !ok { //如果摘要存在，继续检查 p.requestPool 中是否存在与该摘要对应的请求消息，如果不存在，则记录错误日志并中断循环
			p.pl.Plog.Printf("S%dN%d : has no this message to this digest %d\n", p.ShardID, p.NodeID, height)
			break
		} else { //如果摘要和请求消息都存在，则将请求消息添加到 oldR 中
			oldR = append(oldR, r)
		}
	}
	p.pl.Plog.Printf("S%dN%d : has generated the message to be sent\n", p.ShardID, p.NodeID) //打印一条日志消息，指示节点已生成要发送的消息。

	p.ihm.HandleReqestforOldSeq(rom) //使用HandleReqestforOldSeq函数处理旧消息请求。它需要一个参数： rom（类型为*message.RequestOldMessage）：这是一个指向message.RequestOldMessage结构的指针，包含旧消息请求。

	// send the block back
	sb := message.SendOldMessage{ //创建一个 SendOldMessage 结构，包括开始和结束高度，以及旧请求消息 oldR 和发送消息的节点信息
		SeqStartHeight: rom.SeqStartHeight,
		SeqEndHeight:   rom.SeqEndHeight,
		OldRequest:     oldR,
		SenderNode:     p.RunningNode,
	}
	sbByte, err := json.Marshal(sb) //将该结构转换为字节切片 sbByte
	if err != nil {
		log.Panic()
	}
	msg_send := message.MergeMessage(message.CSendOldrequest, sbByte) //使用 message.MergeMessage 函数将消息类型和内容合并为字节切片 msg_send
	networks.TcpDial(msg_send, rom.SenderNode.IPaddr)                 //使用 networks.TcpDial 函数将消息发送回 rom.SenderNode。它需要两个参数： msg_send：这是一个字节切片，包含要发送的消息。 rom.SenderNode.IPaddr：这是一个字符串，表示消息的发送者节点的IP地址。。
	p.pl.Plog.Printf("S%dN%d : send blocks\n", p.ShardID, p.NodeID)   //记录消息发送的日志，包括分片ID和节点ID
}

// 节点向主节点请求区块并接收区块
func (p *PbftConsensusNode) handleSendOldSeq(content []byte) { //handleSendOldSeq函数用于处理SendOldMessage消息。它需要一个参数： content（类型为[]字节）：这是一个字节切片，包含SendOldMessage消息。
	som := new(message.SendOldMessage)  //使用message.SendOldMessage结构创建一个新的SendOldMessage消息。
	err := json.Unmarshal(content, som) //使用json.Unmarshal函数将SendOldMessage消息解组为som。它需要两个参数： content（类型为[]字节）：这是一个字节切片，包含SendOldMessage消息。 som（类型为*message.SendOldMessage）：这是一个指向message.SendOldMessage结构的指针，用于存储解组的消息。
	if err != nil {
		log.Panic()
	}
	p.pl.Plog.Printf("S%dN%d : has received the SendOldMessage message\n", p.ShardID, p.NodeID) //打印一条日志消息，指示节点已收到SendOldMessage消息。

	// 实现新共识的接口
	p.ihm.HandleforSequentialRequest(som) //使用HandleforSequentialRequest函数处理SendOldMessage消息。它需要一个参数： som（类型为*message.SendOldMessage）：这是一个指向message.SendOldMessage结构的指针，包含SendOldMessage消息。
	beginSeq := som.SeqStartHeight        //使用 som.SeqStartHeight 作为开始序列
	for idx, r := range som.OldRequest {  //使用for循环遍历 som.OldRequest 中的所有请求消息。在每次迭代中，将请求消息添加到请求池中，并将高度到摘要的映射添加到高度到摘要的映射中。这样，可以使用高度来检索摘要。
		p.requestPool[string(getDigest(r))] = r
		p.height2Digest[uint64(idx)+beginSeq] = string(getDigest(r))
		p.isReply[string(getDigest(r))] = true
		p.pl.Plog.Printf("this round of pbft %d is end \n", uint64(idx)+beginSeq)
	}
	p.sequenceID = som.SeqEndHeight + 1                     //使用 som.SeqEndHeight 作为下一个序列ID
	if rDigest, ok1 := p.height2Digest[p.sequenceID]; ok1 { //使用 p.sequenceID 作为高度，检查高度到摘要的映射中是否存在与该高度对应的摘要。如果存在，则检查请求池中是否存在与该摘要对应的请求消息。如果存在，则使用该请求消息生成PrePrepare消息，并使用该消息生成Prepare消息。然后，使用Prepare消息生成Commit消息。最后，使用Commit消息生成Reply消息。这些消息将被广播到所有邻居节点。
		if r, ok2 := p.requestPool[rDigest]; ok2 {
			ppmsg := &message.PrePrepare{
				RequestMsg: r,
				SeqID:      p.sequenceID,
				Digest:     getDigest(r),
			}
			flag := false
			flag = p.ihm.HandleinPrePrepare(ppmsg)
			if flag {
				pre := message.Prepare{
					Digest:     ppmsg.Digest,
					SeqID:      ppmsg.SeqID,
					SenderNode: p.RunningNode,
				}
				prepareByte, err := json.Marshal(pre)
				if err != nil {
					log.Panic()
				}
				// broadcast
				msg_send := message.MergeMessage(message.CPrepare, prepareByte)
				networks.Broadcast(p.RunningNode.IPaddr, p.getNeighborNodes(), msg_send)
				p.pl.Plog.Printf("S%dN%d : has broadcast the prepare message \n", p.ShardID, p.NodeID)
			}
		}
	}

	p.askForLock.Unlock()
}

//这个函数的主要目的是处理节点之间的通信，特别是与主节点之间的通信，用于请求旧的区块并接收它们，同时在适当的时候生成共识阶段的消息。
