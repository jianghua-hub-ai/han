//go:build integration

package articleintegration

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/mailer"
)

const noticeSMTPHost = "smtp.notice.test"

type noticeSMTPOptions struct {
	mode                                                                  mailer.TLSMode
	stage                                                                 string
	code                                                                  int
	noSTARTTLS, quitFailure, ackLoss, expired, wrongHost, untrusted, auth bool
	block                                                                 string
	rejectFormer                                                          bool
}
type noticeSMTPServer struct {
	t                                 *testing.T
	listener                          net.Listener
	cert                              tls.Certificate
	roots                             *x509.CertPool
	options                           noticeSMTPOptions
	ctx                               context.Context
	cancel                            context.CancelFunc
	mu                                sync.Mutex
	active                            map[net.Conn]bool
	commands                          []string
	protocol                          []string
	messages                          []string
	connections, handshakes           int
	wg                                sync.WaitGroup
	reached                           chan struct{}
	release                           chan struct{}
	reachOnce, releaseOnce, closeOnce sync.Once
	blocked                           atomic.Bool
	abortBlocked                      atomic.Bool
	closed                            chan struct{}
}

func noticeCertificates(t *testing.T, o noticeSMTPOptions) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	noticeMust(t, err)
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Synthetic S72 CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, key)
	noticeMust(t, err)
	root, err := x509.ParseCertificate(der)
	noticeMust(t, err)
	leafPub, leafKey, err := ed25519.GenerateKey(rand.Reader)
	noticeMust(t, err)
	host := noticeSMTPHost
	if o.wrongHost {
		host = "wrong.notice.test"
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), DNSNames: []string{host}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if o.expired {
		leaf.NotBefore = now.Add(-2 * time.Hour)
		leaf.NotAfter = now.Add(-time.Hour)
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, root, leafPub, key)
	noticeMust(t, err)
	roots := x509.NewCertPool()
	if !o.untrusted {
		roots.AddCert(root)
	}
	return tls.Certificate{Certificate: [][]byte{leafDER, der}, PrivateKey: leafKey}, roots
}
func newNoticeSMTP(t *testing.T, o noticeSMTPOptions) *noticeSMTPServer {
	t.Helper()
	if o.mode == "" {
		o.mode = mailer.TLSModeSTARTTLSRequired
	}
	cert, roots := noticeCertificates(t, o)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	noticeMust(t, err)
	s := &noticeSMTPServer{t: t, listener: l, cert: cert, roots: roots, options: o, active: map[net.Conn]bool{}, reached: make(chan struct{}), release: make(chan struct{}), closed: make(chan struct{}, 16)}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg.Add(1)
	go s.serve()
	t.Cleanup(s.close)
	return s
}
func (s *noticeSMTPServer) close() {
	s.closeOnce.Do(func() {
		s.cancel()
		s.releaseOnce.Do(func() { close(s.release) })
		_ = s.listener.Close()
		s.mu.Lock()
		for c := range s.active {
			_ = c.Close()
		}
		s.mu.Unlock()
	})
	s.wg.Wait()
}
func (s *noticeSMTPServer) serve() {
	defer s.wg.Done()
	for {
		c, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.active[c] = true
		s.connections++
		s.mu.Unlock()
		s.wg.Add(1)
		go s.connection(c)
	}
}
func (s *noticeSMTPServer) connection(raw net.Conn) {
	defer s.wg.Done()
	defer func() { _ = raw.Close(); s.mu.Lock(); delete(s.active, raw); s.mu.Unlock(); s.closed <- struct{}{} }()
	_ = raw.SetDeadline(time.Now().Add(10 * time.Second))
	var c net.Conn = raw
	secure := false
	if s.options.mode == mailer.TLSModeImplicit {
		var err error
		c, err = s.secure(c)
		if err != nil {
			return
		}
		secure = true
	}
	_ = s.session(c, secure)
}
func (s *noticeSMTPServer) secure(c net.Conn) (net.Conn, error) {
	t := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{s.cert}, MinVersion: tls.VersionTLS12})
	if err := t.HandshakeContext(s.ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.handshakes++
	s.protocol = append(s.protocol, "TLS")
	s.mu.Unlock()
	return t, nil
}
func (s *noticeSMTPServer) barrier(stage string) error {
	if s.options.block != stage || !s.blocked.CompareAndSwap(false, true) {
		return nil
	}
	s.reachOnce.Do(func() { close(s.reached) })
	select {
	case <-s.release:
		if s.abortBlocked.Load() {
			return io.EOF
		}
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}
func (s *noticeSMTPServer) reject(c net.Conn, stage string) bool {
	if s.options.stage != stage {
		return false
	}
	_, _ = fmt.Fprintf(c, "%d synthetic response-canary@example.test\r\n", s.options.code)
	return true
}
func (s *noticeSMTPServer) session(c net.Conn, secure bool) error {
	if _, err := fmt.Fprint(c, "220 synthetic ESMTP\r\n"); err != nil {
		return err
	}
	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		verb := strings.ToUpper(strings.Fields(line)[0])
		s.mu.Lock()
		s.commands = append(s.commands, verb)
		s.protocol = append(s.protocol, verb)
		s.mu.Unlock()
		switch verb {
		case "EHLO", "HELO":
			err = s.greet(c, secure, verb)
		case "STARTTLS":
			c, err = s.upgrade(c)
			if err != nil {
				return err
			}
			secure = true
			r = bufio.NewReader(c)
		case "AUTH":
			if !s.reject(c, "auth") {
				_, err = fmt.Fprint(c, "235 OK\r\n")
			}
		case "*":
			_, err = fmt.Fprint(c, "501 canceled\r\n")
		case "MAIL", "RCPT":
			err = s.envelope(c, secure, verb, line)
		case "DATA":
			err = s.data(c, r)
		case "QUIT":
			if s.options.quitFailure {
				return nil
			}
			_, err = fmt.Fprint(c, "221 bye\r\n")
			return err
		default:
			return errors.New("unexpected SMTP verb")
		}
		if err != nil {
			return err
		}
	}
}
func (s *noticeSMTPServer) greet(c net.Conn, secure bool, verb string) error {
	if s.reject(c, "greeting") {
		return nil
	}
	if verb == "EHLO" {
		if _, err := fmt.Fprint(c, "250-synthetic\r\n"); err != nil {
			return err
		}
		if !secure && !s.options.noSTARTTLS {
			if _, err := fmt.Fprint(c, "250-STARTTLS\r\n"); err != nil {
				return err
			}
		}
		if secure && s.options.auth {
			if _, err := fmt.Fprint(c, "250-AUTH PLAIN\r\n"); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprint(c, "250 OK\r\n")
	return err
}
func (s *noticeSMTPServer) upgrade(c net.Conn) (net.Conn, error) {
	if s.options.noSTARTTLS {
		return nil, errors.New("unexpected TLS upgrade")
	}
	if _, err := fmt.Fprint(c, "220 TLS\r\n"); err != nil {
		return nil, err
	}
	return s.secure(c)
}
func (s *noticeSMTPServer) envelope(c net.Conn, secure bool, verb, line string) error {
	if !secure {
		return errors.New("plaintext envelope")
	}
	stage := "mail_from"
	if verb == "RCPT" {
		stage = "rcpt_to"
	}
	if s.options.rejectFormer && strings.Contains(line, "snapshot-1@") {
		_, err := fmt.Fprint(c, "550 synthetic rejection\r\n")
		return err
	}
	if s.reject(c, stage) {
		return nil
	}
	_, err := fmt.Fprint(c, "250 OK\r\n")
	return err
}
func (s *noticeSMTPServer) data(c net.Conn, r *bufio.Reader) error {
	if s.reject(c, "data_command") {
		return nil
	}
	if _, err := fmt.Fprint(c, "354 send\r\n"); err != nil {
		return err
	}
	var body strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		if err := s.barrier("during_data"); err != nil {
			return err
		}
		if line == ".\r\n" {
			break
		}
		body.WriteString(line)
		if body.Len() > 65536 {
			return errors.New("unexpected oversized fixture")
		}
	}
	if s.reject(c, "data_final") {
		return nil
	}
	s.mu.Lock()
	s.messages = append(s.messages, body.String())
	s.mu.Unlock()
	if s.options.ackLoss {
		return io.EOF
	}
	if err := s.barrier("before_accept_ack"); err != nil {
		return err
	}
	_, err := fmt.Fprint(c, "250 queued\r\n")
	return err
}
func (s *noticeSMTPServer) count() (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connections, len(s.messages), len(s.active)
}
func (s *noticeSMTPServer) mailer() mailer.Mailer {
	s.t.Helper()
	cfg := mailer.SMTPConfig{Address: s.listener.Addr().String(), Host: noticeSMTPHost, From: "sender@example.test", Mode: s.options.mode, TLSConfig: &tls.Config{ServerName: noticeSMTPHost, RootCAs: s.roots, MinVersion: tls.VersionTLS12}}
	if s.options.auth {
		cfg.Auth = smtp.PlainAuth("", "synthetic-user", "synthetic-password", noticeSMTPHost)
	}
	m, err := mailer.NewSMTPMailer(cfg, &net.Dialer{})
	noticeMust(s.t, err)
	return m
}

type noticeSMTPCase struct {
	name     string
	options  noticeSMTPOptions
	bucket   string
	terminal bool
	max      int64
}

func TestNoticeSMTP(t *testing.T) {
	for _, tc := range noticeSMTPCases() {
		for _, max := range []int64{1, 3} {
			tc.max = max
			t.Run(fmt.Sprintf("%s_max_%d", tc.name, max), func(t *testing.T) { testNoticeSMTPCase(t, tc) })
		}
	}
	t.Run("PurposeAndTemplate", testNoticeTemplate)
	t.Run("TwoRecipientsIndependent", testNoticeSMTPIndependent)
	t.Run("PoolReleasedDuringSend", testNoticeSMTPPool)
	t.Run("InitialCancellationSharedDeadline", testNoticeShutdown)
	t.Run("RejectInsecureConfiguration", testNoticeInsecure)
	t.Run("GateClosedDuringSend", testNoticeInFlightGate)
	t.Run("BackoffDueAndActualRetry", testNoticeActualRetry)
	t.Run("SenderSecurityStopsRound", testNoticeStopRound)
	t.Run("StopBeforeInitialReconcile", testNoticeStopBeforeInitial)
	t.Run("SharedDeadlineSettlement", testNoticeSharedSettlement)
	t.Run("CancelBeforeSuccessorClaim", testNoticeStopBeforeNext)
}
func noticeSMTPCases() []noticeSMTPCase {
	cases := []noticeSMTPCase{{name: "ImplicitTLS", options: noticeSMTPOptions{mode: mailer.TLSModeImplicit}}, {name: "RequiredSTARTTLS"}, {name: "AcceptedQUITFailure", options: noticeSMTPOptions{quitFailure: true}}, {name: "AcceptedACKLost", options: noticeSMTPOptions{ackLoss: true}, bucket: "smtp_outcome_unknown"}}
	for _, stage := range []string{"mail_from", "rcpt_to", "data_command", "data_final"} {
		for _, code := range []int{421, 550} {
			bucket := "smtp_transient_rejection"
			terminal := false
			if stage == "mail_from" {
				bucket = "smtp_sender_or_policy"
			} else if code >= 500 {
				bucket = "smtp_permanent_rejection"
				terminal = true
			}
			cases = append(cases, noticeSMTPCase{name: fmt.Sprintf("%s_%d", stage, code), options: noticeSMTPOptions{stage: stage, code: code}, bucket: bucket, terminal: terminal})
		}
	}
	for _, tc := range []noticeSMTPCase{{name: "Auth535", options: noticeSMTPOptions{auth: true, stage: "auth", code: 535}}, {name: "Greeting550", options: noticeSMTPOptions{stage: "greeting", code: 550}}, {name: "MissingSTARTTLS", options: noticeSMTPOptions{noSTARTTLS: true}}, {name: "ExpiredCertificate", options: noticeSMTPOptions{expired: true}}, {name: "WrongHostname", options: noticeSMTPOptions{wrongHost: true}}, {name: "UntrustedCA", options: noticeSMTPOptions{untrusted: true}}} {
		tc.bucket = "smtp_security_or_config"
		cases = append(cases, tc)
	}
	return cases
}
func testNoticeSMTPCase(t *testing.T, tc noticeSMTPCase) {
	h := newNoticeHarness(t)
	id := h.seed(tc.max, false)
	s := newNoticeSMTP(t, tc.options)
	var sendErr error
	m := s.mailer()
	d := h.dispatcher(noticeSendFunc(func(ctx context.Context, msg mailer.Message) error { sendErr = m.Send(ctx, msg); return sendErr }))
	noticeMust(t, d.Reconcile(h.ctx))
	state := article.RecipientRetryWait
	if tc.bucket == "" {
		state = article.RecipientAccepted
	}
	if tc.terminal || (tc.bucket != "" && tc.max == 1) {
		state = article.RecipientFailedTerminal
	}
	h.assertState(id, state, 1, tc.bucket)
	noticeAssertSMTPProtocol(t, s, tc)
	if tc.options.stage != "" && tc.options.stage != "auth" && tc.options.stage != "greeting" {
		var stage *mailer.SMTPMessageRejection
		var raw *textproto.Error
		if !errors.Is(sendErr, mailer.ErrMessageRejected) || !errors.As(sendErr, &stage) || !errors.As(sendErr, &raw) || string(stage.Stage) != tc.options.stage || stage.Code != raw.Code {
			t.Fatal("structured SMTP rejection chain mismatch")
		}
		if strings.Contains(sendErr.Error(), "response-canary") {
			t.Fatal("rejection printable cause leak")
		}
	}
	if tc.options.stage == "auth" || tc.options.stage == "greeting" {
		var stage *mailer.SMTPMessageRejection
		if errors.Is(sendErr, mailer.ErrMessageRejected) || errors.As(sendErr, &stage) {
			t.Fatal("global rejection classified as recipient rejection")
		}
	}
	connections, _, _ := s.count()
	if connections != 1 {
		t.Fatal("one claim did not make exactly one network connection")
	}
	if tc.bucket == "smtp_outcome_unknown" && h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.duplicate_risk'") != 1 {
		t.Fatal("unknown outcome missing risk")
	}
	h.log.mu.Lock()
	printed := fmt.Sprint(h.log.values)
	h.log.mu.Unlock()
	if strings.Contains(printed, "@") || strings.Contains(printed, "synthetic-password") {
		t.Fatal("sensitive protocol value reached logger")
	}
}

func noticeAssertSMTPProtocol(t *testing.T, s *noticeSMTPServer, tc noticeSMTPCase) {
	t.Helper()
	s.mu.Lock()
	commands := append([]string(nil), s.commands...)
	protocol := append([]string(nil), s.protocol...)
	handshakes, accepted := s.handshakes, len(s.messages)
	s.mu.Unlock()
	securityFailure := tc.options.noSTARTTLS || tc.options.expired || tc.options.wrongHost || tc.options.untrusted || tc.options.stage == "auth" || tc.options.stage == "greeting"
	for _, command := range commands {
		if securityFailure && (command == "MAIL" || command == "RCPT" || command == "DATA") {
			t.Fatal("F001: security failure emitted SMTP envelope before rejection")
		}
	}
	if securityFailure {
		if accepted != 0 {
			t.Fatal("F001: security failure accepted a message")
		}
		return
	}
	secure, envelope := false, false
	for _, event := range protocol {
		if event == "TLS" {
			secure = true
		}
		if event == "MAIL" || event == "RCPT" || event == "DATA" {
			envelope = true
			if !secure {
				t.Fatal("F001: SMTP envelope preceded verified TLS handshake")
			}
		}
	}
	if handshakes != 1 || !envelope {
		t.Fatal("F001: secure SMTP handshake/envelope evidence missing")
	}
	if tc.bucket == "" && accepted != 1 {
		t.Fatal("F001: successful secure transport did not accept exactly one message")
	}
}
func testNoticeTemplate(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	s := newNoticeSMTP(t, noticeSMTPOptions{})
	d := h.dispatcher(s.mailer())
	noticeMust(t, d.Reconcile(h.ctx))
	s.mu.Lock()
	messages := append([]string(nil), s.messages...)
	s.mu.Unlock()
	if len(messages) != 2 {
		t.Fatal("both snapshots were not sent")
	}
	for _, raw := range messages {
		msg, err := mail.ReadMessage(strings.NewReader(raw))
		noticeMust(t, err)
		body, err := io.ReadAll(msg.Body)
		noticeMust(t, err)
		if msg.Header.Get("X-Han-Purpose") != string(mailer.PurposeOrphanArticleTransferNotice) || msg.Header.Get("Subject") != "Han article transfer notice" || !strings.Contains(string(body), id) {
			t.Fatal("purpose/template references mismatch")
		}
		if strings.Contains(string(body), "@") || strings.Contains(string(body), "http") || strings.Contains(string(body), "<html") {
			t.Fatal("template exceeded minimal disclosure")
		}
	}
}
func testNoticeSMTPIndependent(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	s := newNoticeSMTP(t, noticeSMTPOptions{rejectFormer: true})
	d := h.dispatcher(s.mailer())
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientAccepted, 1, "")
	if h.recipient(id, article.RecipientFormerDeletedAuthor).Status != article.RecipientFailedTerminal {
		t.Fatal("second recipient did not independently terminate")
	}
}
func testNoticeSMTPPool(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	s := newNoticeSMTP(t, noticeSMTPOptions{block: "during_data"})
	out := make(chan error, 1)
	d := h.dispatcher(s.mailer())
	h.group.Add(1)
	go func() { defer h.group.Done(); out <- d.Reconcile(h.ctx) }()
	select {
	case <-s.reached:
	case <-h.ctx.Done():
		t.Fatal("DATA barrier not reached")
	}
	if h.pool.Stat().AcquiredConns() != 0 {
		t.Fatal("SMTP held a pool connection")
	}
	if h.scalar("SELECT count(*) FROM transfer_notice_recipients") != 2 {
		t.Fatal("concurrent database query failed")
	}
	s.releaseOnce.Do(func() { close(s.release) })
	noticeMust(t, <-out)
	h.assertState(id, article.RecipientAccepted, 1, "")
}
func testNoticeShutdown(t *testing.T) {
	for _, stage := range []string{"during_data", "after_accepted"} {
		t.Run(stage, func(t *testing.T) {
			h := newNoticeHarness(t)
			id := h.seed(3, true)
			s := newNoticeSMTP(t, noticeShutdownOptions(stage))
			out := make(chan error, 1)
			var calls atomic.Int64
			m := s.mailer()
			accepted := make(chan struct{})
			release := make(chan struct{})
			var once sync.Once
			d := h.dispatcher(noticeSendFunc(func(ctx context.Context, msg mailer.Message) error {
				calls.Add(1)
				err := m.Send(ctx, msg)
				if stage == "after_accepted" && err == nil {
					close(accepted)
					select {
					case <-release:
					case <-h.ctx.Done():
						return h.ctx.Err()
					}
				}
				return err
			}))
			t.Cleanup(func() { once.Do(func() { close(release) }); s.close() })
			h.group.Add(1)
			go func() { defer h.group.Done(); out <- d.Reconcile(h.ctx) }()
			barrier := s.reached
			if stage == "after_accepted" {
				barrier = accepted
			}
			noticeAwaitShutdownBarrier(t, h, barrier, out)
			shared, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			d.BeginShutdown(shared)
			d.Stop()
			once.Do(func() { close(release) })
			s.close()
			noticeMust(t, <-out)
			noticeAssertShutdown(h, id, d, shared, calls.Load())
			noticeAssertShutdownResult(h, id, stage)
		})
	}
}
func noticeAwaitShutdownBarrier(t *testing.T, h *noticeHarness, barrier <-chan struct{}, out <-chan error) {
	t.Helper()
	select {
	case <-barrier:
	case err := <-out:
		noticeMust(t, err)
		t.Fatal("F002: Reconcile returned without reaching successful SMTP shutdown barrier")
	case <-h.ctx.Done():
		t.Fatal("shutdown barrier missing")
	}
}
func noticeAssertShutdownResult(h *noticeHarness, id, stage string) {
	h.t.Helper()
	state, bucket := article.RecipientAccepted, ""
	actions := []string{"accepted"}
	if stage == "during_data" {
		state, bucket = article.RecipientRetryWait, "smtp_outcome_unknown"
		actions = []string{"retry_scheduled", "duplicate_risk"}
	}
	r := h.assertState(id, state, 1, bucket)
	if r.NoticeID != id || r.Kind != article.RecipientNewAuthor || r.ClaimedBy != nil || r.LockedUntil != nil || r.LastAttemptAt == nil {
		h.t.Fatal("F002: settled shutdown result retained ownership or lost recipient binding")
	}
	if state == article.RecipientAccepted {
		if r.AcceptedAt == nil || r.NextAttemptAt != nil || r.FailureBucket != nil || !r.AcceptedAt.Equal(r.UpdatedAt) {
			h.t.Fatal("F002: accepted shutdown result lacks accepted_at or field cleanup")
		}
	} else if r.AcceptedAt != nil || r.NextAttemptAt == nil || !r.NextAttemptAt.After(r.UpdatedAt) {
		h.t.Fatal("F002: cancelled DATA result lacks bounded unknown retry")
	}
	if h.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1", id) != int64(len(actions)) {
		h.t.Fatal("F002: shutdown result audit count mismatch")
	}
	for _, action := range actions {
		noticeAssertShutdownAudit(h, id, action, r)
	}
}
func noticeAssertShutdownAudit(h *noticeHarness, id, action string, r article.TransferNoticeRecipient) {
	h.t.Helper()
	fullAction := "article.transfer_notice." + action
	if h.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1 AND action=$2", id, fullAction) != 1 {
		h.t.Fatal("F002: shutdown result lacks unique durable audit action")
	}
	var eventID string
	noticeMust(h.t, h.pool.QueryRow(h.ctx, "SELECT id::text FROM audit_events WHERE target_id=$1 AND action=$2", id, fullAction).Scan(&eventID))
	event, err := (audit.Store{}).Get(h.ctx, h.pool, eventID)
	noticeMust(h.t, err)
	if event.TargetID != id || event.TargetType != "article_transfer_notice" || event.Action != fullAction || event.TraceID != noticeTrace || !event.OccurredAt.Equal(r.UpdatedAt) {
		h.t.Fatal("F002: durable shutdown audit notice/trace/time binding mismatch")
	}
	if event.ActorKind != audit.ActorSystem || event.ActorSystem == nil || *event.ActorSystem != "article-notice-dispatcher" || event.Metadata["recipient_kind"] != string(article.RecipientNewAuthor) || event.Metadata["attempt_count"] != "1" || event.Metadata["max_attempts"] != "3" {
		h.t.Fatal("F002: durable shutdown audit kind/attempt/actor binding mismatch")
	}
	if action == "duplicate_risk" && event.Outcome != "risk" {
		h.t.Fatal("F002: unknown shutdown result audit did not record risk")
	}
	if action != "duplicate_risk" && event.Outcome != string(r.Status) {
		h.t.Fatal("F002: shutdown result audit outcome mismatch")
	}
}
func noticeShutdownOptions(stage string) noticeSMTPOptions {
	if stage == "after_accepted" {
		return noticeSMTPOptions{}
	}
	return noticeSMTPOptions{block: "during_data"}
}
func noticeAssertJoined(t *testing.T, d *article.TransferNoticeDispatcher) {
	t.Helper()
	select {
	case <-d.Done():
	default:
		t.Fatal("initial reconcile not joined")
	}
}
func noticeAssertShutdown(h *noticeHarness, id string, d *article.TransferNoticeDispatcher, shared context.Context, calls int64) {
	noticeMust(h.t, d.Wait(shared))
	noticeAssertJoined(h.t, d)
	if calls != 1 {
		h.t.Fatal("shutdown started successor send")
	}
	if h.recipient(id, article.RecipientFormerDeletedAuthor).AttemptCount != 0 {
		h.t.Fatal("shutdown claimed successor")
	}
	if h.pool.Stat().AcquiredConns() != 0 {
		h.t.Fatal("shutdown leaked connection")
	}
}
func testNoticeInsecure(t *testing.T) {
	s := newNoticeSMTP(t, noticeSMTPOptions{})
	cfg := mailer.SMTPConfig{Address: s.listener.Addr().String(), Host: noticeSMTPHost, From: "sender@example.test", Mode: mailer.TLSModeImplicit, TLSConfig: &tls.Config{ServerName: noticeSMTPHost, MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}}
	if _, err := mailer.NewSMTPMailer(cfg, &net.Dialer{}); !errors.Is(err, mailer.ErrInvalidSMTPConfig) {
		t.Fatal("insecure verification was accepted")
	}
	if n, _, _ := s.count(); n != 0 {
		t.Fatal("invalid constructor connected")
	}
}

func testNoticeInFlightGate(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	s := newNoticeSMTP(t, noticeSMTPOptions{block: "during_data"})
	out := make(chan error, 1)
	d := h.dispatcher(s.mailer())
	h.group.Add(1)
	go func() { defer h.group.Done(); out <- d.Reconcile(h.ctx) }()
	select {
	case <-s.reached:
	case <-h.ctx.Done():
		t.Fatal("SMTP barrier missing")
	}
	h.gateRecord("closed")
	s.releaseOnce.Do(func() { close(s.release) })
	noticeMust(t, <-out)
	h.assertState(id, article.RecipientAccepted, 1, "")
	other := h.recipient(id, article.RecipientFormerDeletedAuthor)
	if other.Status != article.RecipientPending || other.AttemptCount != 0 {
		t.Fatal("Gate closure allowed successor claim")
	}
	if n, accepted, _ := s.count(); n != 1 || accepted != 1 {
		t.Fatal("Gate closure discarded accepted in-flight result or started another send")
	}
}
func testNoticeActualRetry(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	reject := newNoticeSMTP(t, noticeSMTPOptions{stage: "rcpt_to", code: 450})
	success := newNoticeSMTP(t, noticeSMTPOptions{})
	var calls atomic.Int64
	a, b := reject.mailer(), success.mailer()
	d := h.dispatcher(noticeSendFunc(func(ctx context.Context, m mailer.Message) error {
		if calls.Add(1) == 1 {
			return a.Send(ctx, m)
		}
		return b.Send(ctx, m)
	}), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
		c.BackoffInitial = time.Second
	})
	noticeMust(t, d.Reconcile(h.ctx))
	r := h.assertState(id, article.RecipientRetryWait, 1, "smtp_transient_rejection")
	if r.NextAttemptAt == nil || r.NextAttemptAt.Sub(r.UpdatedAt) != time.Second {
		t.Fatal("retry due time differs from DB-based backoff")
	}
	noticeMust(t, d.Reconcile(h.ctx))
	if calls.Load() != 1 {
		t.Fatal("retry sent before due")
	}
	h.waitTime(*r.NextAttemptAt)
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientAccepted, 2, "")
	if calls.Load() != 2 || h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.retry_scheduled'") != 1 {
		t.Fatal("actual retry duplicated Send/audit")
	}
}
func testNoticeStopRound(t *testing.T) {
	for _, tc := range []noticeSMTPCase{{name: "sender", options: noticeSMTPOptions{stage: "mail_from", code: 550}, bucket: "smtp_sender_or_policy"}, {name: "auth", options: noticeSMTPOptions{auth: true, stage: "auth", code: 535}, bucket: "smtp_security_or_config"}, {name: "TLS", options: noticeSMTPOptions{wrongHost: true}, bucket: "smtp_security_or_config"}} {
		t.Run(tc.name, func(t *testing.T) {
			h := newNoticeHarness(t)
			id := h.seed(3, true)
			s := newNoticeSMTP(t, tc.options)
			d := h.dispatcher(s.mailer())
			noticeMust(t, d.Reconcile(h.ctx))
			h.assertState(id, article.RecipientRetryWait, 1, tc.bucket)
			other := h.recipient(id, article.RecipientFormerDeletedAuthor)
			if other.AttemptCount != 0 || other.Status != article.RecipientPending {
				t.Fatal("sender/security failure consumed successor")
			}
			if n, _, _ := s.count(); n != 1 {
				t.Fatal("sender/security fault did not end round")
			}
		})
	}
}

func testNoticeStopBeforeInitial(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	var calls atomic.Int64
	var begins atomic.Int64
	x := &noticeTransactions{pool: h.pool, beginHook: func(context.Context) error { begins.Add(1); return nil }}
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) { d.Transactions = x })
	shared, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.BeginShutdown(shared)
	d.Stop()
	noticeMust(t, d.Reconcile(h.ctx))
	d.Start(h.ctx)
	noticeMust(t, d.Wait(shared))
	h.assertState(id, article.RecipientPending, 0, "")
	if calls.Load() != 0 || begins.Load() != 0 {
		t.Fatal("shutdown-before-initial started work")
	}
	select {
	case <-d.Done():
	default:
		t.Fatal("idle startup stop did not close Done")
	}
}
func testNoticeSharedSettlement(t *testing.T) {
	for _, budget := range []time.Duration{150 * time.Millisecond, 2 * time.Second} {
		t.Run(budget.String(), func(t *testing.T) {
			h := newNoticeHarness(t)
			h.expectWorkerError = true
			id := h.seed(3, false)
			s := newNoticeSMTP(t, noticeSMTPOptions{})
			m := s.mailer()
			var armed atomic.Bool
			var deadline time.Time
			var calls atomic.Int64
			var sharedDeadline, shutdownStarted time.Time
			x := &noticeTransactions{pool: h.pool}
			x.beginHook = func(ctx context.Context) error {
				if !armed.Load() {
					return nil
				}
				deadline, _ = ctx.Deadline()
				<-ctx.Done()
				return ctx.Err()
			}
			var d *article.TransferNoticeDispatcher
			d = h.dispatcher(noticeSendFunc(func(ctx context.Context, msg mailer.Message) error {
				calls.Add(1)
				err := m.Send(ctx, msg)
				shared, cancel := context.WithTimeout(context.Background(), budget)
				t.Cleanup(cancel)
				sharedDeadline, _ = shared.Deadline()
				shutdownStarted = time.Now()
				armed.Store(true)
				d.BeginShutdown(shared)
				d.Stop()
				return err
			}), func(deps *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
				deps.Transactions = x
			})
			err := d.Reconcile(h.ctx)
			if err == nil {
				t.Fatal("failed shutdown settlement was hidden")
			}
			if deadline.After(sharedDeadline) || deadline.After(shutdownStarted.Add(time.Second+100*time.Millisecond)) {
				t.Fatal("settlement discarded shared/DB deadline")
			}
			noticeAssertJoined(t, d)
			h.assertState(id, article.RecipientProcessing, 1, "")
			if calls.Load() != 1 || h.scalar("SELECT count(*) FROM audit_events") != 0 || h.pool.Stat().AcquiredConns() != 0 {
				t.Fatal("failed settlement resent/leaked state or connection")
			}
		})
	}
}
func testNoticeStopBeforeNext(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	s := newNoticeSMTP(t, noticeSMTPOptions{})
	shared, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	x := &noticeTransactions{pool: h.pool}
	var d *article.TransferNoticeDispatcher
	x.hook = func(ctx context.Context, tx *noticeTx) error {
		err := tx.Tx.Commit(ctx)
		if err == nil && tx.phase == "result" {
			d.BeginShutdown(shared)
			d.Stop()
		}
		return err
	}
	d = h.dispatcher(s.mailer(), func(deps *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
		deps.Transactions = x
	})
	noticeMust(t, d.Reconcile(h.ctx))
	noticeMust(t, d.Wait(shared))
	h.assertState(id, article.RecipientAccepted, 1, "")
	if h.recipient(id, article.RecipientFormerDeletedAuthor).AttemptCount != 0 {
		t.Fatal("cancellation before successor still claimed")
	}
	if connections, _, _ := s.count(); connections != 1 {
		t.Fatal("successor connected after shutdown")
	}
}
