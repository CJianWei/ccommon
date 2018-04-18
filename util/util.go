package util

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
	"math"
)

// @Time    : 2018/3/27 10:01
// @Author  : chenjw
// @Site    :
// @File    : util.go
// @Software: GoLand

func Timestamp(t time.Time) int64 {
	return t.Local().UnixNano() / 1e6
}

func Time(timestamp int64) time.Time {
	return time.Unix(0, timestamp*1e6)
}

func Now() int64 {
	return Timestamp(time.Now())
}

func GOMAXPROCS(num int) {
	runtime.GOMAXPROCS(num)
}

func CPU() int {
	i := runtime.NumCPU()
	if i < 2 {
		return i
	} else {
		return i - 1
	}
}
func Json2Map(data string) (Map, error) {
	md := Map{}
	d := json.NewDecoder(strings.NewReader(data))
	err := d.Decode(&md)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("decode to json error(%v) by data(%v)", err.Error(), data))
	}
	return md, nil
}

func S2Json(v interface{}) string {
	bys, _ := json.Marshal(v)
	return string(bys)
}

func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func Round(f float64, n int) float64 {
	pow10_n := math.Pow10(n)
	return math.Trunc((f+0.5/pow10_n)*pow10_n) / pow10_n
}