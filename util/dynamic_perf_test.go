package util

import (
	"github.com/CJianWei/ccommon/db"
	"testing"
)

// @Time    : 2018/4/19 9:50
// @Author  : chenjw
// @Site    :
// @File    : dynamic_perf_test.go
// @Software: GoLand

var test_url_dynamic = "http://127.0.0.1:8888/index.html"

/*
result will look like below
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:292] for number are: 1
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:293] config: {"EagerConcurrent":10,"duration":1000,"eagerDuration":20000,"initCount":10,"per":2,"tid":0,"totalCount":10000000}
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:294] single result: {"65_lines":66,"85_lines":86,"95_lines":477,"99_lines":1303,"concurrent":10,"cost":21492,"err_count":0,"err_rate":0,"err_tolerance":0.01,"median":56,"qps":84.31,"success_ave":112,"success_count":1813,"success_max":4936,"success_min":23,"timeout_tolerance":{"0.85":500}}
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:295] tid are: 1813
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:296] TotalCountLeft are: 9998187
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:297] =*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=
2018/04/19 09:55:09.042 [I] [dynamic_perf.go:298] =*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=*=
*/
func TestNewDynamicTest(t *testing.T) {
	var username = "username"
	var password = "password"
	In := db.NewInflux("http://192.168.38.187:8086", username, password, "go_test", "ns", 1<<15)
	In.InitDb()
	var tags = map[string]string{"api": "ccc"}
	var params = Map{
		"tags": tags,
		"In":   In,
		"log":  true,
	}
	p := NewDynamicTest().SetTotalCount(10000000).SetErrRate(0.01).SetPerCentRateEager(map[float64]int64{RATE_85: 500})
	p.LoadTest.SetInitCount(10).SetPer(2).SetEagerDuration(20 * 1000).SetDuration(1000)
	p.Run(func(i int) error {
		_, _, err := HGet(test_url_dynamic, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}, params)
}
