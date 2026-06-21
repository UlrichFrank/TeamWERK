package mailer

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html"
	"log/slog"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/config"
)

//go:embed assets/icon-192.png
var logoPNG []byte

var urlRe = regexp.MustCompile(`https?://\S+`)

type Mailer struct {
	cfg      config.SMTPConfig
	fromAddr string
	baseURL  string
	disabled bool
}

func New(cfg config.SMTPConfig, baseURL string, disabled bool) *Mailer {
	fromAddr := cfg.User
	if addr, err := mail.ParseAddress(cfg.From); err == nil {
		fromAddr = addr.Address
	}
	return &Mailer{cfg: cfg, fromAddr: fromAddr, baseURL: baseURL, disabled: disabled}
}

func (m *Mailer) Send(to, subject, textBody string) error {
	if m.disabled {
		slog.Info("mailer disabled", "to", to, "subject", subject)
		return nil
	}
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)

	b := make([]byte, 12)
	rand.Read(b)
	msgID := fmt.Sprintf("<%x@%s>", b, m.cfg.Host)
	relBoundary := fmt.Sprintf("=_%x_related", b)
	altBoundary := fmt.Sprintf("=_%x_alt", b)

	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", m.cfg.From)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: %s\r\n", msgID)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Precedence: transactional\r\n")
	fmt.Fprintf(&buf, "X-Mailer: TeamWERK\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/related; boundary=\"%s\"\r\n\r\n", relBoundary)

	// multipart/alternative block
	fmt.Fprintf(&buf, "--%s\r\n", relBoundary)
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)

	// text/plain part
	fmt.Fprintf(&buf, "--%s\r\n", altBoundary)
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qpw := quotedprintable.NewWriter(&buf)
	qpw.Write([]byte(textBody)) //nolint:errcheck
	qpw.Close()

	// text/html part
	fmt.Fprintf(&buf, "\r\n--%s\r\n", altBoundary)
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qpw = quotedprintable.NewWriter(&buf)
	qpw.Write([]byte(m.textToHTML(textBody))) //nolint:errcheck
	qpw.Close()
	fmt.Fprintf(&buf, "\r\n--%s--\r\n", altBoundary)

	// inline logo as CID attachment
	fmt.Fprintf(&buf, "\r\n--%s\r\n", relBoundary)
	buf.WriteString("Content-Type: image/png\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("Content-ID: <logo@teamwerk>\r\n")
	buf.WriteString("Content-Disposition: inline; filename=\"icon-192.png\"\r\n\r\n")
	enc := base64.NewEncoder(base64.StdEncoding, &buf)
	enc.Write(logoPNG) //nolint:errcheck
	enc.Close()
	buf.WriteString("\r\n")

	fmt.Fprintf(&buf, "\r\n--%s--\r\n", relBoundary)

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	return smtp.SendMail(addr, auth, m.fromAddr, []string{to}, buf.Bytes())
}

// actionButtonLabel returns a CTA label if the URL is a known action link, otherwise "".
func actionButtonLabel(u string) string {
	switch {
	case strings.Contains(u, "/register"):
		return "Konto erstellen"
	case strings.Contains(u, "/reset-password"):
		return "Passwort zurücksetzen"
	case strings.Contains(u, "/profile/email/confirm"):
		return "E-Mail-Adresse bestätigen"
	default:
		return ""
	}
}

// textToHTML converts a plain-text email body to a minimal branded HTML version.
// Action URLs become CTA buttons; other URLs become inline links.
func (m *Mailer) textToHTML(text string) string {
	locs := urlRe.FindAllStringIndex(text, -1)
	var content strings.Builder
	prev := 0
	for _, loc := range locs {
		content.WriteString(html.EscapeString(text[prev:loc[0]]))
		u := text[loc[0]:loc[1]]
		uEsc := html.EscapeString(u)
		if label := actionButtonLabel(u); label != "" {
			fmt.Fprintf(&content,
				`<p style="text-align:center;margin:24px 0"><a href="%s" style="display:inline-block;background:#FDE400;color:#181310;font-weight:700;padding:12px 28px;border-radius:6px;text-decoration:none;font-size:15px">%s</a></p>`,
				uEsc, label,
			)
		} else {
			fmt.Fprintf(&content,
				`<a href="%s" style="color:#181310;font-weight:600;word-break:break-all">%s</a>`,
				uEsc, uEsc,
			)
		}
		prev = loc[1]
	}
	content.WriteString(html.EscapeString(text[prev:]))

	var pTags []string
	for _, block := range strings.Split(content.String(), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if strings.HasPrefix(block, "<p") || strings.HasPrefix(block, "<div") {
			pTags = append(pTags, block)
		} else {
			pTags = append(pTags, "<p>"+strings.ReplaceAll(block, "\n", "<br>")+"</p>")
		}
	}

	body := strings.Join(pTags, "\n")
	return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:Arial,Helvetica,sans-serif;background:#ffffff;margin:0;padding:24px">
<div style="max-width:520px;margin:0 auto;background:#F9FAFB;border-radius:12px;overflow:hidden;border-top:4px solid #FDE400;box-shadow:0 1px 3px 0 rgba(0,0,0,.1),0 1px 2px -1px rgba(0,0,0,.1)">
  <div style="padding:20px 24px;border-bottom:1px solid #E5E7EB">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" role="presentation"><tr>
      <td width="52" style="vertical-align:middle">
        <img src="cid:logo@teamwerk" alt="Team Stuttgart" width="44" height="44" style="display:block;border-radius:6px">
      </td>
      <td style="vertical-align:middle;padding-left:12px">
        <span style="color:#181310;font-weight:700;font-size:20px;display:block;letter-spacing:-.5px">TeamWERK</span>
        <span style="color:#6B7280;font-size:12px;display:block;margin-top:1px">Team Stuttgart</span>
      </td>
    </tr></table>
  </div>
  <div style="padding:28px 24px;background:#F9FAFB;color:#111827;font-size:15px;line-height:1.7">
` + body + `
  </div>
  <div style="padding:16px 24px;background:#F9FAFB;border-top:1px solid #E5E7EB;font-size:12px;color:#6B7280">
    Diese E-Mail wurde von TeamWERK automatisch versandt.
  </div>
</div>
</body>
</html>`
}
