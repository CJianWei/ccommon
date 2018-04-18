package util

import (
	"sort"
	"sync"
)

// @Time    : 2018/4/10 17:35
// @Author  : chenjw
// @Site    :
// @File    : points.go
// @Software: GoLand

//记录单个节点的信息
type Point struct {
	Btime int64
	Etime int64
	Cost  int64
	Err   bool
}

func NewPoint(Btime, Etime int64, err error) *Point {
	var err_bool = false
	if err != nil {
		err_bool = true
	}
	return &Point{
		Btime: Btime,
		Etime: Etime,
		Cost:  Etime - Btime,
		Err:   err_bool,
	}
}

// 对于节点数据进行操作
type Points struct {
	Ps        []*Point
	Details   []int64     // 正常数据
	AbNormals []int64     // 异常数据
	RecordE   bool        // 记录其他的数据
	PointLock *sync.Mutex // 数组操作加锁
	RawLimit  int64       // 裸数据条数
	Index     int64       // 当前的索引
	Pipe      chan *Point // 节点管道
	End       chan string // 结束记录
}

func NewPoints() *Points {
	return NewPoints2(false)
}

func NewPoints2(recordE bool) *Points {
	ps := &Points{
		Ps:        []*Point{},
		PointLock: &sync.Mutex{},
		Details:   []int64{},
		AbNormals: []int64{},
		RecordE:   recordE,
		RawLimit:  10000000,
		Pipe:      make(chan *Point, 1<<15),
		End:       make(chan string),
	}
	ps.SyncPipe()
	return ps
}

func (ps *Points) Stop() {
	ps.End <- "end"
}

func (ps *Points) sync(single_point *Point) {
	ps.PointLock.Lock()
	defer ps.PointLock.Unlock()
	ps.Index++
	ps.Ps = append(ps.Ps, single_point)
	// record raw data
	if ps.RecordE {
		if single_point.Err == false {
			ps.Details = append(ps.Details, single_point.Cost)
		} else {
			ps.AbNormals = append(ps.AbNormals, single_point.Cost)
		}
	}
	// sub raw data if need of it will cost a lot of mem
	if ps.Index > ps.RawLimit && ps.Index%10000 == 0 {
		var len_d = len(ps.Details)
		var len_a = len(ps.AbNormals)
		if int64(len_d) > ps.RawLimit {
			ps.Details = ps.Details[int(int64(len_d)-ps.RawLimit):]
		}
		if int64(len_a) > ps.RawLimit {
			ps.AbNormals = ps.AbNormals[int(int64(len_a)-ps.RawLimit):]
		}
	}
}

func (ps *Points) SyncPipe() {
	go func() {
		for {
			select {
			case single_point := <-ps.Pipe:
				ps.sync(single_point)
			case <-ps.End:
				break
			}
		}
	}()
}

func (ps *Points) Flush() {
	for {
		if len(ps.Pipe) > 0 {
			single_point := <-ps.Pipe
			ps.sync(single_point)
			continue
		}
		break
	}
}

func (ps *Points) WritePoints(btime, etime int64, err error) {
	ps.Pipe <- NewPoint(btime, etime, err)
}

func (ps *Points) CopyAry(src []int64) []int64 {
	ps.PointLock.Lock()
	defer ps.PointLock.Unlock()
	var dst = []int64{}
	DeepCopy(&dst, &src)
	return dst
}

func (ps *Points) LoadDetails() []int64 {
	return ps.CopyAry(ps.Details)
}

func (ps *Points) LoadAbNormals() []int64 {
	return ps.CopyAry(ps.AbNormals)
}

func (ps *Points) LoadAllDetails() []int64 {
	return append(ps.CopyAry(ps.Details), ps.CopyAry(ps.AbNormals)...)
}

func (ps *Points) Sort(src []int64) []int64 {
	int64Slice := NewInt64Slice(src)
	sort.Sort(int64Slice)
	return int64Slice.Data
}

func (ps *Points) PercentRate(m map[float64]int64, status ...interface{}) (map[float64]int64, int64, float64) {
	var ret_m = map[float64]int64{}
	var median int64 = -1
	var err_rate float64 = 0
	if m == nil {
		return ret_m, median, err_rate
	}
	var default_status = "all"
	if len(status) > 0 {
		default_status, _ = status[0].(string)
	}
	var int64Slice = []int64{}
	if default_status == "all" {
		int64Slice = ps.LoadAllDetails()
	} else if default_status == "success" {
		int64Slice = ps.LoadDetails()
	} else {
		int64Slice = ps.LoadAbNormals()
	}
	int64Slice = ps.Sort(int64Slice)
	var i64_len = len(int64Slice)

	for f_64, _ := range m {
		if i64_len <= 0 {
			ret_m[f_64] = -1
		} else {
			ret_m[f_64] = int64Slice[int(float64(i64_len)*f_64)]
		}
	}

	if i64_len > 0 {
		median = int64Slice[i64_len/2]
	}

	err_len := len(ps.AbNormals)
	Normal_len := len(ps.Details)
	if err_len+Normal_len > 0 {
		err_rate = Round(float64(err_len)/(float64(err_len+Normal_len)), 5)
	}
	return ret_m, median, err_rate

}

//只是截取 某段时间区间内的数据即可
func (ps *Points) SubPoints(b, e int64) {
	ps.Flush()
	ps.PointLock.Lock()
	defer ps.PointLock.Unlock()
	var tmp = []*Point{}
	for _, v := range ps.Ps {
		if v.Etime >= b && v.Etime <= e {
			tmp = append(tmp, v)
		}
	}
	ps.Ps = tmp
}

//计算超时率 或者错误率的百分比
func (ps *Points) isAbove(rate float64, call func(p *Point) int) bool {
	if len(ps.Ps) <= 0 {
		return false
	} else {
		var count = 0
		var eager_count = 0
		for _, v := range ps.Ps {
			count++
			eager_count += call(v)
		}
		if float64(eager_count)/float64(count) > rate {
			return true
		} else {
			return false
		}
	}
}

//计算超时的点是否大于预期
func (ps *Points) IsAboveTimeOutRate(timeOutRate float64, timeOut int64) bool {
	return ps.isAbove(timeOutRate, func(point *Point) int {
		if point.Err == false && point.Cost > timeOut {
			return 1
		} else {
			return 0
		}
	})
}

//计算错误的点是否超过预期
func (ps *Points) IsAboveErrRate(errRate float64) bool {
	return ps.isAbove(errRate, func(point *Point) int {
		if point.Err == true {
			return 1
		} else {
			return 0
		}
	})
}

func (ps *Points) CalPerWithInRate(reCal bool, timeOut int64, defaultPerWithInRate float64) float64 {
	if reCal == true {
		if len(ps.Ps) <= 0 {
			defaultPerWithInRate = 1
		} else {
			var count = 0
			for _, point := range ps.Ps {
				if point.Err == false && point.Cost <= timeOut {
					count++
				}
			}
			defaultPerWithInRate = float64(count) / float64(len(ps.Ps))
		}
	}
	return defaultPerWithInRate
}

//计算超时率
func (ps *Points) CalPerTimeOutRate(reCal bool, timeOut int64, defaultPerTimeOutRate float64) float64 {
	if reCal == true {
		if len(ps.Ps) <= 0 {
			defaultPerTimeOutRate = 0
		} else {
			var count = 0
			for _, point := range ps.Ps {
				if point.Err == false && point.Cost > timeOut {
					count++
				}
			}
			defaultPerTimeOutRate = float64(count) / float64(len(ps.Ps))
		}
	}
	return defaultPerTimeOutRate
}

func (ps *Points) CalPerNormalRate(reCal bool, defaultNormalRate float64) float64 {
	if reCal == true {
		if len(ps.Ps) <= 0 {
			defaultNormalRate = 1
		} else {
			var count = 0
			for _, point := range ps.Ps {
				if point.Err == false {
					count++
				}
			}
			defaultNormalRate = float64(count) / float64(len(ps.Ps))
		}
	}
	return defaultNormalRate
}

func (ps *Points) CalPerErrRate(reCal bool, defaultPerErrRate float64) float64 {
	if reCal == true {
		if len(ps.Ps) <= 0 {
			defaultPerErrRate = 0
		} else {
			var count = 0
			for _, point := range ps.Ps {
				if point.Err == true {
					count++
				}
			}
			defaultPerErrRate = float64(count) / float64(len(ps.Ps))
		}
	}
	return defaultPerErrRate
}

//计算平均耗时
func (ps *Points) CalPerSuccessAve(reCal bool, defaultPerSuccessAve float64) float64 {
	if reCal == true {
		if len(ps.Ps) <= 0 {
			defaultPerSuccessAve = 0
		} else {
			var cost_count int64 = 0
			var index_count int = 0
			for _, point := range ps.Ps {
				if point.Err == false {
					cost_count += point.Cost
					index_count++
				}
			}
			if index_count > 0 {
				defaultPerSuccessAve = float64(cost_count) / float64(index_count)
			} else {
				defaultPerSuccessAve = 0
			}

		}
	}
	return defaultPerSuccessAve
}

func (ps *Points) CalPerCount(reCal bool, defaultPerCount int) int {
	if reCal == true {
		defaultPerCount = len(ps.Ps)
	}
	return defaultPerCount
}
