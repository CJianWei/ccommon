package util

import (
	"fmt"
	"github.com/CJianWei/ccommon/db"
	"github.com/astaxie/beego/logs"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTest struct {
	BaseTest

	/*
		以下为单个测试方法需要使用的参数
	*/

	Index             int               //	下标索引，用于通过取模记录数据
	DenominatorRecord int               //	被除数分母,每过多少个点记录一次
	Points            *Points           //	记录近n个点的状态
	RunningTmp        int               //	阶段性运行的并发数
	per_add           int               //	每秒叠加的请求数
	Duration          int64             // 	每隔多久时长更新并发数
	timeClock         chan int          //	每过一定时间即打印一次
	EagerDuration     int64             //	希望压测试的时长
	EndS              bool              //	是否停止发送
	EndR              chan string       //	是否停止接收
	PerCount          int               //	在过去某段时间内的响应总数
	PerSuccessAve     float64           //	在过去某段时间内成功请求的均值
	PerErrRate        float64           //	在过去某段时间内的错误率
	PerNormalRate     float64           //	在过去某段时间的正确率
	PerCentRate       map[float64]int64 //	截止目前为止，各个纬度的百分比
	Median            int64             //	耗时中位数
	ErrorRate         float64           //	错误率
	tidx_             int32             // 	设置初始 index
	tidx_tmp          int32             // 	备份

}

func NewLoadTest() *LoadTest {
	return &LoadTest{}
}

//新构建的并发demo
func (load *LoadTest) Build() *LoadTest {
	new_load := NewLoadTest()
	// baseTest params init
	new_load.TotalCount = load.getTotalCount(1000)
	new_load.InitCount = load.getInitCount(10)
	new_load.EagerCurrent = load.getEagerCurrent(20)
	new_load.logf = load.getLogF("log.log")
	new_load.lck = &sync.RWMutex{}
	new_load.MonitorObj = NewTimeCost()
	new_load.MonitorObj.StartRecord()
	new_load.MonitorFunc = load.getMonitorFunc(func() error {
		return nil
	})
	// other params init
	new_load.Index = 0
	new_load.DenominatorRecord = load.getDenominatorRecord(20)
	new_load.Points = NewPoints2(true)
	new_load.RunningTmp = new_load.InitCount
	new_load.per_add = load.getPer(10)
	new_load.Duration = load.getDuration(2000)
	new_load.timeClock = make(chan int)
	new_load.EagerDuration = load.getEagerDuration(5 * 60 * 1000) // default 5 min
	new_load.EndS = false
	new_load.EndR = make(chan string)
	new_load.PerCentRate = map[float64]int64{}
	new_load.ErrorRate = 0
	new_load.tidx_ = load.GetTid(0)
	new_load.tidx_tmp = new_load.tidx_
	new_load.AdjustEagerCurrent()
	return new_load
}

func (load *LoadTest) Clear() {
	load.EndS = true
	load.EndR <- "end"
	load.MonitorObj.StopRecord()
	load.Points.Stop()
	load.MonitorFunc = func() error {
		return nil
	}
}

func (load *LoadTest) ExportConf() Map {
	return Map{
		"initCount":       load.InitCount,
		"totalCount":      load.TotalCount,
		"per":             load.per_add,
		"EagerConcurrent": load.EagerCurrent,
		"duration":        load.Duration,
		"eagerDuration":   load.EagerDuration,
		"tid":             load.tidx_,
	}
}

func (load *LoadTest) GetTid(defaultTid int32) int32 {
	if load.tidx_ <= 0 {
		load.SetTid(defaultTid)
	}
	if load.tidx_ > int32(load.TotalCount-1) {
		return int32(load.TotalCount - 1)
	} else {
		return load.tidx_
	}

}

func (load *LoadTest) SetTid(tid_ int32) *LoadTest {
	load.tidx_ = tid_
	load.tidx_tmp = tid_
	return load
}

// 设置 截取节点的模基数
func (load *LoadTest) SetDenominatorRecord(denominatorRecord int) *LoadTest {
	if denominatorRecord < 20 {
		denominatorRecord = 20
	}
	load.DenominatorRecord = denominatorRecord
	return load
}

func (load *LoadTest) getDenominatorRecord(defaultDenominatorRecord int) int {
	if load.DenominatorRecord <= 0 {
		load.SetDenominatorRecord(defaultDenominatorRecord)
	}
	return load.DenominatorRecord
}

//设置迭代量
func (load *LoadTest) SetPer(per_add int) *LoadTest {
	load.per_add = per_add
	return load
}

func (load *LoadTest) getPer(defaultPer int) int {
	if load.per_add <= 0 {
		load.SetPer(defaultPer)
	}
	return load.per_add
}

// 设置监控方法
func (load *LoadTest) SetMonitorFunc(monitor func() error) *LoadTest {
	load.MonitorFunc = monitor
	return load
}

func (load *LoadTest) getMonitorFunc(defaultMonitor func() error) func() error {
	if load.MonitorFunc == nil {
		load.MonitorFunc = defaultMonitor
	}
	return load.MonitorFunc
}

func (load *LoadTest) DefaultMonitorFunc(arg map[string]interface{}) func() error {
	tags, ok1 := arg["tags"].(map[string]string)
	In, ok2 := arg["In"].(*db.Influx)
	_, ok3 := arg["log"].(bool)
	var f = func() error {
		if ok1 && ok2 {
			In.AddPointSync("Running", tags, map[string]interface{}{"value": load.Running})
			In.AddPointSync("EagerCurrent", tags, map[string]interface{}{"value": load.EagerCurrent})
			In.AddPointSync("RunningTmp", tags, map[string]interface{}{"value": load.RunningTmp})
			In.AddPointSync("PerErrRate", tags, map[string]interface{}{"value": load.CalPerErrRate()})
			In.AddPointSync("PerNormalRate", tags, map[string]interface{}{"value": load.CalPerNormalRate()})
			In.AddPointSync("PerCount", tags, map[string]interface{}{"value": load.CalPerCount()})
			In.AddPointSync("PerSuccessAve", tags, map[string]interface{}{"value": load.CalPerSuccessAve()})
			In.AddPointSync(fmt.Sprintf("%v", RATE_65), tags, map[string]interface{}{"value": load.CalPerSuccessAve()})
			In.AddPointSync(fmt.Sprintf("%v", RATE_85), tags, map[string]interface{}{"value": load.CalPerSuccessAve()})
			In.AddPointSync(fmt.Sprintf("%v", RATE_95), tags, map[string]interface{}{"value": load.CalPerSuccessAve()})
			In.AddPointSync(fmt.Sprintf("%v", RATE_99), tags, map[string]interface{}{"value": load.CalPerSuccessAve()})
			In.AddPointSync("Median", tags, map[string]interface{}{"value": load.Median})
		}
		if ok3 {
			logs.Informational("Running(%v) EagerCurrent(%v) RunningTmp(%v) PerErrRate(%v) PerNormalRate(%v) PerCount(%v) PerSuccessAve(%v) %v(%vms)  %v(%vms)  %v(%vms)  %v(%vms) Median(%v)",
				load.Running,
				load.EagerCurrent,
				load.RunningTmp,
				load.CalPerErrRate(),
				load.CalPerNormalRate(),
				load.CalPerCount(),
				load.CalPerSuccessAve(),
				RATE_65, load.PerCentRate[RATE_65],
				RATE_85, load.PerCentRate[RATE_85],
				RATE_95, load.PerCentRate[RATE_95],
				RATE_99, load.PerCentRate[RATE_99],
				load.Median,
			)
		}

		return nil
	}
	return f
}

func (load *LoadTest) CalPerErrRate() float64 {
	return load.Points.CalPerErrRate(false, load.PerErrRate)
}

func (load *LoadTest) CalPerNormalRate() float64 {
	return load.Points.CalPerNormalRate(false, load.PerNormalRate)
}

func (load *LoadTest) CalPerSuccessAve() float64 {
	return load.Points.CalPerSuccessAve(false, load.PerSuccessAve)
}

func (load *LoadTest) CalPerCount() int {
	return load.Points.CalPerCount(false, load.PerCount)
}

// 设置运行时长
func (load *LoadTest) SetTotalCount(totalCount int) *LoadTest {
	load.TotalCount = totalCount
	return load
}

func (load *LoadTest) getTotalCount(defaultTotalCount int) int {
	if load.TotalCount <= 0 {
		load.SetTotalCount(defaultTotalCount)
	}
	return load.TotalCount
}

// 设置初始运行次数
func (load *LoadTest) SetInitCount(initCount int) *LoadTest {
	load.InitCount = initCount
	return load
}

func (load *LoadTest) getInitCount(defaultInitCount int) int {
	if load.InitCount <= 0 {
		load.SetInitCount(defaultInitCount)
	}
	return load.InitCount
}

// 设置预期的并发数
func (load *LoadTest) SetEagerCurrent(eagerCurrent int) *LoadTest {
	load.EagerCurrent = eagerCurrent
	return load
}

func (load *LoadTest) getEagerCurrent(defaultEagerCurrent int) int {
	if load.EagerCurrent <= 0 {
		load.SetEagerCurrent(defaultEagerCurrent)
	}
	return load.EagerCurrent
}

// 设置 日志路径
func (load *LoadTest) SetLogF(logf string) *LoadTest {
	load.logf = logf
	return load
}

func (load *LoadTest) getLogF(defaultLogf string) string {
	if load.logf == "" {
		load.SetLogF(defaultLogf)
	}
	return load.logf
}

// 设置预期运行时长
func (load *LoadTest) SetEagerDuration(eagerDuration int64) *LoadTest {
	load.EagerDuration = eagerDuration
	return load
}

func (load *LoadTest) getEagerDuration(defaultEagerDuration int64) int64 {
	if load.EagerDuration <= 0 {
		load.SetEagerDuration(defaultEagerDuration)
	}
	return load.EagerDuration
}

//获取测试方法总的耗时时长
func (load *LoadTest) RecordRunfCost(btime int64) {
	load.RunCost = Now() - btime
}

func (load *LoadTest) GetRunCost() int64 {
	return load.RunCost
}

//设置并发数更新时长
func (load *LoadTest) SetDuration(duration int64) *LoadTest {
	load.Duration = duration
	return load
}

func (load *LoadTest) getDuration(defaultDuration int64) int64 {
	if load.Duration <= 0 {
		load.SetDuration(defaultDuration)
	}
	return load.Duration
}

func (load *LoadTest) FlushPointsRate() {
	// 计算各个百分比对应的 耗时情况
	load.PerCentRate, load.Median, load.ErrorRate = load.Points.PercentRate(map[float64]int64{
		RATE_65: 1,
		RATE_85: 1,
		RATE_95: 1,
		RATE_99: 1,
	})
}

func (load *LoadTest) QPS() float64 {
	rs := float64(load.tidx_ - load.tidx_tmp)
	cost := float64(load.GetRunCost())
	if cost == 0 {
		return -1
	} else {
		return Round(1000*rs/cost, 2)
	}
}

//动态调整并发数
func (load *LoadTest) AdjustEagerCurrent() {
	btime := Now()
	go func() {
		for {
			if load.EndS {
				break
			}
			time.Sleep(time.Duration(load.Duration) * time.Millisecond)
			if load.EndS {
				break
			}
			load.timeClock <- 1
		}
	}()

	go func() {
		for {
			select {
			case <-load.EndR:
				break
			case <-load.timeClock:
				load.FlushPointsRate()

				// time out quit directorly
				if Now()-btime >= load.EagerDuration {
					load.RunningTmp = 0
					continue
				}
				// adjust the concurrent
				if load.RunningTmp < load.EagerCurrent {
					tmp := load.RunningTmp + load.per_add
					if tmp > load.EagerCurrent {
						load.RunningTmp = load.EagerCurrent
					} else {
						load.RunningTmp = tmp
					}
				}
				// sub recent points
				etime := Now()
				btime := etime - int64(load.Duration)
				load.Points.SubPoints(btime, etime)
				func() {
					load.Points.PointLock.Lock()
					defer load.Points.PointLock.Unlock()
					load.PerSuccessAve = load.Points.CalPerSuccessAve(true, load.PerSuccessAve)
					load.PerCount = load.Points.CalPerCount(true, load.PerCount)
					load.PerErrRate = load.Points.CalPerErrRate(true, load.PerErrRate)
					load.PerNormalRate = load.Points.CalPerNormalRate(true, load.PerNormalRate)
				}()
			}
		}
	}()
}

//单个用例执行结束之后的操作
func (load *LoadTest) perDone(btime, etime int64, err error) {
	load.lck.Lock()
	defer load.lck.Unlock()
	load.Index++
	if load.Running > load.RunningMax {
		load.RunningMax = load.Running
	}
	//load.Running = load.Running - 1
	atomic.AddInt32(&load.Running, -1)
	var monitor_key = ""
	if err == nil {
		load.SuccessCost += etime - btime
		load.SuccessCount += 1
		monitor_key = SUCCESS
	} else {
		monitor_key = FAILED
		load.FailedCount += 1
	}
	load.MonitorObj.MonitorRecord(monitor_key, btime, etime)
	load.Points.WritePoints(btime, etime, err)
	//当开启监控的时候，每过n个点记录一次数据
	if load.Index%load.DenominatorRecord == 0 && load.Index/load.DenominatorRecord > 0 {
		load.MonitorFunc()
	}
}

func (load *LoadTest) BeforeRun() {
	if len(load.logf) > 0 {
		logs.Reset()
		logs.SetLogger(logs.AdapterFile, fmt.Sprintf(`{"filename":"%s","level":7,"maxlines":0,"maxsize":0,"daily":false}`, load.logf))
		logs.EnableFuncCallDepth(true)
		logs.SetLogFuncCallDepth(3)
		logs.Async(1e3)
	}
}

func (load *LoadTest) AfterRun(btime int64) {
	//记录方法总的耗时时长
	load.RecordRunfCost(btime)
	// 刷新 monitor
	load.MonitorObj.Flush()
	//刷新 ps
	load.Points.Flush()
	// 最后计算结果
	load.FlushPointsRate()
	// 将结果打印出来
	logs.Informational("reslut of call LoadTest %v", S2Json(load.MonitorObj.Load()))

	if len(load.logf) > 0 {
		logs.Reset()
		logs.SetLogger(logs.AdapterConsole)
	}
}

func (load *LoadTest) Run(call func(int) error) error {
	load.BeforeRun()
	defer load.AfterRun(Now())
	var total_count = load.TotalCount
	var init_count = load.InitCount

	var ws = sync.WaitGroup{}

	var run_call func(int)
	var run_next func(int, int)
	var pre_run_next func(int)

	run_call = func(v int) {
		defer ws.Done()
		atomic.AddInt32(&load.Running, 1)
		perbeg := Now()
		terr := call(v)
		load.perDone(perbeg, Now(), terr)
		pre_run_next(v)
	}

	var added = &sync.Mutex{}
	pre_run_next = func(v int) {
		added.Lock()
		defer added.Unlock()
		//RunningTmp 为阶段兵法书目标
		if load.Running < int32(load.RunningTmp) && load.EndS == false {
			run_next(v, int(int32(load.RunningTmp)-load.Running))
		}
	}

	run_next = func(v int, per_add_int int) {
		for i := 0; i < per_add_int; i++ {
			ridx := int(atomic.AddInt32(&load.tidx_, 1))
			if ridx >= total_count {
				break
			}
			ws.Add(1)
			go run_call(ridx)
		}
	}

	atomic.AddInt32(&load.tidx_, int32(init_count-1))
	b_index := int(0 + load.tidx_)
	e_index := int(int32(init_count) + load.tidx_)
	for i := b_index; i < e_index; i++ {
		ws.Add(1)
		go run_call(i)
	}
	ws.Wait()

	return nil
}
