package util

import (
	"fmt"
	"github.com/CJianWei/ccommon/db"
	"github.com/astaxie/beego/logs"
)

// @Time    : 2018/3/27 10:09
// @Author  : chenjw
// @Site    :
// @File    : dynamic_perf.go
// @Software: GoLand
// @desc	: dynamic count concurrent base on load test

type DynamicTest struct {
	LoadTest *LoadTest
	/*
		以下为单个测试方法需要使用的属性
	*/
	PerCentRateEager map[float64]int64 //各个响应节点的耗时上限
	ErrRate          float64           //允许的错误百分比
	NextConcurrent   int               //下一步的并发数
	TotalCount       int               //总的运行次数
	TotalCountLeft   int               //剩下运行次数
	logf             string
}

func NewDynamicTest() *DynamicTest {
	new_p := &DynamicTest{}
	new_p.LoadTest = NewLoadTest()
	new_p.NextConcurrent = new_p.LoadTest.getInitCount(10)
	new_p.PerCentRateEager = new_p.getPerCentRateEager()
	new_p.ErrRate = new_p.getErrRate(0.001)
	new_p.TotalCount = new_p.getTotalCount(10000)
	new_p.TotalCountLeft = new_p.TotalCount
	new_p.logf = new_p.getLogF("logD.log")
	return new_p
}

// 设置运行时长
func (p *DynamicTest) SetTotalCount(totalCount int) *DynamicTest {
	p.TotalCount = totalCount
	return p
}

func (p *DynamicTest) getTotalCount(defaultTotalCount int) int {
	if p.TotalCount <= 0 {
		p.SetTotalCount(defaultTotalCount)
	}
	return p.TotalCount
}

// 设置 日志路径
func (p *DynamicTest) SetLogF(logf string) *DynamicTest {
	p.logf = logf
	return p
}

func (p *DynamicTest) getLogF(defaultLogf string) string {
	if p.logf == "" {
		p.SetLogF(defaultLogf)
	}
	return p.logf
}

// 设置百分比响应耗时上限
func (p *DynamicTest) SetPerCentRateEager(perCentRateEager map[float64]int64) *DynamicTest {
	p.PerCentRateEager = perCentRateEager
	return p
}

func (p *DynamicTest) getPerCentRateEager(defaultPerCentRateEager ...map[float64]int64) map[float64]int64 {
	if p.PerCentRateEager == nil || len(p.PerCentRateEager) == 0 {
		if len(defaultPerCentRateEager) > 0 {
			p.SetPerCentRateEager(defaultPerCentRateEager[0])
		} else {
			p.SetPerCentRateEager(map[float64]int64{RATE_95: 500})
		}

	}
	return p.PerCentRateEager
}

//设置错误百分比
func (p *DynamicTest) SetErrRate(errRate float64) *DynamicTest {
	p.ErrRate = errRate
	return p
}

func (p *DynamicTest) getErrRate(defaultErrRate float64) float64 {
	if p.ErrRate <= 0 {
		p.SetErrRate(defaultErrRate)
	}
	return p.ErrRate
}

func (p *DynamicTest) formate(m map[float64]int64) map[string]interface{} {
	ret_m := map[string]interface{}{}
	for k, v := range m {
		ret_m[fmt.Sprintf("%v", k)] = v
	}
	return ret_m
}

func (p *DynamicTest) ExportConf() Map {
	return Map{
		"PerCentRateEager": p.formate(p.PerCentRateEager),
		"ErrRate":          p.ErrRate,
		"TotalCount":       p.TotalCount,
	}
}

func (p *DynamicTest) AdjustNextConcurrent(rate map[float64]int64, errRate float64) {
	var down = func() {
		p.NextConcurrent = p.NextConcurrent - p.LoadTest.per_add
		if p.NextConcurrent <= 0 {
			p.NextConcurrent = 2
		}
	}
	var up = func() {
		p.NextConcurrent = p.NextConcurrent + p.LoadTest.per_add
	}

	for f_64, i_64 := range p.PerCentRateEager {
		for f_64_i, i_64_i := range rate {
			// 某个指标超出上限了
			if f_64 == f_64_i && i_64 < i_64_i {
				down()
				return
			}
		}
	}
	//错误率比较高
	if errRate > p.ErrRate {
		down()
		return
	}
	up()
	return
}

func (p *DynamicTest) BeforeRecord() {
	if len(p.logf) > 0 {
		logs.Reset()
		logs.SetLogger(logs.AdapterFile, fmt.Sprintf(`{"filename":"%s","level":7,"maxlines":0,"maxsize":0,"daily":false}`, p.logf))
		logs.EnableFuncCallDepth(true)
		logs.SetLogFuncCallDepth(3)
		logs.Async(1e3)
	}
}

func (p *DynamicTest) AfterRecord() {
	if len(p.logf) > 0 {
		logs.Reset()
		logs.SetLogger(logs.AdapterConsole)
	}
}

func (p *DynamicTest) Run(call func(int) error, extras ...Map) error {
	var tidx_ int32 = 0
	p.NextConcurrent = p.LoadTest.InitCount
	var for_count int = 0

	var open_log = true
	var open_In = false
	var tags = map[string]string{}
	var In *db.Influx = nil
	if len(extras) > 0 {
		var ok1, ok2 bool
		tags, ok1 = extras[0]["tags"].(map[string]string)
		In, ok2 = extras[0]["In"].(*db.Influx)
		if ok1 && ok2 && len(tags) > 0 && In != nil {
			open_In = true
		}
		_, open_log = extras[0]["log"].(bool)
	}

	for {
		for_count++
		if p.TotalCount-1 <= int(tidx_) {
			return nil
		}

		if p.TotalCountLeft < p.LoadTest.InitCount || p.TotalCountLeft < p.NextConcurrent {
			return nil
		}

		loadTest := p.LoadTest.Build()
		loadTest.SetTid(tidx_)
		loadTest.SetTotalCount(p.TotalCount)

		if p.NextConcurrent < p.LoadTest.InitCount {
			loadTest.SetInitCount(p.NextConcurrent)
		} else {
			loadTest.SetInitCount(p.LoadTest.InitCount)
		}
		loadTest.SetEagerCurrent(p.NextConcurrent)
		var params = map[string]interface{}{"log": true}
		loadTest.SetMonitorFunc(loadTest.DefaultMonitorFunc(params))
		conf := loadTest.ExportConf()

		loadTest.Run(call)
		loadTest.Clear()
		stepRecord := Map{
			"concurrent":        loadTest.EagerCurrent,
			"success_ave":       loadTest.MonitorObj.LoadSingle(SUCCESS).Ave,
			"success_max":       loadTest.MonitorObj.LoadSingle(SUCCESS).Max,
			"success_min":       loadTest.MonitorObj.LoadSingle(SUCCESS).Min,
			"success_count":     loadTest.MonitorObj.LoadSingle(SUCCESS).Times,
			"err_count":         loadTest.MonitorObj.LoadSingle(FAILED).Times,
			"err_rate":          loadTest.ErrorRate,
			"median":            loadTest.Median,
			"95_lines":          loadTest.PerCentRate[RATE_95],
			"65_lines":          loadTest.PerCentRate[RATE_65],
			"85_lines":          loadTest.PerCentRate[RATE_85],
			"99_lines":          loadTest.PerCentRate[RATE_99],
			"cost":              loadTest.RunCost,
			"qps":               loadTest.QPS(),
			"err_tolerance":     p.ErrRate,
			"timeout_tolerance": p.formate(p.PerCentRateEager),
		}

		p.AdjustNextConcurrent(loadTest.PerCentRate, loadTest.ErrorRate)
		tidx_ = loadTest.GetTid(0) + 1
		p.TotalCountLeft = p.TotalCount - int(tidx_)

		var log_to_file = func() {
			p.BeforeRecord()
			defer p.AfterRecord()
			logs.Informational("for number are: %v", for_count)
			logs.Informational("config: %v", S2Json(conf))
			logs.Informational("single result: %v", S2Json(stepRecord))
			logs.Informational("tid are: %v", tidx_)
			logs.Informational("TotalCountLeft are: %v", p.TotalCountLeft)
			logs.Informational("=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=")
			logs.Informational("=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=")
			logs.Informational("")
		}

		var log_to_Influx = func() {
			for _k, _v := range stepRecord {
				if _k == "timeout_tolerance" {
					continue
				} else {
					In.AddPointSync(_k, tags, map[string]interface{}{"value": _v})
				}
			}

		}
		if open_log {
			log_to_file()
		}
		if open_In {
			log_to_Influx()
		}
	}
	return nil
}
