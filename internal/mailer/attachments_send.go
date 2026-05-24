package mailer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"time"
)

type Attachment struct {
	Filename string
	Data     []byte
	MIMEType string
}

func (m *Mailer) SendWithAttachments(to, subject, body string, attachments []Attachment) error {
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)

	b := make([]byte, 12)
	rand.Read(b)
	msgID := fmt.Sprintf("<%x@%s>", b, m.cfg.Host)
	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="

	var buf bytes.Buffer

	// Headers
	fmt.Fprintf(&buf, "From: %s\r\n", m.cfg.From)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: %s\r\n", msgID)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	mw := multipart.NewWriter(&buf)
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", mw.Boundary())

	// Text part
	th := textproto.MIMEHeader{}
	th.Set("Content-Type", "text/plain; charset=utf-8")
	th.Set("Content-Transfer-Encoding", "8bit")
	pw, _ := mw.CreatePart(th)
	pw.Write([]byte(body))

	// Attachment parts
	for _, a := range attachments {
		ah := textproto.MIMEHeader{}
		ah.Set("Content-Type", a.MIMEType)
		ah.Set("Content-Transfer-Encoding", "base64")
		ah.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.Filename))
		aw, _ := mw.CreatePart(ah)
		enc := base64.NewEncoder(base64.StdEncoding, aw)
		enc.Write(a.Data)
		enc.Close()
	}

	mw.Close()

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	return smtp.SendMail(addr, auth, m.fromAddr, []string{to}, buf.Bytes())
}
