package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

type SMTPMailer struct {
	config Config
}

func NewSMTPMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{config: cfg}
}

func (m *SMTPMailer) SendAdminInvite(ctx context.Context, toEmail, inviterEmail, activationURL string, expiresAt time.Time) error {
	if err := m.validate(); err != nil {
		log.Printf("smtp invite validation failed: to=%s from=%s host=%s port=%d err=%v", toEmail, m.config.FromEmail, m.config.Host, m.config.Port, err)
		return err
	}

	subject := "You've been invited to Prism Admin"
	body := strings.Join([]string{
		fmt.Sprintf("You've been invited to Prism Admin by %s.", inviterEmail),
		"",
		"Click the link below to activate your account and set your password:",
		activationURL,
		"",
		fmt.Sprintf("This link expires on %s.", expiresAt.Format(time.RFC1123)),
		"",
		"If you were not expecting this invitation, you can ignore this email.",
	}, "\r\n")

	message := buildPlainTextMessage(m.fromHeader(), toEmail, subject, body)
	address := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))
	log.Printf("smtp invite send start: to=%s from=%s host=%s port=%d", toEmail, m.config.FromEmail, m.config.Host, m.config.Port)

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		log.Printf("smtp invite dial failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("dial smtp server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		log.Printf("smtp invite client creation failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.config.Host}); err != nil {
			log.Printf("smtp invite starttls failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
			return fmt.Errorf("start tls: %w", err)
		}
		log.Printf("smtp invite starttls ok: to=%s host=%s port=%d", toEmail, m.config.Host, m.config.Port)
	} else {
		log.Printf("smtp invite starttls unavailable: to=%s host=%s port=%d", toEmail, m.config.Host, m.config.Port)
	}

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
		if err := client.Auth(auth); err != nil {
			log.Printf("smtp invite auth failed: to=%s username=%s host=%s port=%d err=%v", toEmail, m.config.Username, m.config.Host, m.config.Port, err)
			return fmt.Errorf("smtp auth: %w", err)
		}
		log.Printf("smtp invite auth ok: to=%s username=%s host=%s port=%d", toEmail, m.config.Username, m.config.Host, m.config.Port)
	} else {
		log.Printf("smtp invite auth unavailable: to=%s host=%s port=%d", toEmail, m.config.Host, m.config.Port)
	}

	if err := client.Mail(m.config.FromEmail); err != nil {
		log.Printf("smtp invite sender rejected: to=%s from=%s host=%s port=%d err=%v", toEmail, m.config.FromEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("smtp from: %w", err)
	}
	if err := client.Rcpt(toEmail); err != nil {
		log.Printf("smtp invite recipient rejected: to=%s from=%s host=%s port=%d err=%v", toEmail, m.config.FromEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		log.Printf("smtp invite data command failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		log.Printf("smtp invite message write failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		log.Printf("smtp invite message close failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("close smtp message: %w", err)
	}

	if err := client.Quit(); err != nil {
		log.Printf("smtp invite quit failed: to=%s host=%s port=%d err=%v", toEmail, m.config.Host, m.config.Port, err)
		return fmt.Errorf("smtp quit: %w", err)
	}

	log.Printf("smtp invite send ok: to=%s from=%s host=%s port=%d", toEmail, m.config.FromEmail, m.config.Host, m.config.Port)

	return nil
}

func (m *SMTPMailer) validate() error {
	if strings.TrimSpace(m.config.Host) == "" ||
		m.config.Port == 0 ||
		strings.TrimSpace(m.config.Username) == "" ||
		strings.TrimSpace(m.config.Password) == "" ||
		strings.TrimSpace(m.config.FromEmail) == "" {
		return fmt.Errorf("smtp mailer is not fully configured")
	}

	return nil
}

func (m *SMTPMailer) fromHeader() string {
	name := strings.TrimSpace(m.config.FromName)
	if name == "" {
		return m.config.FromEmail
	}

	return fmt.Sprintf("%s <%s>", name, m.config.FromEmail)
}

func buildPlainTextMessage(from, to, subject, body string) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="UTF-8"`,
		"",
		body,
	}, "\r\n")
}
