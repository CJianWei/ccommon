package email

// @Time    : 2018/4/23 上午9:52
// @Author  : chenjw
// @Site    :
// @File    : smtp.go
// @Software: GoLand

import (
	"crypto/tls"
	"errors"
	"net"
	"net/smtp"
	"strings"
)

type SMTP struct {
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	Host               string   `json:"host"`
	Port               string   `json:"port"`
	Subject            string   `json:"subject"`
	FromAddress        string   `json:"fromAddress"`
	RecipientAddresses []string `json:"sendTos"`
}

func NewSmtp() *SMTP {
	return &SMTP{}
}

func (this *SMTP) Init(username string, fromAddress string, password string, host string, port string) *SMTP {
	this.Username = username
	this.Password = password
	this.Host = host
	this.Port = port
	this.FromAddress = fromAddress
	return this
}

func (this *SMTP) IsEmpty(str string) bool {
	if strings.Trim(str, " ") == "" {
		return true
	} else {
		return false
	}
}

func (this *SMTP) getAuth() (smtp.Auth, error) {
	if this.IsEmpty(this.Username) || this.IsEmpty(this.Password) || this.IsEmpty(this.Host) {
		return nil, errors.New("call [getAuth] but some of the params are empty")
	}
	return smtp.PlainAuth("", this.Username, this.Password, this.Host), nil
}

func (this *SMTP) SendEmail(subject string, msg string, recipients []string, sendByTls bool, content_types ...string) error {
	if this.IsEmpty(this.FromAddress) {
		this.FromAddress = this.Username
	}
	if this.IsEmpty(this.Port) {
		return errors.New("call [SendEmail] by port do not be init yeah")
	}

	auth, err := this.getAuth()
	if err != nil {
		return err
	}
	var hostAddressWithPort = this.Host + ":" + this.Port
	var content_type = "Content-Type: text/plain; charset=UTF-8"
	if len(content_types) > 0 {
		content_type = content_types[0]
	}
	var content = []byte("To: " + strings.Join(recipients, ",") + "\r\nFrom: " + this.FromAddress +
		"<" + this.FromAddress + ">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + msg)
	if sendByTls {
		return this.SendEmailV(hostAddressWithPort, auth, this.FromAddress, recipients, content)
	} else {
		return smtp.SendMail(hostAddressWithPort, auth, this.FromAddress, recipients, content)
	}

}

func (this *SMTP) SendEmailV(hostAddressWithPort string, auth smtp.Auth, fromAddress string, recipients []string, content []byte) error {
	client, err := smtp.Dial(hostAddressWithPort)
	if err != nil {
		return err
	}

	host, _, _ := net.SplitHostPort(hostAddressWithPort)
	tlsConn := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}
	if err = client.StartTLS(tlsConn); err != nil {
		return err
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(fromAddress); err != nil {
		return err
	}

	for _, rec := range recipients {
		if err = client.Rcpt(rec); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()

}
