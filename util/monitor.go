package util

import (
	"fmt"
	"sync"
)

// @Time    : 2018/3/27 11:18
// @Author  : chenjw
// @Site    :
// @File    : monitor.go
// @Software: GoLand

var RecordExtra bool = true

var Cost *TimeCost

type Record struct {
	Max         int64            //最大耗时
	Min         int64            //最小耗时
	Ave         int64            //平均耗时
	Total       int64            //总耗时
	Times       int64            //总的记录次数
	Range       map[string]int64 //每一秒的访问次数
	HourRange   map[string]int64 //每个小时的访问次数
	MinuteRange map[string]int64 //每十分钟的访问次数
	MsRange     map[string]int64 //线阶段以每100 ms 为一个阶段点
}

type Sign struct {
	Key   string //记录耗时的关键词
	Btime int64  //记录耗时的开始时间
	Etime int64  //记录耗时的结束时间
}

type TimeCost struct {
	Mval        map[string]*Record //所有对象的耗时记录
	Lock        *sync.Mutex        //锁
	Sign        chan *Sign         //管道信号量
	End         chan string        //结束
	RecordExtra map[string]int     //额外的记录信息
}

//拷贝记录对象
func (r *Record) CopyRecord() *Record {
	return &Record{
		Max:         r.Max,
		Min:         r.Min,
		Ave:         r.Ave,
		Total:       r.Total,
		Times:       r.Times,
		Range:       r.Range,
		HourRange:   r.HourRange,
		MinuteRange: r.MinuteRange,
		MsRange:     r.MsRange,
	}
}

//初始化监控工具
func NewTimeCost() *TimeCost {
	return &TimeCost{
		Mval: map[string]*Record{},
		Lock: &sync.Mutex{},
		Sign: make(chan *Sign, 1<<15),
		End:  make(chan string),
		RecordExtra: map[string]int{
			"Range":       1,
			"HourRange":   1,
			"MinuteRange": 1,
			"MsRange":     1,
		},
	}
}

//记录某个接口的耗时,异步操作，不阻塞
func (timec *TimeCost) MonitorRecord(key string, btime int64, etime int64) {
	timec.Sign <- &Sign{Key: key, Btime: btime, Etime: etime}
}

//开启异步线程 记录具体的耗时情况
func (timec *TimeCost) StartRecord() {
	go func() {
		for {
			select {
			case sign := <-timec.Sign:
				timec.RecordDetail(sign.Key, sign.Btime, sign.Etime)
			case <-timec.End:
				break
			}
		}
	}()
}

func (timec *TimeCost) Flush() {
	for {
		if len(timec.Sign) > 0 {
			sign := <-timec.Sign
			timec.RecordDetail(sign.Key, sign.Btime, sign.Etime)
			continue
		}
		break
	}
}

func (timec *TimeCost) StopRecord() {
	timec.End <- "end"
}

//实际记录详情的方法
func (timec *TimeCost) RecordDetail(key string, btime int64, etime int64) {
	timec.Lock.Lock()
	defer timec.Lock.Unlock()
	if timec.Mval[key] == nil {
		timec.Mval[key] = &Record{
			Min:         -1,
			Range:       map[string]int64{},
			HourRange:   map[string]int64{},
			MinuteRange: map[string]int64{},
			MsRange:     map[string]int64{},
		}
	}
	var r = timec.Mval[key]
	v := etime - btime
	r.Times++
	r.Total += v
	if v > r.Max {
		r.Max = v
	}
	if r.Min < 0 {
		r.Min = v
	} else if r.Min > v {
		r.Min = v
	}
	if timec.RecordExtra["Range"] > 0 {
		r.Range[fmt.Sprintf("%d~%d(s)", int64(v/1000), int64(v/1000+1))]++
	}

	//获取结束时间戳 对应的 年月日 小时 分钟
	t := Time(etime)

	if timec.RecordExtra["HourRange"] > 0 {
		hour_str := fmt.Sprintf("%v-%v-%v %v(h)", int(t.Year()), int(t.Month()), int(t.Day()), int(t.Hour()))
		r.HourRange[hour_str]++
	}

	if timec.RecordExtra["MinuteRange"] > 0 {
		minute_str := fmt.Sprintf("%v-%v-%v %v:%v(10min)", int(t.Year()), int(t.Month()), int(t.Day()), int(t.Hour()), int(t.Minute())/10*10)
		r.MinuteRange[minute_str]++
	}

	if timec.RecordExtra["MsRange"] > 0 {
		r.MsRange[fmt.Sprintf("%d~%d(ms)", int64(v/100*100), int64((v/100+1)*100))]++
	}

}

//加载监控数据，不过 这种模型的设计不适合永久性监控，因为内存的 占用空间会无限制的上升
func (timec *TimeCost) Load() map[string]*Record {
	timec.Lock.Lock()
	defer timec.Lock.Unlock()
	var dst = map[string]*Record{}
	DeepCopy(&dst, &timec.Mval)
	for _, v := range dst {
		if v.Times > 0 {
			v.Ave = int64(v.Total / v.Times)
		}
	}
	return dst
}

func (timec *TimeCost) LoadSingle(key string) *Record {
	load := timec.Load()
	if load[key] == nil {
		return &Record{
			Min:         -1,
			Range:       map[string]int64{},
			HourRange:   map[string]int64{},
			MinuteRange: map[string]int64{},
			MsRange:     map[string]int64{},
		}
	} else {
		return load[key]
	}
}

func InitMonitor() {
	Cost = NewTimeCost()
}

func MonitorRecord(key string, btime int64) {
	Cost.MonitorRecord(key, btime, Now())
}

func Load() map[string]*Record {
	return Cost.Load()
}
