package util

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// @Time    : 2018/3/27 9:49
// @Author  : chenjw
// @Site    :
// @File    : http.go
// @Software: GoLand

func DefaultHttpConfig() Map {
	return Map{
		"timeout": 20000,
	}
}

func getClient(extra Map) *http.Client {
	c := &http.Client{}
	transport := &http.Transport{}
	var needTransport bool = false
	if extra != nil {
		if extra.Exist("proxy") && extra.StrVal("proxy") != "" {
			urli := url.URL{}
			urlproxy, _ := urli.Parse(extra.StrVal("proxy"))
			transport.Proxy = http.ProxyURL(urlproxy)
			needTransport = true
		}

		if extra.Exist("timeout") && extra.IntVal("timeout") > 0 {
			time_out := extra.IntVal("timeout")
			transport.Dial = func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(time.Duration(time_out) * time.Millisecond)
				conn, err := net.DialTimeout(netw, addr, time.Millisecond*time.Duration(int(time_out/2)))
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(deadline)
				return conn, nil
			}
			needTransport = true
		}

	}
	if needTransport {
		c.Transport = transport
	}
	return c
}

func HGet(url_addr string, header map[string]string, extra Map) (int, string, error) {
	c := getClient(extra)

	req, err := http.NewRequest("GET", url_addr, nil)
	if err != nil {
		return 0, "", err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, "", err
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return resp.StatusCode, string(body), nil
	}
}

func HGetM(url_addr string, header map[string]string, extra Map) (int, Map, error) {
	code, res, err := HGet(url_addr, header, extra)
	if err != nil {
		return code, Map{}, err
	} else {
		m, err := Json2Map(res)
		if err != nil {
			return 0, nil, err
		}
		return code, m, err
	}
}

func CreateFormBody(fields map[string]string) (string, *bytes.Buffer) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	for k, v := range fields {
		bodyWriter.WriteField(k, v)
	}
	ctype := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	return ctype, bodyBuf
}

func CreateFileForm(bodyWriter *multipart.Writer, fkey, fp string) error {
	fileWriter, err := bodyWriter.CreateFormFile(fkey, fp)
	if err != nil {
		return err
	}
	fh, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer fh.Close()
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}
	return nil
}
func run(bodyWriter *multipart.Writer, fields map[string]string, fkey string, fp string) error {
	for k, v := range fields {
		bodyWriter.WriteField(k, v)
	}
	if len(fkey) > 0 {
		err := CreateFileForm(bodyWriter, fkey, fp)
		if err != nil {
			return err
		}
	}
	return nil
}

func Run(fields map[string]string, fkey string, fp string) (io.Reader, string) {
	pr, pw := io.Pipe()
	bodyWriter := multipart.NewWriter(pw)
	go func() {
		err := run(bodyWriter, fields, fkey, fp)
		bodyWriter.Close()
		if err == nil {
			pw.Close()
		} else {
			pw.CloseWithError(err)
		}
	}()
	return pr, bodyWriter.FormDataContentType()
}

// http post
func HPost(url_addr string, fields map[string]string, header map[string]string, fkey string, fp string, extra Map) (int, string, error) {
	c := getClient(extra)
	var ctype string
	var bodyBuf io.Reader
	if len(fkey) > 0 {
		bodyBuf, ctype = Run(fields, fkey, fp)
	} else {
		ctype, bodyBuf = CreateFormBody(fields)
	}
	req, err := http.NewRequest("POST", url_addr, bodyBuf)
	if err != nil {
		return 0, "", err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", ctype)
	res, err := c.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	bys, err := ioutil.ReadAll(res.Body)
	return res.StatusCode, string(bys), err
}

func HPostM(url string, fields map[string]string, header map[string]string, fkey string, fp string, extra Map) (int, Map, error) {
	code, res, err := HPost(url, fields, header, fkey, fp, extra)
	if err != nil {
		return code, Map{}, err
	} else {
		m, err := Json2Map(res)
		if err != nil {
			return 0, nil, err
		}
		return code, m, err
	}
}

func HPostBodyM(url_addr string, headers map[string]string, buf io.Reader, extra Map) (int, Map, error) {
	code, res, err := HPostBody(url_addr, headers, buf, extra)
	if err != nil {
		return code, Map{}, err
	} else {
		m, err := Json2Map(res)
		if err != nil {
			return 0, nil, err
		}
		return code, m, err
	}
}

func HPostBody(url_addr string, headers map[string]string, buf io.Reader, extra Map) (int, string, error) {
	c := getClient(extra)
	req, err := http.NewRequest("POST", url_addr, buf)
	if err != nil {
		return 0, "", err
	}
	for key, val := range headers {
		req.Header.Set(key, val)
	}
	res, err := c.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	str, err := readAllStr(res.Body)
	return res.StatusCode, str, err
}

func readAllStr(r io.Reader) (string, error) {
	if r == nil {
		return "", nil
	}
	bys, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil
	}
	return string(bys), nil
}
