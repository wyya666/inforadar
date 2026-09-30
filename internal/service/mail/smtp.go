package mail

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"inforadar/internal/config"
	"inforadar/internal/constants"
	serviceinterface "inforadar/internal/service/interface"
)

type SMTPMailer struct {
	cfg         config.SMTP
	frontendURL string
}

func NewSMTPMailer(cfg config.SMTP, frontendURL string) (serviceinterface.Mailer, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil || from.Address == "" {
		return nil, fmt.Errorf("invalid smtp.from address")
	}
	return &SMTPMailer{cfg: cfg, frontendURL: strings.TrimRight(frontendURL, "/")}, nil
}

func (m *SMTPMailer) SendPasswordReset(email, token string) error {
	link := m.frontendURL + constants.ResetPasswordRoute + "?token=" + url.QueryEscape(token)
	return m.send(email, "重置 InfoRadar 密码", "请打开以下链接设置新密码（30 分钟内有效）：\n\n"+link+"\n\n如果这不是你发起的操作，请忽略此邮件。")
}

func (m *SMTPMailer) send(recipient, subject, body string) error {
	from, _ := mail.ParseAddress(m.cfg.From)
	to, err := mail.ParseAddress(recipient)
	if err != nil || to.Address != recipient {
		return fmt.Errorf("invalid recipient address")
	}
	address := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	if m.cfg.TLSMode == constants.SMTPTLSImplicit {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return fmt.Errorf("connect to SMTP: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if m.cfg.TLSMode == constants.SMTPTLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if m.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	message := "From: " + from.String() + "\r\nTo: " + to.String() +
		"\r\nSubject: " + mime.QEncoding.Encode("utf-8", subject) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		strings.ReplaceAll(body, "\n", "\r\n")
	if _, err := w.Write([]byte(message)); err != nil {
		w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
