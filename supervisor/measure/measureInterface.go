package measure

import "blockEmulator/message"

type MeasureModule interface { //MeasureModule接口定义了测量模块的接口
	UpdateMeasureRecord(*message.BlockInfoMsg)
	HandleExtraMessage([]byte)
	OutputMetricName() string           //OutputMetricName()函数用于输出测量的数据名称
	OutputRecord() ([]float64, float64) //OutputRecord()函数用于输出测量记录
}
