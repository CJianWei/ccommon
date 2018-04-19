package util

import (
	"testing"
)

// @Time    : 2018/4/19 9:48
// @Author  : chenjw
// @Site    :
// @File    : load_perf_test.go.go
// @Software: GoLand

var test_url_load = "http://127.0.0.1:8888/index.html"

func TestNewLoadTest(t *testing.T) {
	load := NewLoadTest()
	load = load.SetLogF("log.log").
		SetTotalCount(1000).
		SetInitCount(5).
		SetEagerCurrent(10).
		SetPer(1).
		SetDuration(2000).
		SetEagerDuration(5 * 60 * 1000).
		SetDenominatorRecord(100).
		Build()
	var param = map[string]interface{}{
		"log": true,
	}
	load.SetMonitorFunc(load.DefaultMonitorFunc(param))
	load.Run(func(i int) error {
		_, _, err := HGet(test_url_load, nil, nil)
		if err != nil {
			return err
		}
		return nil
	})
	load.Clear()
}
