package util

import (
	"errors"
	"fmt"
)

// @Time    : 2018/3/27 9:55
// @Author  : chenjw
// @Site    :
// @File    : err.go
// @Software: GoLand

func Err(f string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(f, args...))
}
