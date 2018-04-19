package util

import (
	"github.com/astaxie/beego/logs"
	"sort"
	"testing"
)

func TestNewInt64Slice(t *testing.T) {
	d := NewInt64Slice([]int64{6, 5, 4, 3, 2, 1})
	sort.Sort(d)
	logs.Informational("data :%v", d.Data)
}
