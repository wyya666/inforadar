package mail

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"inforadar/internal/config"
)

func TestSMTPMailerSendsPasswordResetLink(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messages := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 test SMTP\r\n")
		var body strings.Builder
		data := false
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if data {
				if line == "." {
					data = false
					messages <- body.String()
					fmt.Fprint(conn, "250 stored\r\n")
				} else {
					body.WriteString(line + "\n")
				}
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				fmt.Fprint(conn, "250-test\r\n250 PIPELINING\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(conn, "250 ok\r\n")
			case line == "DATA":
				data = true
				fmt.Fprint(conn, "354 continue\r\n")
			case line == "QUIT":
				fmt.Fprint(conn, "221 bye\r\n")
				return
			default:
				fmt.Fprint(conn, "250 ok\r\n")
			}
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	mailer, err := NewSMTPMailer(config.SMTP{Host: "127.0.0.1", Port: port, From: "InfoRadar <no-reply@example.test>", TLSMode: "none"}, "http://127.0.0.1:5173")
	if err != nil {
		t.Fatal(err)
	}
	if err := mailer.SendPasswordReset("user@example.test", "secret-token"); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-messages:
		if !strings.Contains(message, "http://127.0.0.1:5173/reset-password?token=secret-token") || !strings.Contains(message, "To: <user@example.test>") {
			t.Fatalf("unexpected mail content: %s", message)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP server did not receive message")
	}
}
