package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"han/internal/recovery"
)

// restoreDeploymentExclusive requires an explicitly supplied, external facility.
// No operating-system backend or production fallback is registered here.
type restoreDeploymentExclusive struct {
	path    string
	acquire func(context.Context, string) (recovery.ExclusiveLease, error)
}

func (e *restoreDeploymentExclusive) Acquire(ctx context.Context) (recovery.ExclusiveLease, error) {
	if err := restoreContext(ctx); err != nil {
		return nil, err
	}
	if e == nil || e.acquire == nil {
		return nil, recovery.ErrAdmissionUnsupported
	}
	if e.path == "" || !filepath.IsAbs(e.path) || filepath.Clean(e.path) != e.path || strings.TrimSpace(e.path) != e.path {
		return nil, recovery.ErrAdmissionBinding
	}
	lease, err := e.acquire(ctx, e.path)
	if lease == nil {
		return nil, errors.Join(recovery.ErrAdmissionClosed, err)
	}
	owned := &restoreExclusiveLease{actual: lease}
	if err != nil {
		return owned, errors.Join(recovery.ErrAdmissionClosed, err)
	}
	if err = owned.Check(ctx); err != nil {
		return owned, err
	}
	return owned, nil
}

type restoreExclusiveLease struct {
	mu       sync.Mutex
	actual   recovery.ExclusiveLease
	closed   bool
	closeErr error
}

func (l *restoreExclusiveLease) Check(ctx context.Context) error {
	if err := restoreContext(ctx); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return recovery.ErrAdmissionClosed
	}
	return errors.Join(l.actual.Check(ctx), ctx.Err())
}

func (l *restoreExclusiveLease) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		l.closeErr = l.actual.Close()
	}
	return l.closeErr
}
