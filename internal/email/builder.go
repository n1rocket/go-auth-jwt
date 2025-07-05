package email

import (
	"fmt"
	"strings"
	"time"
)

// MessageBuilder builds MIME-formatted email messages
type MessageBuilder struct {
	from     string
	to       string
	subject  string
	textBody string
	htmlBody string
	headers  map[string]string
	boundary string
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		headers:  make(map[string]string),
		boundary: fmt.Sprintf("boundary-%d", time.Now().UnixNano()),
	}
}

// From sets the sender address
func (b *MessageBuilder) From(address string) *MessageBuilder {
	b.from = address
	return b
}

// To sets the recipient address
func (b *MessageBuilder) To(address string) *MessageBuilder {
	b.to = address
	return b
}

// Subject sets the email subject
func (b *MessageBuilder) Subject(subject string) *MessageBuilder {
	b.subject = subject
	return b
}

// TextBody sets the plain text body
func (b *MessageBuilder) TextBody(body string) *MessageBuilder {
	b.textBody = body
	return b
}

// HTMLBody sets the HTML body
func (b *MessageBuilder) HTMLBody(body string) *MessageBuilder {
	b.htmlBody = body
	return b
}

// Header adds a custom header
func (b *MessageBuilder) Header(key, value string) *MessageBuilder {
	b.headers[key] = value
	return b
}

// Build constructs the MIME message
func (b *MessageBuilder) Build() string {
	var message strings.Builder

	// Standard headers
	b.writeHeader(&message, "From", b.from)
	b.writeHeader(&message, "To", b.to)
	b.writeHeader(&message, "Subject", b.subject)
	b.writeHeader(&message, "MIME-Version", "1.0")

	// Custom headers
	for key, value := range b.headers {
		b.writeHeader(&message, key, value)
	}

	// Body content
	if b.htmlBody != "" {
		b.writeMultipartMessage(&message)
	} else {
		b.writePlainMessage(&message)
	}

	return message.String()
}

// writeHeader writes a header line
func (b *MessageBuilder) writeHeader(w *strings.Builder, key, value string) {
	if value != "" {
		w.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
}

// writePlainMessage writes a plain text message
func (b *MessageBuilder) writePlainMessage(w *strings.Builder) {
	w.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	w.WriteString("\r\n")
	w.WriteString(b.textBody)
}

// writeMultipartMessage writes a multipart message with text and HTML
func (b *MessageBuilder) writeMultipartMessage(w *strings.Builder) {
	w.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", b.boundary))
	w.WriteString("\r\n")

	// Plain text part
	w.WriteString(fmt.Sprintf("--%s\r\n", b.boundary))
	w.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	w.WriteString("\r\n")
	w.WriteString(b.textBody)
	w.WriteString("\r\n")

	// HTML part
	w.WriteString(fmt.Sprintf("--%s\r\n", b.boundary))
	w.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	w.WriteString("\r\n")
	w.WriteString(b.htmlBody)
	w.WriteString("\r\n")

	// End boundary
	w.WriteString(fmt.Sprintf("--%s--\r\n", b.boundary))
}

// FormatAddress formats an email address with optional name
func FormatAddress(email, name string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}
