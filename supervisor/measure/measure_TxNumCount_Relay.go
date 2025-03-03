package measure

import "blockEmulator/message"

// to test cross-transaction rate
type TestTxNumCount_Relay struct { //TestTxNumCount_Relay结构包含中继消息的各种信息
	epochID int       //标识测试场景中的特定时期
	txNum   []float64 //每个时期的交易数量
}

func NewTestTxNumCount_Relay() *TestTxNumCount_Relay { //NewTestTxNumCount_Relay函数用于创建和配置中继消息的测试场景
	return &TestTxNumCount_Relay{ //返回一个新的TestTxNumCount_Relay结构
		epochID: -1,
		txNum:   make([]float64, 0),
	}
}

func (ttnc *TestTxNumCount_Relay) OutputMetricName() string { //OutputMetricName()函数用于测量的数据名称
	return "Tx_number"
}

func (ttnc *TestTxNumCount_Relay) UpdateMeasureRecord(b *message.BlockInfoMsg) { //UpdateMeasureRecord()函数用于更新测量记录
	if b.BlockBodyLength == 0 { // empty block
		return
	} //如果区块体长度为0，则表明该块体没有交易
	epochid := b.Epoch //获取当前时期
	// extend
	for ttnc.epochID < epochid { //如果当前时期大于测试场景中的特定时期，则将测试场景中的特定时期加1，并将0添加到ttnc.txNum中
		ttnc.txNum = append(ttnc.txNum, 0)
		ttnc.epochID++
	}

	ttnc.txNum[epochid] += float64(len(b.ExcutedTxs)) //将已完全执行的交易数量添加到ttnc.txNum中
}

func (ttnc *TestTxNumCount_Relay) HandleExtraMessage([]byte) {} //HandleExtraMessage()函数用于处理额外的消息

func (ttnc *TestTxNumCount_Relay) OutputRecord() (perEpochCTXs []float64, totTxNum float64) { //OutputRecord()函数用于输出测量记录
	perEpochCTXs = make([]float64, 0) //创建一个新的浮点数切片，表示每个时期的中继事务数
	totTxNum = 0.0                    //初始化totTxNum，代表所有时期的中继交易总数
	for _, tn := range ttnc.txNum {   //遍历ttnc.txNum
		perEpochCTXs = append(perEpochCTXs, tn) //使用 append 将ttnc.txNum 中的每个值（在循环中表示为 tn）附加到 perEpochCTXs 切片。
		totTxNum += tn                          //通过将 tn 的值添加到 totTxNum 变量来累积中继交易计数。
	}
	return perEpochCTXs, totTxNum //该方法返回 perEpochCTXs 切片（包含每个纪元的交易计数）和 totTxNum 变量（包含总交易计数）。
}
