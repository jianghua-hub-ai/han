package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"han/internal/recovery"
)

type recoveryStartedWorker interface {
	lifecycleWorker
	Started() <-chan struct{}
}
type recoveryStartupKey struct{}
type recoveryStartupScope struct {
	owner  *recoveryOrdinaryLifecycle
	permit recovery.PhasePermit
}

// recoveryOrdinaryLifecycle bridges the three registered owners to one Runtime.
// The root supplies actual joins for HTTP, maintenance, transactions and children.
// It owns resource closure after StopAndJoin; this adapter never calls Runtime.Stop.
type recoveryOrdinaryLifecycle struct {
	mu             sync.Mutex
	runtime        recovery.Runtime
	workers        [3]recoveryStartedWorker
	budget         time.Duration
	joins          []func(context.Context) error
	work           context.Context
	cancel         context.CancelFunc
	initial        chan struct{}
	stopOnce       sync.Once
	shutdownOnce   sync.Once
	stopped        chan struct{}
	shutdown       context.Context
	cancelShutdown context.CancelFunc
	stopErr        error
	coldOnce       sync.Once
	coldErr        error
}

func newRecoveryOrdinaryLifecycle(budget time.Duration, joins ...func(context.Context) error) (*recoveryOrdinaryLifecycle, error) {
	if budget <= 0 {
		return nil, errors.New("ordinary shutdown budget is required")
	}
	for _, join := range joins {
		if join == nil {
			return nil, errors.New("ordinary join owner is required")
		}
	}
	return &recoveryOrdinaryLifecycle{budget: budget, joins: joins, stopped: make(chan struct{})}, nil
}

func (l *recoveryOrdinaryLifecycle) bind(r recovery.Runtime, article, reset, notice recoveryStartedWorker) error {
	if missingRecoveryOrdinaryPort(r) || missingRecoveryOrdinaryPort(article) || missingRecoveryOrdinaryPort(reset) || missingRecoveryOrdinaryPort(notice) || noticeLifecycleIncomplete(notice) {
		return recovery.ErrAdmissionBinding
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.runtime != nil || l.initial != nil {
		return recovery.ErrAdmissionBusy
	}
	l.runtime = r
	l.workers = [3]recoveryStartedWorker{article, reset, notice}
	return nil
}

func missingRecoveryOrdinaryPort(p any) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Chan, reflect.Map, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func (l *recoveryOrdinaryLifecycle) ReconcileAndStart(ctx context.Context, permit recovery.PhasePermit) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	l.mu.Lock()
	if l.runtime == nil || l.initial != nil {
		l.mu.Unlock()
		return recovery.ErrAdmissionBinding
	}
	l.initial = make(chan struct{})
	initial := l.initial
	l.work, l.cancel = context.WithCancel(context.WithoutCancel(ctx))
	l.mu.Unlock()
	defer close(initial)
	if err := l.runtime.CheckPhase(ctx, permit, recovery.PhaseStartupReconcile); err != nil {
		return err
	}
	for _, worker := range l.workers {
		if err := l.reconcile(ctx, permit, worker); err != nil {
			return err
		}
	}
	for _, worker := range l.workers {
		if err := l.startWorker(ctx, worker); err != nil {
			return err
		}
	}
	return l.checkWorkers()
}

func (l *recoveryOrdinaryLifecycle) reconcile(ctx context.Context, permit recovery.PhasePermit, w recoveryStartedWorker) error {
	phase, err := l.runtime.PhaseContext(ctx, permit)
	if err != nil {
		return err
	}
	work, release := l.forwardCancellation(phase, ctx)
	defer release()
	return w.Reconcile(context.WithValue(work, recoveryStartupKey{}, recoveryStartupScope{l, permit}))
}

func (l *recoveryOrdinaryLifecycle) startWorker(ctx context.Context, w recoveryStartedWorker) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := l.checkWorkers(); err != nil {
		return err
	}
	w.Start(l.work)
	select {
	case <-w.Started():
		return l.checkWorkers()
	case <-w.Done():
		return lifecycleWorkerTrigger("ordinary", w, lifecycleStopPriority).err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *recoveryOrdinaryLifecycle) checkWorkers() error {
	for _, index := range []int{1, 0, 2} {
		w := l.workers[index]
		if err := w.Err(); err != nil {
			return err
		}
		select {
		case <-w.Done():
			return errors.New("ordinary worker ended before activation")
		default:
		}
	}
	return nil
}

func (l *recoveryOrdinaryLifecycle) acquire(kind recovery.WorkKind) func(context.Context) (context.Context, func(), error) {
	return func(ctx context.Context) (context.Context, func(), error) {
		if kind != recovery.WorkArticleDeletion && kind != recovery.WorkPasswordReset && kind != recovery.WorkArticleNotice {
			return nil, nil, recovery.ErrAdmissionBinding
		}
		if ctx == nil || l.runtime == nil {
			return nil, nil, recovery.ErrAdmissionBinding
		}
		if scope, ok := ctx.Value(recoveryStartupKey{}).(recoveryStartupScope); ok {
			if scope.owner != l {
				return nil, nil, recovery.ErrAdmissionBinding
			}
			if err := l.runtime.CheckPhase(ctx, scope.permit, recovery.PhaseStartupReconcile); err != nil {
				return nil, nil, err
			}
			return ctx, func() {}, nil
		}
		lease, err := l.runtime.Acquire(ctx, kind)
		if err != nil {
			return nil, nil, err
		}
		if missingRecoveryOrdinaryPort(lease) {
			return nil, nil, recovery.ErrAdmissionBinding
		}
		if missingRecoveryOrdinaryPort(lease.Context()) {
			lease.Release()
			return nil, nil, recovery.ErrAdmissionBinding
		}
		work, release := l.forwardCancellation(lease.Context(), ctx)
		return work, func() { release(); lease.Release() }, nil
	}
}

// Cancellation of a revoked generation first binds notice's fenced-save budget.
// A caller's own operation timeout remains local and does not stop other workers.
func (l *recoveryOrdinaryLifecycle) forwardCancellation(authorized, caller context.Context) (context.Context, func()) {
	work, cancel := context.WithCancel(context.WithoutCancel(authorized))
	finished := make(chan struct{})
	stop := context.AfterFunc(authorized, func() {
		defer close(finished)
		if shared, selected := l.runtime.ShutdownContext(); selected {
			l.bindShutdown(shared)
		}
		cancel()
	})
	var once sync.Once
	return recoveryWorkerContext{Context: work, deadline: caller, authorized: authorized}, func() {
		once.Do(func() {
			if !stop() {
				<-finished
			}
			cancel()
		})
	}
}

type recoveryWorkerContext struct {
	context.Context
	deadline   context.Context
	authorized context.Context
}

func (c recoveryWorkerContext) Deadline() (time.Time, bool) { return c.deadline.Deadline() }
func (c recoveryWorkerContext) Err() error {
	if c.Context.Err() != nil && c.authorized.Err() != nil {
		return c.authorized.Err()
	}
	return c.Context.Err()
}

func (l *recoveryOrdinaryLifecycle) bindShutdown(ctx context.Context) context.Context {
	l.shutdownOnce.Do(func() {
		deadline := time.Now().Add(l.budget)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		l.shutdown, l.cancelShutdown = context.WithDeadline(context.WithoutCancel(ctx), deadline)
		l.workers[2].(interface{ BeginShutdown(context.Context) }).BeginShutdown(l.shutdown)
	})
	return l.shutdown
}

func (l *recoveryOrdinaryLifecycle) StopAndJoin(ctx context.Context) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	l.mu.Lock()
	initial := l.initial
	l.mu.Unlock()
	// Cold recovery drains root-owned calls before any ordinary loop has existed.
	if initial == nil {
		l.coldOnce.Do(func() { l.coldErr = l.joinOwners(ctx) })
		return l.coldErr
	}
	l.stopOnce.Do(func() {
		shared := l.bindShutdown(ctx)
		defer l.cancelShutdown()
		for _, w := range l.workers {
			w.Stop()
		}
		l.cancel()
		<-initial
		result := lifecycleRun{priority: lifecycleUnsetPriority}
		for i, w := range l.workers {
			err := w.Wait(shared)
			<-w.Done()
			if err != nil {
				result.add(fmt.Errorf("ordinary worker %d join: %w", i, err), ordinaryWorkerPriority(i))
			}
		}
		result.add(l.joinOwners(shared), lifecycleCleanupPriority)
		result.add(shared.Err(), lifecycleCleanupPriority)
		l.stopErr = result.result.err()
		close(l.stopped)
	})
	<-l.stopped
	return l.stopErr
}
func ordinaryWorkerPriority(index int) int {
	return []int{lifecycleDeletionPriority, lifecyclePasswordResetPriority, lifecycleNoticePriority}[index]
}

func (l *recoveryOrdinaryLifecycle) joinOwners(ctx context.Context) error {
	var failures []error
	for _, join := range l.joins {
		failures = append(failures, join(ctx))
	}
	return errors.Join(failures...)
}

var _ recovery.OrdinaryLifecycle = (*recoveryOrdinaryLifecycle)(nil)
