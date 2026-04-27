package auth

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(to []string, subject, body string) error
}

type LogMailer struct {
	Prefix string
}

func (l LogMailer) Send(to []string, subject, body string) error {
	log.Printf("[mail] %s to=%v subject=%q\n%s", l.Prefix, to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, Port, User, Password, From string
}

func (s SMTPMailer) Send(to []string, subject, body string) error {
	if s.Host == "" || s.From == "" {
		return fmt.Errorf("smtp not configured")
	}
	addr := s.Host + ":" + s.Port
	if s.Port == "" {
		addr = s.Host + ":587"
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		s.From, strings.Join(to, ","), subject, body))
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
	}
	return smtp.SendMail(addr, auth, s.From, to, msg)
}
