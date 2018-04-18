package db

import (
	"github.com/Centny/gwf/log"
	"github.com/Centny/gwf/util"
	"github.com/influxdata/influxdb/client/v2"
	"testing"
	"github.com/Centny/gwf/tutil"
)

// @Time    : 2018/3/26 18:06
// @Author  : chenjw
// @Site    :
// @File    : influx_test.go
// @Software: GoLand

func TestA(t *testing.T) {
	In := NewInflux("http://192.168.38.187:8086", "chenjianwei", "chenjianwei369", "go_test", "ns", 1<<15)
	In.InitDb()
	go In.Tick()


	tags := map[string]string{
		"cpu": "cpu-total",
	}
	fields := map[string]interface{}{
		"value": 10.2,
	}
	measurement := "cpu_usage"
	now := util.Now()
	for j:=0;j<5;j++{
		tutil.DoPerf(10,"log.log", func(i int) {
			pts := []*client.Point{}
			for index := 0; index < 100; index++ {
				//time.Sleep(time.Millisecond)
				pt, err := In.newPt(measurement, tags, fields)
				if err != nil {
					t.Error(err)
					return
				}
				pts = append(pts, pt)
			}
			In.AddPoints(pts)
		})
	}
	log.D("cost %v", util.Now()-now)

	now2 := util.Now()
	for j:=0;j<5;j++{
		tutil.DoPerf(10,"log.log", func(i int){
			pts := []*client.Point{}
			for index := 0; index < 100; index++ {
				//time.Sleep(time.Millisecond)
				pt, err := In.newPt(measurement, tags, fields)
				if err != nil {
					t.Error(err)
					return
				}
				pts = append(pts, pt)
				In.AddPoints(pts)
			}
		})

	}
	log.D("cost %v", util.Now()-now2)


	now3 := util.Now()
	for j:=0;j<5;j++{
		tutil.DoPerf(10,"log.log", func(i int){
			pts := []*client.Point{}
			for index := 0; index < 100; index++ {
				//time.Sleep(time.Millisecond)
				pt, err := In.newPt(measurement, tags, fields)
				if err != nil {
					t.Error(err)
					return
				}
				pts = append(pts, pt)
			}
			In.AddPointsSync(pts)
		})

	}
	In.Flush()
	log.D("cost %v", util.Now()-now3)




	now4 := util.Now()
	for j:=0;j<5;j++{
		tutil.DoPerf(10,"log.log", func(i int){
			pts := []*client.Point{}
			for index := 0; index < 100; index++ {
				//time.Sleep(time.Millisecond)
				pt, err := In.newPt(measurement, tags, fields)
				if err != nil {
					t.Error(err)
					return
				}
				pts = append(pts, pt)
			}
			In.AddPointsSync(pts)
		})
	}
	In.Flush()
	log.D("cost %v", util.Now()-now4)


}
