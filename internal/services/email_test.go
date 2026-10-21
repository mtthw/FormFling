package services

import (
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"

	"formfling/internal/config"
	"formfling/internal/models"
)

func TestBuildMessagePlainTextAndReplyTo(t *testing.T) {
	service := NewEmailService(&config.Config{
		FromEmail: "contact@mtthw.xyz",
		ToEmail:   "owner@example.com",
		ToName:    "Owner",
		Timezone:  "UTC",
	})
	form := models.FormData{
		Name:    "Éva Smith\r\nBcc: somebody@example.com",
		Email:   "eva@example.com",
		Subject: "Project enquiry\r\nBcc: somebody@example.com",
		Message: "Hello,\nHere is my café idea.",
		Phone:   "+353 1234567",
	}
	raw, err := service.buildMessage(form, "https://example.com/contact")
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	from, err := mail.ParseAddress(message.Header.Get("From"))
	if err != nil || from.Address != "contact@mtthw.xyz" || from.Name != "Éva Smith Bcc: somebody@example.com" {
		t.Fatalf("unexpected From: %q (%v)", message.Header.Get("From"), err)
	}
	replyTo, err := mail.ParseAddress(message.Header.Get("Reply-To"))
	if err != nil || replyTo.Address != "eva@example.com" {
		t.Errorf("unexpected Reply-To: %q (%v)", message.Header.Get("Reply-To"), err)
	}
	if message.Header.Get("Bcc") != "" {
		t.Error("form values injected a Bcc header")
	}
	if message.Header.Get("Date") == "" {
		t.Error("missing Date header")
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	if err != nil || subject != "Project enquiry Bcc: somebody@example.com" {
		t.Errorf("unexpected Subject: %q (%v)", subject, err)
	}
	if got := message.Header.Get("Content-Type"); got != "text/plain; charset=UTF-8" {
		t.Errorf("unexpected Content-Type: %q", got)
	}
	decoded, err := io.ReadAll(quotedprintable.NewReader(message.Body))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Hello,\r\nHere is my café idea.", "From: Éva Smith", "Phone: +353 1234567", "Submitted from: https://example.com/contact"} {
		if !strings.Contains(string(decoded), want) {
			t.Errorf("body does not contain %q: %q", want, decoded)
		}
	}
}

func TestBuildMessageRejectsInvalidReplyTo(t *testing.T) {
	service := NewEmailService(&config.Config{
		FromEmail: "contact@mtthw.xyz",
		ToEmail:   "owner@example.com",
		Timezone:  "UTC",
	})
	_, err := service.buildMessage(models.FormData{Email: "alice@example.com\r\nBcc: target@example.com"}, "")
	if err == nil {
		t.Fatal("expected invalid reply address to be rejected")
	}
}

func TestBuildMessageFallbackSubjectAndOptionalFields(t *testing.T) {
	service := NewEmailService(&config.Config{
		FromEmail: "contact@mtthw.xyz",
		ToEmail:   "owner@example.com",
		Timezone:  "UTC",
	})
	raw, err := service.buildMessage(models.FormData{
		Name: "Alice", Email: "alice@example.com", Message: "Hello",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got := message.Header.Get("Content-Type"); got != "text/plain; charset=UTF-8" {
		t.Errorf("unexpected Content-Type: %q", got)
	}
	if got := message.Header.Get("Subject"); got != "Message from Alice" {
		t.Errorf("unexpected fallback subject: %q", got)
	}
	decoded, err := io.ReadAll(quotedprintable.NewReader(message.Body))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(decoded), "Phone:") || strings.Contains(string(decoded), "Website:") || strings.Contains(string(decoded), "Submitted from:") {
		t.Error("empty optional fields appeared in body")
	}
}
