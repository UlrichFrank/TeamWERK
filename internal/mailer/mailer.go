package mailer

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

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

	b := make([]byte, 12)
	rand.Read(b)
	msgID := fmt.Sprintf("<%x@%s>", b, m.cfg.Host)

	// RFC 2047 encode non-ASCII subject (required for UTF-8 content like em dashes)
	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="

	msg := strings.Join([]string{
		"From: " + m.cfg.From,
		"To: " + to,
		"Subject: " + encodedSubject,
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Message-ID: " + msgID,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	return smtp.SendMail(addr, auth, m.fromAddr, []string{to}, []byte(msg))
}
