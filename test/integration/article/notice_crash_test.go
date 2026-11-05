//go:build integration

package articleintegration

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"han/internal/article"
	"han/internal/mailer"
)

const noticeChildMode = "HAN_S72_CHILD_MODE"
const noticeChildExit = 23

func TestNoticeCrash(t *testing.T) {
	if mode := os.Getenv(noticeChildMode); mode != "" {
		noticeCrashChild(t, mode)
		return
	}
	for _, point := range []string{"AfterClaim", "DuringDATA", "AcceptedBeforeSave", "AfterResultCommit"} {
		for _, max := range []int64{1, 3} {
			t.Run(fmt.Sprintf("%s_max_%d", point, max), func(t *testing.T) { testNoticeCrashPoint(t, point, max) })
		}
	}
}

func noticeCrashChild(t *testing.T, point string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	schema := os.Getenv("HAN_S72_CHILD_SCHEMA")
	if !strings.HasPrefix(schema, "han_s72_") {
		t.Fatal("invalid child schema")
	}
	pool, err := pgxpool.NewWithConfig(ctx, noticePoolConfig(t, schema))
	noticeMust(t, err)
	defer pool.Close()
	h := &noticeHarness{t: t, ctx: ctx, cancel: cancel, pool: pool, schema: schema, log: &noticeLog{}}
	h.gate = &noticeGate{h: h}
	m := noticeChildMailer(t)
	x := &noticeTransactions{pool: pool}
	x.hook = func(c context.Context, tx *noticeTx) error {
		err := tx.Tx.Commit(c)
		if err != nil {
			return err
		}
		if (point == "AfterClaim" && tx.phase == "claim") || (point == "AfterResultCommit" && tx.phase == "result") {
			noticeChildTerminate(point)
		}
		return nil
	}
	d := h.dispatcher(noticeSendFunc(func(c context.Context, msg mailer.Message) error {
		err := m.Send(c, msg)
		if err == nil && point == "AcceptedBeforeSave" {
			noticeChildTerminate(point)
		}
		return err
	}), func(deps *article.NoticeDispatcherDependencies, cfg *article.NoticeDispatcherConfig) {
		deps.Transactions = x
	})
	fmt.Println("S72_CHILD_READY")
	noticeMust(t, d.Reconcile(ctx))
	t.Fatal("child did not reach the requested crash point")
}

func noticeChildTerminate(point string) {
	fmt.Println("S72_CHILD_EXIT " + point)
	os.Exit(noticeChildExit)
}
func noticeChildMailer(t *testing.T) mailer.Mailer {
	t.Helper()
	der, err := base64.StdEncoding.DecodeString(os.Getenv("HAN_S72_CHILD_CA"))
	noticeMust(t, err)
	ca, err := x509.ParseCertificate(der)
	noticeMust(t, err)
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	address := os.Getenv("HAN_S72_CHILD_SMTP")
	host, _, err := net.SplitHostPort(address)
	noticeMust(t, err)
	if host != "127.0.0.1" {
		t.Fatal("child SMTP not loopback")
	}
	m, err := mailer.NewSMTPMailer(mailer.SMTPConfig{Address: address, Host: noticeSMTPHost, From: "sender@example.test", Mode: mailer.TLSModeSTARTTLSRequired, TLSConfig: &tls.Config{ServerName: noticeSMTPHost, RootCAs: roots, MinVersion: tls.VersionTLS12}}, &net.Dialer{})
	noticeMust(t, err)
	return m
}

func noticeChildEnvironment(h *noticeHarness, s *noticeSMTPServer, point string) []string {
	values := []string{}
	for _, key := range []string{"SystemRoot", "TEMP", "TMP", "PATH"} {
		if value, ok := os.LookupEnv(key); ok {
			values = append(values, key+"="+value)
		}
	}
	return append(values, "HAN_TEST_POSTGRES_DSN="+os.Getenv("HAN_TEST_POSTGRES_DSN"), noticeChildMode+"="+point, "HAN_S72_CHILD_SCHEMA="+h.schema, "HAN_S72_CHILD_SMTP="+s.listener.Addr().String(), "HAN_S72_CHILD_CA="+base64.StdEncoding.EncodeToString(s.cert.Certificate[1]))
}

func testNoticeCrashPoint(t *testing.T, point string, max int64) {
	h := newNoticeHarness(t)
	id := h.seed(max, false)
	options := noticeSMTPOptions{}
	if point == "DuringDATA" {
		options.block = "during_data"
	}
	s := newNoticeSMTP(t, options)
	stage := noticeRunChild(t, h, s, point)
	if !stage {
		t.Fatal("child failure was not the selected behavioral crash point")
	}
	r := h.recipient(id, article.RecipientNewAuthor)
	noticeAssertCrashState(t, point, r)
	before, acceptedBefore, _ := s.count()
	if r.Status == article.RecipientProcessing {
		h.waitTime(*r.LockedUntil)
	}
	d := h.dispatcher(s.mailer())
	noticeMust(t, d.Reconcile(h.ctx))
	if point != "AfterResultCommit" && max > 1 {
		noticeWaitSMTPClosed(t, h, s)
	}
	after, acceptedAfter, _ := s.count()
	if point == "AfterResultCommit" {
		h.assertState(id, article.RecipientAccepted, 1, "")
		if after != before {
			t.Fatal("committed terminal was resent")
		}
	} else if max == 1 {
		h.assertState(id, article.RecipientFailedTerminal, 1, "attempts_exhausted_unknown")
		if after != before || acceptedAfter != acceptedBefore {
			t.Fatal("exhausted crashed attempt resent")
		}
	} else {
		h.assertState(id, article.RecipientAccepted, 2, "")
		if after != before+1 || acceptedAfter != acceptedBefore+1 {
			t.Fatal("crashed lease did not recover once")
		}
	}
	if point != "AfterResultCommit" && h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.duplicate_risk'") != 1 {
		t.Fatal("crashed unknown attempt lacks durable risk")
	}
	if point == "AcceptedBeforeSave" && max == 3 && acceptedAfter != 2 {
		t.Fatal("accepted/crash duplicate window not observed")
	}
	t.Logf("child_point=%s actual_exit=true smtp_connections=%d accepted=%d", point, after, acceptedAfter)
}
func noticeAssertCrashState(t *testing.T, point string, r article.TransferNoticeRecipient) {
	t.Helper()
	if point == "AfterResultCommit" {
		if r.Status != article.RecipientAccepted || r.AttemptCount != 1 {
			t.Fatal("committed child result missing")
		}
	} else if r.Status != article.RecipientProcessing || r.AttemptCount != 1 || r.LockedUntil == nil {
		t.Fatal("crash did not retain committed processing")
	}
}

func noticeRunChild(t *testing.T, h *noticeHarness, s *noticeSMTPServer, point string) bool {
	t.Helper()
	cmd := exec.CommandContext(h.ctx, os.Args[0], "-test.run=^TestNoticeCrash$", "-test.v", "-test.timeout=60s")
	cmd.Env = noticeChildEnvironment(h, s, point)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	noticeMust(t, cmd.Start())
	done := make(chan error, 1)
	h.group.Add(1)
	go func() { defer h.group.Done(); done <- cmd.Wait() }()
	if point == "DuringDATA" {
		select {
		case <-s.reached:
			s.abortBlocked.Store(true)
			noticeMust(t, cmd.Process.Kill())
		case <-done:
			t.Fatal("child exited before real DATA server barrier")
		case <-h.ctx.Done():
			_ = cmd.Process.Kill()
			<-done
			t.Fatal("child DATA handshake timed out")
		}
		s.releaseOnce.Do(func() { close(s.release) })
	}
	exit := noticeWaitChild(t, h, cmd, done)
	if point != "AfterClaim" {
		noticeWaitSMTPClosed(t, h, s)
	}
	if stderr.Len() != 0 {
		t.Fatal("child emitted unexpected stderr (raw content withheld)")
	}
	t.Logf("child_point=%s exit_code=%d output_joined=true socket_joined=true", point, exit.ExitCode())
	if point == "DuringDATA" {
		return true
	}
	if exit.ExitCode() != noticeChildExit {
		t.Fatalf("child failed before crash stage: exit=%d", exit.ExitCode())
	}
	found := false
	for _, line := range strings.Split(stdout.String(), "\n") {
		if line == "S72_CHILD_EXIT "+point {
			found = true
		}
	}
	return found
}
func noticeWaitChild(t *testing.T, h *noticeHarness, cmd *exec.Cmd, done <-chan error) *exec.ExitError {
	t.Helper()
	var err error
	select {
	case err = <-done:
	case <-h.ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("child failed to exit")
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal("crash child did not have a nonzero actual exit")
	}
	return exit
}

func noticeWaitSMTPClosed(t *testing.T, h *noticeHarness, s *noticeSMTPServer) {
	t.Helper()
	select {
	case <-s.closed:
	case <-h.ctx.Done():
		t.Fatal("SMTP connection handler did not exit")
	}
}
