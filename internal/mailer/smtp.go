package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
)

var (
	// ErrTLSRequired indicates that verified SMTP transport security was not established.
	ErrTLSRequired = errors.New("verified SMTP TLS is required")
	// ErrInvalidSMTPConfig indicates incomplete or insecure adapter configuration.
	ErrInvalidSMTPConfig = errors.New("SMTP configuration is invalid")
	// ErrDeadlineRequired indicates that SMTP I/O was requested without a caller deadline.
	ErrDeadlineRequired = errors.New("SMTP context deadline is required")
)

// SMTPMessageStage identifies an explicit message rejection's protocol stage.
type SMTPMessageStage string

const (
	// SMTPMessageStageMAILFrom identifies a MAIL FROM rejection.
	SMTPMessageStageMAILFrom SMTPMessageStage = "mail_from"
	// SMTPMessageStageRCPTTo identifies a RCPT TO rejection.
	SMTPMessageStageRCPTTo SMTPMessageStage = "rcpt_to"
	// SMTPMessageStageDATACommand identifies a rejection before the message body.
	SMTPMessageStageDATACommand SMTPMessageStage = "data_command"
	// SMTPMessageStageDATAFinal identifies a rejection after the message body.
	SMTPMessageStageDATAFinal SMTPMessageStage = "data_final"
)

// SMTPMessageRejection preserves an explicit rejection without printing its cause.
type SMTPMessageRejection struct {
	Stage SMTPMessageStage
	Code  int
	cause error
}

// Error returns only a fixed stage label and the numeric SMTP reply code.
func (e *SMTPMessageRejection) Error() string {
	stage := e.Stage
	switch stage {
	case SMTPMessageStageMAILFrom, SMTPMessageStageRCPTTo,
		SMTPMessageStageDATACommand, SMTPMessageStageDATAFinal:
	default:
		stage = "unknown"
	}
	return fmt.Sprintf("SMTP message rejected: stage=%s code=%d", stage, e.Code)
}

// Unwrap preserves the original error for errors.Is and errors.As consumers.
func (e *SMTPMessageRejection) Unwrap() error {
	return e.cause
}

// Is retains the existing explicit-message-rejection sentinel contract.
func (e *SMTPMessageRejection) Is(target error) bool {
	return target == ErrMessageRejected
}

// TLSMode is the only permitted SMTP transport mode.
type TLSMode string

const (
	// TLSModeImplicit establishes TLS before reading the SMTP greeting.
	TLSModeImplicit TLSMode = "implicit_tls"
	// TLSModeSTARTTLSRequired requires an advertised and successful STARTTLS upgrade.
	TLSModeSTARTTLSRequired TLSMode = "starttls_required"
)

// Dialer opens the injected SMTP transport endpoint.
type Dialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

// SMTPConfig contains explicit non-default SMTP transport inputs.
type SMTPConfig struct {
	Address   string
	Host      string
	From      string
	Mode      TLSMode
	TLSConfig *tls.Config
	Auth      smtp.Auth
}

// SMTPMailer sends through verified implicit TLS or required STARTTLS.
type SMTPMailer struct {
	config SMTPConfig
	dialer Dialer
}

// NewSMTPMailer creates a fail-closed SMTP adapter.
func NewSMTPMailer(config SMTPConfig, dialer Dialer) (*SMTPMailer, error) {
	if dialer == nil ||
		strings.TrimSpace(config.Address) == "" ||
		strings.TrimSpace(config.Host) == "" ||
		validateMailbox(config.From) != nil ||
		config.TLSConfig == nil ||
		config.TLSConfig.InsecureSkipVerify ||
		strings.TrimSpace(config.TLSConfig.ServerName) == "" ||
		config.TLSConfig.MinVersion == 0 ||
		(config.TLSConfig.MaxVersion != 0 &&
			config.TLSConfig.MaxVersion < config.TLSConfig.MinVersion) {
		return nil, ErrInvalidSMTPConfig
	}
	if config.Mode != TLSModeImplicit && config.Mode != TLSModeSTARTTLSRequired {
		return nil, ErrInvalidSMTPConfig
	}
	config.Address = strings.TrimSpace(config.Address)
	config.Host = strings.TrimSpace(config.Host)
	config.From = strings.TrimSpace(config.From)
	config.TLSConfig = config.TLSConfig.Clone()
	return &SMTPMailer{config: config, dialer: dialer}, nil
}

// Send establishes verified TLS before issuing MAIL FROM.
func (m *SMTPMailer) Send(ctx context.Context, message Message) error {
	if m == nil || m.dialer == nil || m.config.TLSConfig == nil {
		return ErrInvalidSMTPConfig
	}
	if err := ValidateMessage(message); err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return ErrDeadlineRequired
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	connection, err := m.dialer.DialContext(ctx, "tcp", m.config.Address)
	if err != nil {
		return fmt.Errorf("dial SMTP endpoint: %w", err)
	}
	closeConnection := true
	defer func() {
		if closeConnection {
			_ = connection.Close()
		}
	}()
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}
	stopCancellationWatcher := interruptConnectionOnCancel(ctx, connection)
	defer stopCancellationWatcher()

	client, err := m.secureClient(ctx, connection)
	if err != nil {
		return err
	}
	defer client.Close()
	closeConnection = false

	if m.config.Auth != nil {
		if err := client.Auth(m.config.Auth); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := client.Mail(m.config.From); err != nil {
		return classifySMTPMessageError("send SMTP MAIL command", SMTPMessageStageMAILFrom, err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return classifySMTPMessageError("send SMTP RCPT command", SMTPMessageStageRCPTTo, err)
	}
	writer, err := client.Data()
	if err != nil {
		return classifySMTPMessageError("start SMTP DATA", SMTPMessageStageDATACommand, err)
	}
	if _, err := io.WriteString(writer, formatMessage(m.config.From, message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP DATA: %w", err)
	}
	if err := writer.Close(); err != nil {
		return classifySMTPMessageError("finish SMTP DATA", SMTPMessageStageDATAFinal, err)
	}
	// DATA has already received the server's acceptance reply. A subsequent
	// connection-close failure must not turn an accepted message into a retry.
	_ = client.Quit()
	return nil
}

func classifySMTPMessageError(action string, stage SMTPMessageStage, err error) error {
	var protocolError *textproto.Error
	if errors.As(err, &protocolError) &&
		protocolError.Code >= 400 &&
		protocolError.Code <= 599 {
		return &SMTPMessageRejection{Stage: stage, Code: protocolError.Code, cause: err}
	}
	return fmt.Errorf("%s: %w", action, err)
}

func interruptConnectionOnCancel(ctx context.Context, connection net.Conn) func() {
	watcherDone := make(chan struct{})
	stopWatcher := context.AfterFunc(ctx, func() {
		defer close(watcherDone)
		_ = connection.Close()
	})
	return func() {
		if stopWatcher() {
			close(watcherDone)
		}
		<-watcherDone
	}
}

func (m *SMTPMailer) secureClient(
	ctx context.Context,
	connection net.Conn,
) (*smtp.Client, error) {
	tlsConfig := m.config.TLSConfig.Clone()
	switch m.config.Mode {
	case TLSModeImplicit:
		tlsConnection := tls.Client(connection, tlsConfig)
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			return nil, fmt.Errorf("%w: implicit handshake failed: %w", ErrTLSRequired, err)
		}
		client, err := smtp.NewClient(tlsConnection, m.config.Host)
		if err != nil {
			return nil, fmt.Errorf("create implicit TLS SMTP client: %w", err)
		}
		if err := client.Hello("localhost"); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("negotiate implicit TLS SMTP greeting: %w", err)
		}
		return client, nil
	case TLSModeSTARTTLSRequired:
		client, err := smtp.NewClient(connection, m.config.Host)
		if err != nil {
			return nil, fmt.Errorf("create STARTTLS SMTP client: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, fmt.Errorf("%w: STARTTLS extension is absent", ErrTLSRequired)
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("%w: STARTTLS upgrade failed: %w", ErrTLSRequired, err)
		}
		return client, nil
	default:
		return nil, ErrInvalidSMTPConfig
	}
}

func formatMessage(from string, message Message) string {
	var builder strings.Builder
	builder.WriteString("From: ")
	builder.WriteString(from)
	builder.WriteString("\r\nTo: ")
	builder.WriteString(message.To)
	builder.WriteString("\r\nSubject: ")
	builder.WriteString(message.Subject)
	builder.WriteString("\r\nContent-Type: text/plain; charset=UTF-8")
	builder.WriteString("\r\nX-Han-Purpose: ")
	builder.WriteString(string(message.Purpose))
	builder.WriteString("\r\n\r\n")
	body := strings.ReplaceAll(message.Body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")
	builder.WriteString(body)
	if !strings.HasSuffix(body, "\r\n") {
		builder.WriteString("\r\n")
	}
	return builder.String()
}
