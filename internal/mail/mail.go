// Package mail sends transactional email (currently just password-reset links)
// over plain SMTP using only the standard library - no external dependency.
package mail

import (
	"fmt"
	"net/smtp"
)

// Mailer sends email through a single SMTP account.
type Mailer struct {
	Host string
	Port string // e.g. "587" (STARTTLS, most providers)
	User string
	Pass string
	From string
}

// Configured reports whether enough SMTP settings are present to attempt sending.
func (m *Mailer) Configured() bool {
	return m != nil && m.Host != "" && m.From != ""
}

// Send delivers a plain-text email to a single recipient.
//
// This uses smtp.SendMail, which negotiates STARTTLS automatically when the
// server advertises it - the common case on port 587. A provider that instead
// requires *implicit* TLS on port 465 is not supported here; that would need a
// manual tls.Dial + smtp.NewClient variant instead.
func (m *Mailer) Send(to, subject, body string) error {
	if !m.Configured() {
		return fmt.Errorf("mail: SMTP is not configured")
	}
	addr := m.Host + ":" + m.Port
	msg := "From: " + m.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"\r\n" + body

	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Pass, m.Host)
	}
	return smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg))
}
