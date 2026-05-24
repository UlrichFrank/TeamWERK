package mailer

import "embed"

//go:embed attachments/satzung.pdf attachments/gebuehrenordnung.pdf attachments/leitbild.pdf attachments/logo.svg
var AttachmentFS embed.FS
