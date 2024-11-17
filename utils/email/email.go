package email

import (
	"io"
	"io/ioutil"
	"net/smtp"
	"os"
	"strings"

	gomail "gopkg.in/gomail.v2"
)

var (
	SMTPServer = ""
	SMTPPort   = 0
)

func SendMail(host, from, to, subject, body, mailType string) error {
	var contentType string
	if mailType == "html" {
		contentType = "Content-Type: text/" + mailType + "; charset=UTF-8"
	} else {
		contentType = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := []byte("To: " + to + "\r\nFrom: " + from + "<" + from + ">\r\nSubject: " + subject + "\r\n" + contentType + "\r\n\r\n" + body)
	sendTo := strings.Split(to, ";")
	err := smtp.SendMail(host, nil, from, sendTo, msg)
	return err
}

func SendSystemEmails(sendTo []string, subject, body string, files []*os.File) (err error) {
	dialer := gomail.NewDialer(SMTPServer, SMTPPort, "", "")
	m := gomail.NewMessage()
	var from = "grapery@qq.com"
	m.SetHeader("From", from)
	m.SetHeader("To", sendTo...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	for _, f := range files {
		fileName := f.Name()
		var data []byte
		data, err = io.ReadAll(f)
		if err != nil {
			return
		}
		err = ioutil.WriteFile("./"+fileName, data, os.ModeTemporary)
		if err != nil {
			return err
		}
		m.Attach("./" + fileName)
	}
	if err = dialer.DialAndSend(m); err != nil {
		return err
	}
	return
}
