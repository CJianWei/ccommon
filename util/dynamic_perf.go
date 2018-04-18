package util

import (
	"fmt"
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
	StepRecord       []Map             //每次运行的记录
}

func NewDynamicTest() *DynamicTest {
	new_p := &DynamicTest{}
	new_p.LoadTest = NewLoadTest()
	new_p.NextConcurrent = new_p.LoadTest.getInitCount(10)
	new_p.PerCentRateEager = new_p.getPerCentRateEager()
	new_p.ErrRate = new_p.getErrRate(0.001)
	new_p.TotalCount = new_p.getTotalCount(10000)
	new_p.TotalCountLeft = new_p.TotalCount
	new_p.StepRecord = []Map{}
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

func (p *DynamicTest) Run(call func(int) error) error {
	var tidx_ int32 = 0
	p.NextConcurrent = p.LoadTest.InitCount
	var for_count int = 0
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

		//p.StepRecord = append(p.StepRecord, stepRecord)
		p.AdjustNextConcurrent(loadTest.PerCentRate, loadTest.ErrorRate)
		tidx_ = loadTest.GetTid(0) + 1
		p.TotalCountLeft = p.TotalCount - int(tidx_)

		fmt.Println("for number are:", for_count)
		fmt.Println("config:", S2Json(conf))
		fmt.Println("single result:", S2Json(stepRecord))
		fmt.Println("tid are:", tidx_)
		fmt.Println("TotalCountLeft are:", p.TotalCountLeft)
		fmt.Println("=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=")
		fmt.Println("=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=")
		fmt.Println()
	}
	return nil
}
