package util

import (
	"github.com/astaxie/beego/logs"
	"testing"
)

func TestNewPoint(t *testing.T) {
	var ps = NewPoints2(true)

	for i := 20000000; i > 2; i-- {
		ps.WritePoints(1, int64(i), nil)
	}
	now := Now()
	end := Now() - now

	ps.LoadDetails()

	logs.Informational("data %v", end)

}

