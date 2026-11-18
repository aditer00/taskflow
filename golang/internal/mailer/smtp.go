package mailer

import (
	"fmt"
	"net/smtp"
)

type Mailer struct {
	addr string
	from string
}

func New(host, port, from string) *Mailer {
	return &Mailer{
		addr: fmt.Sprintf("%s:%s", host, port),
		from: from,
	}
}

func (m *Mailer) Send(to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		m.from, to, subject, body)
	return smtp.SendMail(m.addr, nil, m.from, []string{to}, []byte(msg))
}
