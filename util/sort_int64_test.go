package util

import (
	"github.com/astaxie/beego/logs"
	"sort"
	"testing"
)

func TestNewInt64Slice(t *testing.T) {
	d := NewInt64Slice([]int64{6, 5, 4, 3, 2, 1})
	sort.Sort(d)
	if d.Data[0] > d.Data[1]{
		t.Error("sort err")
		return
	}
	logs.Informational("data :%v", d.Data)
}

