package mailer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime/quotedprintable"
	"net/smtp"
	"time"
)

type Attachment struct {
	Filename string
	Data     []byte
	MIMEType string
}

func (m *Mailer) SendWithAttachments(to, subject, textBody string, attachments []Attachment) error {
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)

	b := make([]byte, 12)
	rand.Read(b)
	msgID := fmt.Sprintf("<%x@%s>", b, m.cfg.Host)
	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="

	mixedBoundary := fmt.Sprintf("=_%x_mixed", b)
	altBoundary := fmt.Sprintf("=_%x_alt", b)

	var buf bytes.Buffer

	// Outer headers
	fmt.Fprintf(&buf, "From: %s\r\n", m.cfg.From)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: %s\r\n", msgID)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", mixedBoundary)

	// multipart/alternative block (text + html)
	fmt.Fprintf(&buf, "--%s\r\n", mixedBoundary)
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)

	fmt.Fprintf(&buf, "--%s\r\n", altBoundary)
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qpw := quotedprintable.NewWriter(&buf)
	qpw.Write([]byte(textBody)) //nolint:errcheck
	qpw.Close()

	fmt.Fprintf(&buf, "\r\n--%s\r\n", altBoundary)
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qpw = quotedprintable.NewWriter(&buf)
	qpw.Write([]byte(m.textToHTML(textBody))) //nolint:errcheck
	qpw.Close()

	fmt.Fprintf(&buf, "\r\n--%s--\r\n", altBoundary)

	// Attachment parts
	for _, a := range attachments {
		fmt.Fprintf(&buf, "\r\n--%s\r\n", mixedBoundary)
		fmt.Fprintf(&buf, "Content-Type: %s\r\n", a.MIMEType)
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", a.Filename)
		enc := base64.NewEncoder(base64.StdEncoding, &buf)
		enc.Write(a.Data) //nolint:errcheck
		enc.Close()
		buf.WriteString("\r\n")
	}

	fmt.Fprintf(&buf, "\r\n--%s--\r\n", mixedBoundary)

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	return smtp.SendMail(addr, auth, m.fromAddr, []string{to}, buf.Bytes())
}
