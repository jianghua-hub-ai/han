package mailer

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSMTPMailerCancellationInterruptsBlockedSMTPIO(t *testing.T) {
	tests := []struct {
		name         string
		stage        smtpBlockStage
		withAuth     bool
		wantCommands string
	}{
		{
			name:         "AUTH reply",
			stage:        smtpBlockAuth,
			withAuth:     true,
			wantCommands: "EHLO,AUTH",
		},
		{
			name:         "MAIL reply",
			stage:        smtpBlockMail,
			wantCommands: "EHLO,MAIL",
		},
		{
			name:         "RCPT reply",
			stage:        smtpBlockRCPT,
			wantCommands: "EHLO,MAIL,RCPT",
		},
		{
			name:         "DATA command reply",
			stage:        smtpBlockDATACommand,
			wantCommands: "EHLO,MAIL,RCPT,DATA",
		},
		{
			name:         "DATA acceptance reply",
			stage:        smtpBlockDATAAcceptance,
			wantCommands: "EHLO,MAIL,RCPT,DATA",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const serverName = "smtp.example.test"
			certificate, roots := newTestCertificate(
				t,
				serverName,
				time.Now().Add(-time.Hour),
				time.Now().Add(time.Hour),
			)
			server := newBlockingSMTPTestServer(t, certificate, test.stage)
			config := SMTPConfig{
				Address:   "pipe:cancellation",
				Host:      serverName,
				From:      "sender@example.test",
				Mode:      TLSModeImplicit,
				TLSConfig: testClientTLSConfig(serverName, roots),
			}
			if test.withAuth {
				config.Auth = smtp.PlainAuth("", "synthetic-user", "synthetic-password", serverName)
			}
			mailer := newTestSMTPMailer(t, server.smtpTestServer, config)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)
			result := make(chan error, 1)

			go func() {
				result <- mailer.Send(ctx, testMessage())
			}()
			server.waitBlocked(t)
			cancel()

			select {
			case err := <-result:
				if err == nil {
					t.Fatal("Send() error = nil after cancellation")
				}
				if errors.Is(err, ErrMessageRejected) {
					t.Fatalf("Send() error = %v, interrupted SMTP I/O is not message rejection", err)
				}
				assertNoSMTPMessageRejection(t, err)
			case <-time.After(time.Second):
				t.Fatal("Send() did not return promptly after cancellation")
			}
			server.wait(t)
			if got := strings.Join(server.commands, ","); got != test.wantCommands {
				t.Fatalf("SMTP commands = %q, want %q", got, test.wantCommands)
			}
		})
	}
}

func TestSMTPMailerCancellationAfterDataAcceptanceKeepsSuccess(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	server := newBlockingSMTPTestServer(t, certificate, smtpBlockQUIT)
	mailer := newTestSMTPMailer(t, server.smtpTestServer, SMTPConfig{
		Address:   "pipe:accepted-cancellation",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeImplicit,
		TLSConfig: testClientTLSConfig(serverName, roots),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	result := make(chan error, 1)

	go func() {
		result <- mailer.Send(ctx, testMessage())
	}()
	server.waitBlocked(t)
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Send() error after DATA acceptance = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Send() did not return promptly after cancellation")
	}
	server.wait(t)
}

func TestSMTPMailerAcceptedDATA_QUITFailure_KeepsSuccess(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(t, serverName,
		time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	for _, test := range []struct {
		name string
		code int
	}{
		{"rejected", 550},
		{"disconnected", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := newSMTPTestServer(t, smtpServerOptions{
				implicitTLS: true, certificate: certificate,
				rejectStage: smtpBlockQUIT, rejectCode: test.code,
			})
			mailer := newTestSMTPMailer(t, server, SMTPConfig{
				Address: "pipe:accepted-quit-failure", Host: serverName,
				From: "sender@example.test", Mode: TLSModeImplicit,
				TLSConfig: testClientTLSConfig(serverName, roots),
			})
			if err := mailer.Send(testContext(t), testMessage()); err != nil {
				t.Error("QUIT failure changed an accepted DATA result")
			}
			server.wait(t)
		})
	}
}

func TestSMTPMailerNormalCompletionStopsCancellationWatcher(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	server := newSMTPTestServer(t, smtpServerOptions{
		implicitTLS: true,
		certificate: certificate,
	})
	connection := &smtpActivityConn{
		Conn:     server.client,
		activity: make(chan struct{}, 16),
	}
	mailer, err := NewSMTPMailer(SMTPConfig{
		Address:   "pipe:watcher-cleanup",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeImplicit,
		TLSConfig: testClientTLSConfig(serverName, roots),
	}, pipeDialer{connection: connection})
	if err != nil {
		t.Fatalf("NewSMTPMailer() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	if err := mailer.Send(ctx, testMessage()); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	server.wait(t)
	for {
		select {
		case <-connection.activity:
		default:
			goto drained
		}
	}

drained:
	cancel()
	select {
	case <-connection.activity:
		t.Fatal("SMTP connection changed after normal Send returned and context was canceled")
	case <-time.After(250 * time.Millisecond):
	}
}

func TestSMTPMailerAllowsOnlyApprovedPurposes(t *testing.T) {
	for _, purpose := range []Purpose{
		PurposeNewAdministratorCredential,
		PurposePasswordResetCredential,
		PurposeTemporaryCredentialResend,
		PurposeOrphanArticleTransferNotice,
	} {
		if err := ValidateMessage(Message{
			Purpose: purpose,
			To:      "recipient@example.test",
			Subject: "subject",
			Body:    "body",
		}); err != nil {
			t.Fatalf("ValidateMessage(%q) error = %v", purpose, err)
		}
	}
	err := ValidateMessage(Message{
		Purpose: Purpose("unsupported"),
		To:      "recipient@example.test",
		Subject: "subject",
		Body:    "body",
	})
	if !errors.Is(err, ErrUnsupportedPurpose) {
		t.Fatalf("ValidateMessage(unsupported) error = %v, want %v", err, ErrUnsupportedPurpose)
	}
}

func TestSMTPMailerClassifiesExplicitMessageRejection(t *testing.T) {
	const rejectionCanary = " synthetic-response-canary recipient@example.test"
	tests := []struct {
		name      string
		stage     smtpBlockStage
		wantStage SMTPMessageStage
		code      int
	}{
		{"temporary MAIL rejection", smtpBlockMail, SMTPMessageStageMAILFrom, 450},
		{"permanent MAIL rejection", smtpBlockMail, SMTPMessageStageMAILFrom, 550},
		{"temporary RCPT rejection", smtpBlockRCPT, SMTPMessageStageRCPTTo, 421},
		{"permanent RCPT rejection", smtpBlockRCPT, SMTPMessageStageRCPTTo, 550},
		{"temporary DATA command rejection", smtpBlockDATACommand, SMTPMessageStageDATACommand, 451},
		{"permanent DATA command rejection", smtpBlockDATACommand, SMTPMessageStageDATACommand, 554},
		{"temporary DATA final rejection", smtpBlockDATAAcceptance, SMTPMessageStageDATAFinal, 450},
		{"permanent DATA final rejection", smtpBlockDATAAcceptance, SMTPMessageStageDATAFinal, 554},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const serverName = "smtp.example.test"
			certificate, roots := newTestCertificate(
				t,
				serverName,
				time.Now().Add(-time.Hour),
				time.Now().Add(time.Hour),
			)
			server := newSMTPTestServer(t, smtpServerOptions{
				implicitTLS:   true,
				certificate:   certificate,
				rejectStage:   test.stage,
				rejectCode:    test.code,
				rejectMessage: rejectionCanary,
			})
			mailer := newTestSMTPMailer(t, server, SMTPConfig{
				Address:   "pipe:message-rejection",
				Host:      serverName,
				From:      "sender@example.test",
				Mode:      TLSModeImplicit,
				TLSConfig: testClientTLSConfig(serverName, roots),
			})

			err := mailer.Send(testContext(t), testMessage())
			assertSMTPMessageRejection(t, err, test.wantStage, test.code)
			var protocolError *textproto.Error
			if !errors.As(err, &protocolError) || protocolError.Code != test.code {
				t.Fatal("Send() did not preserve the SMTP protocol error and code")
			}
			if !strings.Contains(protocolError.Msg, rejectionCanary) {
				t.Error("Send() did not preserve the original SMTP response in its cause")
			}
			server.wait(t)
		})
	}
}

func TestSMTPMailerMessageRejection_Cause_PreservesIdentity(t *testing.T) {
	stages := []SMTPMessageStage{
		SMTPMessageStageMAILFrom, SMTPMessageStageRCPTTo,
		SMTPMessageStageDATACommand, SMTPMessageStageDATAFinal,
	}
	for _, stage := range stages {
		for _, code := range []int{400, 421, 499, 500, 550, 599} {
			t.Run(fmt.Sprintf("%s/%d", stage, code), func(t *testing.T) {
				protocolError := &textproto.Error{
					Code: code, Msg: "synthetic-cause-canary recipient@example.test",
				}
				cause := fmt.Errorf("synthetic transport context: %w", protocolError)
				err := classifySMTPMessageError("fixed action", stage, cause)
				assertSMTPMessageRejection(t, err, stage, code)
				var got *textproto.Error
				if !errors.As(err, &got) || got != protocolError ||
					!errors.Is(err, cause) || !errors.Is(err, protocolError) {
					t.Error("rejection did not retain the original cause and protocol error identities")
				}
				var rejection *SMTPMessageRejection
				if errors.As(err, &rejection) && rejection.Unwrap() != cause {
					t.Error("rejection did not unwrap to the original cause")
				}
			})
		}
	}
}

func TestSMTPMailerMessageRejection_NonRejection_PreservesCause(t *testing.T) {
	for _, test := range []struct {
		name  string
		cause error
	}{
		{"success reply", &textproto.Error{Code: 250, Msg: "unexpected success"}},
		{"intermediate reply", &textproto.Error{Code: 354, Msg: "unexpected intermediate reply"}},
		{"below range", &textproto.Error{Code: 399, Msg: "below negative range"}},
		{"above range", &textproto.Error{Code: 600, Msg: "above negative range"}},
		{"EOF", io.EOF},
		{"canceled", context.Canceled},
		{"deadline exceeded", context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := classifySMTPMessageError("fixed action", SMTPMessageStageRCPTTo, test.cause)
			assertNoSMTPMessageRejection(t, err)
			if errors.Is(err, ErrMessageRejected) || !errors.Is(err, test.cause) {
				t.Error("non-rejection changed its sentinel or original cause")
			}
		})
	}
}

func TestSMTPMailerMessageRejection_UnknownStage_PrintsFixedLabel(t *testing.T) {
	err := &SMTPMessageRejection{
		Stage: "synthetic-stage-canary recipient@example.test\r\n", Code: 550,
		cause: &textproto.Error{Code: 550, Msg: "synthetic-cause-canary"},
	}
	if err.Error() != "SMTP message rejected: stage=unknown code=550" {
		t.Error("unknown stage did not print a fixed label")
	}
	if err.Is(ErrTLSRequired) {
		t.Error("rejection matched an unrelated sentinel")
	}
}

func assertSMTPMessageRejection(t *testing.T, err error, stage SMTPMessageStage, code int) {
	t.Helper()
	var rejection *SMTPMessageRejection
	if !errors.As(err, &rejection) {
		t.Error("explicit rejection is missing SMTPMessageRejection")
	} else if rejection.Stage != stage || rejection.Code != code {
		t.Error("explicit rejection has an incorrect stage or code")
	}
	if !errors.Is(err, ErrMessageRejected) {
		t.Error("explicit rejection no longer matches ErrMessageRejected")
	}
	want := fmt.Sprintf("SMTP message rejected: stage=%s code=%d", stage, code)
	for _, format := range []string{"%s", "%v", "%+v"} {
		if fmt.Sprintf(format, err) != want {
			t.Errorf("rejection printable format %s contains more than the fixed stage and code", format)
		}
	}
	if fmt.Sprint(fmt.Errorf("deliver notification: %w", err)) != "deliver notification: "+want {
		t.Error("wrapped rejection prints more than the fixed context, stage and code")
	}
}

func assertNoSMTPMessageRejection(t *testing.T, err error) {
	t.Helper()
	var rejection *SMTPMessageRejection
	if errors.As(err, &rejection) {
		t.Error("non-message failure acquired SMTPMessageRejection")
	}
}

func TestSMTPMailerDoesNotClassifyAuthenticationRejectionAsMessageRejection(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	server := newSMTPTestServer(t, smtpServerOptions{
		implicitTLS: true,
		certificate: certificate,
		auth:        true,
		rejectStage: smtpBlockAuth,
		rejectCode:  535,
	})
	mailer := newTestSMTPMailer(t, server, SMTPConfig{
		Address:   "pipe:authentication-rejection",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeImplicit,
		TLSConfig: testClientTLSConfig(serverName, roots),
		Auth:      smtp.PlainAuth("", "synthetic-user", "synthetic-password", serverName),
	})

	err := mailer.Send(testContext(t), testMessage())
	if err == nil {
		t.Fatal("Send() error = nil, want authentication rejection")
	}
	if errors.Is(err, ErrMessageRejected) {
		t.Fatalf("Send() error = %v, authentication failure is not message rejection", err)
	}
	assertNoSMTPMessageRejection(t, err)
	server.wait(t)
}

func TestSMTPMailerDoesNotClassifyGreetingRejectionAsMessageRejection(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	server := newSMTPTestServer(t, smtpServerOptions{
		implicitTLS:    true,
		certificate:    certificate,
		rejectGreeting: true,
		rejectCode:     550,
	})
	mailer := newTestSMTPMailer(t, server, SMTPConfig{
		Address:   "pipe:greeting-rejection",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeImplicit,
		TLSConfig: testClientTLSConfig(serverName, roots),
	})

	err := mailer.Send(testContext(t), testMessage())
	if err == nil {
		t.Fatal("Send() error = nil, want EHLO/HELO rejection")
	}
	if errors.Is(err, ErrMessageRejected) {
		t.Fatalf("Send() error = %v, greeting failure is not message rejection", err)
	}
	assertNoSMTPMessageRejection(t, err)
	var protocolError *textproto.Error
	if !errors.As(err, &protocolError) || protocolError.Code != 550 {
		t.Fatalf("Send() protocol error = %#v, want greeting code 550", protocolError)
	}
	server.wait(t)
	if !server.ehloSeen.Load() || !server.heloSeen.Load() {
		t.Fatal("server did not receive both EHLO and HELO rejection attempts")
	}
	if server.mailSeen.Load() {
		t.Fatal("server received MAIL after EHLO/HELO rejection")
	}
}

func TestSMTPMailerDoesNotClassifyUnexpectedNonNegativeReplyAsMessageRejection(
	t *testing.T,
) {
	tests := []struct {
		name  string
		stage smtpBlockStage
		code  int
	}{
		{name: "MAIL receives intermediate reply", stage: smtpBlockMail, code: 354},
		{name: "RCPT receives intermediate reply", stage: smtpBlockRCPT, code: 354},
		{name: "DATA command receives success reply", stage: smtpBlockDATACommand, code: 250},
		{name: "DATA acceptance receives intermediate reply", stage: smtpBlockDATAAcceptance, code: 354},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const serverName = "smtp.example.test"
			certificate, roots := newTestCertificate(
				t,
				serverName,
				time.Now().Add(-time.Hour),
				time.Now().Add(time.Hour),
			)
			server := newSMTPTestServer(t, smtpServerOptions{
				implicitTLS: true,
				certificate: certificate,
				rejectStage: test.stage,
				rejectCode:  test.code,
			})
			mailer := newTestSMTPMailer(t, server, SMTPConfig{
				Address:   "pipe:unexpected-reply",
				Host:      serverName,
				From:      "sender@example.test",
				Mode:      TLSModeImplicit,
				TLSConfig: testClientTLSConfig(serverName, roots),
			})

			err := mailer.Send(testContext(t), testMessage())
			if err == nil {
				t.Fatal("Send() error = nil, want unexpected SMTP reply")
			}
			if errors.Is(err, ErrMessageRejected) {
				t.Fatalf("Send() error = %v, non-negative reply is not message rejection", err)
			}
			assertNoSMTPMessageRejection(t, err)
			var protocolError *textproto.Error
			if !errors.As(err, &protocolError) || protocolError.Code != test.code {
				t.Fatalf("Send() protocol error = %#v, want code %d", protocolError, test.code)
			}
			server.wait(t)
		})
	}
}

func TestSMTPMailerImplicitTLSSendsOnlyAfterVerifiedHandshake(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	server := newSMTPTestServer(t, smtpServerOptions{
		implicitTLS: true,
		certificate: certificate,
	})
	mailer := newTestSMTPMailer(t, server, SMTPConfig{
		Address:   "pipe:implicit",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeImplicit,
		TLSConfig: testClientTLSConfig(serverName, roots),
	})

	if err := mailer.Send(testContext(t), testMessage()); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	server.wait(t)
	if !server.mailSeen.Load() {
		t.Fatal("server did not receive MAIL after verified TLS")
	}
}

func TestSMTPMailerSTARTTLSRequiredSendsOnlyAfterUpgrade(t *testing.T) {
	const serverName = "smtp.example.test"
	certificate, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	server := newSMTPTestServer(t, smtpServerOptions{
		startTLS:    true,
		certificate: certificate,
	})
	mailer := newTestSMTPMailer(t, server, SMTPConfig{
		Address:   "pipe:starttls",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeSTARTTLSRequired,
		TLSConfig: testClientTLSConfig(serverName, roots),
	})

	if err := mailer.Send(testContext(t), testMessage()); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	server.wait(t)
	if !server.tlsEstablished.Load() || !server.mailSeen.Load() {
		t.Fatalf(
			"server states: tlsEstablished=%v mailSeen=%v",
			server.tlsEstablished.Load(),
			server.mailSeen.Load(),
		)
	}
}

func TestSMTPMailerSTARTTLSMissingOrRejectedFailsBeforeMAIL(t *testing.T) {
	tests := []struct {
		name    string
		options smtpServerOptions
	}{
		{name: "extension missing"},
		{name: "upgrade rejected", options: smtpServerOptions{startTLS: true, rejectSTARTTLS: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const serverName = "smtp.example.test"
			certificate, roots := newTestCertificate(
				t,
				serverName,
				time.Now().Add(-time.Hour),
				time.Now().Add(time.Hour),
			)
			test.options.certificate = certificate
			server := newSMTPTestServer(t, test.options)
			mailer := newTestSMTPMailer(t, server, SMTPConfig{
				Address:   "pipe:starttls-failure",
				Host:      serverName,
				From:      "sender@example.test",
				Mode:      TLSModeSTARTTLSRequired,
				TLSConfig: testClientTLSConfig(serverName, roots),
			})

			err := mailer.Send(testContext(t), testMessage())
			if !errors.Is(err, ErrTLSRequired) {
				t.Fatalf("Send() error = %v, want %v", err, ErrTLSRequired)
			}
			if errors.Is(err, ErrMessageRejected) {
				t.Fatalf("Send() error = %v, TLS failure is not message rejection", err)
			}
			assertNoSMTPMessageRejection(t, err)
			server.wait(t)
			if server.mailSeen.Load() {
				t.Fatal("server received MAIL after STARTTLS failure")
			}
		})
	}
}

func TestSMTPMailerCertificateFailuresFailClosedBeforeMAIL(t *testing.T) {
	const correctServerName = "smtp.example.test"
	now := time.Now()
	tests := []struct {
		name             string
		certificateName  string
		clientServerName string
		notBefore        time.Time
		notAfter         time.Time
		trustCertificate bool
	}{
		{
			name:             "untrusted certificate",
			certificateName:  correctServerName,
			clientServerName: correctServerName,
			notBefore:        now.Add(-time.Hour),
			notAfter:         now.Add(time.Hour),
		},
		{
			name:             "wrong host",
			certificateName:  "other.example.test",
			clientServerName: correctServerName,
			notBefore:        now.Add(-time.Hour),
			notAfter:         now.Add(time.Hour),
			trustCertificate: true,
		},
		{
			name:             "expired certificate",
			certificateName:  correctServerName,
			clientServerName: correctServerName,
			notBefore:        now.Add(-2 * time.Hour),
			notAfter:         now.Add(-time.Hour),
			trustCertificate: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			certificate, roots := newTestCertificate(
				t,
				test.certificateName,
				test.notBefore,
				test.notAfter,
			)
			if !test.trustCertificate {
				roots = x509.NewCertPool()
			}
			server := newSMTPTestServer(t, smtpServerOptions{
				startTLS:    true,
				certificate: certificate,
			})
			mailer := newTestSMTPMailer(t, server, SMTPConfig{
				Address:   "pipe:certificate-failure",
				Host:      test.clientServerName,
				From:      "sender@example.test",
				Mode:      TLSModeSTARTTLSRequired,
				TLSConfig: testClientTLSConfig(test.clientServerName, roots),
			})

			err := mailer.Send(testContext(t), testMessage())
			if err == nil {
				t.Fatal("Send() error = nil, want certificate verification failure")
			}
			assertNoSMTPMessageRejection(t, err)
			server.wait(t)
			if server.mailSeen.Load() {
				t.Fatal("server received MAIL after certificate verification failure")
			}
		})
	}
}

func TestSMTPMailerRejectsPlaintextInsecureTLSAndMissingDeadline(t *testing.T) {
	const serverName = "smtp.example.test"
	_, roots := newTestCertificate(
		t,
		serverName,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
	valid := SMTPConfig{
		Address:   "pipe:validation",
		Host:      serverName,
		From:      "sender@example.test",
		Mode:      TLSModeImplicit,
		TLSConfig: testClientTLSConfig(serverName, roots),
	}

	for _, mutate := range []func(*SMTPConfig){
		func(config *SMTPConfig) { config.Mode = TLSMode("plaintext") },
		func(config *SMTPConfig) { config.TLSConfig.InsecureSkipVerify = true },
		func(config *SMTPConfig) { config.TLSConfig.ServerName = "" },
		func(config *SMTPConfig) { config.TLSConfig.MinVersion = 0 },
	} {
		config := valid
		config.TLSConfig = valid.TLSConfig.Clone()
		mutate(&config)
		if _, err := NewSMTPMailer(config, pipeDialer{}); err == nil {
			t.Fatal("NewSMTPMailer() error = nil, want fail-closed validation")
		}
	}

	mailer, err := NewSMTPMailer(valid, errorDialer{err: errors.New("must not dial")})
	if err != nil {
		t.Fatalf("NewSMTPMailer(valid) error = %v", err)
	}
	if err := mailer.Send(context.Background(), testMessage()); !errors.Is(err, ErrDeadlineRequired) {
		t.Fatalf("Send(without deadline) error = %v, want %v", err, ErrDeadlineRequired)
	} else {
		assertNoSMTPMessageRejection(t, err)
	}
}

func newTestSMTPMailer(t *testing.T, server *smtpTestServer, config SMTPConfig) *SMTPMailer {
	t.Helper()
	mailer, err := NewSMTPMailer(config, server.dialer())
	if err != nil {
		t.Fatalf("NewSMTPMailer() error = %v", err)
	}
	return mailer
}

func testMessage() Message {
	return Message{
		Purpose: PurposePasswordResetCredential,
		To:      "recipient@example.test",
		Subject: "Temporary credential",
		Body:    "controlled test body",
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func testClientTLSConfig(serverName string, roots *x509.CertPool) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
		RootCAs:    roots,
	}
}

type smtpTestServer struct {
	tlsEstablished atomic.Bool
	ehloSeen       atomic.Bool
	heloSeen       atomic.Bool
	mailSeen       atomic.Bool
	client         net.Conn
	server         net.Conn
	options        smtpServerOptions
	done           chan error
}

type smtpServerOptions struct {
	implicitTLS    bool
	startTLS       bool
	rejectSTARTTLS bool
	certificate    tls.Certificate
	auth           bool
	rejectGreeting bool
	rejectStage    smtpBlockStage
	rejectCode     int
	rejectMessage  string
}

type smtpBlockStage string

const (
	smtpBlockAuth           smtpBlockStage = "AUTH"
	smtpBlockMail           smtpBlockStage = "MAIL"
	smtpBlockRCPT           smtpBlockStage = "RCPT"
	smtpBlockDATACommand    smtpBlockStage = "DATA command"
	smtpBlockDATAAcceptance smtpBlockStage = "DATA acceptance"
	smtpBlockQUIT           smtpBlockStage = "QUIT"
)

type blockingSMTPTestServer struct {
	*smtpTestServer
	stage    smtpBlockStage
	blocked  chan struct{}
	commands []string
}

func newBlockingSMTPTestServer(
	t *testing.T,
	certificate tls.Certificate,
	stage smtpBlockStage,
) *blockingSMTPTestServer {
	t.Helper()
	client, server := net.Pipe()
	testServer := &blockingSMTPTestServer{
		smtpTestServer: &smtpTestServer{
			client: client,
			server: server,
			options: smtpServerOptions{
				implicitTLS: true,
				certificate: certificate,
			},
			done: make(chan error, 1),
		},
		stage:   stage,
		blocked: make(chan struct{}),
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	go func() {
		testServer.done <- testServer.serveBlocked()
	}()
	return testServer
}

func (s *blockingSMTPTestServer) waitBlocked(t *testing.T) {
	t.Helper()
	select {
	case <-s.blocked:
	case err := <-s.done:
		t.Fatalf("SMTP test server stopped before blocking: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("SMTP test server did not reach blocking stage")
	}
}

func (s *blockingSMTPTestServer) serveBlocked() error {
	defer s.server.Close()
	connection := tls.Server(s.server, &tls.Config{
		Certificates: []tls.Certificate{s.options.certificate},
		MinVersion:   tls.VersionTLS12,
	})
	if err := connection.Handshake(); err != nil {
		return err
	}
	reader := bufio.NewReader(connection)
	if _, err := fmt.Fprint(connection, "220 smtp.example.test ESMTP\r\n"); err != nil {
		return err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		command := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			s.commands = append(s.commands, "EHLO")
			if _, err := fmt.Fprint(
				connection,
				"250-smtp.example.test\r\n250 AUTH PLAIN\r\n",
			); err != nil {
				return err
			}
		case strings.HasPrefix(command, "AUTH"):
			s.commands = append(s.commands, "AUTH")
			if s.stage == smtpBlockAuth {
				return s.blockUntilDisconnect(reader)
			}
			if _, err := fmt.Fprint(connection, "235 authenticated\r\n"); err != nil {
				return err
			}
		case strings.HasPrefix(command, "MAIL FROM:"):
			s.commands = append(s.commands, "MAIL")
			if s.stage == smtpBlockMail {
				return s.blockUntilDisconnect(reader)
			}
			if _, err := fmt.Fprint(connection, "250 sender accepted\r\n"); err != nil {
				return err
			}
		case strings.HasPrefix(command, "RCPT TO:"):
			s.commands = append(s.commands, "RCPT")
			if s.stage == smtpBlockRCPT {
				return s.blockUntilDisconnect(reader)
			}
			if _, err := fmt.Fprint(connection, "250 recipient accepted\r\n"); err != nil {
				return err
			}
		case command == "DATA":
			s.commands = append(s.commands, "DATA")
			if s.stage == smtpBlockDATACommand {
				return s.blockUntilDisconnect(reader)
			}
			if _, err := fmt.Fprint(connection, "354 send data\r\n"); err != nil {
				return err
			}
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				if dataLine == ".\r\n" {
					break
				}
			}
			if s.stage == smtpBlockDATAAcceptance {
				return s.blockUntilDisconnect(reader)
			}
			if _, err := fmt.Fprint(connection, "250 queued\r\n"); err != nil {
				return err
			}
		case command == "QUIT":
			s.commands = append(s.commands, "QUIT")
			if s.stage == smtpBlockQUIT {
				return s.blockUntilDisconnect(reader)
			}
			_, err := fmt.Fprint(connection, "221 bye\r\n")
			return err
		default:
			return fmt.Errorf("unexpected SMTP command %q", command)
		}
	}
}

func (s *blockingSMTPTestServer) blockUntilDisconnect(reader *bufio.Reader) error {
	close(s.blocked)
	command, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	return fmt.Errorf("received SMTP data after blocked %s: %q", s.stage, command)
}

func newSMTPTestServer(t *testing.T, options smtpServerOptions) *smtpTestServer {
	t.Helper()
	client, server := net.Pipe()
	testServer := &smtpTestServer{
		client:  client,
		server:  server,
		options: options,
		done:    make(chan error, 1),
	}
	go func() {
		testServer.done <- testServer.serve()
	}()
	return testServer
}

func (s *smtpTestServer) dialer() Dialer {
	return pipeDialer{connection: s.client}
}

func (s *smtpTestServer) wait(t *testing.T) {
	t.Helper()
	select {
	case err := <-s.done:
		if err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, net.ErrClosed) &&
			!strings.Contains(err.Error(), "tls: bad certificate") {
			t.Fatalf("SMTP test server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SMTP test server did not stop")
	}
}

func (s *smtpTestServer) serve() error {
	defer s.server.Close()
	connection := s.server
	if s.options.implicitTLS {
		tlsConnection := tls.Server(connection, &tls.Config{
			Certificates: []tls.Certificate{s.options.certificate},
			MinVersion:   tls.VersionTLS12,
		})
		if err := tlsConnection.Handshake(); err != nil {
			return err
		}
		s.tlsEstablished.Store(true)
		connection = tlsConnection
	}

	reader := bufio.NewReader(connection)
	if _, err := fmt.Fprint(connection, "220 smtp.example.test ESMTP\r\n"); err != nil {
		return err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		command := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			if strings.HasPrefix(command, "EHLO") {
				s.ehloSeen.Store(true)
			} else {
				s.heloSeen.Store(true)
			}
			if s.options.rejectGreeting {
				if _, err := fmt.Fprintf(connection, "%d greeting rejected\r\n", s.options.rejectCode); err != nil {
					return err
				}
			} else if s.options.startTLS && !s.tlsEstablished.Load() {
				if _, err := fmt.Fprint(connection, "250-smtp.example.test\r\n250 STARTTLS\r\n"); err != nil {
					return err
				}
			} else if s.options.auth {
				if _, err := fmt.Fprint(connection, "250-smtp.example.test\r\n250 AUTH PLAIN\r\n"); err != nil {
					return err
				}
			} else if _, err := fmt.Fprint(connection, "250 smtp.example.test\r\n"); err != nil {
				return err
			}
		case command == "STARTTLS":
			if s.options.rejectSTARTTLS {
				if _, err := fmt.Fprint(connection, "454 TLS unavailable\r\n"); err != nil {
					return err
				}
				continue
			}
			if !s.options.startTLS {
				return fmt.Errorf("unexpected STARTTLS")
			}
			if _, err := fmt.Fprint(connection, "220 ready for TLS\r\n"); err != nil {
				return err
			}
			tlsConnection := tls.Server(connection, &tls.Config{
				Certificates: []tls.Certificate{s.options.certificate},
				MinVersion:   tls.VersionTLS12,
			})
			if err := tlsConnection.Handshake(); err != nil {
				return err
			}
			s.tlsEstablished.Store(true)
			connection = tlsConnection
			reader = bufio.NewReader(connection)
		case strings.HasPrefix(command, "AUTH"):
			if s.options.rejectStage == smtpBlockAuth {
				if _, err := fmt.Fprintf(connection, "%d authentication rejected\r\n", s.options.rejectCode); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprint(connection, "235 authenticated\r\n"); err != nil {
				return err
			}
		case command == "*":
			_, err := fmt.Fprint(connection, "501 authentication canceled\r\n")
			return err
		case strings.HasPrefix(command, "MAIL FROM:"):
			s.mailSeen.Store(true)
			if s.options.rejectStage == smtpBlockMail {
				if _, err := fmt.Fprintf(connection, "%d sender rejected%s\r\n", s.options.rejectCode, s.options.rejectMessage); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprint(connection, "250 sender accepted\r\n"); err != nil {
				return err
			}
		case strings.HasPrefix(command, "RCPT TO:"):
			if s.options.rejectStage == smtpBlockRCPT {
				if _, err := fmt.Fprintf(connection, "%d recipient rejected%s\r\n", s.options.rejectCode, s.options.rejectMessage); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprint(connection, "250 recipient accepted\r\n"); err != nil {
				return err
			}
		case command == "DATA":
			if s.options.rejectStage == smtpBlockDATACommand {
				if _, err := fmt.Fprintf(connection, "%d data rejected%s\r\n", s.options.rejectCode, s.options.rejectMessage); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprint(connection, "354 send data\r\n"); err != nil {
				return err
			}
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				if dataLine == ".\r\n" {
					break
				}
			}
			if s.options.rejectStage == smtpBlockDATAAcceptance {
				if _, err := fmt.Fprintf(connection, "%d message rejected%s\r\n", s.options.rejectCode, s.options.rejectMessage); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprint(connection, "250 queued\r\n"); err != nil {
				return err
			}
		case command == "QUIT":
			if s.options.rejectStage == smtpBlockQUIT {
				if s.options.rejectCode == 0 {
					return nil
				}
				_, err := fmt.Fprintf(connection, "%d QUIT rejected\r\n", s.options.rejectCode)
				return err
			}
			_, err := fmt.Fprint(connection, "221 bye\r\n")
			return err
		default:
			return fmt.Errorf("unexpected SMTP command %q", command)
		}
	}
}

type pipeDialer struct {
	connection net.Conn
}

func (d pipeDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	if d.connection == nil {
		return nil, errors.New("pipe connection is not configured")
	}
	return d.connection, nil
}

type smtpActivityConn struct {
	net.Conn
	activity chan struct{}
}

func (c *smtpActivityConn) Close() error {
	c.activity <- struct{}{}
	return c.Conn.Close()
}

func (c *smtpActivityConn) SetDeadline(deadline time.Time) error {
	c.activity <- struct{}{}
	return c.Conn.SetDeadline(deadline)
}

type errorDialer struct {
	err error
}

func (d errorDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, d.err
}

func newTestCertificate(
	t *testing.T,
	serverName string,
	notBefore time.Time,
	notAfter time.Time,
) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Han Test CA"},
		NotBefore:             notBefore.Add(-time.Hour),
		NotAfter:              notAfter.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create test CA: %v", err)
	}
	caCertificate, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse test CA: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: serverName},
		DNSNames:     []string{serverName},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	serverDER, err := x509.CreateCertificate(
		rand.Reader,
		serverTemplate,
		caCertificate,
		&serverKey.PublicKey,
		caKey,
	)
	if err != nil {
		t.Fatalf("create test server certificate: %v", err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
	})
	certificate, err := tls.X509KeyPair(certificatePEM, privateKeyPEM)
	if err != nil {
		t.Fatalf("load test server certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCertificate)
	return certificate, roots
}
