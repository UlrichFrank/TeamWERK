package mailer

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/config"
)

type Mailer struct {
	cfg      config.SMTPConfig
	fromAddr string // bare email extracted from cfg.From
}

func New(cfg config.SMTPConfig) *Mailer {
	fromAddr := cfg.User
	if addr, err := mail.ParseAddress(cfg.From); err == nil {
		fromAddr = addr.Address
	}
	return &Mailer{cfg: cfg, fromAddr: fromAddr}
}

func (m *Mailer) Send(to, subject, body string) error {
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	msg := strings.Join([]string{
		"From: " + m.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	return smtp.SendMail(addr, auth, m.fromAddr, []string{to}, []byte(msg))
}
