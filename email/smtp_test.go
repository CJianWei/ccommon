package email

import "testing"

// @Time    : 2018/4/23 上午9:53
// @Author  : chenjw
// @Site    :
// @File    : smtp_test.go
// @Software: GoLand

func TestCaseSent(t *testing.T) {

	username := "username"
	password := "password"
	host := "smtp.qq.com"
	port := "25"
	recipients := []string{"aa@qq.com"}

	msg := "<html><body><p> 你好 <body><html>"
	err := NewSmtp().
		Init(username, "", password, host, port).
		SendEmail("标题", msg, recipients, true)
	if err != nil {
		t.Error(err)
		return
	}
}
