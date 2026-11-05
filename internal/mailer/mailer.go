// Package mailer defines purpose-bound email delivery contracts.
package mailer

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

var (
	// ErrInvalidMessage indicates incomplete or unsafe message input.
	ErrInvalidMessage = errors.New("mail message is invalid")
	// ErrUnsupportedPurpose indicates a purpose outside the approved allowlist.
	ErrUnsupportedPurpose = errors.New("mail purpose is unsupported")
	// ErrMessageRejected indicates that the provider explicitly did not accept the message.
	ErrMessageRejected = errors.New("mail message was rejected")
)

// Purpose is the approved business reason for one delivery.
type Purpose string

const (
	// PurposeNewAdministratorCredential delivers an initial administrator credential.
	PurposeNewAdministratorCredential Purpose = "new_administrator_credential"
	// PurposePasswordResetCredential delivers a password-reset credential.
	PurposePasswordResetCredential Purpose = "password_reset_credential"
	// PurposeTemporaryCredentialResend delivers a replacement temporary credential.
	PurposeTemporaryCredentialResend Purpose = "temporary_credential_resend"
	// PurposeOrphanArticleTransferNotice delivers an orphan transfer notice.
	PurposeOrphanArticleTransferNotice Purpose = "orphan_article_transfer_notice"
)

// Message is one single-recipient purpose-bound email.
type Message struct {
	Purpose Purpose
	To      string
	Subject string
	Body    string
}

// Mailer sends one message or returns without claiming acceptance.
type Mailer interface {
	Send(context.Context, Message) error
}

// ValidateMessage validates the approved purpose and injection-safe envelope.
func ValidateMessage(message Message) error {
	switch message.Purpose {
	case PurposeNewAdministratorCredential,
		PurposePasswordResetCredential,
		PurposeTemporaryCredentialResend,
		PurposeOrphanArticleTransferNotice:
	default:
		return ErrUnsupportedPurpose
	}
	if err := validateMailbox(message.To); err != nil {
		return fmt.Errorf("%w: recipient", ErrInvalidMessage)
	}
	if strings.TrimSpace(message.Subject) == "" ||
		strings.ContainsAny(message.Subject, "\r\n") ||
		strings.TrimSpace(message.Body) == "" {
		return ErrInvalidMessage
	}
	return nil
}

func validateMailbox(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return ErrInvalidMessage
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return ErrInvalidMessage
	}
	return nil
}
