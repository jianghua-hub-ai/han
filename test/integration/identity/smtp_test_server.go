//go:build integration

package identity

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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/mailer"
)

const smtpTestServerName = "smtp.integration.test"

const (
	smtpTestIOTimeout      = 5 * time.Second
	smtpTestCleanupTimeout = 2 * time.Second
)

type smtpTestStage string

const (
	smtpStageTLS      smtpTestStage = "tls"
	smtpStageGreeting smtpTestStage = "greeting"
	smtpStageAuth     smtpTestStage = "auth"
	smtpStageMail     smtpTestStage = "mail"
	smtpStageRCPT     smtpTestStage = "rcpt"
	smtpStageDATA     smtpTestStage = "data"
	smtpStageResult   smtpTestStage = "result"
)

type smtpTestServerOptions struct {
	mode               mailer.TLSMode
	advertiseSTARTTLS  bool
	advertiseAuth      bool
	rejectStage        smtpTestStage
	rejectCode         int
	disconnectStage    smtpTestStage
	expectedCloseStage smtpTestStage
}

type smtpTestSnapshot struct {
	connections        int
	messages           int
	commands           []string
	protocolViolations []string
	authBeforeTLS      bool
	envelopeBeforeTLS  bool
	tlsVersions        []uint16
	closeStage         smtpTestStage
}

type smtpTestServer struct {
	t           *testing.T
	listener    net.Listener
	certificate tls.Certificate
	roots       *x509.CertPool
	options     smtpTestServerOptions

	mu                sync.Mutex
	snapshot          smtpTestSnapshot
	activeConnections map[net.Conn]struct{}
	closing           bool
	done              chan struct{}
	accepted          chan struct{}
	closeOnce         sync.Once
	reportOnce        sync.Once
	serveErr          error
}

func newSMTPTestServer(t *testing.T, options smtpTestServerOptions) *smtpTestServer {
	t.Helper()
	certificate, roots := newSMTPTestCertificate(t, smtpTestServerName)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on loopback SMTP endpoint: %v", err)
	}
	server := &smtpTestServer{
		t:                 t,
		listener:          listener,
		certificate:       certificate,
		roots:             roots,
		options:           options,
		activeConnections: make(map[net.Conn]struct{}),
		done:              make(chan struct{}),
		accepted:          make(chan struct{}),
	}
	go server.serve()
	t.Cleanup(server.close)
	return server
}

func (s *smtpTestServer) address() string {
	return s.listener.Addr().String()
}

func (s *smtpTestServer) clientTLSConfig(serverName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
		RootCAs:    s.roots,
	}
}

func (s *smtpTestServer) wait(t *testing.T) smtpTestSnapshot {
	t.Helper()
	if !s.waitForDone(smtpTestIOTimeout) {
		t.Fatal("local SMTP server did not stop")
	}
	s.reportServeError(t)
	return s.readSnapshot()
}

func (s *smtpTestServer) readSnapshot() smtpTestSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.snapshot
	result.commands = append([]string(nil), s.snapshot.commands...)
	result.protocolViolations = append(
		[]string(nil),
		s.snapshot.protocolViolations...,
	)
	result.tlsVersions = append([]uint16(nil), s.snapshot.tlsVersions...)
	return result
}

func (s *smtpTestServer) close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closing = true
		connections := make([]net.Conn, 0, len(s.activeConnections))
		for connection := range s.activeConnections {
			connections = append(connections, connection)
		}
		s.mu.Unlock()

		_ = s.listener.Close()
		for _, connection := range connections {
			_ = connection.Close()
		}
	})
	if !s.waitForDone(smtpTestCleanupTimeout) {
		s.t.Errorf("local SMTP server cleanup timed out with active connections")
		return
	}
	s.reportServeError(s.t)
}

func (s *smtpTestServer) serve() {
	defer close(s.done)
	connection, err := s.listener.Accept()
	if err != nil {
		s.setServeError(smtpStageGreeting, err)
		return
	}
	_ = s.listener.Close()
	rawConnection := connection
	s.trackConnection(rawConnection)
	close(s.accepted)
	defer func() {
		s.untrackConnection(rawConnection)
		_ = rawConnection.Close()
	}()
	if err := connection.SetDeadline(time.Now().Add(smtpTestIOTimeout)); err != nil {
		s.setServeError(smtpStageGreeting, err)
		return
	}

	tlsEstablished := false
	if s.options.mode == mailer.TLSModeImplicit {
		tlsConnection, tlsErr := s.upgradeTLS(connection)
		err = tlsErr
		if err != nil {
			s.setServeError(smtpStageTLS, err)
			return
		}
		connection = tlsConnection
		tlsEstablished = true
	}
	s.setServeError(smtpStageGreeting, s.serveConnection(connection, tlsEstablished))
}

func (s *smtpTestServer) serveConnection(connection net.Conn, tlsEstablished bool) error {
	reader := bufio.NewReader(connection)
	state := smtpTestSessionState{tlsEstablished: tlsEstablished}
	if _, err := fmt.Fprint(connection, "220 smtp.integration.test ESMTP\r\n"); err != nil {
		return err
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		command := strings.TrimSpace(line)
		fields := strings.Fields(command)
		if len(fields) == 0 {
			if _, err := fmt.Fprint(connection, "500 empty command\r\n"); err != nil {
				return err
			}
			continue
		}
		verb := strings.ToUpper(fields[0])
		s.recordCommand(verb)
		if reason := state.protocolViolation(verb, s.options); reason != "" {
			s.recordProtocolViolation(verb + ": " + reason)
			if _, err := fmt.Fprint(connection, "503 bad command sequence\r\n"); err != nil {
				return err
			}
			continue
		}

		switch verb {
		case "EHLO":
			if s.options.rejectStage == smtpStageGreeting {
				code := smtpTestReplyCode(s.options.rejectCode, 550)
				if _, err := fmt.Fprintf(connection, "%d EHLO rejected\r\n", code); err != nil {
					return err
				}
				continue
			}
			state.greeted = true
			extensions := []string{"250-smtp.integration.test"}
			if !tlsEstablished && s.options.advertiseSTARTTLS {
				extensions = append(extensions, "250-STARTTLS")
			}
			if tlsEstablished && s.options.advertiseAuth {
				extensions = append(extensions, "250-AUTH PLAIN")
			}
			extensions = append(extensions, "250 PIPELINING")
			if _, err := fmt.Fprint(connection, strings.Join(extensions, "\r\n")+"\r\n"); err != nil {
				return err
			}
		case "HELO":
			if s.options.rejectStage == smtpStageGreeting {
				code := smtpTestReplyCode(s.options.rejectCode, 550)
				if _, err := fmt.Fprintf(connection, "%d HELO rejected\r\n", code); err != nil {
					return err
				}
				continue
			}
			state.greeted = true
			if _, err := fmt.Fprint(connection, "250 smtp.integration.test\r\n"); err != nil {
				return err
			}
		case "STARTTLS":
			if !s.options.advertiseSTARTTLS {
				return errors.New("received STARTTLS without advertising it")
			}
			if _, err := fmt.Fprint(connection, "220 ready for TLS\r\n"); err != nil {
				return err
			}
			connection, err = s.upgradeTLS(connection)
			if err != nil {
				return err
			}
			reader = bufio.NewReader(connection)
			state = smtpTestSessionState{tlsEstablished: true}
		case "AUTH":
			if !state.tlsEstablished {
				s.mu.Lock()
				s.snapshot.authBeforeTLS = true
				s.mu.Unlock()
			}
			if s.options.rejectStage == smtpStageAuth {
				code := smtpTestReplyCode(s.options.rejectCode, 535)
				if _, err := fmt.Fprintf(connection, "%d authentication rejected\r\n", code); err != nil {
					return err
				}
				continue
			}
			state.authenticated = true
			if _, err := fmt.Fprint(connection, "235 authenticated\r\n"); err != nil {
				return err
			}
		case "*":
			if _, err := fmt.Fprint(connection, "501 authentication canceled\r\n"); err != nil {
				return err
			}
		case "MAIL":
			s.recordEnvelopeTLS(state.tlsEstablished)
			if s.options.disconnectStage == smtpStageMail {
				s.recordCloseStage(smtpStageMail)
				return nil
			}
			if s.options.rejectStage == smtpStageMail {
				return writeSMTPTestReply(connection, s.options.rejectCode, 550, "sender rejected")
			}
			if _, err := fmt.Fprint(connection, "250 sender accepted\r\n"); err != nil {
				return err
			}
			state.mailAccepted = true
		case "RCPT":
			s.recordEnvelopeTLS(state.tlsEstablished)
			if s.options.disconnectStage == smtpStageRCPT {
				s.recordCloseStage(smtpStageRCPT)
				return nil
			}
			if s.options.rejectStage == smtpStageRCPT {
				return writeSMTPTestReply(connection, s.options.rejectCode, 550, "recipient rejected")
			}
			if _, err := fmt.Fprint(connection, "250 recipient accepted\r\n"); err != nil {
				return err
			}
			state.recipientAccepted = true
		case "DATA":
			s.recordEnvelopeTLS(state.tlsEstablished)
			if s.options.rejectStage == smtpStageDATA {
				return writeSMTPTestReply(connection, s.options.rejectCode, 554, "DATA rejected")
			}
			if _, err := fmt.Fprint(connection, "354 send message\r\n"); err != nil {
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
			if s.options.disconnectStage == smtpStageResult {
				s.recordCloseStage(smtpStageResult)
				return nil
			}
			if s.options.rejectStage == smtpStageResult {
				return writeSMTPTestReply(connection, s.options.rejectCode, 554, "message rejected")
			}
			s.mu.Lock()
			s.snapshot.messages++
			s.mu.Unlock()
			state.dataAccepted = true
			if _, err := fmt.Fprint(connection, "250 queued\r\n"); err != nil {
				return err
			}
		case "QUIT":
			_, err := fmt.Fprint(connection, "221 bye\r\n")
			return err
		default:
			return fmt.Errorf("unexpected SMTP command %q", command)
		}
	}
}

type smtpTestSessionState struct {
	tlsEstablished    bool
	greeted           bool
	authenticated     bool
	mailAccepted      bool
	recipientAccepted bool
	dataAccepted      bool
}

func (s smtpTestSessionState) protocolViolation(
	verb string,
	options smtpTestServerOptions,
) string {
	switch verb {
	case "EHLO", "HELO":
		if s.mailAccepted || s.recipientAccepted || s.dataAccepted {
			return "greeting after envelope"
		}
	case "STARTTLS":
		if s.tlsEstablished || !s.greeted || !options.advertiseSTARTTLS {
			return "STARTTLS before greeting or after TLS"
		}
	case "AUTH":
		if !s.tlsEstablished || !s.greeted || !options.advertiseAuth ||
			s.mailAccepted || s.authenticated {
			return "AUTH outside authenticated greeting phase"
		}
	case "MAIL":
		if !s.tlsEstablished || !s.greeted || s.mailAccepted ||
			(options.advertiseAuth && !s.authenticated) {
			return "MAIL before secure greeting and required AUTH"
		}
	case "RCPT":
		if !s.mailAccepted || s.recipientAccepted {
			return "RCPT before MAIL"
		}
	case "DATA":
		if !s.recipientAccepted || s.dataAccepted {
			return "DATA before RCPT"
		}
	}
	return ""
}

func (s *smtpTestServer) upgradeTLS(connection net.Conn) (net.Conn, error) {
	tlsConnection := tls.Server(connection, &tls.Config{
		Certificates: []tls.Certificate{s.certificate},
		MinVersion:   tls.VersionTLS12,
	})
	if err := tlsConnection.Handshake(); err != nil {
		return nil, err
	}
	if err := tlsConnection.SetDeadline(time.Now().Add(smtpTestIOTimeout)); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.snapshot.tlsVersions = append(s.snapshot.tlsVersions, tlsConnection.ConnectionState().Version)
	s.mu.Unlock()
	return tlsConnection, nil
}

func (s *smtpTestServer) recordCommand(command string) {
	s.mu.Lock()
	s.snapshot.commands = append(s.snapshot.commands, command)
	s.mu.Unlock()
}

func (s *smtpTestServer) recordProtocolViolation(violation string) {
	s.mu.Lock()
	s.snapshot.protocolViolations = append(s.snapshot.protocolViolations, violation)
	s.mu.Unlock()
}

func (s *smtpTestServer) recordCloseStage(stage smtpTestStage) {
	s.mu.Lock()
	s.snapshot.closeStage = stage
	s.mu.Unlock()
}

func (s *smtpTestServer) recordEnvelopeTLS(tlsEstablished bool) {
	if tlsEstablished {
		return
	}
	s.mu.Lock()
	s.snapshot.envelopeBeforeTLS = true
	s.mu.Unlock()
}

func (s *smtpTestServer) trackConnection(connection net.Conn) {
	s.mu.Lock()
	s.activeConnections[connection] = struct{}{}
	s.snapshot.connections++
	s.mu.Unlock()
}

func (s *smtpTestServer) untrackConnection(connection net.Conn) {
	s.mu.Lock()
	delete(s.activeConnections, connection)
	s.mu.Unlock()
}

func (s *smtpTestServer) setServeError(stage smtpTestStage, err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return
	}
	if s.options.expectedCloseStage == stage {
		s.snapshot.closeStage = stage
		return
	}
	if s.serveErr == nil {
		s.serveErr = fmt.Errorf("%s stage: %w", stage, err)
	}
}

func (s *smtpTestServer) waitForDone(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-s.done:
		return true
	case <-timer.C:
		return false
	}
}

func (s *smtpTestServer) waitForConnection(t *testing.T) {
	t.Helper()
	timer := time.NewTimer(smtpTestIOTimeout)
	defer timer.Stop()
	select {
	case <-s.accepted:
	case <-timer.C:
		t.Fatal("local SMTP server did not accept the client")
	}
}

func (s *smtpTestServer) reportServeError(t *testing.T) {
	t.Helper()
	s.reportOnce.Do(func() {
		s.mu.Lock()
		err := s.serveErr
		s.mu.Unlock()
		if err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, net.ErrClosed) {
			t.Errorf("local SMTP server error: %v", err)
		}
	})
}

func (s *smtpTestServer) activeConnectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.activeConnections)
}

type smtpTestProtocolClient struct {
	connection net.Conn
	reader     *bufio.Reader
}

func newSMTPTestProtocolClient(
	t *testing.T,
	server *smtpTestServer,
	implicitTLS bool,
) *smtpTestProtocolClient {
	t.Helper()
	connection, err := net.DialTimeout("tcp", server.address(), smtpTestIOTimeout)
	if err != nil {
		t.Fatalf("dial local SMTP fixture: %v", err)
	}
	if err := connection.SetDeadline(time.Now().Add(smtpTestIOTimeout)); err != nil {
		_ = connection.Close()
		t.Fatalf("set local SMTP client deadline: %v", err)
	}
	if implicitTLS {
		connection = smtpTestClientTLS(t, connection, server)
	}
	client := &smtpTestProtocolClient{
		connection: connection,
		reader:     bufio.NewReader(connection),
	}
	t.Cleanup(func() {
		_ = client.connection.Close()
	})
	if code := client.readReplyCode(t); code != 220 {
		t.Fatalf("SMTP greeting code = %d, want 220", code)
	}
	return client
}

func (c *smtpTestProtocolClient) command(t *testing.T, command string) int {
	t.Helper()
	if _, err := fmt.Fprintf(c.connection, "%s\r\n", command); err != nil {
		t.Fatalf("write SMTP command %q: %v", command, err)
	}
	return c.readReplyCode(t)
}

func (c *smtpTestProtocolClient) upgradeTLS(
	t *testing.T,
	server *smtpTestServer,
) {
	t.Helper()
	c.connection = smtpTestClientTLS(t, c.connection, server)
	c.reader = bufio.NewReader(c.connection)
}

func (c *smtpTestProtocolClient) readReplyCode(t *testing.T) int {
	t.Helper()
	var code int
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SMTP reply: %v", err)
		}
		if len(line) < 4 {
			t.Fatalf("malformed SMTP reply %q", line)
		}
		current, err := strconv.Atoi(line[:3])
		if err != nil {
			t.Fatalf("parse SMTP reply code %q: %v", line[:3], err)
		}
		if code == 0 {
			code = current
		}
		if current != code {
			t.Fatalf("inconsistent SMTP multiline reply code %d, want %d", current, code)
		}
		if line[3] != '-' {
			return code
		}
	}
}

func smtpTestClientTLS(
	t *testing.T,
	connection net.Conn,
	server *smtpTestServer,
) net.Conn {
	t.Helper()
	tlsConnection := tls.Client(
		connection,
		server.clientTLSConfig(smtpTestServerName),
	)
	if err := tlsConnection.Handshake(); err != nil {
		_ = connection.Close()
		t.Fatalf("negotiate local SMTP client TLS: %v", err)
	}
	return tlsConnection
}

func writeSMTPTestReply(
	connection net.Conn,
	code int,
	fallback int,
	message string,
) error {
	_, err := fmt.Fprintf(
		connection,
		"%d %s\r\n",
		smtpTestReplyCode(code, fallback),
		message,
	)
	return err
}

func smtpTestReplyCode(code int, fallback int) int {
	if code == 0 {
		return fallback
	}
	return code
}

func newSMTPTestMailer(
	t *testing.T,
	server *smtpTestServer,
	mode mailer.TLSMode,
	withAuth bool,
) *mailer.SMTPMailer {
	t.Helper()
	config := mailer.SMTPConfig{
		Address:   server.address(),
		Host:      smtpTestServerName,
		From:      "sender@example.test",
		Mode:      mode,
		TLSConfig: server.clientTLSConfig(smtpTestServerName),
	}
	if withAuth {
		config.Auth = smtp.PlainAuth(
			"",
			"synthetic-user",
			"synthetic-password",
			smtpTestServerName,
		)
	}
	result, err := mailer.NewSMTPMailer(config, &net.Dialer{})
	if err != nil {
		t.Fatalf("create SMTP mailer: %v", err)
	}
	return result
}

func newSMTPTestCertificate(
	t *testing.T,
	serverName string,
) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	now := time.Now().UTC()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate SMTP test CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Han SMTP Integration Test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(
		rand.Reader,
		caTemplate,
		caTemplate,
		&caKey.PublicKey,
		caKey,
	)
	if err != nil {
		t.Fatalf("create SMTP test CA: %v", err)
	}
	caCertificate, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse SMTP test CA: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate SMTP test server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: serverName},
		DNSNames:     []string{serverName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
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
		t.Fatalf("create SMTP test server certificate: %v", err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverDER,
	})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
	})
	certificate, err := tls.X509KeyPair(certificatePEM, privateKeyPEM)
	if err != nil {
		t.Fatalf("load SMTP test certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCertificate)
	return certificate, roots
}

func smtpTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func containsSMTPCommand(commands []string, command string) bool {
	for _, current := range commands {
		if current == command {
			return true
		}
	}
	return false
}

func compactSMTPTestSQL(statement string) string {
	return strings.Join(strings.Fields(statement), " ")
}

func smtpTestID(prefix string, value int) string {
	return prefix + "-" + strconv.Itoa(value)
}
