//go:build integration

package identity

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	identitypkg "han/internal/identity"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCredentialSMTPProtocolClassification(t *testing.T) {
	tests := []struct {
		name         string
		options      smtpTestServerOptions
		mode         mailer.TLSMode
		withAuth     bool
		serverName   string
		wantErr      error
		wantRejected bool
		wantMessages int
		wantCommands []string
		wantClose    smtpTestStage
	}{
		{
			name: "implicit TLS accepted after authenticated negotiation",
			options: smtpTestServerOptions{
				mode:          mailer.TLSModeImplicit,
				advertiseAuth: true,
			},
			mode:         mailer.TLSModeImplicit,
			withAuth:     true,
			serverName:   smtpTestServerName,
			wantMessages: 1,
			wantCommands: []string{"EHLO", "AUTH", "MAIL", "RCPT", "DATA", "QUIT"},
		},
		{
			name: "required STARTTLS accepted only after upgrade",
			options: smtpTestServerOptions{
				mode:              mailer.TLSModeSTARTTLSRequired,
				advertiseSTARTTLS: true,
			},
			mode:         mailer.TLSModeSTARTTLSRequired,
			serverName:   smtpTestServerName,
			wantMessages: 1,
			wantCommands: []string{
				"EHLO", "STARTTLS", "EHLO", "MAIL", "RCPT", "DATA", "QUIT",
			},
		},
		{
			name: "RCPT rejection is explicit non-acceptance",
			options: smtpTestServerOptions{
				mode:        mailer.TLSModeImplicit,
				rejectStage: smtpStageRCPT,
				rejectCode:  550,
			},
			mode:         mailer.TLSModeImplicit,
			serverName:   smtpTestServerName,
			wantRejected: true,
		},
		{
			name: "disconnect before DATA result is unknown",
			options: smtpTestServerOptions{
				mode:            mailer.TLSModeImplicit,
				disconnectStage: smtpStageResult,
			},
			mode:       mailer.TLSModeImplicit,
			serverName: smtpTestServerName,
			wantErr:    errors.New("unknown"),
			wantClose:  smtpStageResult,
		},
		{
			name: "implicit TLS EHLO and HELO rejection with nil Auth is unknown",
			options: smtpTestServerOptions{
				mode:        mailer.TLSModeImplicit,
				rejectStage: smtpStageGreeting,
				rejectCode:  550,
			},
			mode:       mailer.TLSModeImplicit,
			serverName: smtpTestServerName,
			wantErr:    errors.New("unknown"),
		},
		{
			name: "unexpected non-negative MAIL reply is unknown",
			options: smtpTestServerOptions{
				mode:        mailer.TLSModeImplicit,
				rejectStage: smtpStageMail,
				rejectCode:  354,
			},
			mode:       mailer.TLSModeImplicit,
			serverName: smtpTestServerName,
			wantErr:    errors.New("unknown"),
		},
		{
			name: "missing STARTTLS fails closed",
			options: smtpTestServerOptions{
				mode: mailer.TLSModeSTARTTLSRequired,
			},
			mode:       mailer.TLSModeSTARTTLSRequired,
			serverName: smtpTestServerName,
			wantErr:    mailer.ErrTLSRequired,
		},
		{
			name: "certificate hostname mismatch fails closed",
			options: smtpTestServerOptions{
				mode:               mailer.TLSModeImplicit,
				expectedCloseStage: smtpStageTLS,
			},
			mode:       mailer.TLSModeImplicit,
			serverName: "wrong.integration.test",
			wantErr:    mailer.ErrTLSRequired,
			wantClose:  smtpStageTLS,
		},
		{
			name: "authentication rejection remains unknown",
			options: smtpTestServerOptions{
				mode:          mailer.TLSModeImplicit,
				advertiseAuth: true,
				rejectStage:   smtpStageAuth,
				rejectCode:    535,
			},
			mode:       mailer.TLSModeImplicit,
			withAuth:   true,
			serverName: smtpTestServerName,
			wantErr:    errors.New("unknown"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newSMTPTestServer(t, test.options)
			smtpMailer := newSMTPTestMailer(
				t,
				server,
				test.mode,
				test.withAuth,
			)
			if test.serverName != smtpTestServerName {
				var err error
				smtpMailer, err = mailer.NewSMTPMailer(
					mailer.SMTPConfig{
						Address: server.address(),
						Host:    smtpTestServerName,
						From:    "sender@example.test",
						Mode:    test.mode,
						TLSConfig: server.clientTLSConfig(
							test.serverName,
						),
					},
					&net.Dialer{},
				)
				if err != nil {
					t.Fatalf("create mismatched SMTP mailer: %v", err)
				}
			}

			err := smtpMailer.Send(smtpTestContext(t), smtpCredentialMessage())
			switch {
			case test.wantRejected && !errors.Is(err, mailer.ErrMessageRejected):
				t.Fatalf("Send() error = %v, want explicit rejection", err)
			case test.wantRejected:
			case test.wantErr == nil && err != nil:
				t.Fatalf("Send() error = %v, want nil", err)
			case test.wantErr != nil && errors.Is(test.wantErr, mailer.ErrTLSRequired) &&
				!errors.Is(err, mailer.ErrTLSRequired):
				t.Fatalf("Send() error = %v, want %v", err, mailer.ErrTLSRequired)
			case test.wantErr != nil && err == nil:
				t.Fatal("Send() error = nil, want unknown failure")
			case test.wantErr != nil && errors.Is(err, mailer.ErrMessageRejected):
				t.Fatalf("Send() error = %v, want unknown rather than rejection", err)
			}

			snapshot := server.wait(t)
			if snapshot.messages != test.wantMessages {
				t.Fatalf(
					"accepted messages = %d, want %d; commands=%v",
					snapshot.messages,
					test.wantMessages,
					snapshot.commands,
				)
			}
			if snapshot.authBeforeTLS || snapshot.envelopeBeforeTLS {
				t.Fatalf("SMTP command crossed plaintext boundary: %+v", snapshot)
			}
			if len(snapshot.protocolViolations) != 0 {
				t.Fatalf("SMTP protocol violations = %v", snapshot.protocolViolations)
			}
			if test.wantCommands != nil &&
				!reflect.DeepEqual(snapshot.commands, test.wantCommands) {
				t.Fatalf("SMTP commands = %v, want %v",
					snapshot.commands, test.wantCommands)
			}
			if snapshot.closeStage != test.wantClose {
				t.Fatalf("SMTP close stage = %q, want %q",
					snapshot.closeStage, test.wantClose)
			}
			if test.wantMessages == 1 &&
				(len(snapshot.tlsVersions) == 0 ||
					snapshot.tlsVersions[len(snapshot.tlsVersions)-1] < tls.VersionTLS12) {
				t.Fatalf("negotiated TLS versions = %v", snapshot.tlsVersions)
			}
			if test.options.rejectStage == smtpStageGreeting {
				if !containsSMTPCommand(snapshot.commands, "EHLO") ||
					!containsSMTPCommand(snapshot.commands, "HELO") ||
					containsSMTPCommand(snapshot.commands, "MAIL") {
					t.Fatalf("greeting negotiation commands = %v", snapshot.commands)
				}
			}
		})
	}
}

func TestCredentialSMTPFixtureRejectsOutOfOrderCommands(t *testing.T) {
	tests := []struct {
		name       string
		options    smtpTestServerOptions
		implicit   bool
		prepare    func(*testing.T, *smtpTestServer, *smtpTestProtocolClient)
		command    string
		wantBefore []string
	}{
		{
			name: "STARTTLS before greeting",
			options: smtpTestServerOptions{
				mode:              mailer.TLSModeSTARTTLSRequired,
				advertiseSTARTTLS: true,
			},
			command: "STARTTLS",
		},
		{
			name: "MAIL before implicit TLS greeting",
			options: smtpTestServerOptions{
				mode: mailer.TLSModeImplicit,
			},
			implicit: true,
			command:  "MAIL FROM:<sender@example.test>",
		},
		{
			name: "MAIL before post TLS greeting",
			options: smtpTestServerOptions{
				mode:              mailer.TLSModeSTARTTLSRequired,
				advertiseSTARTTLS: true,
			},
			prepare: func(t *testing.T, server *smtpTestServer, client *smtpTestProtocolClient) {
				if code := client.command(t, "EHLO client.integration.test"); code != 250 {
					t.Fatalf("EHLO code = %d, want 250", code)
				}
				if code := client.command(t, "STARTTLS"); code != 220 {
					t.Fatalf("STARTTLS code = %d, want 220", code)
				}
				client.upgradeTLS(t, server)
			},
			command:    "MAIL FROM:<sender@example.test>",
			wantBefore: []string{"EHLO", "STARTTLS"},
		},
		{
			name: "AUTH before greeting",
			options: smtpTestServerOptions{
				mode:          mailer.TLSModeImplicit,
				advertiseAuth: true,
			},
			implicit: true,
			command:  "AUTH PLAIN AHVzZXIAcGFzcw==",
		},
		{
			name: "MAIL before required AUTH",
			options: smtpTestServerOptions{
				mode:          mailer.TLSModeImplicit,
				advertiseAuth: true,
			},
			implicit: true,
			prepare: func(t *testing.T, _ *smtpTestServer, client *smtpTestProtocolClient) {
				if code := client.command(t, "EHLO client.integration.test"); code != 250 {
					t.Fatalf("EHLO code = %d, want 250", code)
				}
			},
			command:    "MAIL FROM:<sender@example.test>",
			wantBefore: []string{"EHLO"},
		},
		{
			name: "RCPT before MAIL",
			options: smtpTestServerOptions{
				mode: mailer.TLSModeImplicit,
			},
			implicit: true,
			prepare: func(t *testing.T, _ *smtpTestServer, client *smtpTestProtocolClient) {
				if code := client.command(t, "EHLO client.integration.test"); code != 250 {
					t.Fatalf("EHLO code = %d, want 250", code)
				}
			},
			command:    "RCPT TO:<recipient@example.test>",
			wantBefore: []string{"EHLO"},
		},
		{
			name: "DATA before RCPT",
			options: smtpTestServerOptions{
				mode: mailer.TLSModeImplicit,
			},
			implicit: true,
			prepare: func(t *testing.T, _ *smtpTestServer, client *smtpTestProtocolClient) {
				if code := client.command(t, "EHLO client.integration.test"); code != 250 {
					t.Fatalf("EHLO code = %d, want 250", code)
				}
				if code := client.command(t, "MAIL FROM:<sender@example.test>"); code != 250 {
					t.Fatalf("MAIL code = %d, want 250", code)
				}
			},
			command:    "DATA",
			wantBefore: []string{"EHLO", "MAIL"},
		},
		{
			name: "greeting after MAIL",
			options: smtpTestServerOptions{
				mode: mailer.TLSModeImplicit,
			},
			implicit: true,
			prepare: func(t *testing.T, _ *smtpTestServer, client *smtpTestProtocolClient) {
				if code := client.command(t, "EHLO client.integration.test"); code != 250 {
					t.Fatalf("EHLO code = %d, want 250", code)
				}
				if code := client.command(t, "MAIL FROM:<sender@example.test>"); code != 250 {
					t.Fatalf("MAIL code = %d, want 250", code)
				}
			},
			command:    "EHLO client.integration.test",
			wantBefore: []string{"EHLO", "MAIL"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newSMTPTestServer(t, test.options)
			client := newSMTPTestProtocolClient(t, server, test.implicit)
			if test.prepare != nil {
				test.prepare(t, server, client)
			}
			if code := client.command(t, test.command); code != 503 {
				t.Fatalf("out-of-order command code = %d, want 503", code)
			}
			if code := client.command(t, "QUIT"); code != 221 {
				t.Fatalf("QUIT code = %d, want 221", code)
			}
			snapshot := server.wait(t)
			wantCommands := append(append([]string(nil), test.wantBefore...),
				strings.ToUpper(strings.Fields(test.command)[0]), "QUIT")
			if !reflect.DeepEqual(snapshot.commands, wantCommands) {
				t.Fatalf("SMTP commands = %v, want %v",
					snapshot.commands, wantCommands)
			}
			if len(snapshot.protocolViolations) != 1 {
				t.Fatalf("SMTP protocol violations = %v, want exactly one",
					snapshot.protocolViolations)
			}
			if snapshot.messages != 0 {
				t.Fatalf("accepted messages = %d, want 0", snapshot.messages)
			}
		})
	}
}

func TestCredentialSMTPFixtureClassifiesExpectedServerClose(t *testing.T) {
	synthetic := errors.New("synthetic structured server failure")
	expected := &smtpTestServer{
		options: smtpTestServerOptions{expectedCloseStage: smtpStageTLS},
	}
	expected.setServeError(smtpStageTLS, synthetic)
	if expected.serveErr != nil || expected.snapshot.closeStage != smtpStageTLS {
		t.Fatalf("expected TLS close classification = err:%v snapshot:%+v",
			expected.serveErr, expected.snapshot)
	}

	unexpected := &smtpTestServer{}
	unexpected.setServeError(smtpStageGreeting, synthetic)
	if !errors.Is(unexpected.serveErr, synthetic) {
		t.Fatalf("unexpected server error = %v, want retained structured cause",
			unexpected.serveErr)
	}
}

func TestCredentialSMTPFixtureCleanupClosesAbnormalClients(t *testing.T) {
	t.Run("stalled TLS handshake", func(t *testing.T) {
		server := newSMTPTestServer(t, smtpTestServerOptions{
			mode: mailer.TLSModeImplicit,
		})
		connection, err := net.DialTimeout("tcp", server.address(), smtpTestIOTimeout)
		if err != nil {
			t.Fatalf("dial stalled SMTP client: %v", err)
		}
		defer connection.Close()
		server.waitForConnection(t)
		server.close()
		if server.activeConnectionCount() != 0 {
			t.Fatalf("active connections = %d, want 0",
				server.activeConnectionCount())
		}
	})

	t.Run("partial SMTP command", func(t *testing.T) {
		server := newSMTPTestServer(t, smtpTestServerOptions{
			mode: mailer.TLSModeImplicit,
		})
		client := newSMTPTestProtocolClient(t, server, true)
		if _, err := fmt.Fprint(client.connection, "MA"); err != nil {
			t.Fatalf("write partial SMTP command: %v", err)
		}
		server.close()
		if server.activeConnectionCount() != 0 {
			t.Fatalf("active connections = %d, want 0",
				server.activeConnectionCount())
		}
	})
}

func TestCredentialSMTPPasswordResetCoordinator(t *testing.T) {
	tests := []struct {
		name               string
		resend             bool
		options            smtpTestServerOptions
		wantCode           platform.Code
		wantDelivery       identitypkg.PasswordResetDeliveryState
		wantSettlement     identitypkg.PasswordResetSettlementState
		wantOperation      identitypkg.PasswordResetOperationState
		wantCredential     identitypkg.TemporaryCredentialState
		wantRequestKind    identitypkg.PasswordResetRequestKind
		wantAuthGeneration int64
		wantAccepted       int
		wantRejected       int
		wantUnknown        int
	}{
		{
			name: "first reset accepted enters existing settlement path",
			options: smtpTestServerOptions{
				mode: mailer.TLSModeImplicit,
			},
			wantDelivery:       identitypkg.PasswordResetDeliveryAccepted,
			wantSettlement:     identitypkg.PasswordResetSettlementPending,
			wantOperation:      identitypkg.PasswordResetOperationRecoveryRequired,
			wantCredential:     identitypkg.TemporaryCredentialPendingDelivery,
			wantRequestKind:    identitypkg.PasswordResetInitialReset,
			wantAuthGeneration: 3,
			wantAccepted:       1,
		},
		{
			name: "first reset explicit rejection records rejected evidence",
			options: smtpTestServerOptions{
				mode:        mailer.TLSModeImplicit,
				rejectStage: smtpStageRCPT,
				rejectCode:  550,
			},
			wantCode:           platform.CodeIdentityEmailDeliveryFailed,
			wantDelivery:       identitypkg.PasswordResetDeliveryRejected,
			wantSettlement:     identitypkg.PasswordResetSettlementInvalidated,
			wantOperation:      identitypkg.PasswordResetOperationFailed,
			wantCredential:     identitypkg.TemporaryCredentialRecoveryInvalidated,
			wantRequestKind:    identitypkg.PasswordResetInitialReset,
			wantAuthGeneration: 3,
			wantRejected:       1,
		},
		{
			name: "first reset unknown remains unresolved",
			options: smtpTestServerOptions{
				mode:            mailer.TLSModeImplicit,
				disconnectStage: smtpStageResult,
			},
			wantCode:           platform.CodeIdentityOperationUncertain,
			wantDelivery:       identitypkg.PasswordResetDeliveryUnknown,
			wantSettlement:     identitypkg.PasswordResetSettlementPending,
			wantOperation:      identitypkg.PasswordResetOperationCommitUnknown,
			wantCredential:     identitypkg.TemporaryCredentialPendingDelivery,
			wantRequestKind:    identitypkg.PasswordResetInitialReset,
			wantAuthGeneration: 3,
			wantUnknown:        1,
		},
		{
			name:   "ordinary resend accepted preserves prior authority until settlement",
			resend: true,
			options: smtpTestServerOptions{
				mode: mailer.TLSModeImplicit,
			},
			wantDelivery:       identitypkg.PasswordResetDeliveryAccepted,
			wantSettlement:     identitypkg.PasswordResetSettlementPending,
			wantOperation:      identitypkg.PasswordResetOperationRecoveryRequired,
			wantCredential:     identitypkg.TemporaryCredentialPendingDelivery,
			wantRequestKind:    identitypkg.PasswordResetResend,
			wantAuthGeneration: 4,
			wantAccepted:       1,
		},
		{
			name:   "ordinary resend rejection invalidates only the new credential",
			resend: true,
			options: smtpTestServerOptions{
				mode:        mailer.TLSModeImplicit,
				rejectStage: smtpStageRCPT,
				rejectCode:  550,
			},
			wantCode:           platform.CodeIdentityEmailDeliveryFailed,
			wantDelivery:       identitypkg.PasswordResetDeliveryRejected,
			wantSettlement:     identitypkg.PasswordResetSettlementInvalidated,
			wantOperation:      identitypkg.PasswordResetOperationFailed,
			wantCredential:     identitypkg.TemporaryCredentialRecoveryInvalidated,
			wantRequestKind:    identitypkg.PasswordResetResend,
			wantAuthGeneration: 4,
			wantRejected:       1,
		},
		{
			name:   "ordinary resend unknown preserves pending candidate and prior authority",
			resend: true,
			options: smtpTestServerOptions{
				mode:            mailer.TLSModeImplicit,
				disconnectStage: smtpStageResult,
			},
			wantCode:           platform.CodeIdentityOperationUncertain,
			wantDelivery:       identitypkg.PasswordResetDeliveryUnknown,
			wantSettlement:     identitypkg.PasswordResetSettlementPending,
			wantOperation:      identitypkg.PasswordResetOperationCommitUnknown,
			wantCredential:     identitypkg.TemporaryCredentialPendingDelivery,
			wantRequestKind:    identitypkg.PasswordResetResend,
			wantAuthGeneration: 4,
			wantUnknown:        1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newSMTPPasswordResetHarness(t, test.options, test.resend)
			result, err := harness.execute(smtpTestContext(t), test.resend)
			if test.wantCode == "" && err != nil {
				t.Fatalf("password reset error = %v, want nil", err)
			}
			if test.wantCode != "" {
				if got := platform.PublicCode(err); got != test.wantCode {
					t.Fatalf("password reset code = %q, want %q; err=%v", got, test.wantCode, err)
				}
			}
			if test.wantCode == "" && result.DeliveryStatus != "accepted" {
				t.Fatalf("password reset result = %#v", result)
			}
			state := harness.state.snapshot()
			if state.delivery != test.wantDelivery ||
				state.settlement != test.wantSettlement ||
				state.operation != test.wantOperation ||
				state.credential != test.wantCredential ||
				state.requestKind != test.wantRequestKind ||
				state.authGeneration != test.wantAuthGeneration ||
				state.acceptedEvidence != test.wantAccepted ||
				state.rejectedEvidence != test.wantRejected ||
				state.unknownEvidence != test.wantUnknown ||
				!state.oldCredentialUsable ||
				state.sessionRevocations != 0 {
				t.Fatalf("password reset durable evidence = %+v", state)
			}
			if state.credentialID == "" {
				t.Fatal("password reset candidate credential ID is empty")
			}
			if test.wantRejected == 1 {
				if state.credentialInvalidations != 1 ||
					state.invalidatedCredentialID != state.credentialID {
					t.Fatalf("rejected candidate invalidation = %+v", state)
				}
			} else if state.credentialInvalidations != 0 ||
				state.invalidatedCredentialID != "" {
				t.Fatalf("non-rejected candidate was invalidated: %+v", state)
			}
			if test.resend {
				if state.oldCredentialID == "" ||
					state.oldCredentialID == state.credentialID ||
					state.oldCredentialState != identitypkg.TemporaryCredentialActive {
					t.Fatalf("resend changed prior credential authority: %+v", state)
				}
			}
			if state.dispatchClaims != 1 || state.mailAttempts != 1 {
				t.Fatalf("dispatch owner/mail attempts = %d/%d, want 1/1",
					state.dispatchClaims, state.mailAttempts)
			}
			serverSnapshot := harness.server.wait(t)
			wantMessages := 0
			if test.wantAccepted == 1 {
				wantMessages = 1
			}
			if serverSnapshot.messages != wantMessages {
				t.Fatalf("SMTP accepted messages = %d, want %d",
					serverSnapshot.messages, wantMessages)
			}

			if !test.resend && test.wantAccepted == 1 {
				_, replayErr := harness.execute(smtpTestContext(t), false)
				if got := platform.PublicCode(replayErr); got !=
					platform.CodeIdentityOperationUncertain {
					t.Fatalf("unsettled replay code = %q, want operation uncertain; err=%v",
						got, replayErr)
				}
				replayed := harness.state.snapshot()
				if replayed.dispatchClaims != 1 ||
					replayed.mailAttempts != 1 ||
					replayed.credential != test.wantCredential ||
					replayed.authGeneration != test.wantAuthGeneration ||
					!replayed.oldCredentialUsable ||
					replayed.sessionRevocations != 0 ||
					replayed.credentialInvalidations != state.credentialInvalidations ||
					replayed.invalidatedCredentialID != state.invalidatedCredentialID {
					t.Fatalf("replay changed credential authority or redispatched SMTP: %+v",
						replayed)
				}
			}
		})
	}
}

func TestSMTPResidualWindowNewAdministrator(t *testing.T) {
	server := newSMTPTestServer(t, smtpTestServerOptions{
		mode: mailer.TLSModeImplicit,
	})
	smtpMailer := newSMTPTestMailer(
		t,
		server,
		mailer.TLSModeImplicit,
		false,
	)
	mainTx := &smtpCoordinatorTx{commitErr: errors.New("synthetic commit acknowledgement loss")}
	reconcileTx := &smtpCoordinatorTx{}
	transactions := &smtpSequenceTransactor{
		transactions: []pgx.Tx{mainTx, reconcileTx, &smtpCoordinatorTx{}},
	}
	repository := &smtpAdministratorRepository{}
	idempotencyStore := &smtpAdministratorIdempotency{}
	now := time.Date(2026, time.October, 25, 9, 0, 0, 0, time.UTC)
	service, err := identitypkg.NewAdministratorCreateService(
		identitypkg.AdministratorCreateDependencies{
			Transactions: transactions,
			Repository:   repository,
			InformationNotices: smtpAdministratorNotice{
				notice: identitypkg.AdministratorInformationNotice{
					Version:     "administrator-notice-v1",
					EffectiveAt: now.Add(-time.Hour),
					Content:     "controlled integration notice",
					ContentHash: "sha256:controlled-integration-notice",
				},
			},
			Gate:           smtpOpenGate{},
			PasswordPolicy: security.PasswordPolicy{},
			PasswordHasher: smtpCredentialHasher{},
			Mailer:         smtpMailer,
			Idempotency:    idempotencyStore,
			IdempotencyKeys: idempotency.DigestKeyring{
				ActiveVersion: "v1",
				Keys: map[string][]byte{
					"v1": []byte("smtp-residual-window-integration-key"),
				},
			},
			IdempotencyLock:     time.Minute,
			IdempotencyTTL:      24 * time.Hour,
			SMTPOperationBudget: 3 * time.Second,
			Audit:               smtpAuditAppender{},
			IDs:                 &smtpIDGenerator{},
			Clock:               smtpFixedClock{now: now},
		},
	)
	if err != nil {
		t.Fatalf("NewAdministratorCreateService() error = %v", err)
	}
	request := identitypkg.CreateAdministratorRequest{
		Principal: session.Principal{
			SessionID:      "initial-session",
			UserID:         "initial-admin",
			Role:           string(identitypkg.RoleAdmin),
			IsInitialAdmin: true,
			Status:         string(identitypkg.UserStatusActive),
		},
		IdempotencyKey:                    "0123456789abcdef",
		Username:                          "ops",
		Email:                             "ops@example.test",
		Phone:                             "+10000000000",
		TemporaryPassword:                 "Temporary123",
		InformationProvisionNoticeVersion: "administrator-notice-v1",
		InformationProvisionDeclared:      true,
		TraceID:                           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}

	_, err = service.CreateAdministrator(smtpTestContext(t), request)
	if got := platform.PublicCode(err); got != platform.CodeIdentityOperationUncertain {
		t.Fatalf("CreateAdministrator() code = %q, want operation uncertain; err=%v", got, err)
	}
	snapshot := server.wait(t)
	if snapshot.messages != 1 {
		t.Fatalf("accepted SMTP messages = %d, want 1", snapshot.messages)
	}
	if mainTx.committed || repository.credentialUsable(mainTx) {
		t.Fatal("accepted credential became usable despite failed database commit")
	}
	if !idempotencyStore.uncertain {
		t.Fatal("accepted/commit-failed operation was not marked uncertain")
	}

	_, retryErr := service.CreateAdministrator(smtpTestContext(t), request)
	if got := platform.PublicCode(retryErr); got != platform.CodeIdentityOperationUncertain {
		t.Fatalf("retry code = %q, want operation uncertain; err=%v", got, retryErr)
	}
	if transactions.beginCount() != 3 {
		t.Fatalf("transaction begins = %d, want main/reconcile/retry only", transactions.beginCount())
	}
	if idempotencyStore.executeCalls != 1 {
		t.Fatalf("idempotency executions = %d, want 1", idempotencyStore.executeCalls)
	}
}

func smtpCredentialMessage() mailer.Message {
	return mailer.Message{
		Purpose: mailer.PurposePasswordResetCredential,
		To:      "recipient@example.test",
		Subject: "Temporary credential",
		Body:    "controlled integration body",
	}
}

type smtpCoordinatorTx struct {
	pgx.Tx
	commitErr  error
	committed  bool
	rolledBack bool
}

func (tx *smtpCoordinatorTx) Commit(context.Context) error {
	if tx.commitErr != nil {
		return tx.commitErr
	}
	tx.committed = true
	return nil
}

func (tx *smtpCoordinatorTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

type smtpSequenceTransactor struct {
	mu           sync.Mutex
	transactions []pgx.Tx
	next         int
}

func (t *smtpSequenceTransactor) Begin(context.Context) (pgx.Tx, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.next >= len(t.transactions) {
		return nil, errors.New("unexpected SMTP integration transaction")
	}
	tx := t.transactions[t.next]
	t.next++
	return tx, nil
}

func (t *smtpSequenceTransactor) beginCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.next
}

type smtpAdministratorRepository struct {
	credential identitypkg.TemporaryCredential
	activated  bool
}

func (r *smtpAdministratorRepository) FindAdministratorCreationActor(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) (identitypkg.User, error) {
	return identitypkg.User{
		ID:             userID,
		Role:           identitypkg.RoleAdmin,
		IsInitialAdmin: true,
		Status:         identitypkg.UserStatusActive,
		CredentialMode: identitypkg.CredentialModePermanent,
	}, nil
}

func (*smtpAdministratorRepository) InsertUser(
	context.Context,
	pgx.Tx,
	identitypkg.User,
) error {
	return nil
}

func (*smtpAdministratorRepository) InsertIdentifier(
	context.Context,
	pgx.Tx,
	identitypkg.IdentityIdentifier,
) error {
	return nil
}

func (r *smtpAdministratorRepository) InsertTemporaryCredential(
	_ context.Context,
	_ pgx.Tx,
	credential identitypkg.TemporaryCredential,
) error {
	r.credential = credential
	return nil
}

func (*smtpAdministratorRepository) InsertAdministratorInformationDeclaration(
	context.Context,
	pgx.Tx,
	identitypkg.AdministratorInformationDeclaration,
) error {
	return nil
}

func (r *smtpAdministratorRepository) ActivateTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
) error {
	r.activated = true
	return nil
}

func (*smtpAdministratorRepository) FindCreatedAdministrator(
	context.Context,
	pgx.Tx,
	string,
) (identitypkg.CreatedAdministrator, error) {
	return identitypkg.CreatedAdministrator{}, errors.New("unexpected replay read")
}

func (r *smtpAdministratorRepository) credentialUsable(tx *smtpCoordinatorTx) bool {
	return r.activated && tx.committed
}

type smtpAdministratorNotice struct {
	notice identitypkg.AdministratorInformationNotice
}

func (n smtpAdministratorNotice) CurrentAdministratorInformationNotice(
	context.Context,
	pgx.Tx,
) (identitypkg.AdministratorInformationNotice, error) {
	return n.notice, nil
}

type smtpOpenGate struct{}

func (smtpOpenGate) RequireOpen(context.Context) error { return nil }

type smtpCredentialHasher struct{}

func (smtpCredentialHasher) Hash(string) (string, error) {
	return "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0", nil
}

func (smtpCredentialHasher) Verify(string, string) (bool, bool, error) {
	return true, false, nil
}

type smtpAdministratorIdempotency struct {
	mu           sync.Mutex
	uncertain    bool
	executeCalls int
}

func (s *smtpAdministratorIdempotency) ResolveExisting(
	context.Context,
	idempotency.Transaction,
	idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.uncertain {
		return idempotency.BeginResult{}, false, nil
	}
	return idempotency.BeginResult{
		Action: idempotency.BeginOperationUncertain,
		Record: smtpIdempotencyRecord(),
	}, true, nil
}

func (s *smtpAdministratorIdempotency) Begin(
	context.Context,
	idempotency.Transaction,
	idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uncertain {
		return idempotency.BeginResult{
			Action: idempotency.BeginOperationUncertain,
			Record: smtpIdempotencyRecord(),
		}, nil
	}
	if s.executeCalls == 0 {
		s.executeCalls++
		return idempotency.BeginResult{
			Action: idempotency.BeginExecute,
			Record: smtpIdempotencyRecord(),
		}, nil
	}
	return idempotency.BeginResult{
		Action: idempotency.BeginInProgress,
		Record: smtpIdempotencyRecord(),
	}, nil
}

func (*smtpAdministratorIdempotency) Complete(
	context.Context,
	idempotency.Transaction,
	idempotency.Completion,
) error {
	return nil
}

func (s *smtpAdministratorIdempotency) MarkUncertain(
	context.Context,
	idempotency.Transaction,
	idempotency.Transition,
) error {
	s.mu.Lock()
	s.uncertain = true
	s.mu.Unlock()
	return nil
}

func smtpIdempotencyRecord() idempotency.Record {
	now := time.Date(2026, time.October, 25, 9, 0, 0, 0, time.UTC)
	lockedUntil := now.Add(time.Minute)
	return idempotency.Record{
		ScopeHash:           []byte("scope"),
		KeyLocator:          []byte("locator"),
		KeyDigest:           []byte("key"),
		RequestDigest:       []byte("request"),
		DigestKeyVersion:    "v1",
		Status:              idempotency.StatusProcessing,
		ExecutionGeneration: 1,
		LockedUntil:         &lockedUntil,
		ExpiresAt:           now.Add(24 * time.Hour),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

type smtpAuditAppender struct{}

func (smtpAuditAppender) Append(context.Context, pgx.Tx, audit.Event) error {
	return nil
}

type smtpIDGenerator struct {
	mu   sync.Mutex
	next int
}

func (g *smtpIDGenerator) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return smtpTestID("smtp-integration", g.next), nil
}

type smtpFixedClock struct {
	now time.Time
}

func (c smtpFixedClock) Now() time.Time { return c.now }

const (
	smtpPasswordResetActorID  = "11111111-1111-4111-8111-111111111111"
	smtpPasswordResetTargetID = "22222222-2222-4222-8222-222222222222"
)

type smtpPasswordResetHarness struct {
	server  *smtpTestServer
	service *identitypkg.PasswordResetCoordinator
	state   *smtpPasswordResetState
}

func newSMTPPasswordResetHarness(
	t *testing.T,
	options smtpTestServerOptions,
	resend bool,
) *smtpPasswordResetHarness {
	t.Helper()
	server := newSMTPTestServer(t, options)
	state := newSMTPPasswordResetState(resend)
	repository := &smtpPasswordResetRepository{
		actor: identitypkg.User{
			ID:             smtpPasswordResetActorID,
			Role:           identitypkg.RoleAdmin,
			IsInitialAdmin: true,
			Status:         identitypkg.UserStatusActive,
			CredentialMode: identitypkg.CredentialModePermanent,
		},
		target: identitypkg.User{
			ID:             smtpPasswordResetTargetID,
			Role:           identitypkg.RoleMember,
			Status:         identitypkg.UserStatusActive,
			CredentialMode: identitypkg.CredentialModePermanent,
			Email:          smtpPointer("recipient@example.test"),
			AuthGeneration: state.authGeneration,
		},
	}
	if resend {
		repository.latest = state.oldCredential
		repository.latestFound = true
		repository.target.CredentialMode = identitypkg.CredentialModeTemporary
	}
	now := time.Date(2026, time.October, 25, 10, 0, 0, 0, time.UTC)
	service, err := identitypkg.NewPasswordResetService(
		identitypkg.PasswordResetDependencies{
			Transactions:   &smtpPasswordResetTransactor{state: state},
			Repository:     repository,
			Store:          &identitypkg.PasswordResetStore{},
			PasswordPolicy: smtpPasswordPolicy{},
			PasswordHasher: smtpCredentialHasher{},
			Secrets:        smtpPasswordSecret{},
			Mailer: newSMTPTestMailer(
				t,
				server,
				mailer.TLSModeImplicit,
				false,
			),
			IdempotencyKeys: idempotency.DigestKeyring{
				ActiveVersion: "v1",
				Keys: map[string][]byte{
					"v1": []byte("smtp-password-reset-integration-key"),
				},
			},
			IdempotencyLock:     5 * time.Second,
			IdempotencyTTL:      time.Hour,
			SMTPOperationBudget: 2 * time.Second,
			Audit:               smtpAuditAppender{},
			IDs:                 &smtpIDGenerator{},
			Clock:               smtpFixedClock{now: now},
		},
	)
	if err != nil {
		t.Fatalf("NewPasswordResetService() error = %v", err)
	}
	return &smtpPasswordResetHarness{
		server:  server,
		service: service,
		state:   state,
	}
}

func (h *smtpPasswordResetHarness) execute(
	ctx context.Context,
	resend bool,
) (identitypkg.PasswordResetResult, error) {
	request := identitypkg.PasswordResetRequest{
		Actor: session.Principal{
			SessionID:      "initial-session",
			UserID:         smtpPasswordResetActorID,
			Role:           string(identitypkg.RoleAdmin),
			IsInitialAdmin: true,
			Status:         string(identitypkg.UserStatusActive),
		},
		TargetID:       smtpPasswordResetTargetID,
		TargetRole:     identitypkg.RoleMember,
		IdempotencyKey: "password-reset-integration-key",
		TraceID:        "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}
	if resend {
		request.IdempotencyKey = "password-reset-resend-integration-key"
		return h.service.ResendPasswordResetCredential(ctx, request)
	}
	return h.service.ResetPassword(ctx, request)
}

type smtpPasswordResetSnapshot struct {
	operation               identitypkg.PasswordResetOperationState
	delivery                identitypkg.PasswordResetDeliveryState
	settlement              identitypkg.PasswordResetSettlementState
	requestKind             identitypkg.PasswordResetRequestKind
	credentialID            string
	credential              identitypkg.TemporaryCredentialState
	authGeneration          int64
	dispatchClaims          int
	mailAttempts            int
	acceptedEvidence        int
	rejectedEvidence        int
	unknownEvidence         int
	credentialInvalidations int
	invalidatedCredentialID string
	oldCredentialUsable     bool
	oldCredentialID         string
	oldCredentialState      identitypkg.TemporaryCredentialState
	sessionRevocations      int
}

type smtpPasswordResetState struct {
	mu sync.Mutex

	userID                  string
	authGeneration          int64
	executionGeneration     int64
	currentOperationID      string
	operationID             string
	credentialID            string
	requestKind             identitypkg.PasswordResetRequestKind
	operation               identitypkg.PasswordResetOperationState
	delivery                identitypkg.PasswordResetDeliveryState
	settlement              identitypkg.PasswordResetSettlementState
	credential              identitypkg.TemporaryCredentialState
	targetAuth              int64
	storedClaim             *idempotency.Record
	dispatchClaims          int
	mailAttempts            int
	acceptedEvidence        int
	rejectedEvidence        int
	unknownEvidence         int
	credentialInvalidations int
	invalidatedCredentialID string
	oldCredentialUsable     bool
	sessionRevocations      int
	oldCredential           identitypkg.TemporaryCredential
}

func newSMTPPasswordResetState(resend bool) *smtpPasswordResetState {
	state := &smtpPasswordResetState{
		userID:              smtpPasswordResetTargetID,
		authGeneration:      3,
		executionGeneration: 0,
		oldCredentialUsable: true,
	}
	if resend {
		now := time.Date(2026, time.October, 25, 9, 0, 0, 0, time.UTC)
		state.authGeneration = 4
		state.executionGeneration = 1
		state.oldCredential = identitypkg.TemporaryCredential{
			ID:                "password-reset-old-credential",
			UserID:            smtpPasswordResetTargetID,
			Purpose:           identitypkg.TemporaryCredentialPasswordReset,
			Generation:        1,
			AuthGeneration:    4,
			SourceOperationID: smtpPointer("password-reset-old-operation"),
			State:             identitypkg.TemporaryCredentialActive,
			SecretDigest:      "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
			DeliveredToEmail:  "recipient@example.test",
			IssuedAt:          now,
			ExpiresAt:         now.Add(7 * 24 * time.Hour),
		}
	}
	return state
}

func (s *smtpPasswordResetState) snapshot() smtpPasswordResetSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return smtpPasswordResetSnapshot{
		operation:               s.operation,
		delivery:                s.delivery,
		settlement:              s.settlement,
		requestKind:             s.requestKind,
		credentialID:            s.credentialID,
		credential:              s.credential,
		authGeneration:          s.authGeneration,
		dispatchClaims:          s.dispatchClaims,
		mailAttempts:            s.mailAttempts,
		acceptedEvidence:        s.acceptedEvidence,
		rejectedEvidence:        s.rejectedEvidence,
		unknownEvidence:         s.unknownEvidence,
		credentialInvalidations: s.credentialInvalidations,
		invalidatedCredentialID: s.invalidatedCredentialID,
		oldCredentialUsable:     s.oldCredentialUsable,
		oldCredentialID:         s.oldCredential.ID,
		oldCredentialState:      s.oldCredential.State,
		sessionRevocations:      s.sessionRevocations,
	}
}

type smtpPasswordResetTransactor struct {
	state *smtpPasswordResetState
}

func (t *smtpPasswordResetTransactor) Begin(context.Context) (pgx.Tx, error) {
	return &smtpPasswordResetTx{state: t.state}, nil
}

type smtpPasswordResetTx struct {
	pgx.Tx
	state *smtpPasswordResetState
}

func (*smtpPasswordResetTx) Commit(context.Context) error   { return nil }
func (*smtpPasswordResetTx) Rollback(context.Context) error { return nil }

func (tx *smtpPasswordResetTx) QueryRow(
	_ context.Context,
	statement string,
	_ ...any,
) pgx.Row {
	sql := compactSMTPTestSQL(statement)
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()

	switch {
	case strings.Contains(sql, "LEFT JOIN password_reset_operations o"):
		return smtpTestRow{values: tx.state.bindingValues()}
	case strings.Contains(sql, "FROM idempotency_records") &&
		strings.Contains(sql, "WHERE scope_hash = $1"):
		if tx.state.storedClaim == nil {
			return smtpTestRow{err: pgx.ErrNoRows}
		}
		return smtpTestRow{values: tx.state.idempotencyValues()}
	case strings.Contains(sql, "SELECT id, auth_generation, password_reset_execution_generation"):
		return smtpTestRow{values: []any{
			tx.state.userID,
			tx.state.authGeneration,
			tx.state.executionGeneration,
			false,
		}}
	case strings.Contains(sql, "SELECT operation_id, user_id FROM password_reset_operations"):
		return smtpTestRow{err: pgx.ErrNoRows}
	case strings.Contains(sql, "NOT current_account.auth_integrity_hold"):
		return smtpTestRow{values: []any{true}}
	case strings.Contains(sql, "state NOT IN ('succeeded', 'invalidated', 'failed')"):
		return smtpTestRow{values: []any{false}}
	case strings.Contains(sql, "WITH RECURSIVE current_credential AS"):
		return smtpTestRow{values: tx.state.lineageValues()}
	case sql == "SELECT id FROM users WHERE id=$1 FOR UPDATE":
		return smtpTestRow{values: []any{tx.state.userID}}
	default:
		return smtpTestRow{err: fmt.Errorf("unexpected password-reset query: %s", sql)}
	}
}

func (tx *smtpPasswordResetTx) Exec(
	_ context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	sql := compactSMTPTestSQL(statement)
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()

	switch {
	case strings.HasPrefix(sql, "UPDATE users SET password_reset_execution_generation"):
		tx.state.executionGeneration = arguments[2].(int64)
		tx.state.currentOperationID = arguments[3].(string)
	case strings.HasPrefix(sql, "INSERT INTO password_reset_operations"):
		tx.state.operationID = arguments[0].(string)
		tx.state.credentialID = arguments[2].(string)
		tx.state.requestKind = arguments[9].(identitypkg.PasswordResetRequestKind)
		tx.state.targetAuth = arguments[13].(int64)
		tx.state.settlement = arguments[14].(identitypkg.PasswordResetSettlementState)
		tx.state.delivery = arguments[15].(identitypkg.PasswordResetDeliveryState)
		tx.state.operation = arguments[16].(identitypkg.PasswordResetOperationState)
	case strings.HasPrefix(sql, "INSERT INTO temporary_credentials"):
		tx.state.credential = arguments[6].(identitypkg.TemporaryCredentialState)
	case strings.HasPrefix(sql, "INSERT INTO password_reset_recovery_jobs"):
	case strings.HasPrefix(sql, "INSERT INTO idempotency_records") &&
		strings.Contains(sql, "password_reset_operation_id"):
		lockedUntil := arguments[8].(time.Time)
		record := idempotency.Record{
			ScopeHash:           append([]byte(nil), arguments[0].([]byte)...),
			KeyLocator:          append([]byte(nil), arguments[1].([]byte)...),
			KeyDigest:           append([]byte(nil), arguments[2].([]byte)...),
			RequestDigest:       append([]byte(nil), arguments[3].([]byte)...),
			DigestKeyVersion:    arguments[4].(string),
			Status:              idempotency.StatusProcessing,
			ExecutionGeneration: arguments[5].(int64),
			LockedUntil:         &lockedUntil,
			ExpiresAt:           arguments[9].(time.Time),
			CreatedAt:           arguments[10].(time.Time),
			UpdatedAt:           arguments[10].(time.Time),
		}
		tx.state.storedClaim = &record
	case strings.Contains(sql, "SET state='dispatching',delivery_state='dispatching'"):
		tx.state.operation = identitypkg.PasswordResetOperationDispatching
		tx.state.delivery = identitypkg.PasswordResetDeliveryDispatching
		tx.state.dispatchClaims++
		tx.state.mailAttempts++
	case strings.Contains(sql, "SET state='recovery_required',delivery_state='accepted'"):
		tx.state.operation = identitypkg.PasswordResetOperationRecoveryRequired
		tx.state.delivery = identitypkg.PasswordResetDeliveryAccepted
		tx.state.acceptedEvidence++
	case strings.Contains(sql, "SET state='commit_unknown',delivery_state='unknown'"):
		tx.state.operation = identitypkg.PasswordResetOperationCommitUnknown
		tx.state.delivery = identitypkg.PasswordResetDeliveryUnknown
		tx.state.unknownEvidence++
	case strings.Contains(sql, "SET state='failed',delivery_state='rejected'"):
		tx.state.operation = identitypkg.PasswordResetOperationFailed
		tx.state.delivery = identitypkg.PasswordResetDeliveryRejected
		tx.state.settlement = identitypkg.PasswordResetSettlementInvalidated
		tx.state.rejectedEvidence++
	case strings.Contains(sql, "SET state='recovery_invalidated',invalidated_at="):
		credentialID := arguments[1].(string)
		tx.state.credentialInvalidations++
		tx.state.invalidatedCredentialID = credentialID
		switch credentialID {
		case tx.state.credentialID:
			tx.state.credential = identitypkg.TemporaryCredentialRecoveryInvalidated
		case tx.state.oldCredential.ID:
			tx.state.oldCredential.State = identitypkg.TemporaryCredentialRecoveryInvalidated
			tx.state.oldCredentialUsable = false
		default:
			return pgconn.CommandTag{}, fmt.Errorf(
				"invalidated unknown password-reset credential %q",
				credentialID,
			)
		}
	case strings.Contains(sql, "SET status='succeeded',updated_at="):
	case strings.Contains(sql, "SET status='ready',next_attempt_at="):
	default:
		return pgconn.CommandTag{}, fmt.Errorf("unexpected password-reset exec: %s", sql)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (s *smtpPasswordResetState) idempotencyValues() []any {
	record := *s.storedClaim
	return []any{
		record.ScopeHash,
		record.KeyLocator,
		record.KeyDigest,
		record.RequestDigest,
		record.DigestKeyVersion,
		record.Status,
		record.ExecutionGeneration,
		nil,
		nil,
		record.LockedUntil,
		nil,
		nil,
		record.ExpiresAt,
		record.CreatedAt,
		record.UpdatedAt,
	}
}

func (s *smtpPasswordResetState) bindingValues() []any {
	record := *s.storedClaim
	return []any{
		record.ScopeHash,
		record.KeyLocator,
		record.KeyDigest,
		record.DigestKeyVersion,
		record.RequestDigest,
		record.ExecutionGeneration,
		smtpPointer(s.operationID),
		smtpPointer(s.userID),
		smtpPointer(s.requestKind),
		smtpPointer(s.operation),
		smtpPointer(s.settlement),
		smtpPointer(s.delivery),
		s.targetAuth,
		nil,
		nil,
		nil,
		nil,
		record.ExecutionGeneration,
		true,
		true,
	}
}

func (s *smtpPasswordResetState) lineageValues() []any {
	now := time.Date(2026, time.October, 25, 9, 0, 0, 0, time.UTC)
	oldOperationID := *s.oldCredential.SourceOperationID
	state := identitypkg.PasswordResetOperationSucceeded
	settlement := identitypkg.PasswordResetSettlementTemporaryActive
	delivery := identitypkg.PasswordResetDeliveryAccepted
	requestKind := identitypkg.PasswordResetInitialReset
	purpose := identitypkg.TemporaryCredentialPasswordReset
	credentialState := identitypkg.TemporaryCredentialActive
	baseAuth := int64(3)
	targetAuth := int64(4)
	generation := int64(1)
	attemptID := "old-delivery-attempt"

	values := []any{
		smtpPointer(s.userID),
		identitypkg.CredentialModeTemporary,
		s.authGeneration,
		false,
		nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil,
	}
	values = append(values,
		smtpPointer(oldOperationID),
		smtpPointer(s.userID),
		smtpPointer(s.oldCredential.ID),
		smtpPointer(state),
		smtpPointer(settlement),
		smtpPointer(delivery),
		smtpPointer(targetAuth),
		smtpPointer(targetAuth),
		nil,
		smtpPointer(now),
		smtpPointer(now),
		smtpPointer(attemptID),
		smtpPointer(now),
		smtpPointer(now),
		smtpPointer(s.oldCredential.ID),
		smtpPointer(s.userID),
		smtpPointer(purpose),
		smtpPointer(targetAuth),
		smtpPointer(oldOperationID),
		smtpPointer(credentialState),
	)
	values = append(values,
		smtpPointer(oldOperationID),
		smtpPointer(s.userID),
		smtpPointer(s.oldCredential.ID),
		smtpPointer(requestKind),
		nil,
		nil,
		smtpPointer(state),
		smtpPointer(settlement),
		smtpPointer(delivery),
		smtpPointer(baseAuth),
		smtpPointer(targetAuth),
		smtpPointer(targetAuth),
		nil,
		smtpPointer(now),
		smtpPointer(now),
		smtpPointer(attemptID),
		smtpPointer(now),
		smtpPointer(now),
		smtpPointer(s.oldCredential.ID),
		smtpPointer(s.userID),
		smtpPointer(purpose),
		smtpPointer(generation),
		smtpPointer(generation),
		smtpPointer(targetAuth),
		smtpPointer(oldOperationID),
		smtpPointer(credentialState),
		true,
		true,
		smtpPointer(baseAuth),
	)
	return values
}

type smtpTestRow struct {
	values []any
	err    error
}

func (r smtpTestRow) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(destinations) != len(r.values) {
		return fmt.Errorf("scan destinations = %d, values = %d", len(destinations), len(r.values))
	}
	for index, destination := range destinations {
		target := reflect.ValueOf(destination)
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return fmt.Errorf("scan destination %d is invalid", index)
		}
		target = target.Elem()
		value := r.values[index]
		if value == nil {
			target.Set(reflect.Zero(target.Type()))
			continue
		}
		source := reflect.ValueOf(value)
		if source.Type().AssignableTo(target.Type()) {
			target.Set(source)
			continue
		}
		if source.Type().ConvertibleTo(target.Type()) {
			target.Set(source.Convert(target.Type()))
			continue
		}
		return fmt.Errorf(
			"scan value %d type %s cannot assign to %s",
			index,
			source.Type(),
			target.Type(),
		)
	}
	return nil
}

type smtpPasswordResetRepository struct {
	actor       identitypkg.User
	target      identitypkg.User
	latest      identitypkg.TemporaryCredential
	latestFound bool
}

func (r *smtpPasswordResetRepository) LockUser(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) (identitypkg.User, error) {
	switch userID {
	case r.actor.ID:
		return r.actor, nil
	case r.target.ID:
		return r.target, nil
	default:
		return identitypkg.User{}, pgx.ErrNoRows
	}
}

func (r *smtpPasswordResetRepository) LockLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (identitypkg.TemporaryCredential, error) {
	if !r.latestFound {
		return identitypkg.TemporaryCredential{}, pgx.ErrNoRows
	}
	return r.latest, nil
}

func (*smtpPasswordResetRepository) UpdateTemporaryCredential(
	context.Context,
	pgx.Tx,
	identitypkg.TemporaryCredential,
) error {
	return errors.New("unexpected password-reset credential update before settlement")
}

func (*smtpPasswordResetRepository) InsertTemporaryCredential(
	context.Context,
	pgx.Tx,
	identitypkg.TemporaryCredential,
) error {
	return errors.New("unexpected repository insert; durable Store owns candidate")
}

func (*smtpPasswordResetRepository) SetCredentialMode(
	context.Context,
	pgx.Tx,
	string,
	identitypkg.CredentialMode,
	time.Time,
) error {
	return errors.New("unexpected credential-mode change before settlement")
}

func (*smtpPasswordResetRepository) RevokeSessions(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
	string,
) error {
	return errors.New("unexpected session revocation before settlement")
}

type smtpPasswordPolicy struct{}

func (smtpPasswordPolicy) Validate(string) error { return nil }

type smtpPasswordSecret struct{}

func (smtpPasswordSecret) Generate() (string, error) { return "Temporary123", nil }

func smtpPointer[T any](value T) *T { return &value }
