package signal

import (
	"sync"
)

// 判断监听者何时向领导发送停止消息
type StopSignal struct { // StopSignal用于控制停止消息发送
	stoplock sync.Mutex // stoplock用于保护stopGap

	stopGap       int // stopGap用于记录接收到的空交易列表的数量
	stopThreshold int // stopThreshold用于记录停止消息发送的阈值，如果 StopGap 不小于 stopThreshold，则应向领导者发送停止消息。
}

func NewStopSignal(stop_Threshold int) *StopSignal { // NewStopSignal用于创建一个新的停止信号
	return &StopSignal{
		stopGap:       0,              // stopGap初始化为0
		stopThreshold: stop_Threshold, // stopThreshold初始化为stop_Threshold
	}
}

// 当接收到一个空的交易列表时，调用该函数来增加stopGap
func (ss *StopSignal) StopGap_Inc() {
	ss.stoplock.Lock()
	defer ss.stoplock.Unlock()
	ss.stopGap++
}

// 当收到一条已执行txs的消息时，调用此函数重置stopGap
func (ss *StopSignal) StopGap_Reset() {
	ss.stoplock.Lock()
	defer ss.stoplock.Unlock()
	ss.stopGap = 0
}

// 检查stopGap是否足够
// 如果 StopGap 不小于 stopThreshold，则应向领导者发送停止消息。
func (ss *StopSignal) GapEnough() bool {
	ss.stoplock.Lock()
	defer ss.stoplock.Unlock()
	return ss.stopGap >= ss.stopThreshold
}
