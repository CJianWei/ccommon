package util

import "sync"

// @Time    : 2018/4/10 16:27
// @Author  : chenjw
// @Site    :
// @File    : common_perf.go
// @Software: GoLand

var adapters = make(map[string]newIgTest)

type newIgTest func() IgTest

type IgTest interface {
	SetLogF(logf string)                         // set file path to store log
	SetExtraParams(m Map)                        // set other params
	SetMonitorFunc()                             // set monitor func to monitor the program
	AdjustEagerCurrent()                         // to adjust the concurrent due to acutal situation
	Run(call func(int) error)                    // run case
	perDone(btime, etime int64, err error) error // do when single case finished
	GetRunCost(btime int64) int64                // duration of case cost
}

const (
	SUCCESS_AND_TIMEOUT          = "success_and_timeout"
	SUCCESS_AND_WITHIN_TIMELIMIT = "success_and_with_timeLimit"
	FAILED                       = "Failed"
	SUCCESS                      = "success"
)

const (
	RATE_65 float64 = 0.65
	RATE_85 float64 = 0.85
	RATE_95 float64 = 0.95
	RATE_99 float64 = 0.99
)

func Register(name string, ig newIgTest) {
	if ig == nil {
		panic("logs: Register provide is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("logs: Register called twice for provider " + name)
	}
	adapters[name] = ig
}

type BaseTest struct {
	SuccessCost  int64        //成功请求的耗时
	SuccessCount int          //成功请求的次数
	FailedCount  int          //失败请求的次数
	RunCost      int64        //执行所有的请求的总耗时
	MonitorObj   *TimeCost    // 数据监控方式
	MonitorFunc  func() error //自定义监控方式
	TotalCount   int          //总的运行次数
	InitCount    int          //初始化值
	logf         string
	Running      int32         //正在运行的并发数
	RunningMax   int32         //最高并发数
	EagerCurrent int           //每个阶段预期的并发数
	lck          *sync.RWMutex //读写加锁
}
