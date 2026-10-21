package services

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"formfling/internal/config"
	"formfling/internal/models"
	"formfling/internal/utils"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{config: cfg}
}

// getLocalTime returns the current time in the timezone set by TZ environment variable
func (s *EmailService) getLocalTime(cfg *config.Config) time.Time {
	now := time.Now()

	// Try to load the location from TZ
	if loc, err := time.LoadLocation(cfg.Timezone); err == nil {
		return now.In(loc)
	} else {
		log.Printf("Warning: Failed to load timezone '%s': %v. Using system local time.", cfg.Timezone, err)
	}

	// Fall back to system local time
	return now
}

func (s *EmailService) SendEmail(formData models.FormData, origin string) error {
	msg, err := s.buildMessage(formData, origin)
	if err != nil {
		return err
	}

	// Set up authentication information
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	// Connect to server and send email
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	deadline := time.Now().Add(config.SMTPTimeout)

	// Handle different SMTP configurations
	if s.config.SMTPPort == 465 {
		return s.sendEmailSSL(addr, auth, msg, deadline)
	}
	return s.sendEmailSTARTTLS(addr, auth, msg, deadline)
}

func (s *EmailService) buildMessage(formData models.FormData, origin string) (string, error) {
	if !utils.ValidateEmail(formData.Email) || !utils.ValidateEmail(s.config.FromEmail) || !utils.ValidateEmail(s.config.ToEmail) {
		return "", fmt.Errorf("invalid email address in message")
	}
	now := s.getLocalTime(s.config)
	body := buildEmailBody(formData, origin, now)

	subject := sanitizeHeaderText(formData.Subject)
	if subject == "" {
		subject = "Message from " + sanitizeHeaderText(formData.Name)
	}
	from := (&mail.Address{Name: sanitizeHeaderText(formData.Name), Address: s.config.FromEmail}).String()
	to := (&mail.Address{Name: sanitizeHeaderText(s.config.ToName), Address: s.config.ToEmail}).String()
	replyTo := (&mail.Address{Address: formData.Email}).String()

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "Date: %s\r\nFrom: %s\r\nTo: %s\r\nReply-To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		now.Format(time.RFC1123Z), from, to, replyTo, mime.QEncoding.Encode("UTF-8", subject))
	body = strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\r", "\n")
	qp := quotedprintable.NewWriter(&msg)
	if _, err := io.WriteString(qp, strings.ReplaceAll(body, "\n", "\r\n")); err != nil {
		return "", fmt.Errorf("error encoding email body: %w", err)
	}
	if err := qp.Close(); err != nil {
		return "", fmt.Errorf("error closing email body: %w", err)
	}
	return msg.String(), nil
}

func buildEmailBody(formData models.FormData, origin string, now time.Time) string {
	var body strings.Builder
	body.WriteString(formData.Message)
	fmt.Fprintf(&body, "\n\nFrom: %s <%s>\n", formData.Name, formData.Email)
	if formData.Phone != "" {
		fmt.Fprintf(&body, "Phone: %s\n", formData.Phone)
	}
	if formData.Website != "" {
		fmt.Fprintf(&body, "Website: %s\n", formData.Website)
	}
	if origin != "" {
		fmt.Fprintf(&body, "Submitted from: %s\n", origin)
	}
	fmt.Fprintf(&body, "Submitted: %s at %s\n", now.Format("02 January 2006"), now.Format("03:04 PM"))
	return body.String()
}

func sanitizeHeaderText(value string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, value)), " ")
}

func dialSMTP(addr string, deadline time.Time) (net.Conn, error) {
	dialer := net.Dialer{Timeout: config.SMTPConnectTimeout, Deadline: deadline}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (s *EmailService) sendEmailSSL(addr string, auth smtp.Auth, msg string, deadline time.Time) error {
	rawConn, err := dialSMTP(addr, deadline)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	conn := tls.Client(rawConn, &tls.Config{
		ServerName: s.config.SMTPHost,
	})
	defer conn.Close()
	if err := conn.Handshake(); err != nil {
		return fmt.Errorf("failed to establish SMTP TLS connection: %v", err)
	}

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()

	// Authenticate
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	// Send email
	return s.sendEmailData(client, msg)
}

func (s *EmailService) sendEmailSTARTTLS(addr string, auth smtp.Auth, msg string, deadline time.Time) error {
	rawConn, err := dialSMTP(addr, deadline)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer rawConn.Close()
	conn, err := smtp.NewClient(rawConn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer conn.Quit()

	// Start TLS if available
	if ok, _ := conn.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.config.SMTPHost,
		}
		if err := conn.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %v", err)
		}
	}

	// Authenticate
	if err := conn.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	// Send email
	return s.sendEmailData(conn, msg)
}

func (s *EmailService) sendEmailData(client *smtp.Client, msg string) error {
	// Set sender and recipient
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	if err := client.Rcpt(s.config.ToEmail); err != nil {
		return fmt.Errorf("failed to set recipient: %v", err)
	}

	// Send email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %v", err)
	}

	_, err = writer.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write email data: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close email writer: %v", err)
	}

	return nil
}
