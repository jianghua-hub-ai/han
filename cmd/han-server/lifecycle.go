package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"
)

const (
	lifecycleBusinessPriority = iota
	lifecycleMetricsPriority
	lifecyclePasswordResetPriority
	lifecycleDeletionPriority
	lifecycleNoticePriority
	lifecycleStopPriority
	lifecycleCleanupPriority
	lifecycleUnsetPriority
)

type lifecycleServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
	Close() error
}

type lifecycleWorker interface {
	Reconcile(context.Context) error
	Start(context.Context)
	Stop()
	Wait(context.Context) error
	Done() <-chan struct{}
	Err() error
}

type lifecycleConfig struct {
	// joinWork joins root-owned HTTP/maintenance/transaction/child calls.
	joinWork            func(context.Context) error
	additionalServers   []*lifecycleServerState
	businessServer      lifecycleServer
	metricsServer       lifecycleServer
	businessListener    net.Listener
	metricsListener     net.Listener
	articleWorker       lifecycleWorker
	passwordResetWorker lifecycleWorker
	noticeWorker        lifecycleWorker
	closeArticle        func() error
	closePool           func() error
	shutdownBudget      time.Duration
	signals             <-chan os.Signal
	deadlineContext     func(context.Context, time.Duration) (context.Context, context.CancelFunc)
}

type lifecycleServeResult struct {
	name string
	err  error
}
type lifecycleOperationResult struct {
	name   string
	err    error
	worker lifecycleWorker
}

func runLifecycle(ctx context.Context, config lifecycleConfig) error {
	if err := validateLifecycleConfig(ctx, config); err != nil {
		return err
	}
	// The coordinator binds the shared shutdown deadline before propagating a parent stop.
	work, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	run := lifecycleRun{ctx: ctx, work: work, cancelWork: cancel, config: config,
		serveResults: make(chan lifecycleServeResult, 2+len(config.additionalServers)), initialResults: make(chan error, 1), priority: lifecycleUnsetPriority}
	run.workers = []*lifecycleWorkerState{
		{name: "Article deletion", worker: config.articleWorker, priority: lifecycleDeletionPriority},
		{name: "PasswordReset recovery", worker: config.passwordResetWorker, priority: lifecyclePasswordResetPriority},
		{name: "Article notice", worker: config.noticeWorker, priority: lifecycleNoticePriority},
	}
	run.servers = []*lifecycleServerState{
		{name: "business", server: config.businessServer, listener: config.businessListener},
		{name: "metrics", server: config.metricsServer, listener: config.metricsListener},
	}
	run.servers = append(run.servers, config.additionalServers...)
	run.start()
	if !run.stopping {
		run.stop(waitForLifecycleTrigger(ctx, config, run.serveResults))
	}
	return run.shutdown()
}

type lifecycleWorkerState struct {
	name     string
	worker   lifecycleWorker
	priority int
	started  bool
}

type lifecycleServerState struct {
	name     string
	server   lifecycleServer
	listener net.Listener
	served   bool
}

type lifecycleRun struct {
	ctx, work, shutdownCtx     context.Context
	cancelWork, cancelShutdown context.CancelFunc
	config                     lifecycleConfig
	workers                    []*lifecycleWorkerState
	servers                    []*lifecycleServerState
	serveResults               chan lifecycleServeResult
	initialResults             chan error
	initial                    *lifecycleWorkerState
	serveRemaining             int
	stopping                   bool
	result                     lifecycleErrors
	priority                   int
}

func (run *lifecycleRun) start() {
	for _, item := range run.workers {
		if run.checkStop() {
			return
		}
		if item.worker != nil && !run.reconcile(item) {
			return
		}
	}
	for _, item := range run.workers {
		if run.checkStop() {
			return
		}
		if item.worker != nil {
			item.worker.Start(run.work)
			item.started = true
			if !run.awaitStarted(item) {
				return
			}
		}
	}
	for _, item := range run.servers {
		if run.checkStop() {
			return
		}
		item.served = true
		run.serveRemaining++
		go serveLifecycleServer(item.name, item.server, item.listener, run.serveResults)
	}
}

func (run *lifecycleRun) reconcile(item *lifecycleWorkerState) bool {
	run.initial = item
	go func() { run.initialResults <- item.worker.Reconcile(run.work) }()
	select {
	case err := <-run.initialResults:
		run.initial = nil
		run.checkStop()
		if err != nil && !run.stopping {
			run.stop(lifecycleTrigger{})
		}
		run.recordWorkerError(item, "initial reconciliation", err)
		return !run.stopping
	case <-run.ctx.Done():
		run.stop(lifecycleContextTrigger(run.ctx))
	case signal := <-run.config.signals:
		run.stop(lifecycleSignalTrigger(signal))
	}
	return false
}

func (run *lifecycleRun) checkStop() bool {
	if run.stopping {
		return true
	}
	select {
	case <-run.ctx.Done():
		run.stop(lifecycleContextTrigger(run.ctx))
		return true
	default:
	}
	select {
	case signal := <-run.config.signals:
		run.stop(lifecycleSignalTrigger(signal))
		return true
	default:
	}
	if trigger, ok := readyLifecycleServerTrigger(run.serveResults); ok {
		run.stop(trigger)
		return true
	}
	for _, item := range run.workers {
		if !item.started {
			continue
		}
		select {
		case <-item.worker.Done():
			run.stop(lifecycleWorkerTrigger(item.name, item.worker, item.priority))
			return true
		default:
		}
	}
	return false
}

func (run *lifecycleRun) stop(trigger lifecycleTrigger) {
	if run.stopping {
		return
	}
	run.stopping = true
	startedAt := time.Now()
	if run.config.deadlineContext != nil {
		run.shutdownCtx, run.cancelShutdown = run.config.deadlineContext(context.WithoutCancel(run.ctx), run.config.shutdownBudget)
	} else {
		run.shutdownCtx, run.cancelShutdown = context.WithDeadline(context.WithoutCancel(run.ctx), startedAt.Add(run.config.shutdownBudget))
	}
	if run.config.noticeWorker != nil {
		run.config.noticeWorker.(interface{ BeginShutdown(context.Context) }).BeginShutdown(run.shutdownCtx)
	}
	run.cancelWork()
	for _, item := range run.workers {
		if item.worker != nil {
			item.worker.Stop()
		}
	}
	run.serveRemaining -= trigger.serveCount
	run.add(trigger.err, trigger.priority)
	for _, additional := range trigger.additional {
		run.add(additional, trigger.priority)
	}
	if run.ctx.Err() != nil {
		run.add(lifecycleContextTrigger(run.ctx).err, lifecycleStopPriority)
	}
}

func (run *lifecycleRun) add(err error, priority int) {
	if err == nil {
		return
	}
	if run.result.primary != nil && priority < run.priority {
		run.result.additional = append(run.result.additional, run.result.primary)
		run.result.primary = err
		run.priority = priority
		return
	}
	run.result.add(err)
	if priority < run.priority {
		run.priority = priority
	}
}

func (run *lifecycleRun) recordWorkerError(item *lifecycleWorkerState, stage string, err error) {
	if authoritative := item.worker.Err(); authoritative != nil {
		run.add(fmt.Errorf("%s %s failed: %w", item.name, stage, authoritative), item.priority)
	}
	if err == nil {
		return
	}
	if item.worker.Err() == nil && run.work.Err() != nil && pureLifecycleCancellation(err) {
		return
	}
	run.add(fmt.Errorf("%s %s failed: %w", item.name, stage, err), item.priority)
}

// Only cancellation-only chains without an authoritative worker error are normal stop results.
func pureLifecycleCancellation(err error) bool {
	if err == context.Canceled {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !pureLifecycleCancellation(cause) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return pureLifecycleCancellation(wrapped.Unwrap())
	}
	return false
}

func (run *lifecycleRun) joinInitial() {
	if run.initial == nil {
		return
	}
	var err error
	select {
	case err = <-run.initialResults:
	case <-run.shutdownCtx.Done():
		run.add(fmt.Errorf("initial %s join deadline: %w", run.initial.name, run.shutdownCtx.Err()), lifecycleCleanupPriority)
		err = <-run.initialResults
	}
	run.recordWorkerError(run.initial, "initial reconciliation", err)
	run.initial = nil
}

func (run *lifecycleRun) shutdown() error {
	defer run.cancelShutdown()
	results := make(chan lifecycleOperationResult, len(run.servers)+len(run.workers))
	var group sync.WaitGroup
	for _, item := range run.servers {
		if !item.served {
			continue
		}
		group.Add(1)
		go shutdownLifecycleServer(item.name, item.server, run.shutdownCtx, results, &group)
	}
	for _, item := range run.workers {
		if item.worker == nil || (!item.started && item.priority != lifecycleNoticePriority) {
			continue
		}
		group.Add(1)
		go waitLifecycleWorker(item.name, item.worker, run.shutdownCtx, results, &group)
	}
	run.joinInitial()
	group.Wait()
	close(results)
	failed := run.collectShutdown(results)
	run.finishServers(failed)
	if run.config.joinWork != nil {
		run.add(run.config.joinWork(run.shutdownCtx), lifecycleCleanupPriority)
	}
	for _, item := range run.servers {
		run.add(closeLifecycleListener(item.name, item.listener), lifecycleCleanupPriority)
	}
	if run.config.closeArticle != nil {
		run.add(wrapCleanupError("close Article resources", run.config.closeArticle()), lifecycleCleanupPriority)
	}
	run.add(wrapCleanupError("close database pool", run.config.closePool()), lifecycleCleanupPriority)
	return run.result.err()
}

func (run *lifecycleRun) collectShutdown(results <-chan lifecycleOperationResult) bool {
	failed := false
	for operation := range results {
		if operation.err == nil {
			continue
		}
		if operation.worker != nil {
			for _, item := range run.workers {
				if operation.name == "wait for "+item.name+" worker" {
					run.recordWorkerError(item, "worker join", operation.err)
				}
			}
			continue
		}
		failed = true
		run.add(fmt.Errorf("%s server shutdown failed: %w", operation.name, operation.err), lifecycleCleanupPriority)
	}
	for _, item := range run.workers {
		if item.worker != nil && (item.started || item.priority == lifecycleNoticePriority) {
			run.recordWorkerError(item, "worker join", nil)
		}
	}
	if err := run.shutdownCtx.Err(); err != nil {
		failed = true
		run.add(fmt.Errorf("shared shutdown deadline failed: %w", err), lifecycleCleanupPriority)
	}
	return failed
}

func (run *lifecycleRun) finishServers(failed bool) {
	if failed {
		for _, item := range run.servers {
			if item.served {
				run.add(wrapCleanupError(item.name+" server close failed", item.server.Close()), lifecycleCleanupPriority)
			}
		}
	}
	for range run.serveRemaining {
		value := <-run.serveResults
		if value.err != nil && !errors.Is(value.err, http.ErrServerClosed) {
			priority := lifecycleMetricsPriority
			if value.name == "business" {
				priority = lifecycleBusinessPriority
			}
			run.add(fmt.Errorf("%s server failed during shutdown: %w", value.name, value.err), priority)
		}
	}
}

func lifecycleContextTrigger(ctx context.Context) lifecycleTrigger {
	return lifecycleTrigger{err: fmt.Errorf("lifecycle context stopped: %w", ctx.Err()), priority: lifecycleStopPriority}
}

func lifecycleSignalTrigger(signal os.Signal) lifecycleTrigger {
	trigger := lifecycleTrigger{priority: lifecycleStopPriority}
	if signal == nil {
		trigger.err = errors.New("shutdown signal source closed")
	}
	return trigger
}

func noticeLifecycleIncomplete(worker lifecycleWorker) bool {
	if worker == nil {
		return true
	}
	if value := reflect.ValueOf(worker); value.Kind() == reflect.Pointer && value.IsNil() {
		return true
	}
	_, ok := worker.(interface{ BeginShutdown(context.Context) })
	return !ok
}

type lifecycleTrigger struct {
	err        error
	serveCount int
	priority   int
	additional []error
}

func waitForLifecycleTrigger(ctx context.Context, config lifecycleConfig, serveResults <-chan lifecycleServeResult) lifecycleTrigger {
	if trigger, ok := readyLifecycleServerTrigger(serveResults); ok {
		return trigger
	}
	select {
	case <-workerDone(config.passwordResetWorker):
		return prioritizeLifecycleTrigger(lifecycleWorkerTrigger("PasswordReset recovery", config.passwordResetWorker, lifecyclePasswordResetPriority), config, serveResults)
	default:
	}
	select {
	case <-workerDone(config.articleWorker):
		return prioritizeLifecycleTrigger(lifecycleWorkerTrigger("Article deletion", config.articleWorker, lifecycleDeletionPriority), config, serveResults)
	default:
	}
	var selected lifecycleTrigger
	select {
	case value := <-serveResults:
		selected = lifecycleServerTriggerWithReady(value, serveResults)
	case <-workerDone(config.passwordResetWorker):
		selected = lifecycleWorkerTrigger("PasswordReset recovery", config.passwordResetWorker, lifecyclePasswordResetPriority)
	case <-workerDone(config.articleWorker):
		selected = lifecycleWorkerTrigger("Article deletion", config.articleWorker, lifecycleDeletionPriority)
	case <-workerDone(config.noticeWorker):
		selected = lifecycleWorkerTrigger("Article notice", config.noticeWorker, lifecycleNoticePriority)
	case <-ctx.Done():
		selected = lifecycleContextTrigger(ctx)
	case signal := <-config.signals:
		selected = lifecycleTrigger{priority: lifecycleStopPriority}
		if signal == nil {
			selected.err = errors.New("shutdown signal source closed")
		}
	}
	return prioritizeLifecycleTrigger(selected, config, serveResults)
}

func prioritizeLifecycleTrigger(selected lifecycleTrigger, config lifecycleConfig, serveResults <-chan lifecycleServeResult) lifecycleTrigger {
	if selected.priority > lifecycleMetricsPriority {
		if server, ok := readyLifecycleServerTrigger(serveResults); ok {
			return overrideLifecycleTrigger(server, selected)
		}
	}
	if selected.priority > lifecyclePasswordResetPriority {
		select {
		case <-workerDone(config.passwordResetWorker):
			return overrideLifecycleTrigger(lifecycleWorkerTrigger("PasswordReset recovery", config.passwordResetWorker, lifecyclePasswordResetPriority), selected)
		default:
		}
	}
	if selected.priority > lifecycleDeletionPriority {
		select {
		case <-workerDone(config.articleWorker):
			return overrideLifecycleTrigger(lifecycleWorkerTrigger("Article deletion", config.articleWorker, lifecycleDeletionPriority), selected)
		default:
		}
	}
	if selected.priority > lifecycleNoticePriority {
		select {
		case <-workerDone(config.noticeWorker):
			return overrideLifecycleTrigger(lifecycleWorkerTrigger("Article notice", config.noticeWorker, lifecycleNoticePriority), selected)
		default:
		}
	}
	return selected
}

func overrideLifecycleTrigger(primary, lower lifecycleTrigger) lifecycleTrigger {
	primary.serveCount += lower.serveCount
	primary.additional = append(primary.additional, lower.additional...)
	if lower.err != nil {
		primary.additional = append(primary.additional, lower.err)
	}
	return primary
}

func readyLifecycleServerTrigger(results <-chan lifecycleServeResult) (lifecycleTrigger, bool) {
	select {
	case value := <-results:
		return lifecycleServerTriggerWithReady(value, results), true
	default:
		return lifecycleTrigger{}, false
	}
}

func lifecycleServerTriggerWithReady(first lifecycleServeResult, results <-chan lifecycleServeResult) lifecycleTrigger {
	values := []lifecycleServeResult{first}
	select {
	case second := <-results:
		values = append(values, second)
	default:
	}
	return chooseLifecycleServerTrigger(values)
}

func chooseLifecycleServerTrigger(values []lifecycleServeResult) lifecycleTrigger {
	primaryIndex := 0
	for index, value := range values {
		if value.name == "business" {
			primaryIndex = index
			break
		}
	}
	primary := values[primaryIndex]
	if primary.err == nil {
		primary.err = errors.New("server stopped without an error")
	}
	priority := lifecycleMetricsPriority
	if primary.name == "business" {
		priority = lifecycleBusinessPriority
	}
	trigger := lifecycleTrigger{err: fmt.Errorf("%s server failed: %w", primary.name, primary.err), serveCount: len(values), priority: priority}
	for index, value := range values {
		if index == primaryIndex {
			continue
		}
		if value.err == nil {
			value.err = errors.New("server stopped without an error")
		}
		trigger.additional = append(trigger.additional, fmt.Errorf("%s server failed: %w", value.name, value.err))
	}
	return trigger
}

func lifecycleWorkerTrigger(name string, worker lifecycleWorker, priority int) lifecycleTrigger {
	workerErr := worker.Err()
	if workerErr == nil {
		workerErr = fmt.Errorf("%s worker stopped without an error", name)
	}
	return lifecycleTrigger{err: fmt.Errorf("%s worker failed: %w", name, workerErr), priority: priority}
}

func closeLifecycleStartupResources(config lifecycleConfig, result *lifecycleErrors) {
	if err := closeLifecycleListener("business", config.businessListener); err != nil {
		result.add(err)
	}
	if err := closeLifecycleListener("metrics", config.metricsListener); err != nil {
		result.add(err)
	}
	if config.closeArticle != nil {
		if err := config.closeArticle(); err != nil {
			result.add(fmt.Errorf("close Article resources: %w", err))
		}
	}
	if err := config.closePool(); err != nil {
		result.add(fmt.Errorf("close database pool: %w", err))
	}
}

func waitLifecycleWorker(name string, worker lifecycleWorker, ctx context.Context, results chan<- lifecycleOperationResult, group *sync.WaitGroup) {
	defer group.Done()
	err := worker.Wait(ctx)
	<-worker.Done()
	results <- lifecycleOperationResult{name: "wait for " + name + " worker", err: err, worker: worker}
}

func validateLifecycleConfig(ctx context.Context, config lifecycleConfig) error {
	if ctx == nil {
		return errors.New("lifecycle context is required")
	}
	if config.businessServer == nil || config.metricsServer == nil || config.businessListener == nil || config.metricsListener == nil || config.closePool == nil || config.shutdownBudget <= 0 || config.signals == nil {
		return errors.New("lifecycle dependencies are incomplete")
	}
	for _, server := range config.additionalServers {
		if server == nil || server.name == "" || server.server == nil || server.listener == nil {
			return errors.New("additional lifecycle server is incomplete")
		}
	}
	if (config.articleWorker == nil) != (config.closeArticle == nil) {
		return errors.New("Article lifecycle dependencies are incomplete")
	}
	if config.noticeWorker != nil && noticeLifecycleIncomplete(config.noticeWorker) {
		return errors.New("notice shutdown capability is required")
	}
	if config.passwordResetWorker != nil && config.articleWorker == nil {
		return errors.New("password reset lifecycle requires Article lifecycle")
	}
	return nil
}

func workerDone(worker lifecycleWorker) <-chan struct{} {
	if worker == nil {
		return nil
	}
	return worker.Done()
}

func serveLifecycleServer(
	name string,
	server lifecycleServer,
	listener net.Listener,
	results chan<- lifecycleServeResult,
) {
	results <- lifecycleServeResult{name: name, err: server.Serve(listener)}
}

func shutdownLifecycleServer(
	name string,
	server lifecycleServer,
	ctx context.Context,
	results chan<- lifecycleOperationResult,
	group *sync.WaitGroup,
) {
	defer group.Done()
	results <- lifecycleOperationResult{name: name, err: server.Shutdown(ctx)}
}

func closeLifecycleServer(
	name string,
	server lifecycleServer,
	results chan<- lifecycleOperationResult,
	group *sync.WaitGroup,
) {
	defer group.Done()
	results <- lifecycleOperationResult{name: name, err: server.Close()}
}

func closeLifecycleListener(name string, listener net.Listener) error {
	err := listener.Close()
	if err == nil || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return fmt.Errorf("close %s listener: %w", name, err)
}

type lifecycleErrors struct {
	primary    error
	additional []error
}

func (errorsSet *lifecycleErrors) addPrimary(err error) {
	if err == nil {
		return
	}
	if errorsSet.primary == nil {
		errorsSet.primary = err
		return
	}
	errorsSet.additional = append(errorsSet.additional, err)
}

func (errorsSet *lifecycleErrors) add(err error) {
	if err == nil {
		return
	}
	if errorsSet.primary == nil {
		errorsSet.primary = err
		return
	}
	errorsSet.additional = append(errorsSet.additional, err)
}

func (errorsSet lifecycleErrors) err() error {
	if errorsSet.primary == nil {
		return nil
	}
	return lifecycleError{
		primary:    errorsSet.primary,
		additional: append([]error(nil), errorsSet.additional...),
	}
}

type lifecycleError struct {
	primary    error
	additional []error
}

func (err lifecycleError) Error() string {
	if len(err.additional) == 0 {
		return err.primary.Error()
	}
	return fmt.Sprintf("%v (additional lifecycle errors: %v)", err.primary, errors.Join(err.additional...))
}

func (err lifecycleError) Unwrap() []error {
	causes := make([]error, 0, 1+len(err.additional))
	causes = append(causes, err.primary)
	return append(causes, err.additional...)
}

func firstLifecycleError(err error) error {
	var lifecycleErr lifecycleError
	if errors.As(err, &lifecycleErr) {
		return lifecycleErr.primary
	}
	return err
}

func lifecycleExitCode(err error) int {
	if err == nil {
		return 0
	}
	return 1
}

// awaitStarted never interprets a returned Start call as a loop handshake.
func (run *lifecycleRun) awaitStarted(item *lifecycleWorkerState) bool {
	worker, ok := item.worker.(interface{ Started() <-chan struct{} })
	if !ok {
		return !run.checkStop()
	} // Existing non-runtime lifecycle consumers.
	select {
	case <-worker.Started():
		if err := item.worker.Err(); err != nil {
			run.stop(lifecycleWorkerTrigger(item.name, item.worker, item.priority))
			return false
		}
		return !run.checkStop()
	case <-item.worker.Done():
		run.stop(lifecycleWorkerTrigger(item.name, item.worker, item.priority))
	case <-run.ctx.Done():
		run.stop(lifecycleContextTrigger(run.ctx))
	case signal := <-run.config.signals:
		run.stop(lifecycleSignalTrigger(signal))
	case value := <-run.serveResults:
		run.stop(lifecycleServerTriggerWithReady(value, run.serveResults))
	}
	return false
}
