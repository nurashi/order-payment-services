package provider

import (
	"fmt"
	"net/smtp"
)

type SMTPEmailSender struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewSMTPEmailSender(host, port, user, password, from string) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (s *SMTPEmailSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.from, to, subject, body,
	))

	if err := smtp.SendMail(addr, auth, s.from, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	return nil
}
