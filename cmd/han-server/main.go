package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/mail"
	"net/netip"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/privacy"
	"han/internal/recovery"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	databaseDSNEnvironment                         = "HAN_DATABASE_DSN"
	httpAddressEnvironment                         = "HAN_HTTP_ADDR"
	articleTransferNoticeMaxAttemptsEnvironment    = "HAN_ARTICLE_TRANSFER_NOTICE_MAX_ATTEMPTS"
	noticePollIntervalEnvironment                  = "HAN_ARTICLE_TRANSFER_NOTICE_POLL_INTERVAL"
	noticeBatchLimitEnvironment                    = "HAN_ARTICLE_TRANSFER_NOTICE_BATCH_LIMIT"
	noticeLeaseDurationEnvironment                 = "HAN_ARTICLE_TRANSFER_NOTICE_LEASE_DURATION"
	noticeDBOperationBudgetEnvironment             = "HAN_ARTICLE_TRANSFER_NOTICE_DB_OPERATION_BUDGET"
	noticeBackoffInitialEnvironment                = "HAN_ARTICLE_TRANSFER_NOTICE_BACKOFF_INITIAL"
	noticeBackoffMaxEnvironment                    = "HAN_ARTICLE_TRANSFER_NOTICE_BACKOFF_MAX"
	metricsAddressEnvironment                      = "HAN_METRICS_ADDR"
	loginObservationBudgetEnvironment              = "HAN_LOGIN_OBSERVATION_BUDGET"
	originObservationBudgetEnvironment             = "HAN_ORIGIN_OBSERVATION_BUDGET"
	failureAuditBudgetEnvironment                  = "HAN_LOGIN_FAILURE_AUDIT_BUDGET"
	shutdownBudgetEnvironment                      = "HAN_SHUTDOWN_BUDGET"
	smtpOperationBudgetEnvironment                 = "HAN_SMTP_OPERATION_BUDGET"
	passwordResetWorkerLeaseDurationEnvironment    = "HAN_PASSWORD_RESET_WORKER_LEASE_DURATION"
	passwordResetWorkerRetryBackoffEnvironment     = "HAN_PASSWORD_RESET_WORKER_RETRY_BACKOFF"
	passwordResetWorkerMaxRetriesEnvironment       = "HAN_PASSWORD_RESET_WORKER_MAX_RETRIES"
	passwordResetWorkerClaimBatchLimitEnvironment  = "HAN_PASSWORD_RESET_WORKER_CLAIM_BATCH_LIMIT"
	passwordResetWorkerConnectionBudgetEnvironment = "HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET"
	articleQueryDefaultLimitEnvironment            = "HAN_ARTICLE_QUERY_DEFAULT_LIMIT"
	articleQueryMaxLimitEnvironment                = "HAN_ARTICLE_QUERY_MAX_LIMIT"
	articleQueryCandidateBatchSizeEnvironment      = "HAN_ARTICLE_QUERY_CANDIDATE_BATCH_SIZE"
	loginDurationBucketsEnvironment                = "HAN_LOGIN_DURATION_BUCKETS_SECONDS"
	loginMaxBodyBytesEnvironment                   = "HAN_LOGIN_MAX_BODY_BYTES"
	argon2MemoryEnvironment                        = "HAN_ARGON2_MEMORY_KIB"
	argon2IterationsEnvironment                    = "HAN_ARGON2_ITERATIONS"
	argon2ParallelismEnvironment                   = "HAN_ARGON2_PARALLELISM"
	argon2SaltBytesEnvironment                     = "HAN_ARGON2_SALT_BYTES"
	argon2KeyBytesEnvironment                      = "HAN_ARGON2_KEY_BYTES"
	argon2ProfileIDEnvironment                     = "HAN_ARGON2_PROFILE_ID"
	argon2MaxConcurrencyEnvironment                = "HAN_ARGON2_MAX_CONCURRENCY"
	loginBucketKeyVersionEnvironment               = "HAN_LOGIN_BUCKET_CURRENT_KEY_VERSION"
	loginBucketKeyPrefix                           = "HAN_LOGIN_BUCKET_HMAC_KEY_"
	loginAbuseWindowEnvironment                    = "HAN_LOGIN_ABUSE_WINDOW"
	loginAbuseThresholdEnvironment                 = "HAN_LOGIN_ABUSE_FAILURE_THRESHOLD"
	loginAbuseBlockEnvironment                     = "HAN_LOGIN_ABUSE_BLOCK_DURATION"
	loginAbuseActionEnvironment                    = "HAN_LOGIN_ABUSE_SUCCESS_ACTION"
	loginAbuseDecayEnvironment                     = "HAN_LOGIN_ABUSE_SUCCESS_DECAY"
	sessionTokenEntropyEnvironment                 = "HAN_SESSION_TOKEN_ENTROPY"
	csrfTokenEntropyEnvironment                    = "HAN_CSRF_TOKEN_ENTROPY"
	sessionAbsoluteTTLEnvironment                  = "HAN_SESSION_ABSOLUTE_TTL"
	sessionIdleTTLEnvironment                      = "HAN_SESSION_IDLE_TTL"
	sourceKeyVersionEnvironment                    = "HAN_SOURCE_FINGERPRINT_CURRENT_KEY_VERSION"
	sourceKeyPrefix                                = "HAN_SOURCE_FINGERPRINT_HMAC_KEY_"
	recoveryModeEnvironment                        = "HAN_RECOVERY_MODE"
	administratorNoticeVersionEnvironment          = "HAN_ADMINISTRATOR_INFORMATION_NOTICE_VERSION"
	administratorNoticeEffectiveAtEnvironment      = "HAN_ADMINISTRATOR_INFORMATION_NOTICE_EFFECTIVE_AT"
	administratorNoticeContentEnvironment          = "HAN_ADMINISTRATOR_INFORMATION_NOTICE_CONTENT"
	administratorNoticeSHA256Environment           = "HAN_ADMINISTRATOR_INFORMATION_NOTICE_SHA256"
	idempotencyCurrentKeyVersionEnvironment        = "HAN_IDEMPOTENCY_CURRENT_KEY_VERSION"
	idempotencyRetainedKeyVersionsEnvironment      = "HAN_IDEMPOTENCY_RETAINED_KEY_VERSIONS"
	idempotencyKeyPrefix                           = "HAN_IDEMPOTENCY_HMAC_KEY_"
	idempotencyLockDurationEnvironment             = "HAN_IDEMPOTENCY_LOCK_DURATION"
	idempotencyRecordTTLEnvironment                = "HAN_IDEMPOTENCY_RECORD_TTL"
	smtpPortEnvironment                            = "HAN_SMTP_PORT"
	smtpFromEnvironment                            = "HAN_SMTP_FROM"
	smtpAuthModeEnvironment                        = "HAN_SMTP_AUTH_MODE"
	smtpUsernameEnvironment                        = "HAN_SMTP_USERNAME"
	smtpPasswordEnvironment                        = "HAN_SMTP_PASSWORD"
	initialAdminPasswordEnvironment                = "HAN_INITIAL_ADMIN_PASSWORD"
	articleDeletionJournalRootEnvironment          = "HAN_ARTICLE_DELETION_JOURNAL_ROOT"
	articleDeletionJournalFileModeEnvironment      = "HAN_ARTICLE_DELETION_JOURNAL_FILE_MODE"
	articleDeletionCurrentKeyVersionEnvironment    = "HAN_ARTICLE_DELETION_KEY_CURRENT_VERSION"
	articleDeletionRetainedKeyVersionsEnvironment  = "HAN_ARTICLE_DELETION_RETAINED_KEY_VERSIONS"
	articleDeletionKeyPrefix                       = "HAN_ARTICLE_DELETION_HMAC_KEY_"
	articleBackupInventoryPathEnvironment          = "HAN_ARTICLE_BACKUP_INVENTORY_PATH"
	articleFinalizerWorkerIDEnvironment            = "HAN_ARTICLE_FINALIZER_WORKER_ID"
	articleFinalizerPollIntervalEnvironment        = "HAN_ARTICLE_FINALIZER_POLL_INTERVAL"
	articleFinalizerBatchSizeEnvironment           = "HAN_ARTICLE_FINALIZER_BATCH_SIZE"
	articleFinalizerLeaseDurationEnvironment       = "HAN_ARTICLE_FINALIZER_LEASE_DURATION"
	articleFinalizerOperationBudgetEnvironment     = "HAN_ARTICLE_FINALIZER_OPERATION_BUDGET"
	articleFinalizerBackoffInitialEnvironment      = "HAN_ARTICLE_FINALIZER_BACKOFF_INITIAL"
	articleFinalizerBackoffMaxEnvironment          = "HAN_ARTICLE_FINALIZER_BACKOFF_MAX"
	articleFinalizerMaxAttemptsEnvironment         = "HAN_ARTICLE_FINALIZER_MAX_ATTEMPTS_PER_WAKE"
	articleFinalizerShutdownBudgetEnvironment      = "HAN_ARTICLE_FINALIZER_SHUTDOWN_BUDGET"
	passwordResetTemporaryPasswordCharacters       = 32
	migrationsDirectory                            = "migrations"
	startupFailureMessage                          = "han-server startup failed"
	serviceName                                    = "han-server"
	noticeLogModule                                = "article"
	noticeWarningEvent                             = "article.transfer_notice.warning"
)

var errArticleDeletionJournalUnavailable = errors.New(
	"Article deletion journal unavailable",
)

type startupDependencies struct {
	getenv                     func(string) string
	lookupEnvironment          platform.EnvironmentLookup
	parseDatabase              func(string) (*pgxpool.Config, error)
	connect                    func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error)
	ping                       func(context.Context, *pgxpool.Pool) error
	migrate                    func(context.Context, *pgxpool.Pool) error
	listen                     func(string, string) (net.Listener, error)
	serve                      func(net.Listener) error
	closePool                  func(*pgxpool.Pool)
	composeBusiness            func(*pgxpool.Pool, businessComposition) (http.Handler, error)
	newServer                  func(string, http.Handler) lifecycleServer
	signals                    <-chan os.Signal
	coordinate                 func(context.Context, lifecycleConfig) error
	logOutput                  io.Writer
	prepareBusiness            func(startupConfig) (*preparedLoginDependencies, error)
	prepareArticle             func(articleDeletionStartupConfig) (*preparedArticleDeletion, error)
	openArticle                func(*preparedArticleDeletion) (articleDeletionJournalOwner, error)
	requirePasswordResetWorker bool
	requireNoticeWorker        bool
	recoveryCommand            *recoveryCommand
	composeRecoveryControl     func(*pgxpool.Pool, recoveryControlOptions, *recoveryControlWiring) (http.Handler, error)
	bootstrapInitialAdmin      func(
		context.Context,
		*pgxpool.Pool,
		*preparedLoginDependencies,
	) error
}

func main() {
	command, recoverySelected, commandErr := selectRecoveryCommand(os.Args[1:])
	if recoverySelected {
		if commandErr != nil {
			fmt.Fprintln(os.Stderr, startupFailureMessage)
			os.Exit(2)
		}
		dependencies := productionDependencies()
		dependencies.recoveryCommand = &command
		err := run(context.Background(), dependencies)
		if lifecycleExitCode(err) != 0 {
			fmt.Fprintln(os.Stderr, startupFailureMessage)
			os.Exit(1)
		}
		return
	}
	maintenance, routeErr := selectMaintenanceMode(os.Args[1:])
	if maintenance {
		code := maintenanceMain(routeErr, os.Stdin, os.Stdout, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
		return
	}
	err := run(context.Background(), productionDependencies())
	if lifecycleExitCode(err) != 0 {
		fmt.Fprintln(os.Stderr, startupFailureMessage)
		os.Exit(1)
	}
}

func run(ctx context.Context, dependencies startupDependencies) (err error) {
	if dependencies.composeBusiness != nil {
		return runComposed(ctx, dependencies)
	}
	return runLegacy(ctx, dependencies)
}

func runLegacy(ctx context.Context, dependencies startupDependencies) (err error) {
	dsn := strings.TrimSpace(dependencies.getenv(databaseDSNEnvironment))
	httpAddress := dependencies.getenv(httpAddressEnvironment)
	if dsn == "" {
		return errors.New("database DSN is required")
	}
	if _, err := platform.LoadDeploymentConfig(dependencies.lookupEnvironment); err != nil {
		return fmt.Errorf("validate deployment security configuration: %w", err)
	}

	databaseConfig, err := dependencies.parseDatabase(dsn)
	if err != nil {
		return fmt.Errorf("validate database configuration: %w", err)
	}
	if err := validateHTTPAddress(httpAddress); err != nil {
		return fmt.Errorf("validate HTTP listen address: %w", err)
	}

	pool, err := dependencies.connect(ctx, databaseConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer dependencies.closePool(pool)

	if err := dependencies.ping(ctx, pool); err != nil {
		return fmt.Errorf("validate database connection: %w", err)
	}
	if err := dependencies.migrate(ctx, pool); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}

	listener, err := dependencies.listen("tcp", httpAddress)
	if err != nil {
		return fmt.Errorf("listen for HTTP requests: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeHTTPListener(listener))
	}()

	if err := dependencies.serve(listener); err != nil {
		return fmt.Errorf("serve HTTP requests: %w", err)
	}
	return errors.New("HTTP service stopped")
}

type startupConfig struct {
	noticeDispatcher                 article.NoticeDispatcherConfig
	articleTransferNoticeMaxAttempts int64
	databaseDSN                      string
	httpAddress                      string
	metricsAddress                   string
	loginObservationBudget           time.Duration
	originObservationBudget          time.Duration
	failureAuditBudget               time.Duration
	shutdownBudget                   time.Duration
	smtpOperationBudget              time.Duration
	loginDurationBuckets             []float64
	deployment                       platform.DeploymentConfig
	loginMaxBodyBytes                int64
	argon2                           security.Argon2idConfig
	argon2ProfileID                  string
	argon2MaxConcurrency             uint8
	loginBucketKeyVersion            string
	loginBucketKey                   []byte
	loginAbuse                       identity.LoginAbuseConfig
	sessionTokenEntropy              uint32
	csrfTokenEntropy                 uint32
	sessionAbsoluteTTL               time.Duration
	sessionIdleTTL                   time.Duration
	sourceKeyVersion                 string
	sourceKey                        []byte
	recoveryMode                     string
	administratorNotice              identity.AdministratorInformationNotice
	idempotencyKeys                  idempotency.DigestKeyring
	idempotencyLock                  time.Duration
	idempotencyTTL                   time.Duration
	smtpPort                         uint16
	smtpFrom                         string
	smtpAuthMode                     string
	smtpUsername                     string
	smtpPassword                     string
	articleQuery                     article.QueryConfig
	articleBackupInventory           string
	articleDeletion                  articleDeletionStartupConfig
	passwordResetWorker              identity.PasswordResetWorkerConfig
}

type articleDeletionStartupConfig struct {
	journalRoot string
	fileMode    fs.FileMode
	keyring     article.DeletionJournalKeyring
	workerID    string
	lease       time.Duration
	worker      article.DeletionWorkerConfig
}

type preparedArticleDeletion struct {
	config articleDeletionStartupConfig
}

type articleDeletionJournalOwner interface {
	article.ArticleDeletionJournal
	Close() error
}

type businessRuntime struct {
	noticeWorker        lifecycleWorker
	articleWorker       lifecycleWorker
	passwordResetWorker lifecycleWorker
	closeArticle        func() error
}

type businessComposition struct {
	noticeDispatcher                 article.NoticeDispatcherConfig
	articleTransferNoticeMaxAttempts int64
	logger                           *slog.Logger
	metrics                          *platform.MetricsRegistry
	loginObserver                    identity.LoginOutcomeObserver
	originGuard                      *platform.OriginGuard
	failureAuditor                   identity.LoginFailureAuditor
	failureAuditFor                  time.Duration
	loginObserveFor                  time.Duration
	loginMaxBodyBytes                int64
	smtpOperationBudget              time.Duration
	processingGate                   platform.ProcessingGateConfig
	prepared                         *preparedLoginDependencies
	articleJournal                   article.ArticleDeletionJournal
	articleJournalOwner              articleDeletionJournalOwner
	articleBackupInv                 string
	articleEvidence                  *article.PostgresReplayEvidenceStore
	articleReplaySource              *articleReplaySourceSlot
	articleDeletion                  *articleDeletionStartupConfig
	articleQuery                     article.QueryConfig
	passwordResetWorker              identity.PasswordResetWorkerConfig
	runtime                          *businessRuntime
	recoveryHandle                   func() recovery.Runtime
	articleTestSeam                  *articleProductionTestSeam
}

// articleReplaySourceSlot publishes the one Article deletion replay source the
// business composition built to the recovery root, so the root seals the same
// evidence Store with that same source before any Store method runs.
type articleReplaySourceSlot struct {
	source *article.ProductionDeletionReplaySource
}

type articleProductionTestSeam struct {
	adminQueryHandlers        [2]http.Handler
	createHandler             http.Handler
	updateHandler             http.Handler
	authorDetailHandler       http.Handler
	orphanHandlers            [4]http.Handler
	principalResolver         platform.PrincipalResolver
	protectOrigin             func(http.Handler) http.Handler
	trace                     func(*http.Request)
	passwordReset             identity.PasswordResetService
	temporaryCredentialResend identity.TemporaryCredentialResendService
	handlers                  [3]http.Handler
	queryHandlers             [2]http.Handler
	authorized                func(string)
	shared                    func(articleSharedDependency)
}

type articleSharedDependency struct {
	Kind     string
	Consumer string
	Instance any
}

type preparedLoginDependencies struct {
	passwords            identity.LoginPasswordHasher
	sessionIssuer        identity.LoginSessionIssuer
	abuseDigester        identity.LoginAbuseDigester
	abuseConfig          identity.LoginAbuseConfig
	source               identity.RegistrationSourceProvider
	recoveryMode         identity.LoginRecoveryMode
	ids                  identity.LoginIDGenerator
	clock                identity.LoginClock
	fakeDigest           string
	fakeProbe            string
	administratorNotices identity.AdministratorInformationNoticeReader
	idempotencyKeys      idempotency.DigestKeyring
	idempotencyLock      time.Duration
	idempotencyTTL       time.Duration
	administratorMailer  mailer.Mailer
}

func runComposed(
	ctx context.Context,
	dependencies startupDependencies,
) (resultErr error) {
	config, err := loadStartupConfig(dependencies.getenv, dependencies.lookupEnvironment)
	if err != nil {
		return err
	}
	var prepared *preparedLoginDependencies
	var recoveryPrepared preparedLoginDependencies
	if dependencies.prepareBusiness != nil {
		prepared, err = dependencies.prepareBusiness(config)
		if err != nil {
			clearStartupSecrets(&config)
			return fmt.Errorf("prepare business dependencies: %w", err)
		}
		if prepared == nil {
			clearStartupSecrets(&config)
			return errors.New("business dependency preparation returned no result")
		}
		defer prepared.cleanup()
		recoveryPrepared = *prepared
	}
	var preparedArticle *preparedArticleDeletion
	if dependencies.prepareArticle != nil {
		preparedArticle, err = dependencies.prepareArticle(config.articleDeletion)
		if err != nil {
			clearStartupSecrets(&config)
			return fmt.Errorf("prepare Article deletion dependencies: %w", err)
		}
		if preparedArticle == nil {
			clearStartupSecrets(&config)
			return errors.New("Article deletion preparation returned no result")
		}
		defer preparedArticle.cleanup()
	}
	clearStartupSecrets(&config)
	if dependencies.parseDatabase == nil || dependencies.connect == nil ||
		dependencies.ping == nil || dependencies.migrate == nil ||
		dependencies.bootstrapInitialAdmin == nil ||
		dependencies.listen == nil || dependencies.newServer == nil ||
		dependencies.coordinate == nil || dependencies.closePool == nil ||
		dependencies.signals == nil {
		return errors.New("startup dependencies are incomplete")
	}

	databaseConfig, err := dependencies.parseDatabase(config.databaseDSN)
	if err != nil {
		return fmt.Errorf("validate database configuration: %w", err)
	}
	if databaseConfig == nil || config.passwordResetWorker.ConnectionBudget > int(databaseConfig.MaxConns) {
		return fmt.Errorf("validate %s: exceeds database MaxConns", passwordResetWorkerConnectionBudgetEnvironment)
	}
	registry, err := platform.NewMetricsRegistry(config.loginDurationBuckets)
	if err != nil {
		return fmt.Errorf("create metrics registry: %w", err)
	}
	output := dependencies.logOutput
	if output == nil {
		output = os.Stdout
	}
	logger := slog.New(slog.NewJSONHandler(output, nil))
	loginLogger, err := identity.NewSlogLoginEventLogger(logger, serviceName)
	if err != nil {
		return fmt.Errorf("create login logger: %w", err)
	}
	loginObserver, err := identity.NewLoginOutcomeObserver(
		loginLogger,
		loginMetricsAdapter{registry: registry},
	)
	if err != nil {
		return fmt.Errorf("create login observer: %w", err)
	}
	originLogger, err := platform.NewSlogOriginRejectionLogger(logger, serviceName)
	if err != nil {
		return fmt.Errorf("create origin logger: %w", err)
	}
	originObserver, err := platform.NewOriginRejectionObserver(
		originLogger,
		originMetricsAdapter{registry: registry},
	)
	if err != nil {
		return fmt.Errorf("create origin observer: %w", err)
	}
	originGuard, err := platform.NewObservedOriginGuard(
		config.deployment.PublicOrigin,
		originObserver,
		config.originObservationBudget,
	)
	if err != nil {
		return fmt.Errorf("create origin guard: %w", err)
	}

	pool, err := dependencies.connect(ctx, databaseConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	poolOwned := true
	defer func() {
		if poolOwned {
			dependencies.closePool(pool)
		}
	}()
	if err := dependencies.ping(ctx, pool); err != nil {
		return fmt.Errorf("validate database connection: %w", err)
	}
	if err := dependencies.migrate(ctx, pool); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	if err := dependencies.bootstrapInitialAdmin(ctx, pool, prepared); err != nil {
		return fmt.Errorf("bootstrap initial admin: %w", err)
	}
	var articleJournal articleDeletionJournalOwner
	articleOwned := false
	if preparedArticle != nil {
		if dependencies.openArticle == nil {
			return errors.New("Article deletion journal factory is required")
		}
		articleJournal, err = dependencies.openArticle(preparedArticle)
		if err != nil {
			return fmt.Errorf(
				"open Article deletion journal: %w",
				sanitizeArticleDeletionJournalError(err),
			)
		}
		if articleJournal == nil {
			return errors.New("Article deletion journal factory returned no result")
		}
		articleOwned = true
		defer func() {
			if articleOwned {
				resultErr = errors.Join(
					resultErr,
					wrapCleanupError(
						"close Article deletion journal",
						sanitizeArticleDeletionJournalError(articleJournal.Close()),
					),
				)
			}
		}()
	}
	failureAuditor, err := identity.NewPostgresLoginFailureAuditor(pool, audit.Store{})
	if err != nil {
		return fmt.Errorf("create login failure auditor: %w", err)
	}
	boundedAuditor := boundedLoginFailureAuditor{
		delegate: failureAuditor,
		budget:   config.failureAuditBudget,
	}
	runtime := &businessRuntime{}
	var recoveryHandleMu sync.RWMutex
	var recoveryRuntime recovery.Runtime
	recoveryHandle := func() recovery.Runtime {
		recoveryHandleMu.RLock()
		defer recoveryHandleMu.RUnlock()
		return recoveryRuntime
	}
	var articleJournalDependency article.ArticleDeletionJournal
	if articleJournal != nil {
		runtime.closeArticle = func() error {
			return sanitizeArticleDeletionJournalError(articleJournal.Close())
		}
		articleJournalDependency = articleJournal
	}
	// One Article replay evidence Store serves both the recovery runtime's
	// published-closure reader and the deletion replay provider. It is created
	// here, before any consumer exists, so neither side can open a second Store.
	// The slot shares the one Article deletion replay source with the recovery
	// root so the same evidence Store is sealed with the same source.
	replaySlot := &articleReplaySourceSlot{}
	var articleEvidence *article.PostgresReplayEvidenceStore
	if pool != nil {
		articleEvidence, err = article.NewPostgresReplayEvidenceStore(pool)
		if err != nil {
			return fmt.Errorf("create Article deletion replay store: %w", err)
		}
	}
	businessHandler, err := dependencies.composeBusiness(pool, businessComposition{
		logger:                           logger,
		metrics:                          registry,
		loginObserver:                    loginObserver,
		originGuard:                      &originGuard,
		failureAuditor:                   boundedAuditor,
		failureAuditFor:                  config.failureAuditBudget,
		loginObserveFor:                  config.loginObservationBudget,
		loginMaxBodyBytes:                config.loginMaxBodyBytes,
		smtpOperationBudget:              config.smtpOperationBudget,
		processingGate:                   config.deployment.ProcessingGate,
		prepared:                         prepared,
		articleJournal:                   articleJournalDependency,
		articleJournalOwner:              articleJournal,
		articleBackupInv:                 config.articleBackupInventory,
		articleEvidence:                  articleEvidence,
		articleReplaySource:              replaySlot,
		articleQuery:                     config.articleQuery,
		articleTransferNoticeMaxAttempts: config.articleTransferNoticeMaxAttempts,
		passwordResetWorker:              config.passwordResetWorker,
		noticeDispatcher:                 config.noticeDispatcher,
		articleDeletion: func() *articleDeletionStartupConfig {
			if preparedArticle == nil {
				return nil
			}
			return &preparedArticle.config
		}(),
		runtime:        runtime,
		recoveryHandle: recoveryHandle,
	})
	if err != nil {
		return fmt.Errorf("compose business routes: %w", err)
	}
	if businessHandler == nil {
		return errors.New("business route composition returned no handler")
	}
	if preparedArticle != nil && runtime.articleWorker == nil {
		return errors.New("Article deletion worker construction returned no result")
	}
	if dependencies.requirePasswordResetWorker && runtime.passwordResetWorker == nil {
		return errors.New("password reset worker construction returned no result")
	}
	if dependencies.requireNoticeWorker && noticeLifecycleIncomplete(runtime.noticeWorker) {
		return errors.New("notice worker construction returned no result or shutdown capability")
	}
	var recoveryPlan recoveryStartupPlan
	var recoveryAssembly *recoveryRootAssembly
	var closeArticleOnce sync.Once
	var closeArticleErr error
	closeArticle := func() error {
		if runtime.closeArticle == nil {
			return nil
		}
		closeArticleOnce.Do(func() { closeArticleErr = runtime.closeArticle() })
		return closeArticleErr
	}
	recoveryPlan, err = planRecoveryStartup(dependencies.getenv, dependencies.recoveryCommand)
	if err != nil {
		return fmt.Errorf("load recovery configuration: %w", err)
	}
	if recoveryPlan.present {
		assembly, assemblyErr := newRecoveryRootAssembly(ctx, recoveryRootInputs{
			pool: pool, plan: recoveryPlan, prepared: &recoveryPrepared,
			closures: articleEvidence,
			seal: func(sealed recovery.Runtime) error {
				if articleEvidence == nil || replaySlot.source == nil {
					return nil
				}
				_, err := article.NewReplayPublicationPorts(articleEvidence, article.ReplayPublicationDependencies{
					Source:         replaySlot.source,
					Runtime:        sealed,
					InstallationID: recoveryPlan.config.installation,
					PublicationIDs: recoveryPrepared.ids,
					Clock:          recoveryPrepared.clock,
				})
				return err
			},
		})
		if assemblyErr != nil {
			return fmt.Errorf("compose recovery runtime: %w", assemblyErr)
		}
		recoveryAssembly = assembly
		recoveryHandleMu.Lock()
		recoveryRuntime = assembly.runtime
		recoveryHandleMu.Unlock()
		if !recoveryPlan.verify {
			if runtime.closeArticle == nil {
				return errors.New("recovery startup requires the Article deletion journal owner")
			}
			articleWorker, articleWorkerErr := recoveryStartedWorkerFrom(runtime.articleWorker)
			if articleWorkerErr != nil {
				return fmt.Errorf("bind recovery Article worker: %w", articleWorkerErr)
			}
			resetWorker, resetWorkerErr := recoveryStartedWorkerFrom(runtime.passwordResetWorker)
			if resetWorkerErr != nil {
				return fmt.Errorf("bind recovery password reset worker: %w", resetWorkerErr)
			}
			noticeWorker, noticeWorkerErr := recoveryStartedWorkerFrom(runtime.noticeWorker)
			if noticeWorkerErr != nil {
				return fmt.Errorf("bind recovery notice worker: %w", noticeWorkerErr)
			}
			if bindErr := assembly.bindWorkers(articleWorker, resetWorker, noticeWorker); bindErr != nil {
				return fmt.Errorf("bind recovery workers: %w", bindErr)
			}
			if bindErr := assembly.bindArticleOwner(closeArticle, newRecoveryArticleJoin(runtime)); bindErr != nil {
				return fmt.Errorf("bind recovery Article handle: %w", bindErr)
			}
		}
	}

	metricsHandler := platform.NewMetricsHandler(registry)
	if recoveryPlan.verify {
		if verifyErr := recoveryAssembly.verify(ctx, recoveryPlan.config.proofBudget); verifyErr != nil {
			return fmt.Errorf("verify recovery runtime: %w", verifyErr)
		}
		return nil
	}

	businessListener, err := dependencies.listen("tcp", config.httpAddress)
	if err != nil {
		return fmt.Errorf("listen for business HTTP requests: %w", err)
	}
	metricsListener, err := dependencies.listen("tcp", config.metricsAddress)
	if err != nil {
		return errors.Join(
			fmt.Errorf("listen for metrics HTTP requests: %w", err),
			wrapCleanupError(
				"close business HTTP listener",
				closeHTTPListener(businessListener),
			),
		)
	}
	businessServer := dependencies.newServer("business", businessHandler)
	metricsServer := dependencies.newServer("metrics", metricsHandler)
	if businessServer == nil || metricsServer == nil {
		return errors.Join(
			errors.New("HTTP server construction failed"),
			wrapCleanupError(
				"close business HTTP listener",
				closeHTTPListener(businessListener),
			),
			wrapCleanupError(
				"close metrics HTTP listener",
				closeHTTPListener(metricsListener),
			),
		)
	}

	var additionalServers []*lifecycleServerState
	var recoveryJoinWork func(context.Context) error
	if recoveryAssembly != nil {
		recoveryJoinWork = recoveryAssembly.joinWork
		if recoveryAssembly.maintenanceMode() {
			wiring, wiringErr := recoveryAssembly.wiring()
			if wiringErr != nil {
				return fmt.Errorf("compose recovery control wiring: %w", wiringErr)
			}
			if dependencies.composeRecoveryControl == nil {
				return errors.New("recovery control composition is unavailable")
			}
			controlHandler, controlErr := dependencies.composeRecoveryControl(pool, recoveryControlOptions{
				prepared:     &recoveryPrepared,
				logger:       logger,
				observer:     loginObserver,
				auditor:      boundedAuditor,
				auditFor:     config.failureAuditBudget,
				observeFor:   config.loginObservationBudget,
				maxBodyBytes: config.loginMaxBodyBytes,
				ready:        recoveryAssembly.identityReady,
			}, wiring)
			if controlErr != nil {
				return fmt.Errorf("compose recovery control surface: %w", controlErr)
			}
			controlListener, controlListenErr := dependencies.listen("tcp", recoveryAssembly.controlAddress())
			if controlListenErr != nil {
				return fmt.Errorf("listen for recovery control requests: %w", controlListenErr)
			}
			controlServer := dependencies.newServer("recovery-control", controlHandler)
			if controlServer == nil {
				return errors.Join(
					errors.New("recovery control server construction failed"),
					wrapCleanupError("close recovery control listener", closeHTTPListener(controlListener)),
				)
			}
			controlJoined, bindErr := recoveryAssembly.bindControl(controlServer, controlListener)
			if bindErr != nil {
				return errors.Join(
					fmt.Errorf("bind recovery control surface: %w", bindErr),
					wrapCleanupError("close recovery control listener", closeHTTPListener(controlListener)),
				)
			}
			additionalServers = append(additionalServers, &lifecycleServerState{
				// The lifecycle serves the joined wrapper, never the raw server: only
				// the wrapper's Serve closes done for the recovery exit join.
				name: "recovery-control", server: controlJoined, listener: controlListener,
			})
		}
		if startErr := recoveryAssembly.start(ctx, recoveryPlan.config.ioBudget); startErr != nil {
			return fmt.Errorf("start recovery runtime: %w", startErr)
		}
		if recoveryAssembly.maintenanceMode() {
			if readyErr := recoveryAssembly.awaitControlReady(ctx); readyErr != nil {
				return fmt.Errorf("await recovery control readiness: %w", readyErr)
			}
		}
	}
	poolOwned = false
	articleOwned = false
	return dependencies.coordinate(ctx, lifecycleConfig{
		businessServer:   businessServer,
		metricsServer:    metricsServer,
		businessListener: businessListener,
		metricsListener:  metricsListener,
		closePool: func() error {
			dependencies.closePool(pool)
			return nil
		},
		articleWorker:       runtime.articleWorker,
		passwordResetWorker: runtime.passwordResetWorker,
		noticeWorker:        runtime.noticeWorker,
		closeArticle:        closeArticle,
		joinWork:            recoveryJoinWork,
		additionalServers:   additionalServers,
		shutdownBudget:      config.shutdownBudget,
		signals:             dependencies.signals,
	})
}

func wrapCleanupError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func loadStartupConfig(
	getenv func(string) string,
	lookup platform.EnvironmentLookup,
) (startupConfig, error) {
	if getenv == nil || lookup == nil {
		return startupConfig{}, errors.New("environment readers are required")
	}
	config := startupConfig{
		databaseDSN:    strings.TrimSpace(getenv(databaseDSNEnvironment)),
		httpAddress:    strings.TrimSpace(getenv(httpAddressEnvironment)),
		metricsAddress: strings.TrimSpace(getenv(metricsAddressEnvironment)),
	}
	loaded := false
	defer func() {
		if loaded {
			return
		}
		clearStartupSecrets(&config)
	}()
	if config.databaseDSN == "" {
		return startupConfig{}, errors.New("database DSN is required")
	}
	if err := validateHTTPAddress(config.httpAddress); err != nil {
		return startupConfig{}, fmt.Errorf("validate HTTP listen address: %w", err)
	}
	if err := validateHTTPAddress(config.metricsAddress); err != nil {
		return startupConfig{}, fmt.Errorf("validate metrics listen address: %w", err)
	}
	var err error
	config.articleQuery, err = loadArticleQueryConfig(lookup)
	if err != nil {
		return startupConfig{}, err
	}
	config.articleTransferNoticeMaxAttempts, err = parsePositiveDecimal(getenv(articleTransferNoticeMaxAttemptsEnvironment), articleTransferNoticeMaxAttemptsEnvironment, 64)
	if err != nil {
		return startupConfig{}, err
	}
	config.loginObservationBudget, err = parseRequiredDuration(
		getenv(loginObservationBudgetEnvironment),
		loginObservationBudgetEnvironment,
		time.Millisecond,
		5*time.Second,
	)
	if err != nil {
		return startupConfig{}, err
	}
	config.originObservationBudget, err = parseRequiredDuration(
		getenv(originObservationBudgetEnvironment),
		originObservationBudgetEnvironment,
		time.Millisecond,
		5*time.Second,
	)
	if err != nil {
		return startupConfig{}, err
	}
	config.failureAuditBudget, err = parseRequiredDuration(
		getenv(failureAuditBudgetEnvironment),
		failureAuditBudgetEnvironment,
		10*time.Millisecond,
		30*time.Second,
	)
	if err != nil {
		return startupConfig{}, err
	}
	config.shutdownBudget, err = parseRequiredDuration(
		getenv(shutdownBudgetEnvironment),
		shutdownBudgetEnvironment,
		time.Second,
		60*time.Second,
	)
	if err != nil {
		return startupConfig{}, err
	}
	config.smtpOperationBudget, err = parseRequiredDuration(
		getenv(smtpOperationBudgetEnvironment),
		smtpOperationBudgetEnvironment,
		time.Second,
		60*time.Second,
	)
	if err != nil {
		return startupConfig{}, err
	}
	config.noticeDispatcher, err = loadNoticeDispatcherConfig(getenv, config.smtpOperationBudget, config.shutdownBudget)
	if err != nil {
		return startupConfig{}, err
	}
	config.passwordResetWorker, err = loadPasswordResetWorkerConfig(getenv)
	if err != nil {
		return startupConfig{}, err
	}
	config.loginDurationBuckets, err = platform.ParseHistogramBuckets(
		getenv(loginDurationBucketsEnvironment),
	)
	if err != nil {
		return startupConfig{}, fmt.Errorf(
			"validate %s: %w",
			loginDurationBucketsEnvironment,
			err,
		)
	}
	if err := loadIdentityStartupConfig(&config, getenv); err != nil {
		return startupConfig{}, err
	}
	if err := loadAdministratorStartupConfig(&config, getenv, lookup); err != nil {
		return startupConfig{}, err
	}
	if err := loadArticleDeletionStartupConfig(&config, getenv, lookup); err != nil {
		return startupConfig{}, fmt.Errorf(
			"validate Article deletion configuration: %w",
			err,
		)
	}
	config.deployment, err = loadDeploymentConfig(lookup, config.deployment.TrustedProxyCIDRs)
	if err != nil {
		return startupConfig{}, fmt.Errorf("validate deployment security configuration: %w", err)
	}
	if err := validateSMTPDeploymentConfig(lookup, config.deployment); err != nil {
		return startupConfig{}, err
	}
	loaded = true
	return config, nil
}

func loadArticleQueryConfig(
	lookup platform.EnvironmentLookup,
) (article.QueryConfig, error) {
	defaultLimit, err := parseArticleQueryConfigField(
		lookup,
		articleQueryDefaultLimitEnvironment,
	)
	if err != nil {
		return article.QueryConfig{}, err
	}
	maxLimit, err := parseArticleQueryConfigField(
		lookup,
		articleQueryMaxLimitEnvironment,
	)
	if err != nil {
		return article.QueryConfig{}, err
	}
	candidateBatchSize, err := parseArticleQueryConfigField(
		lookup,
		articleQueryCandidateBatchSizeEnvironment,
	)
	if err != nil {
		return article.QueryConfig{}, err
	}
	if defaultLimit > maxLimit || candidateBatchSize <= maxLimit {
		return article.QueryConfig{}, articleQueryConfigError("", "cross_constraint")
	}
	return article.QueryConfig{
		DefaultLimit:       defaultLimit,
		MaxLimit:           maxLimit,
		CandidateBatchSize: candidateBatchSize,
	}, nil
}

func parseArticleQueryConfigField(
	lookup platform.EnvironmentLookup,
	key string,
) (int, error) {
	if lookup == nil {
		return 0, articleQueryConfigError(key, "missing")
	}
	raw, exists := lookup(key)
	if !exists {
		return 0, articleQueryConfigError(key, "missing")
	}
	if raw == "" {
		return 0, articleQueryConfigError(key, "empty")
	}
	if raw[0] < '1' || raw[0] > '9' {
		return 0, articleQueryConfigError(key, "invalid_format")
	}
	for index := 1; index < len(raw); index++ {
		if raw[index] < '0' || raw[index] > '9' {
			return 0, articleQueryConfigError(key, "invalid_format")
		}
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, articleQueryConfigError(key, "out_of_range")
	}
	return int(value), nil
}

func articleQueryConfigError(key string, category string) error {
	if key == "" {
		return fmt.Errorf(
			"validate Article query configuration: %s",
			category,
		)
	}
	return fmt.Errorf(
		"validate Article query configuration: %s: %s",
		key,
		category,
	)
}

func loadNoticeDispatcherConfig(getenv func(string) string, smtpBudget, shutdownBudget time.Duration) (article.NoticeDispatcherConfig, error) {
	config := article.NoticeDispatcherConfig{SMTPOperationBudget: smtpBudget}
	for _, field := range []struct {
		key   string
		value *time.Duration
	}{
		{noticePollIntervalEnvironment, &config.PollInterval},
		{noticeLeaseDurationEnvironment, &config.LeaseDuration},
		{noticeDBOperationBudgetEnvironment, &config.DBOperationBudget},
		{noticeBackoffInitialEnvironment, &config.BackoffInitial},
		{noticeBackoffMaxEnvironment, &config.BackoffMax},
	} {
		value, err := parsePositiveDuration(getenv(field.key), field.key)
		if err != nil {
			return article.NoticeDispatcherConfig{}, err
		}
		if value%time.Microsecond != 0 {
			return article.NoticeDispatcherConfig{}, fmt.Errorf("validate %s: integer_microseconds_required", field.key)
		}
		*field.value = value
	}
	batch, err := parsePositiveDecimal(getenv(noticeBatchLimitEnvironment), noticeBatchLimitEnvironment, 32)
	if err != nil {
		return article.NoticeDispatcherConfig{}, err
	}
	config.BatchLimit = int(batch)
	if err := config.Validate(); err != nil {
		return article.NoticeDispatcherConfig{}, fmt.Errorf("validate notice configuration: %w", err)
	}
	if config.DBOperationBudget > (time.Duration(math.MaxInt64)-smtpBudget)/2 ||
		shutdownBudget <= smtpBudget+2*config.DBOperationBudget {
		return article.NoticeDispatcherConfig{}, fmt.Errorf("validate %s: notice_cross_constraint", shutdownBudgetEnvironment)
	}
	return config, nil
}

func loadPasswordResetWorkerConfig(getenv func(string) string) (identity.PasswordResetWorkerConfig, error) {
	if getenv == nil {
		return identity.PasswordResetWorkerConfig{}, errors.New("environment reader is required")
	}
	lease, err := parsePositiveDuration(getenv(passwordResetWorkerLeaseDurationEnvironment), passwordResetWorkerLeaseDurationEnvironment)
	if err != nil {
		return identity.PasswordResetWorkerConfig{}, err
	}
	backoff, err := parsePositiveDuration(getenv(passwordResetWorkerRetryBackoffEnvironment), passwordResetWorkerRetryBackoffEnvironment)
	if err != nil {
		return identity.PasswordResetWorkerConfig{}, err
	}
	maxRetries, err := parsePositiveDecimal(getenv(passwordResetWorkerMaxRetriesEnvironment), passwordResetWorkerMaxRetriesEnvironment, 64)
	if err != nil {
		return identity.PasswordResetWorkerConfig{}, err
	}
	claimLimit, err := parsePositiveDecimal(getenv(passwordResetWorkerClaimBatchLimitEnvironment), passwordResetWorkerClaimBatchLimitEnvironment, 32)
	if err != nil {
		return identity.PasswordResetWorkerConfig{}, err
	}
	budget, err := parsePositiveDecimal(getenv(passwordResetWorkerConnectionBudgetEnvironment), passwordResetWorkerConnectionBudgetEnvironment, 32)
	if err != nil {
		return identity.PasswordResetWorkerConfig{}, err
	}
	return identity.PasswordResetWorkerConfig{LeaseDuration: lease, RetryBackoff: backoff, MaxRetries: maxRetries, ClaimBatchLimit: int(claimLimit), ConnectionBudget: int(budget)}, nil
}

func parsePositiveDuration(raw, key string) (time.Duration, error) {
	if raw == "" {
		return 0, fmt.Errorf("validate %s: value is required", key)
	}
	if raw != strings.TrimSpace(raw) {
		return 0, fmt.Errorf("validate %s: value is invalid", key)
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("validate %s: duration is invalid", key)
	}
	if value <= 0 {
		return 0, fmt.Errorf("validate %s: value must be positive", key)
	}
	return value, nil
}

func parsePositiveDecimal(raw, key string, bitSize int) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("validate %s: value is required", key)
	}
	if raw != strings.TrimSpace(raw) {
		return 0, fmt.Errorf("validate %s: value is invalid", key)
	}
	value, err := strconv.ParseInt(raw, 10, bitSize)
	if err != nil || strconv.FormatInt(value, 10) != raw {
		return 0, fmt.Errorf("validate %s: integer is invalid", key)
	}
	if value <= 0 {
		return 0, fmt.Errorf("validate %s: value must be positive", key)
	}
	return value, nil
}

func loadArticleDeletionStartupConfig(
	config *startupConfig,
	getenv func(string) string,
	lookup platform.EnvironmentLookup,
) error {
	if config == nil || getenv == nil || lookup == nil {
		return errors.New("environment readers are required")
	}
	root := getenv(articleDeletionJournalRootEnvironment)
	if root == "" || root != strings.TrimSpace(root) ||
		root != filepath.Clean(root) ||
		(!filepath.IsAbs(root) &&
			!strings.HasPrefix(root, string(filepath.Separator))) {
		return errors.New("journal root is invalid")
	}
	config.articleDeletion.journalRoot = root

	inventoryPath := getenv(articleBackupInventoryPathEnvironment)
	if inventoryPath == "" || inventoryPath != strings.TrimSpace(inventoryPath) ||
		inventoryPath != filepath.Clean(inventoryPath) ||
		(!filepath.IsAbs(inventoryPath) &&
			!strings.HasPrefix(inventoryPath, string(filepath.Separator))) {
		return errors.New("backup inventory path is invalid")
	}
	config.articleBackupInventory = inventoryPath

	modeRaw := getenv(articleDeletionJournalFileModeEnvironment)
	if modeRaw == "" {
		modeRaw = "0600"
	}
	modeValue, err := strconv.ParseUint(modeRaw, 8, 32)
	if err != nil || len(modeRaw) != 4 {
		return errors.New("journal file mode is invalid")
	}
	mode := fs.FileMode(modeValue).Perm()
	if mode&0o600 != 0o600 || mode&0o077 != 0 {
		return errors.New("journal file mode is invalid")
	}
	config.articleDeletion.fileMode = mode

	current, err := parseArticleDeletionVersion(
		getenv(articleDeletionCurrentKeyVersionEnvironment),
	)
	if err != nil {
		return errors.New("current key version is invalid")
	}
	retainedRaw, exists := lookup(articleDeletionRetainedKeyVersionsEnvironment)
	if !exists {
		return errors.New("retained key versions are missing")
	}
	retainedVersions, err := parseArticleDeletionRetainedVersions(retainedRaw, current)
	if err != nil {
		return errors.New("retained key versions are invalid")
	}
	active, err := loadArticleDeletionKey(getenv, current)
	if err != nil {
		return errors.New("current key is invalid")
	}
	config.articleDeletion.keyring.Active = active
	for _, version := range retainedVersions {
		key, err := loadArticleDeletionKey(getenv, version)
		if err != nil {
			return errors.New("retained key is invalid")
		}
		config.articleDeletion.keyring.Retained = append(
			config.articleDeletion.keyring.Retained,
			key,
		)
	}

	config.articleDeletion.workerID, err = parseArticleDeletionWorkerID(
		getenv(articleFinalizerWorkerIDEnvironment),
	)
	if err != nil {
		return err
	}
	config.articleDeletion.worker.PollInterval, err = parseRequiredDuration(
		getenv(articleFinalizerPollIntervalEnvironment),
		articleFinalizerPollIntervalEnvironment,
		100*time.Millisecond,
		time.Minute,
	)
	if err != nil {
		return err
	}
	batchSize, err := parseRequiredInt(
		getenv,
		articleFinalizerBatchSizeEnvironment,
		1,
		500,
	)
	if err != nil {
		return err
	}
	config.articleDeletion.worker.BatchSize = int(batchSize)
	config.articleDeletion.lease, err = parseRequiredDuration(
		getenv(articleFinalizerLeaseDurationEnvironment),
		articleFinalizerLeaseDurationEnvironment,
		5*time.Second,
		5*time.Minute,
	)
	if err != nil {
		return err
	}
	config.articleDeletion.worker.OperationBudget, err = parseRequiredDuration(
		getenv(articleFinalizerOperationBudgetEnvironment),
		articleFinalizerOperationBudgetEnvironment,
		time.Second,
		time.Minute,
	)
	if err != nil {
		return err
	}
	config.articleDeletion.worker.BackoffInitial, err = parseRequiredDuration(
		getenv(articleFinalizerBackoffInitialEnvironment),
		articleFinalizerBackoffInitialEnvironment,
		50*time.Millisecond,
		10*time.Second,
	)
	if err != nil {
		return err
	}
	config.articleDeletion.worker.BackoffMax, err = parseRequiredDuration(
		getenv(articleFinalizerBackoffMaxEnvironment),
		articleFinalizerBackoffMaxEnvironment,
		time.Second,
		5*time.Minute,
	)
	if err != nil {
		return err
	}
	maxBatches, err := parseRequiredInt(
		getenv,
		articleFinalizerMaxAttemptsEnvironment,
		1,
		10,
	)
	if err != nil {
		return err
	}
	config.articleDeletion.worker.MaxBatchesPerWake = int(maxBatches)
	config.articleDeletion.worker.ShutdownBudget, err = parseRequiredDuration(
		getenv(articleFinalizerShutdownBudgetEnvironment),
		articleFinalizerShutdownBudgetEnvironment,
		time.Second,
		time.Minute,
	)
	if err != nil {
		return err
	}
	if config.articleDeletion.worker.BackoffInitial >
		config.articleDeletion.worker.BackoffMax ||
		config.articleDeletion.worker.OperationBudget >= config.articleDeletion.lease ||
		config.articleDeletion.worker.PollInterval >= config.articleDeletion.lease ||
		config.articleDeletion.worker.ShutdownBudget > config.shutdownBudget {
		return errors.New("worker timing configuration is inconsistent")
	}
	return nil
}

func parseArticleDeletionVersion(raw string) (string, error) {
	if raw == "" || len(raw) > 64 || raw != strings.TrimSpace(raw) {
		return "", errors.New("invalid version")
	}
	for _, value := range []byte(raw) {
		if value < 0x21 || value > 0x7e ||
			value == ',' || value == '=' {
			return "", errors.New("invalid version")
		}
	}
	return raw, nil
}

func parseArticleDeletionRetainedVersions(
	raw string,
	current string,
) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	seen := map[string]struct{}{current: {}}
	versions := strings.Split(raw, ",")
	for index, rawVersion := range versions {
		version, err := parseArticleDeletionVersion(rawVersion)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[version]; exists {
			return nil, errors.New("duplicate version")
		}
		seen[version] = struct{}{}
		versions[index] = version
	}
	return versions, nil
}

func loadArticleDeletionKey(
	getenv func(string) string,
	version string,
) (article.DeletionJournalKey, error) {
	raw := getenv(articleDeletionKeyPrefix + version)
	material, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || raw == "" ||
		base64.RawURLEncoding.EncodeToString(material) != raw ||
		len(material) < 32 || len(material) > 64 {
		clear(material)
		return article.DeletionJournalKey{}, errors.New("invalid key")
	}
	return article.DeletionJournalKey{
		ID:       "article-deletion-hmac",
		Version:  version,
		Material: material,
	}, nil
}

func parseArticleDeletionWorkerID(raw string) (string, error) {
	if raw == "" || len(raw) > 128 || raw != strings.TrimSpace(raw) {
		return "", errors.New("finalizer worker ID is invalid")
	}
	for _, value := range []byte(raw) {
		if value < 0x21 || value > 0x7e {
			return "", errors.New("finalizer worker ID is invalid")
		}
	}
	return raw, nil
}

func loadIdentityStartupConfig(config *startupConfig, getenv func(string) string) error {
	completed := false
	defer func() {
		if completed {
			return
		}
		clear(config.loginBucketKey)
		clear(config.sourceKey)
	}()

	var err error
	config.loginMaxBodyBytes, err = parseRequiredInt(getenv, loginMaxBodyBytesEnvironment, 1024, 65536)
	if err != nil {
		return err
	}
	memory, err := parseRequiredUint(getenv, argon2MemoryEnvironment, 19456, 262144, 32)
	if err != nil {
		return err
	}
	iterations, err := parseRequiredUint(getenv, argon2IterationsEnvironment, 2, 6, 32)
	if err != nil {
		return err
	}
	parallelism, err := parseRequiredUint(getenv, argon2ParallelismEnvironment, 1, 4, 8)
	if err != nil {
		return err
	}
	saltLength, err := parseRequiredUint(getenv, argon2SaltBytesEnvironment, 16, 64, 32)
	if err != nil {
		return err
	}
	keyLength, err := parseRequiredUint(getenv, argon2KeyBytesEnvironment, 32, 64, 32)
	if err != nil {
		return err
	}
	config.argon2 = security.Argon2idConfig{
		MemoryKiB: uint32(memory), Iterations: uint32(iterations),
		Parallelism: uint8(parallelism), SaltLength: uint32(saltLength), KeyLength: uint32(keyLength),
	}
	config.argon2ProfileID, err = parseToken(getenv(argon2ProfileIDEnvironment), argon2ProfileIDEnvironment, true)
	if err != nil {
		return err
	}
	concurrency, err := parseRequiredUint(getenv, argon2MaxConcurrencyEnvironment, 1, 4, 8)
	if err != nil {
		return err
	}
	config.argon2MaxConcurrency = uint8(concurrency)
	if uint64(config.argon2.MemoryKiB)*concurrency > 524288 {
		return fmt.Errorf("environment variables %s and %s exceed the accepted resource bound", argon2MemoryEnvironment, argon2MaxConcurrencyEnvironment)
	}
	config.loginBucketKeyVersion, err = parseToken(getenv(loginBucketKeyVersionEnvironment), loginBucketKeyVersionEnvironment, false)
	if err != nil {
		return err
	}
	config.loginBucketKey, err = parseSecret(getenv(loginBucketKeyPrefix+config.loginBucketKeyVersion), loginBucketKeyPrefix+config.loginBucketKeyVersion)
	if err != nil {
		return err
	}
	config.loginAbuse.Window, err = parseRequiredDuration(getenv(loginAbuseWindowEnvironment), loginAbuseWindowEnvironment, 10*time.Second, 24*time.Hour)
	if err != nil {
		return err
	}
	config.loginAbuse.FailureThreshold, err = parseRequiredInt(getenv, loginAbuseThresholdEnvironment, 3, 100)
	if err != nil {
		return err
	}
	config.loginAbuse.BlockDuration, err = parseRequiredDuration(getenv(loginAbuseBlockEnvironment), loginAbuseBlockEnvironment, time.Second, 24*time.Hour)
	if err != nil {
		return err
	}
	action := getenv(loginAbuseActionEnvironment)
	if action != strings.TrimSpace(action) {
		return fmt.Errorf("environment variable %s has an unsupported value", loginAbuseActionEnvironment)
	}
	switch action {
	case string(identity.LoginAbuseSuccessClear):
		config.loginAbuse.SuccessAction = identity.LoginAbuseSuccessClear
	case string(identity.LoginAbuseSuccessDecay):
		config.loginAbuse.SuccessAction = identity.LoginAbuseSuccessDecay
	default:
		return fmt.Errorf("environment variable %s has an unsupported value", loginAbuseActionEnvironment)
	}
	config.loginAbuse.SuccessDecay, err = parseRequiredInt(getenv, loginAbuseDecayEnvironment, 0, 99)
	if err != nil {
		return err
	}
	if (config.loginAbuse.SuccessAction == identity.LoginAbuseSuccessClear && config.loginAbuse.SuccessDecay != 0) ||
		(config.loginAbuse.SuccessAction == identity.LoginAbuseSuccessDecay &&
			(config.loginAbuse.SuccessDecay < 1 || config.loginAbuse.SuccessDecay >= config.loginAbuse.FailureThreshold)) {
		return fmt.Errorf("environment variables %s and %s are inconsistent", loginAbuseActionEnvironment, loginAbuseDecayEnvironment)
	}
	tokenEntropy, err := parseRequiredUint(getenv, sessionTokenEntropyEnvironment, 32, 64, 32)
	if err != nil {
		return err
	}
	config.sessionTokenEntropy = uint32(tokenEntropy)
	csrfEntropy, err := parseRequiredUint(getenv, csrfTokenEntropyEnvironment, 32, 64, 32)
	if err != nil {
		return err
	}
	config.csrfTokenEntropy = uint32(csrfEntropy)
	config.sessionAbsoluteTTL, err = parseRequiredDuration(getenv(sessionAbsoluteTTLEnvironment), sessionAbsoluteTTLEnvironment, 15*time.Minute, 30*24*time.Hour)
	if err != nil {
		return err
	}
	config.sessionIdleTTL, err = parseRequiredDuration(getenv(sessionIdleTTLEnvironment), sessionIdleTTLEnvironment, 5*time.Minute, 24*time.Hour)
	if err != nil {
		return err
	}
	if config.sessionIdleTTL > config.sessionAbsoluteTTL {
		return fmt.Errorf("environment variables %s and %s are inconsistent", sessionIdleTTLEnvironment, sessionAbsoluteTTLEnvironment)
	}
	config.sourceKeyVersion, err = parseToken(getenv(sourceKeyVersionEnvironment), sourceKeyVersionEnvironment, false)
	if err != nil {
		return err
	}
	config.sourceKey, err = parseSecret(getenv(sourceKeyPrefix+config.sourceKeyVersion), sourceKeyPrefix+config.sourceKeyVersion)
	if err != nil {
		return err
	}
	if bytes.Equal(config.sourceKey, config.loginBucketKey) {
		return fmt.Errorf("source and login bucket keys must be independent")
	}
	proxies, err := parseTrustedProxyConfiguration(getenv(platform.EnvTrustedProxyCIDRs))
	if err != nil {
		return err
	}
	config.deployment.TrustedProxyCIDRs = proxies
	config.recoveryMode = getenv(recoveryModeEnvironment)
	if config.recoveryMode != strings.TrimSpace(config.recoveryMode) {
		return fmt.Errorf("environment variable %s has an unsupported value", recoveryModeEnvironment)
	}
	if config.recoveryMode != "normal" && config.recoveryMode != "maintenance" {
		return fmt.Errorf("environment variable %s has an unsupported value", recoveryModeEnvironment)
	}
	completed = true
	return nil
}

func loadAdministratorStartupConfig(
	config *startupConfig,
	getenv func(string) string,
	lookup platform.EnvironmentLookup,
) error {
	if config == nil || getenv == nil || lookup == nil {
		return errors.New("administrator startup configuration readers are required")
	}
	completed := false
	defer func() {
		if !completed {
			clearDigestKeyring(&config.idempotencyKeys)
			config.smtpUsername = ""
			config.smtpPassword = ""
		}
	}()

	version, err := parseAdministratorNoticeVersion(
		getenv(administratorNoticeVersionEnvironment),
	)
	if err != nil {
		return err
	}
	effectiveAt, err := parseAdministratorNoticeEffectiveAt(
		getenv(administratorNoticeEffectiveAtEnvironment),
	)
	if err != nil {
		return err
	}
	content, err := parseAdministratorNoticeContent(
		getenv(administratorNoticeContentEnvironment),
	)
	if err != nil {
		return err
	}
	contentHash, err := parseAdministratorNoticeHash(
		getenv(administratorNoticeSHA256Environment),
		content,
	)
	if err != nil {
		return err
	}
	config.administratorNotice = identity.AdministratorInformationNotice{
		Version: version, EffectiveAt: effectiveAt,
		Content: content, ContentHash: contentHash,
	}

	config.idempotencyKeys, err = parseIdempotencyKeyring(getenv)
	if err != nil {
		return err
	}
	for version, key := range config.idempotencyKeys.Keys {
		if bytes.Equal(key, config.loginBucketKey) || bytes.Equal(key, config.sourceKey) {
			return fmt.Errorf(
				"environment variable %s%s must use independent key material",
				idempotencyKeyPrefix,
				version,
			)
		}
	}
	config.idempotencyLock, err = parseRequiredDuration(
		getenv(idempotencyLockDurationEnvironment),
		idempotencyLockDurationEnvironment,
		time.Second,
		5*time.Minute,
	)
	if err != nil {
		return err
	}
	if config.idempotencyLock <= config.smtpOperationBudget {
		return fmt.Errorf(
			"environment variables %s and %s are inconsistent",
			idempotencyLockDurationEnvironment,
			smtpOperationBudgetEnvironment,
		)
	}
	config.idempotencyTTL, err = parseRequiredDuration(
		getenv(idempotencyRecordTTLEnvironment),
		idempotencyRecordTTLEnvironment,
		time.Hour,
		30*24*time.Hour,
	)
	if err != nil {
		return err
	}
	if config.idempotencyTTL <= config.idempotencyLock {
		return fmt.Errorf(
			"environment variables %s and %s are inconsistent",
			idempotencyRecordTTLEnvironment,
			idempotencyLockDurationEnvironment,
		)
	}

	port, err := parseRequiredUint(getenv, smtpPortEnvironment, 1, 65535, 16)
	if err != nil {
		return err
	}
	config.smtpPort = uint16(port)
	config.smtpFrom, err = parseSMTPMailbox(getenv(smtpFromEnvironment))
	if err != nil {
		return err
	}
	config.smtpAuthMode = getenv(smtpAuthModeEnvironment)
	if config.smtpAuthMode != "plain" && config.smtpAuthMode != "none" {
		return fmt.Errorf(
			"environment variable %s has an unsupported value",
			smtpAuthModeEnvironment,
		)
	}
	username, usernameExists := lookup(smtpUsernameEnvironment)
	password, passwordExists := lookup(smtpPasswordEnvironment)
	if !usernameExists {
		return fmt.Errorf("required environment variable %s is not set", smtpUsernameEnvironment)
	}
	if !passwordExists {
		return fmt.Errorf("required environment variable %s is not set", smtpPasswordEnvironment)
	}
	switch config.smtpAuthMode {
	case "plain":
		if len(username) < 1 || len(username) > 1024 ||
			username != strings.TrimSpace(username) ||
			containsControl(username) {
			return fmt.Errorf("environment variable %s is invalid", smtpUsernameEnvironment)
		}
		if len(password) < 1 || len(password) > 1024 {
			return fmt.Errorf("environment variable %s is invalid", smtpPasswordEnvironment)
		}
	case "none":
		if username != "" || password != "" {
			return fmt.Errorf(
				"environment variables %s and %s are inconsistent with %s",
				smtpUsernameEnvironment,
				smtpPasswordEnvironment,
				smtpAuthModeEnvironment,
			)
		}
	}
	config.smtpUsername = username
	config.smtpPassword = password
	completed = true
	return nil
}

func parseAdministratorNoticeVersion(raw string) (string, error) {
	if len(raw) < 1 || len(raw) > 128 {
		return "", fmt.Errorf(
			"environment variable %s is invalid",
			administratorNoticeVersionEnvironment,
		)
	}
	for index := range raw {
		character := raw[index]
		if character > 0x7f ||
			!(character >= 'A' && character <= 'Z' ||
				character >= 'a' && character <= 'z' ||
				character >= '0' && character <= '9' ||
				index > 0 && (character == '.' || character == '_' || character == '-')) {
			return "", fmt.Errorf(
				"environment variable %s is invalid",
				administratorNoticeVersionEnvironment,
			)
		}
	}
	return raw, nil
}

func parseAdministratorNoticeEffectiveAt(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	_, offset := parsed.Zone()
	if err != nil || !strings.HasSuffix(raw, "Z") || offset != 0 ||
		parsed.IsZero() || raw != parsed.UTC().Format(time.RFC3339) {
		return time.Time{}, fmt.Errorf(
			"environment variable %s must be a canonical UTC whole-second RFC3339 value",
			administratorNoticeEffectiveAtEnvironment,
		)
	}
	return parsed.UTC(), nil
}

func parseAdministratorNoticeContent(raw string) (string, error) {
	if len(raw) < 1 || len(raw) > 65536 || !utf8.ValidString(raw) ||
		strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf(
			"environment variable %s is invalid",
			administratorNoticeContentEnvironment,
		)
	}
	return raw, nil
}

func parseAdministratorNoticeHash(raw, content string) (string, error) {
	if len(raw) != sha256.Size*2 {
		return "", fmt.Errorf(
			"environment variable %s is invalid",
			administratorNoticeSHA256Environment,
		)
	}
	for _, character := range raw {
		if !(character >= '0' && character <= '9' ||
			character >= 'a' && character <= 'f') {
			return "", fmt.Errorf(
				"environment variable %s is invalid",
				administratorNoticeSHA256Environment,
			)
		}
	}
	digest := sha256.Sum256([]byte(content))
	expected := fmt.Sprintf("%x", digest)
	if subtle.ConstantTimeCompare([]byte(raw), []byte(expected)) != 1 {
		return "", fmt.Errorf(
			"environment variable %s does not match %s",
			administratorNoticeSHA256Environment,
			administratorNoticeContentEnvironment,
		)
	}
	return raw, nil
}

func parseIdempotencyKeyring(getenv func(string) string) (idempotency.DigestKeyring, error) {
	current := getenv(idempotencyCurrentKeyVersionEnvironment)
	if !validIdempotencyVersion(current) || current == "none" {
		return idempotency.DigestKeyring{}, fmt.Errorf(
			"environment variable %s is invalid",
			idempotencyCurrentKeyVersionEnvironment,
		)
	}
	retainedRaw := getenv(idempotencyRetainedKeyVersionsEnvironment)
	var retained []string
	if retainedRaw != "none" {
		retained = strings.Split(retainedRaw, ",")
		if len(retained) < 1 || len(retained) > 16 {
			return idempotency.DigestKeyring{}, fmt.Errorf(
				"environment variable %s has an invalid item count",
				idempotencyRetainedKeyVersionsEnvironment,
			)
		}
		for index, version := range retained {
			if !validIdempotencyVersion(version) || version == "none" ||
				version == current ||
				index > 0 && retained[index-1] >= version {
				return idempotency.DigestKeyring{}, fmt.Errorf(
					"environment variable %s is invalid",
					idempotencyRetainedKeyVersionsEnvironment,
				)
			}
		}
	}
	versions := append([]string{current}, retained...)
	keyring := idempotency.DigestKeyring{
		ActiveVersion: current,
		Keys:          make(map[string][]byte, len(versions)),
	}
	completed := false
	defer func() {
		if !completed {
			clearDigestKeyring(&keyring)
		}
	}()
	for _, version := range versions {
		keyName := idempotencyKeyPrefix + version
		key, err := parseSecret(getenv(keyName), keyName)
		if err != nil {
			return idempotency.DigestKeyring{}, err
		}
		for existingVersion, existing := range keyring.Keys {
			if bytes.Equal(key, existing) {
				clear(key)
				return idempotency.DigestKeyring{}, fmt.Errorf(
					"environment variables %s and %s must use independent key material",
					keyName,
					idempotencyKeyPrefix+existingVersion,
				)
			}
		}
		keyring.Keys[version] = key
	}
	completed = true
	return keyring, nil
}

func validIdempotencyVersion(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for index := range value {
		character := value[index]
		if !(character >= 'A' && character <= 'Z' ||
			character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '_') {
			return false
		}
	}
	return true
}

func parseSMTPMailbox(raw string) (string, error) {
	if raw == "" || raw != strings.TrimSpace(raw) ||
		strings.ContainsAny(raw, "\r\n") {
		return "", fmt.Errorf("environment variable %s is invalid", smtpFromEnvironment)
	}
	address, err := mail.ParseAddress(raw)
	if err != nil || address.Address != raw {
		return "", fmt.Errorf("environment variable %s is invalid", smtpFromEnvironment)
	}
	return raw, nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func validateSMTPDeploymentConfig(
	lookup platform.EnvironmentLookup,
	config platform.DeploymentConfig,
) error {
	host, hostExists := lookup(platform.EnvSMTPHost)
	if !hostExists || !validSMTPHost(host) || config.SMTPHost != host {
		return fmt.Errorf("environment variable %s is invalid", platform.EnvSMTPHost)
	}
	serverName, serverNameExists := lookup(platform.EnvSMTPServerName)
	if !serverNameExists || !validSMTPHost(serverName) ||
		config.SMTPServerName != serverName {
		return fmt.Errorf(
			"environment variable %s is invalid",
			platform.EnvSMTPServerName,
		)
	}
	caPath, caExists := lookup(platform.EnvSMTPCAPath)
	if !caExists || caPath == "" {
		if config.SMTPCAPath != "" {
			return fmt.Errorf("environment variable %s is invalid", platform.EnvSMTPCAPath)
		}
		return nil
	}
	if caPath != strings.TrimSpace(caPath) || !filepath.IsAbs(caPath) ||
		filepath.Clean(caPath) != caPath || config.SMTPCAPath != caPath {
		return fmt.Errorf("environment variable %s is invalid", platform.EnvSMTPCAPath)
	}
	return nil
}

func validSMTPHost(value string) bool {
	if len(value) < 1 || len(value) > 253 || value != strings.TrimSpace(value) {
		return false
	}
	for index := range value {
		if value[index] > 0x7f || value[index] == '/' || value[index] == '@' ||
			value[index] == '[' || value[index] == ']' ||
			value[index] == '%' || value[index] == '\\' {
			return false
		}
	}
	if net.ParseIP(value) != nil {
		return true
	}
	if strings.Contains(value, ":") || strings.HasPrefix(value, ".") ||
		strings.HasSuffix(value, ".") {
		return false
	}
	labels := strings.Split(value, ".")
	for _, label := range labels {
		if len(label) < 1 || len(label) > 63 ||
			label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for index := range label {
			character := label[index]
			if !(character >= 'A' && character <= 'Z' ||
				character >= 'a' && character <= 'z' ||
				character >= '0' && character <= '9' ||
				character == '-') {
				return false
			}
		}
	}
	return true
}

func parseRequiredInt(getenv func(string) string, key string, minimum, maximum int64) (int64, error) {
	original := getenv(key)
	raw := strings.TrimSpace(original)
	value, err := strconv.ParseInt(raw, 10, 64)
	if raw == "" || raw != original || err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("environment variable %s is outside the accepted range", key)
	}
	return value, nil
}

func parseRequiredUint(getenv func(string) string, key string, minimum, maximum uint64, bits int) (uint64, error) {
	original := getenv(key)
	raw := strings.TrimSpace(original)
	value, err := strconv.ParseUint(raw, 10, bits)
	if raw == "" || raw != original || err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("environment variable %s is outside the accepted range", key)
	}
	return value, nil
}

func parseToken(raw, key string, punctuation bool) (string, error) {
	original := raw
	raw = strings.TrimSpace(raw)
	if len(raw) < 1 || len(raw) > 64 {
		return "", fmt.Errorf("environment variable %s is invalid", key)
	}
	if raw != original {
		return "", fmt.Errorf("environment variable %s is invalid", key)
	}
	for index, char := range raw {
		valid := char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' ||
			(!punctuation && char == '_') ||
			(punctuation && index > 0 && (char == '.' || char == '_' || char == '-'))
		if !valid {
			return "", fmt.Errorf("environment variable %s is invalid", key)
		}
	}
	return raw, nil
}

func parseSecret(raw, key string) ([]byte, error) {
	if len(raw) < 32 || len(raw) > 64 {
		return nil, fmt.Errorf("environment variable %s has an invalid length", key)
	}
	return []byte(raw), nil
}

func parseTrustedProxyConfiguration(raw string) ([]netip.Prefix, error) {
	if len(raw) == 0 || len(raw) > 4096 || raw != strings.TrimSpace(raw) {
		return nil, fmt.Errorf("environment variable %s is invalid", platform.EnvTrustedProxyCIDRs)
	}
	if raw == "none" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) < 1 || len(parts) > 64 {
		return nil, fmt.Errorf("environment variable %s has an invalid item count", platform.EnvTrustedProxyCIDRs)
	}
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		if part == "" || part != strings.TrimSpace(part) {
			return nil, fmt.Errorf("environment variable %s contains an invalid prefix", platform.EnvTrustedProxyCIDRs)
		}
		prefix, err := netip.ParsePrefix(part)
		if err != nil || prefix.String() != part || prefix != prefix.Masked() || prefix.Bits() == 0 {
			return nil, fmt.Errorf("environment variable %s contains an invalid prefix", platform.EnvTrustedProxyCIDRs)
		}
		if prefix.Addr().Is4In6() {
			if prefix.Bits() < 96 {
				return nil, fmt.Errorf("environment variable %s contains an invalid mapped prefix", platform.EnvTrustedProxyCIDRs)
			}
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96).Masked()
		}
		if !prefix.IsValid() || prefix != prefix.Masked() || prefix.Bits() == 0 {
			return nil, fmt.Errorf("environment variable %s contains an invalid prefix", platform.EnvTrustedProxyCIDRs)
		}
		for _, existing := range prefixes {
			if existing == prefix {
				return nil, fmt.Errorf("environment variable %s contains a duplicate prefix", platform.EnvTrustedProxyCIDRs)
			}
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}

func loadDeploymentConfig(lookup platform.EnvironmentLookup, proxies []netip.Prefix) (platform.DeploymentConfig, error) {
	proxyValue := "192.0.2.1/32"
	if len(proxies) > 0 {
		values := make([]string, len(proxies))
		for index, prefix := range proxies {
			values[index] = prefix.String()
		}
		proxyValue = strings.Join(values, ",")
	}
	wrapped := func(key string) (string, bool) {
		if key == platform.EnvTrustedProxyCIDRs {
			return proxyValue, true
		}
		return lookup(key)
	}
	config, err := platform.LoadDeploymentConfig(wrapped)
	if err != nil {
		return platform.DeploymentConfig{}, err
	}
	config.TrustedProxyCIDRs = append([]netip.Prefix(nil), proxies...)
	return config, nil
}

func parseRequiredDuration(raw, key string, minimum, maximum time.Duration) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("required environment variable %s is not set", key)
	}
	value, err := time.ParseDuration(trimmed)
	if err != nil || raw != trimmed || value < minimum || value > maximum {
		return 0, fmt.Errorf("environment variable %s is outside the accepted range", key)
	}
	return value, nil
}

type loginMetricsAdapter struct {
	registry *platform.MetricsRegistry
}

func (adapter loginMetricsAdapter) RecordLogin(
	ctx context.Context,
	observation identity.LoginMetricObservation,
) error {
	if adapter.registry == nil {
		return errors.New("login metrics registry is required")
	}
	if err := adapter.registry.RecordLoginAttempt(
		ctx,
		observation.Endpoint,
		string(observation.Outcome),
		observation.ReasonBucket,
		observation.ErrorCode,
	); err != nil {
		return err
	}
	if observation.Outcome == identity.LoginOutcomeInternalFailure {
		if err := adapter.registry.RecordLoginInternalFailure(
			ctx,
			observation.Endpoint,
			observation.ReasonBucket,
			observation.ErrorCode,
			string(observation.FailureStage),
		); err != nil {
			return err
		}
	}
	return adapter.registry.RecordLoginDuration(
		ctx,
		observation.Endpoint,
		string(observation.Outcome),
		observation.ReasonBucket,
		observation.ErrorCode,
		observation.Duration.Seconds(),
	)
}

type originMetricsAdapter struct {
	registry *platform.MetricsRegistry
}

func (adapter originMetricsAdapter) RecordOriginRejection(
	ctx context.Context,
	observation platform.OriginRejectionObservation,
) error {
	if adapter.registry == nil {
		return errors.New("origin metrics registry is required")
	}
	return adapter.registry.RecordOriginRejection(
		ctx,
		string(observation.Reason),
		string(observation.ResponseWrite),
	)
}

type boundedLoginFailureAuditor struct {
	delegate identity.LoginFailureAuditor
	budget   time.Duration
}

func (auditor boundedLoginFailureAuditor) RecordOnce(
	ctx context.Context,
	request identity.LoginFailureAuditRequest,
) (identity.LoginAuditPersistence, error) {
	if auditor.delegate == nil || auditor.budget <= 0 {
		return identity.LoginAuditPersistedUnknown, errors.New("login failure auditor is incomplete")
	}
	bounded, cancel := context.WithTimeout(ctx, auditor.budget)
	defer cancel()
	return auditor.delegate.RecordOnce(bounded, request)
}

func prepareProductionLoginDependencies(
	config startupConfig,
	random io.Reader,
) (prepared *preparedLoginDependencies, err error) {
	hasher, err := security.NewArgon2idHasher(config.argon2, random)
	if err != nil {
		return nil, errors.New("create credential hasher")
	}
	gate, err := security.NewArgon2idProcessGate(config.argon2MaxConcurrency)
	if err != nil {
		return nil, errors.New("create credential process gate")
	}
	passwords, err := security.NewGatedArgon2idHasher(hasher, gate)
	if err != nil {
		return nil, errors.New("create gated credential hasher")
	}
	digester, err := identity.NewHMACLoginAbuseDigester(
		config.loginBucketKeyVersion,
		config.loginBucketKey,
	)
	if err != nil {
		return nil, errors.New("create login abuse digester")
	}
	ids := &identity.RandomUUIDGenerator{}
	clock := &identity.SystemClock{}
	issuer, err := identity.NewRandomLoginSessionIssuer(
		random,
		ids,
		clock,
		config.sessionTokenEntropy,
		config.csrfTokenEntropy,
		config.sessionAbsoluteTTL,
		config.sessionIdleTTL,
	)
	if err != nil {
		return nil, errors.New("create login session issuer")
	}
	source, err := identity.NewTrustedSourceEvidenceProvider(
		config.sourceKey,
		config.sourceKeyVersion,
		config.deployment.TrustedProxyCIDRs,
	)
	if err != nil {
		return nil, errors.New("create login source provider")
	}
	recoveryMode, err := identity.NewConfiguredLoginRecoveryMode(config.recoveryMode)
	if err != nil {
		return nil, errors.New("create login recovery mode")
	}
	probeBytes := make([]byte, config.sessionTokenEntropy)
	defer clear(probeBytes)
	if _, err := io.ReadFull(random, probeBytes); err != nil {
		return nil, errors.New("create fake login credential")
	}
	probe := base64.RawURLEncoding.EncodeToString(probeBytes)
	digest, err := passwords.Hash(probe)
	if err != nil {
		return nil, errors.New("hash fake login credential")
	}

	keyring, err := cloneDigestKeyring(config.idempotencyKeys)
	if err != nil {
		return nil, errors.New("create idempotency keyring")
	}
	prepared = &preparedLoginDependencies{
		passwords: passwords, sessionIssuer: issuer,
		abuseDigester: digester, abuseConfig: config.loginAbuse,
		source: source, recoveryMode: recoveryMode,
		ids: ids, clock: clock, fakeDigest: digest, fakeProbe: probe,
		administratorNotices: identity.StaticAdministratorInformationNoticeReader{
			Notice: config.administratorNotice,
		},
		idempotencyKeys: keyring,
		idempotencyLock: config.idempotencyLock,
		idempotencyTTL:  config.idempotencyTTL,
	}
	defer func() {
		if err != nil {
			prepared.cleanup()
			prepared = nil
		}
	}()

	rootCAs, rootErr := x509.SystemCertPool()
	if rootErr != nil || rootCAs == nil {
		err = fmt.Errorf("load system certificate roots for %s", platform.EnvSMTPCAPath)
		return nil, err
	}
	if config.deployment.SMTPCAPath != "" {
		pemBytes, readErr := os.ReadFile(config.deployment.SMTPCAPath)
		if readErr != nil {
			err = fmt.Errorf("read environment variable %s certificate bundle", platform.EnvSMTPCAPath)
			return nil, err
		}
		validBundle := validCertificateBundle(pemBytes)
		appended := validBundle && rootCAs.AppendCertsFromPEM(pemBytes)
		clear(pemBytes)
		if !appended {
			err = fmt.Errorf("environment variable %s certificate bundle is invalid", platform.EnvSMTPCAPath)
			return nil, err
		}
	}
	var auth smtp.Auth
	if config.smtpAuthMode == "plain" {
		auth = smtp.PlainAuth(
			"",
			config.smtpUsername,
			config.smtpPassword,
			config.deployment.SMTPHost,
		)
	}
	smtpMailer, mailerErr := mailer.NewSMTPMailer(
		mailer.SMTPConfig{
			Address: net.JoinHostPort(
				config.deployment.SMTPHost,
				strconv.FormatUint(uint64(config.smtpPort), 10),
			),
			Host: config.deployment.SMTPHost,
			From: config.smtpFrom,
			Mode: mailer.TLSMode(config.deployment.SMTPTLSMode),
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: config.deployment.SMTPServerName,
				RootCAs:    rootCAs,
			},
			Auth: auth,
		},
		&net.Dialer{},
	)
	if mailerErr != nil {
		err = errors.New("create SMTP mailer")
		return nil, err
	}
	prepared.administratorMailer = smtpMailer
	return prepared, nil
}

func validCertificateBundle(contents []byte) bool {
	remaining := contents
	certificates := 0
	for {
		remaining = bytes.TrimSpace(remaining)
		if len(remaining) == 0 {
			return certificates > 0
		}
		if !bytes.HasPrefix(remaining, []byte("-----BEGIN CERTIFICATE-----")) {
			return false
		}
		block, rest := pem.Decode(remaining)
		if block == nil {
			return false
		}
		if block.Type != "CERTIFICATE" {
			return false
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return false
		}
		certificates++
		remaining = rest
	}
}

func clearStartupSecrets(config *startupConfig) {
	if config == nil {
		return
	}
	clear(config.loginBucketKey)
	clear(config.sourceKey)
	config.loginBucketKey = nil
	config.sourceKey = nil
	clearDigestKeyring(&config.idempotencyKeys)
	clearArticleDeletionKeyring(&config.articleDeletion.keyring)
	config.smtpUsername = ""
	config.smtpPassword = ""
}

func prepareArticleDeletion(
	config articleDeletionStartupConfig,
) (*preparedArticleDeletion, error) {
	if err := article.ValidateArticleDeletionJournalRoot(config.journalRoot); err != nil {
		return nil, err
	}
	keyring, err := cloneArticleDeletionKeyring(config.keyring)
	if err != nil {
		return nil, err
	}
	config.keyring = keyring
	return &preparedArticleDeletion{config: config}, nil
}

func openArticleDeletionJournal(
	prepared *preparedArticleDeletion,
) (articleDeletionJournalOwner, error) {
	if prepared == nil {
		return nil, errors.New("prepared Article deletion configuration is required")
	}
	journal, err := article.NewProductionFileArticleDeletionJournal(
		prepared.config.journalRoot,
		prepared.config.fileMode,
		prepared.config.keyring,
	)
	clearArticleDeletionKeyring(&prepared.config.keyring)
	if err != nil {
		return nil, sanitizeArticleDeletionJournalError(err)
	}
	return journal, nil
}

func sanitizeArticleDeletionJournalError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, article.ErrDeletionJournalBusy):
		return article.ErrDeletionJournalBusy
	case errors.Is(err, article.ErrDeletionJournalCorrupt):
		return article.ErrDeletionJournalCorrupt
	case errors.Is(err, article.ErrDeletionJournalUnsupportedPlatform):
		return article.ErrDeletionJournalUnsupportedPlatform
	case errors.Is(err, article.ErrDeletionTombstoneNotFound):
		return article.ErrDeletionTombstoneNotFound
	default:
		return errArticleDeletionJournalUnavailable
	}
}

func cloneArticleDeletionKeyring(
	source article.DeletionJournalKeyring,
) (article.DeletionJournalKeyring, error) {
	if source.Active.ID != "article-deletion-hmac" ||
		source.Active.Version == "" || len(source.Active.Material) == 0 {
		return article.DeletionJournalKeyring{}, errors.New("Article deletion keyring is invalid")
	}
	cloned := article.DeletionJournalKeyring{
		Active: article.DeletionJournalKey{
			ID:       source.Active.ID,
			Version:  source.Active.Version,
			Material: append([]byte(nil), source.Active.Material...),
		},
		Retained: make([]article.DeletionJournalKey, 0, len(source.Retained)),
	}
	seen := map[string]struct{}{source.Active.Version: {}}
	for _, key := range source.Retained {
		if key.ID != "article-deletion-hmac" || key.Version == "" ||
			len(key.Material) == 0 {
			clearArticleDeletionKeyring(&cloned)
			return article.DeletionJournalKeyring{}, errors.New(
				"Article deletion keyring is invalid",
			)
		}
		if _, exists := seen[key.Version]; exists {
			clearArticleDeletionKeyring(&cloned)
			return article.DeletionJournalKeyring{}, errors.New(
				"Article deletion keyring is invalid",
			)
		}
		seen[key.Version] = struct{}{}
		cloned.Retained = append(cloned.Retained, article.DeletionJournalKey{
			ID:       key.ID,
			Version:  key.Version,
			Material: append([]byte(nil), key.Material...),
		})
	}
	return cloned, nil
}

func clearArticleDeletionKeyring(keyring *article.DeletionJournalKeyring) {
	if keyring == nil {
		return
	}
	clear(keyring.Active.Material)
	keyring.Active = article.DeletionJournalKey{}
	for index := range keyring.Retained {
		clear(keyring.Retained[index].Material)
		keyring.Retained[index] = article.DeletionJournalKey{}
	}
	keyring.Retained = nil
}

func (prepared *preparedArticleDeletion) cleanup() {
	if prepared == nil {
		return
	}
	clearArticleDeletionKeyring(&prepared.config.keyring)
}

func cloneDigestKeyring(source idempotency.DigestKeyring) (idempotency.DigestKeyring, error) {
	if !validIdempotencyVersion(source.ActiveVersion) ||
		source.ActiveVersion == "none" ||
		len(source.Keys) == 0 {
		return idempotency.DigestKeyring{}, errors.New("idempotency keyring is invalid")
	}
	cloned := idempotency.DigestKeyring{
		ActiveVersion: source.ActiveVersion,
		Keys:          make(map[string][]byte, len(source.Keys)),
	}
	for version, key := range source.Keys {
		if !validIdempotencyVersion(version) || len(key) < 32 || len(key) > 64 {
			clearDigestKeyring(&cloned)
			return idempotency.DigestKeyring{}, errors.New("idempotency keyring is invalid")
		}
		cloned.Keys[version] = append([]byte(nil), key...)
	}
	if _, exists := cloned.Keys[cloned.ActiveVersion]; !exists {
		clearDigestKeyring(&cloned)
		return idempotency.DigestKeyring{}, errors.New("idempotency keyring is invalid")
	}
	return cloned, nil
}

func clearDigestKeyring(keyring *idempotency.DigestKeyring) {
	if keyring == nil {
		return
	}
	for version, key := range keyring.Keys {
		clear(key)
		delete(keyring.Keys, version)
	}
	keyring.ActiveVersion = ""
	keyring.Keys = nil
}

func (prepared *preparedLoginDependencies) cleanup() {
	if prepared == nil {
		return
	}
	clearDigestKeyring(&prepared.idempotencyKeys)
	prepared.administratorMailer = nil
	prepared.administratorNotices = nil
	prepared.fakeProbe = ""
}

type productionBusinessBuilder struct {
	processingGate        *privacy.ProcessingGate
	noticeWorker          *article.TransferNoticeDispatcher
	articleIdentity       *articleIdentityLifecycleAdapter
	pool                  *pgxpool.Pool
	composition           businessComposition
	prepared              *preparedLoginDependencies
	passwordResetGate     *identity.PasswordResetStore
	passwordResetPermit   *identity.PasswordResetConnectionPermit
	passwordResetOperator *identity.PasswordResetOperator
	passwordResetWorker   *identity.PasswordResetRecoveryWorker
	sessionStore          *session.Store
	auditStore            *audit.Store
}

type productionIdentityHandlers struct {
	login                     http.Handler
	password                  http.Handler
	administratorNotice       http.Handler
	administratorCreate       http.Handler
	administratorPrivacy      http.Handler
	passwordReset             http.Handler
	temporaryCredentialResend http.Handler
	logout                    http.Handler
	authorizer                platform.HTTPAuthorizer
}

type productionArticleComponents struct {
	handler *article.Handler
	worker  *article.DeletionWorker
	gate    articleArticleDeletionGate
}

// articleArticleDeletionGate is the recovery consumer seam for the article
// replay provider.
type articleArticleDeletionGate interface {
	ReplayAndVerify(context.Context, string, string) ([]byte, error)
	ReadStatus(context.Context, string, string) ([]byte, error)
}

func newProductionBusinessHandler(
	pool *pgxpool.Pool,
	composition businessComposition,
) (http.Handler, error) {
	builder, err := newProductionBusinessBuilder(pool, composition)
	if err != nil {
		return nil, err
	}
	identityHandlers, err := builder.buildIdentityHandlers()
	if err != nil {
		return nil, err
	}
	articleHandlers, err := builder.buildArticleHandlers(identityHandlers.authorizer)
	if err != nil {
		return nil, err
	}
	routerHandlers := [23]http.Handler{
		identityHandlers.password,
		identityHandlers.administratorNotice,
		identityHandlers.administratorCreate,
		identityHandlers.administratorPrivacy,
	}
	copy(routerHandlers[4:7], articleHandlers[:3])
	routerHandlers[7] = identityHandlers.passwordReset
	routerHandlers[8] = identityHandlers.temporaryCredentialResend
	routerHandlers[9] = articleHandlers[3]
	routerHandlers[10] = articleHandlers[4]
	routerHandlers[11] = articleHandlers[5]
	routerHandlers[12] = articleHandlers[6]
	copy(routerHandlers[13:], articleHandlers[7:])
	routerHandlers[22] = identityHandlers.logout
	var trace func(*http.Request)
	if composition.articleTestSeam != nil {
		trace = composition.articleTestSeam.trace
	}
	return newLoginBusinessRouterWithTraceAndRecovery(
		identityHandlers.login,
		trace,
		composition.recoveryHandle,
		routerHandlers[:]...,
	), nil
}

func newProductionBusinessBuilder(
	pool *pgxpool.Pool,
	composition businessComposition,
) (*productionBusinessBuilder, error) {
	if err := validateProductionBusinessComposition(pool, composition); err != nil {
		return nil, err
	}
	observeArticleSharedDependency(composition, "logger", "composition", composition.logger)
	observeArticleSharedDependency(composition, "origin_guard", "identity", composition.originGuard)
	passwordResetGate := &identity.PasswordResetStore{Idempotency: idempotency.Store{}}
	sessionStoreValue, err := session.NewStore(session.StoreDependencies{
		Transactions:      pool,
		PasswordResetGate: passwordResetGate,
	})
	if err != nil {
		return nil, errors.New("create login session store")
	}
	builder := &productionBusinessBuilder{
		pool: pool, composition: composition, prepared: composition.prepared,
		passwordResetGate: passwordResetGate, sessionStore: &sessionStoreValue,
		auditStore: &audit.Store{},
	}
	if err := builder.buildPasswordResetRecovery(); err != nil {
		return nil, err
	}
	if err := builder.buildNoticeDispatcher(); err != nil {
		return nil, err
	}
	builder.observeIdentityCore()
	return builder, nil
}

func validateProductionBusinessComposition(
	pool *pgxpool.Pool,
	composition businessComposition,
) error {
	if pool == nil || composition.prepared == nil ||
		composition.logger == nil || composition.metrics == nil ||
		composition.loginObserver == nil || composition.originGuard == nil ||
		composition.failureAuditor == nil || composition.failureAuditFor <= 0 ||
		composition.loginObserveFor <= 0 || composition.loginMaxBodyBytes <= 0 ||
		composition.smtpOperationBudget <= 0 ||
		composition.passwordResetWorker.LeaseDuration <= 0 ||
		composition.passwordResetWorker.RetryBackoff <= 0 ||
		composition.passwordResetWorker.MaxRetries <= 0 ||
		composition.passwordResetWorker.ClaimBatchLimit <= 0 ||
		composition.passwordResetWorker.ConnectionBudget <= 0 ||
		composition.articleQuery.DefaultLimit <= 0 ||
		composition.articleQuery.MaxLimit < composition.articleQuery.DefaultLimit ||
		composition.articleQuery.CandidateBatchSize <= composition.articleQuery.MaxLimit ||
		composition.prepared.administratorNotices == nil ||
		composition.prepared.administratorMailer == nil ||
		composition.prepared.idempotencyLock <= 0 ||
		composition.prepared.idempotencyTTL <= composition.prepared.idempotencyLock {
		return errors.New("production login composition is incomplete")
	}
	return nil
}

func (b *productionBusinessBuilder) buildPasswordResetRecovery() error {
	permit, err := identity.NewPasswordResetConnectionPermit(b.composition.passwordResetWorker.ConnectionBudget)
	if err != nil {
		return errors.New("create password reset connection permit")
	}
	operator, err := identity.NewPasswordResetOperator(identity.PasswordResetOperatorDependencies{Transactions: b.pool, Store: b.passwordResetGate, Audit: b.auditStore, IDs: b.prepared.ids, Clock: b.prepared.clock, Permit: permit})
	if err != nil {
		return errors.New("create password reset operator")
	}
	workerID, err := b.prepared.ids.NewID()
	if err != nil {
		return errors.New("create password reset worker id")
	}
	worker, err := identity.NewPasswordResetRecoveryWorker(identity.PasswordResetWorkerDependencies{Pool: b.pool, Store: b.passwordResetGate, Operator: operator, Permit: permit, WorkerID: workerID, Clock: b.prepared.clock}, b.composition.passwordResetWorker)
	if err != nil {
		return errors.New("create password reset worker")
	}
	b.passwordResetPermit = permit
	b.passwordResetOperator = operator
	b.passwordResetWorker = worker
	if b.composition.runtime != nil {
		b.composition.runtime.passwordResetWorker = worker
	}
	observeArticleSharedDependency(b.composition, "password_reset_operator_store", "operator", b.passwordResetGate)
	observeArticleSharedDependency(b.composition, "password_reset_worker_store", "worker", b.passwordResetGate)
	observeArticleSharedDependency(b.composition, "password_reset_operator", "composition", operator)
	observeArticleSharedDependency(b.composition, "password_reset_worker", "composition", worker)
	observeArticleSharedDependency(b.composition, "password_reset_permit", "composition", permit)
	return nil
}

func (b *productionBusinessBuilder) observeIdentityCore() {
	observeArticleSharedDependency(b.composition, "pool", "login", b.pool)
	observeArticleSharedDependency(b.composition, "session_store", "login", b.sessionStore)
	observeArticleSharedDependency(
		b.composition,
		"password_reset_gate",
		"session",
		b.passwordResetGate,
	)
	observeArticleSharedDependency(b.composition, "audit_store", "login", b.auditStore)
	observeArticleSharedDependency(b.composition, "id_provider", "login", b.prepared.ids)
	observeArticleSharedDependency(b.composition, "clock", "login", b.prepared.clock)
}

func (b *productionBusinessBuilder) buildIdentityHandlers() (productionIdentityHandlers, error) {
	loginHandler, passwordHandler, authorizer, err := b.newCoreIdentityHandlers()
	if err != nil {
		return productionIdentityHandlers{}, err
	}
	administratorHandler, administratorCreate, err :=
		b.newAdministratorHandlerWithCoordinator()
	if err != nil {
		return productionIdentityHandlers{}, err
	}
	passwordResetHandler, resendHandler, err :=
		b.newCredentialRecoveryHandlers(administratorCreate)
	if err != nil {
		return productionIdentityHandlers{}, err
	}
	logoutHandler, err := identity.NewSessionLogoutHandler(
		postgresSessionLogout{pool: b.pool, store: b.sessionStore},
		time.Now,
	)
	if err != nil {
		return productionIdentityHandlers{}, errors.New("create session logout handler")
	}
	return b.protectIdentityHandlers(
		loginHandler,
		passwordHandler,
		administratorHandler,
		passwordResetHandler,
		resendHandler,
		logoutHandler,
		authorizer,
	), nil
}

func (b *productionBusinessBuilder) newCoreIdentityHandlers() (
	*identity.Handler,
	*identity.Handler,
	platform.HTTPAuthorizer,
	error,
) {
	loginService, err := b.newLoginService()
	if err != nil {
		return nil, nil, platform.HTTPAuthorizer{}, err
	}
	passwordService, err := b.newPasswordChangeService()
	if err != nil {
		return nil, nil, platform.HTTPAuthorizer{}, err
	}
	b.observePasswordDependencies()
	loginHandler, err := b.newLoginHandler(loginService)
	if err != nil {
		return nil, nil, platform.HTTPAuthorizer{}, err
	}
	passwordHandler, err := identity.NewPasswordChangeHandler(
		passwordService,
		b.composition.loginMaxBodyBytes,
	)
	if err != nil {
		return nil, nil, platform.HTTPAuthorizer{},
			errors.New("create password change handler")
	}
	authorizer, err := b.newAuthorizer()
	if err != nil {
		return nil, nil, platform.HTTPAuthorizer{}, err
	}
	return loginHandler, passwordHandler, authorizer, nil
}

func (b *productionBusinessBuilder) newCredentialRecoveryHandlers(
	administratorCreate *identity.AdministratorCreateCoordinator,
) (*identity.Handler, *identity.Handler, error) {
	if administratorCreate == nil {
		return nil, nil, errors.New(
			"administrator creation coordinator is required",
		)
	}
	passwordResetCoordinator, err := b.newPasswordResetCoordinator()
	if err != nil {
		return nil, nil, err
	}
	passwordResetHandler, err := b.newPasswordResetHandlerFor(
		passwordResetCoordinator,
	)
	if err != nil {
		return nil, nil, err
	}
	resendService, err := identity.NewTemporaryCredentialResendService(
		passwordResetCoordinator,
		administratorCreate,
	)
	if err != nil {
		return nil, nil, errors.New(
			"create temporary credential resend service",
		)
	}
	var temporaryCredentialResend identity.TemporaryCredentialResendService = resendService
	if b.composition.articleTestSeam != nil &&
		b.composition.articleTestSeam.temporaryCredentialResend != nil {
		temporaryCredentialResend =
			b.composition.articleTestSeam.temporaryCredentialResend
	}
	resendHandler, err := identity.NewTemporaryCredentialResendHandler(
		temporaryCredentialResend,
	)
	if err != nil {
		return nil, nil, errors.New(
			"create temporary credential resend handler",
		)
	}
	observeArticleSharedDependency(
		b.composition,
		"password_reset_coordinator",
		"temporary_credential_resend",
		passwordResetCoordinator,
	)
	observeArticleSharedDependency(
		b.composition,
		"administrator_create_coordinator",
		"temporary_credential_resend",
		administratorCreate,
	)
	return passwordResetHandler, resendHandler, nil
}

func (b *productionBusinessBuilder) newPasswordResetHandler() (*identity.Handler, error) {
	coordinator, err := b.newPasswordResetCoordinator()
	if err != nil {
		return nil, err
	}
	return b.newPasswordResetHandlerFor(coordinator)
}

func (b *productionBusinessBuilder) newPasswordResetCoordinator() (
	*identity.PasswordResetCoordinator,
	error,
) {
	if b.passwordResetGate == nil || b.passwordResetPermit == nil ||
		b.passwordResetOperator == nil || b.passwordResetWorker == nil {
		return nil, errors.New("password reset durable dependencies are incomplete")
	}
	secrets, err := identity.NewCryptoTemporaryPasswordGenerator(
		rand.Reader,
		passwordResetTemporaryPasswordCharacters,
	)
	if err != nil {
		return nil, errors.New("create password reset secret generator")
	}
	coordinator, err := identity.NewPasswordResetService(identity.PasswordResetDependencies{
		Transactions: b.pool, Repository: identity.PostgresPasswordResetRepository{},
		Store: b.passwordResetGate, PasswordPolicy: security.PasswordPolicy{},
		PasswordHasher: b.prepared.passwords, Secrets: secrets,
		Mailer:              b.prepared.administratorMailer,
		IdempotencyKeys:     b.prepared.idempotencyKeys,
		IdempotencyLock:     b.prepared.idempotencyLock,
		IdempotencyTTL:      b.prepared.idempotencyTTL,
		SMTPOperationBudget: b.composition.smtpOperationBudget,
		Audit:               b.auditStore, IDs: b.prepared.ids, Clock: b.prepared.clock,
	})
	if err != nil {
		return nil, errors.New("create password reset coordinator")
	}
	return coordinator, nil
}

func (b *productionBusinessBuilder) newPasswordResetHandlerFor(
	coordinator *identity.PasswordResetCoordinator,
) (*identity.Handler, error) {
	if coordinator == nil {
		return nil, errors.New("password reset coordinator is required")
	}
	var service identity.PasswordResetService = coordinator
	if b.composition.articleTestSeam != nil &&
		b.composition.articleTestSeam.passwordReset != nil {
		service = b.composition.articleTestSeam.passwordReset
	}
	handler, err := identity.NewPasswordResetHandler(service)
	if err != nil {
		return nil, errors.New("create password reset handler")
	}
	observeArticleSharedDependency(
		b.composition,
		"password_reset_gate",
		"password_reset",
		b.passwordResetGate,
	)
	observeArticleSharedDependency(
		b.composition,
		"password_reset_coordinator",
		"password_reset",
		coordinator,
	)
	return handler, nil
}

func (b *productionBusinessBuilder) newLoginService() (*identity.Service, error) {
	abuse, err := identity.NewPersistentLoginAbuseProtector(
		identity.PostgresLoginAbuseBucketStore{},
		b.prepared.abuseDigester,
		b.prepared.abuseConfig,
	)
	if err != nil {
		return nil, errors.New("create login abuse protector")
	}
	service, err := identity.NewLoginService(identity.LoginDependencies{
		Transactions: b.pool, Repository: identity.PostgresLoginRepository{},
		PasswordResetGate: b.passwordResetGate, Passwords: b.prepared.passwords,
		Sessions: b.sessionStore, SessionIssuer: b.prepared.sessionIssuer,
		Abuse: abuse, RecoveryMode: b.prepared.recoveryMode, Audit: b.auditStore,
		IDs: b.prepared.ids, Clock: b.prepared.clock,
		FakeCredentialDigest: b.prepared.fakeDigest,
		FakeCredentialProbe:  b.prepared.fakeProbe,
	})
	b.prepared.fakeProbe = ""
	if err != nil {
		return nil, errors.New("create login service")
	}
	return service, nil
}

func (b *productionBusinessBuilder) newPasswordChangeService() (*identity.Service, error) {
	service, err := identity.NewPasswordChangeService(identity.PasswordChangeDependencies{
		Transactions: b.pool, Repository: identity.PostgresPasswordChangeRepository{},
		PasswordResetGate: b.passwordResetGate, Passwords: b.prepared.passwords,
		Policy:        security.PasswordPolicy{},
		Sessions:      identity.PostgresPasswordChangeSessions{Store: *b.sessionStore},
		SessionIssuer: b.prepared.sessionIssuer, Audit: b.auditStore,
		IDs: b.prepared.ids, Clock: b.prepared.clock,
	})
	if err != nil {
		return nil, errors.New("create password change service")
	}
	return service, nil
}

func (b *productionBusinessBuilder) observePasswordDependencies() {
	observeArticleSharedDependency(b.composition, "pool", "password", b.pool)
	observeArticleSharedDependency(b.composition, "session_store", "password", b.sessionStore)
	observeArticleSharedDependency(
		b.composition,
		"password_reset_gate",
		"login",
		b.passwordResetGate,
	)
	observeArticleSharedDependency(
		b.composition,
		"password_reset_gate",
		"password",
		b.passwordResetGate,
	)
	observeArticleSharedDependency(b.composition, "audit_store", "password", b.auditStore)
	observeArticleSharedDependency(b.composition, "id_provider", "password", b.prepared.ids)
	observeArticleSharedDependency(b.composition, "clock", "password", b.prepared.clock)
}

func (b *productionBusinessBuilder) newLoginHandler(
	service *identity.Service,
) (*identity.Handler, error) {
	coordinator, err := identity.NewLoginAttemptCoordinator(
		b.prepared.ids,
		b.prepared.clock,
		b.composition.failureAuditor,
	)
	if err != nil {
		return nil, errors.New("create login attempt coordinator")
	}
	handler, err := identity.NewLoginHandler(
		service,
		b.prepared.source,
		b.composition.loginMaxBodyBytes,
		identity.LoginHandlerConfig{
			Coordinator:       coordinator,
			Observer:          b.composition.loginObserver,
			ObservationBudget: b.composition.loginObserveFor,
		},
	)
	if err != nil {
		return nil, errors.New("create login handler")
	}
	return handler, nil
}

func (b *productionBusinessBuilder) newAuthorizer() (platform.HTTPAuthorizer, error) {
	var resolver platform.PrincipalResolver = b.sessionStore
	if b.composition.articleTestSeam != nil &&
		b.composition.articleTestSeam.principalResolver != nil {
		resolver = b.composition.articleTestSeam.principalResolver
	}
	authorizer, err := platform.NewHTTPAuthorizer(
		resolver,
		func() time.Time { return b.prepared.clock.Now() },
	)
	if err != nil {
		return platform.HTTPAuthorizer{}, errors.New("create password change authorizer")
	}
	observeArticleSharedDependency(
		b.composition,
		"session_store",
		"authorizer",
		b.sessionStore,
	)
	observeArticleSharedDependency(b.composition, "clock", "authorizer", b.prepared.clock)
	return authorizer, nil
}

func (b *productionBusinessBuilder) newAdministratorHandler() (*identity.Handler, error) {
	handler, _, err := b.newAdministratorHandlerWithCoordinator()
	return handler, err
}

func (b *productionBusinessBuilder) newAdministratorHandlerWithCoordinator() (
	*identity.Handler,
	*identity.AdministratorCreateCoordinator,
	error,
) {
	processingGate, err := b.newProcessingGate()
	if err != nil {
		return nil, nil, err
	}
	repository := identity.PostgresAdministratorRepository{}
	create, privacyService, err := b.newAdministratorServices(repository, processingGate)
	if err != nil {
		return nil, nil, err
	}
	b.observeAdministratorDependencies()
	handler, err := identity.NewAdministratorHandler(
		create,
		privacyService,
		create,
		b.prepared.source,
		b.composition.loginMaxBodyBytes,
		b.composition.loginMaxBodyBytes,
	)
	if err != nil {
		return nil, nil, errors.New("create administrator handler")
	}
	observeArticleSharedDependency(
		b.composition,
		"administrator_create_coordinator",
		"administrator",
		create,
	)
	return handler, create, nil
}

func (b *productionBusinessBuilder) newProcessingGate() (*privacy.ProcessingGate, error) {
	if b.processingGate != nil {
		return b.processingGate, nil
	}
	store, err := privacy.NewStore(b.pool)
	if err != nil {
		return nil, errors.New("create privacy processing store")
	}
	if err := b.composition.processingGate.Validate(); err != nil {
		return nil, errors.New("validate privacy processing gate configuration")
	}
	gate, err := privacy.NewProcessingGate(b.composition.processingGate, store)
	if err != nil {
		return nil, errors.New("create privacy processing gate")
	}
	b.processingGate = gate
	return gate, nil
}

// noticeProcessingGate binds privacy's sole authority to each caller-owned transaction.
type noticeProcessingGate struct {
	config   platform.ProcessingGateConfig
	poolGate *privacy.ProcessingGate
}

func (g noticeProcessingGate) RequireOpen(ctx context.Context) error {
	return g.poolGate.RequireOpen(ctx)
}

func (g noticeProcessingGate) RequireOpenInTx(ctx context.Context, tx pgx.Tx) error {
	store, err := privacy.NewStore(tx)
	if err != nil {
		return err
	}
	gate, err := privacy.NewProcessingGate(g.config, store)
	if err != nil {
		return err
	}
	return gate.RequireOpen(ctx)
}

func (b *productionBusinessBuilder) buildNoticeDispatcher() error {
	gate, err := b.newProcessingGate()
	if err != nil {
		return err
	}
	workerID, err := b.prepared.ids.NewID()
	if err != nil {
		return errors.New("create notice worker id")
	}
	noticeLogger := b.composition.logger.With(
		slog.String("service", serviceName),
		slog.String("module", noticeLogModule),
		slog.String("event", noticeWarningEvent),
	)
	worker, err := article.NewTransferNoticeDispatcher(article.NoticeDispatcherDependencies{
		Transactions: b.pool, Repository: article.Repository{},
		Gate:   noticeProcessingGate{config: b.composition.processingGate, poolGate: gate},
		Mailer: b.prepared.administratorMailer, Audit: b.auditStore,
		IDs: b.prepared.ids, NewTraceID: platform.NewTraceID,
		Logger: noticeLogger, WorkerID: workerID,
	}, b.composition.noticeDispatcher)
	if err != nil {
		return fmt.Errorf("create notice dispatcher: %w", err)
	}
	b.noticeWorker = worker
	if b.composition.runtime != nil {
		b.composition.runtime.noticeWorker = worker
	}
	for _, dependency := range []articleSharedDependency{
		{Kind: "pool", Instance: b.pool}, {Kind: "repository", Instance: article.Repository{}},
		{Kind: "mailer", Instance: b.prepared.administratorMailer}, {Kind: "audit_store", Instance: b.auditStore},
		{Kind: "id_provider", Instance: b.prepared.ids}, {Kind: "logger", Instance: noticeLogger},
		{Kind: "processing_gate", Instance: gate}, {Kind: "notice_worker", Instance: worker},
	} {
		observeArticleSharedDependency(b.composition, dependency.Kind, "notice", dependency.Instance)
	}
	return nil
}

func (b *productionBusinessBuilder) newAdministratorServices(
	repository identity.PostgresAdministratorRepository,
	processingGate *privacy.ProcessingGate,
) (*identity.AdministratorCreateCoordinator, *identity.AdministratorPrivacyCoordinator, error) {
	create, err := identity.NewAdministratorCreateService(
		identity.AdministratorCreateDependencies{
			Transactions: b.pool, Repository: repository,
			InformationNotices: b.prepared.administratorNotices, Gate: processingGate,
			PasswordPolicy: security.PasswordPolicy{}, PasswordHasher: b.prepared.passwords,
			Mailer: b.prepared.administratorMailer, Idempotency: idempotency.Store{},
			IdempotencyKeys:     b.prepared.idempotencyKeys,
			IdempotencyLock:     b.prepared.idempotencyLock,
			IdempotencyTTL:      b.prepared.idempotencyTTL,
			SMTPOperationBudget: b.composition.smtpOperationBudget,
			Audit:               b.auditStore, IDs: b.prepared.ids, Clock: b.prepared.clock,
		},
	)
	if err != nil {
		return nil, nil, errors.New("create administrator creation coordinator")
	}
	privacyService, err := identity.NewAdministratorPrivacyService(
		identity.AdministratorPrivacyDependencies{
			Transactions: b.pool, Repository: repository,
			Notices: identity.CurrentNoticeReader{}, Sessions: b.sessionStore,
			SessionIssuer: b.prepared.sessionIssuer, Audit: b.auditStore,
			IDs: b.prepared.ids, Clock: b.prepared.clock,
		},
	)
	if err != nil {
		return nil, nil, errors.New("create administrator privacy coordinator")
	}
	return create, privacyService, nil
}

func (b *productionBusinessBuilder) observeAdministratorDependencies() {
	observeArticleSharedDependency(b.composition, "pool", "administrator", b.pool)
	observeArticleSharedDependency(
		b.composition,
		"session_store",
		"administrator",
		b.sessionStore,
	)
	observeArticleSharedDependency(b.composition, "audit_store", "administrator", b.auditStore)
	observeArticleSharedDependency(b.composition, "id_provider", "administrator", b.prepared.ids)
	observeArticleSharedDependency(b.composition, "clock", "administrator", b.prepared.clock)
}

func (b *productionBusinessBuilder) protectIdentityHandlers(
	loginHandler *identity.Handler,
	passwordHandler *identity.Handler,
	administratorHandler *identity.Handler,
	passwordResetHandler *identity.Handler,
	temporaryCredentialResendHandler *identity.Handler,
	logoutHandler *identity.SessionLogoutHandler,
	authorizer platform.HTTPAuthorizer,
) productionIdentityHandlers {
	return productionIdentityHandlers{
		login: b.composition.originGuard.Protect(http.HandlerFunc(loginHandler.Login)),
		password: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessChangePassword,
			http.HandlerFunc(passwordHandler.ChangePassword),
		)),
		administratorNotice: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessCurrentIdentity,
			http.HandlerFunc(administratorHandler.CurrentAdministratorInformationNotice),
		)),
		administratorCreate: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessGeneral,
			http.HandlerFunc(administratorHandler.CreateAdministrator),
		)),
		administratorPrivacy: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessAcknowledgePrivacy,
			http.HandlerFunc(administratorHandler.AcknowledgePrivacy),
		)),
		passwordReset: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessGeneral,
			http.HandlerFunc(passwordResetHandler.ResetPassword),
		)),
		temporaryCredentialResend: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessGeneral,
			http.HandlerFunc(
				temporaryCredentialResendHandler.ResendTemporaryCredential,
			),
		)),
		logout: b.composition.originGuard.Protect(authorizer.Protect(
			session.AccessGeneral,
			http.HandlerFunc(logoutHandler.Logout),
		)),
		authorizer: authorizer,
	}
}

func (b *productionBusinessBuilder) buildArticleHandlers(
	authorizer platform.HTTPAuthorizer,
) ([16]http.Handler, error) {
	var handlers [16]http.Handler
	queryHandlers, err := b.buildArticleQueryHandlers(authorizer)
	if err != nil {
		return handlers, err
	}
	handlers[3] = queryHandlers[0]
	handlers[4] = queryHandlers[1]
	handlers[7], handlers[8] = queryHandlers[2], queryHandlers[3]
	handlers[12] = queryHandlers[4]
	handlers[14], handlers[15] = queryHandlers[5], queryHandlers[6]
	if b.composition.articleJournal == nil && b.composition.articleDeletion == nil {
		return handlers, nil
	}
	if b.composition.articleJournal == nil || b.composition.articleDeletion == nil ||
		b.composition.runtime == nil {
		return handlers, errors.New("production Article composition is incomplete")
	}
	repository := article.Repository{}
	b.observeArticleDependencies()
	components, err := b.newArticleComponents(repository)
	if err != nil {
		return handlers, err
	}
	b.composition.runtime.articleWorker = components.worker
	protected := b.protectArticleHandlers(authorizer, components.handler)
	copy(handlers[:3], protected[:3])
	copy(handlers[5:7], protected[3:5])
	copy(handlers[9:11], protected[5:7])
	handlers[11], handlers[13] = protected[7], protected[8]
	return handlers, nil
}

func (b *productionBusinessBuilder) buildArticleQueryHandlers(
	authorizer platform.HTTPAuthorizer,
) ([7]http.Handler, error) {
	var handlers [7]http.Handler
	repository, err := article.NewPostgresArticleQueryRepository(b.pool)
	if err != nil {
		return handlers, errors.New("create Article query repository")
	}
	authorReader, err := identity.NewArticleIdentityReader(b.pool)
	if err != nil {
		return handlers, errors.New("create Article author state reader")
	}
	service, err := article.NewArticleQueryService(
		repository,
		authorReader,
		b.composition.articleQuery,
	)
	if err != nil {
		return handlers, errors.New("create Article query service")
	}
	handler, err := article.NewQueryHandler(service)
	if err != nil {
		return handlers, errors.New("create Article query handler")
	}
	orphanService, err := article.NewOrphanQueryService(repository, b.composition.articleQuery)
	if err != nil {
		return handlers, err
	}
	orphanHandler, err := article.NewOrphanQueryHandler(orphanService)
	if err != nil {
		return handlers, err
	}
	b.observeArticleQueryDependencies(repository, authorReader, service, handler)
	return b.buildProtectedArticleQueryHandlers(authorizer, service, handler, orphanHandler)
}

func (b *productionBusinessBuilder) buildProtectedArticleQueryHandlers(
	authorizer platform.HTTPAuthorizer,
	service *article.ArticleQueryService,
	handler, orphanHandler *article.Handler,
) ([7]http.Handler, error) {
	authorHandler, err := article.NewAuthorDetailHandler(service)
	if err != nil {
		return [7]http.Handler{}, errors.New("create Article author detail handler")
	}
	adminHandler, err := article.NewAdminQueryHandler(service)
	if err != nil {
		return [7]http.Handler{}, errors.New("create Article admin query handler")
	}
	observeArticleSharedDependency(b.composition, "article_query_service", "article_author_detail_handler", service)
	observeArticleSharedDependency(b.composition, "article_author_detail_handler", "router", authorHandler)
	observeArticleSharedDependency(b.composition, "article_query_service", "article_admin_query_handler", service)
	observeArticleSharedDependency(b.composition, "article_admin_query_handler", "router", adminHandler)
	return b.protectArticleQueryHandlers(authorizer, handler, orphanHandler, authorHandler, adminHandler), nil
}

func (b *productionBusinessBuilder) observeArticleQueryDependencies(
	repository *article.PostgresArticleQueryRepository,
	authorReader *identity.ArticleIdentityReader,
	service *article.ArticleQueryService,
	handler *article.Handler,
) {
	observeArticleSharedDependency(
		b.composition,
		"pool",
		"article_query_repository",
		b.pool,
	)
	observeArticleSharedDependency(
		b.composition,
		"pool",
		"article_author_reader",
		b.pool,
	)
	observeArticleSharedDependency(
		b.composition,
		"article_query_repository",
		"article_query_service",
		repository,
	)
	observeArticleSharedDependency(
		b.composition,
		"article_author_reader",
		"article_query_service",
		authorReader,
	)
	observeArticleSharedDependency(
		b.composition,
		"article_query_service",
		"article_query_handler",
		service,
	)
	observeArticleSharedDependency(
		b.composition,
		"article_query_handler",
		"router",
		handler,
	)
}

func (b *productionBusinessBuilder) protectArticleQueryHandlers(
	authorizer platform.HTTPAuthorizer,
	handler *article.Handler,
	orphanHandler *article.Handler,
	authorHandler *article.Handler,
	adminHandler *article.Handler,
) [7]http.Handler {
	routeHandlers := [7]http.Handler{
		http.HandlerFunc(handler.ListArticles),
		http.HandlerFunc(handler.GetArticle),
		http.HandlerFunc(orphanHandler.ListOrphanArticles),
		http.HandlerFunc(orphanHandler.GetOrphanArticle),
		http.HandlerFunc(authorHandler.GetAuthorArticle),
		http.HandlerFunc(adminHandler.ListAdminArticles),
		http.HandlerFunc(adminHandler.GetAdminArticle),
	}
	if b.composition.articleTestSeam != nil {
		for index, injected := range b.composition.articleTestSeam.adminQueryHandlers {
			if injected != nil {
				routeHandlers[index+5] = injected
			}
		}
		if injected := b.composition.articleTestSeam.authorDetailHandler; injected != nil {
			routeHandlers[4] = injected
		}
		for index, injected := range b.composition.articleTestSeam.queryHandlers {
			if injected != nil {
				routeHandlers[index] = injected
			}
		}
		for index := 0; index < 2; index++ {
			if injected := b.composition.articleTestSeam.orphanHandlers[index]; injected != nil {
				routeHandlers[index+2] = injected
			}
		}
	}
	return [7]http.Handler{
		authorizer.Protect(session.AccessGeneral, routeHandlers[0]),
		authorizer.Protect(session.AccessGeneral, routeHandlers[1]),
		authorizer.Protect(session.AccessGeneral, routeHandlers[2]),
		authorizer.Protect(session.AccessGeneral, routeHandlers[3]),
		authorizer.Protect(session.AccessGeneral, routeHandlers[4]),
		authorizer.Protect(session.AccessGeneral, routeHandlers[5]),
		authorizer.Protect(session.AccessGeneral, routeHandlers[6]),
	}
}

func (b *productionBusinessBuilder) observeArticleDependencies() {
	observeArticleSharedDependency(b.composition, "pool", "article", b.pool)
	observeArticleSharedDependency(b.composition, "audit_store", "article", b.auditStore)
	observeArticleSharedDependency(b.composition, "logger", "article", b.composition.logger)
	observeArticleSharedDependency(b.composition, "id_provider", "article", b.prepared.ids)
	observeArticleSharedDependency(b.composition, "clock", "article", b.prepared.clock)
}

func (b *productionBusinessBuilder) newArticleComponents(
	repository article.Repository,
) (productionArticleComponents, error) {
	lifecycle, err := newArticleIdentityLifecycleAdapter(b.auditStore, b.prepared.ids, b.prepared.clock)
	if err != nil {
		return productionArticleComponents{}, err
	}
	b.articleIdentity = lifecycle
	finalizer, err := b.newArticleFinalizer(repository)
	if err != nil {
		return productionArticleComponents{}, err
	}
	worker, err := b.newArticleWorker(finalizer)
	if err != nil {
		return productionArticleComponents{}, err
	}
	coordinator, err := b.newArticleCoordinator(repository, finalizer)
	if err != nil {
		return productionArticleComponents{}, err
	}
	handler, err := b.newArticleHandler(repository, coordinator, worker)
	if err != nil {
		return productionArticleComponents{}, err
	}
	var gate articleArticleDeletionGate
	if b.composition.articleJournalOwner != nil {
		gate, err = b.newArticleReplayProvider(finalizer)
		if err != nil {
			return productionArticleComponents{}, err
		}
	}
	return productionArticleComponents{handler: handler, worker: worker, gate: gate}, nil
}

// newArticleReplayProvider builds the unique production source and provider in
// the single composition root; a missing inventory path or journal owner fails
// closed instead of falling back to a stub.
func (b *productionBusinessBuilder) newArticleReplayProvider(
	finalizer *article.DeletionFinalizer,
) (articleArticleDeletionGate, error) {
	owner := b.composition.articleJournalOwner
	path := b.composition.articleBackupInv
	if owner == nil || path == "" {
		return nil, errors.New("production Article replay composition is incomplete")
	}
	reader, err := article.NewFileBackupInventoryReader(path)
	if err != nil {
		return nil, errors.New("create Article backup inventory reader")
	}
	verified, ok := owner.(article.VerifiedDeletionJournal)
	if !ok {
		return nil, errors.New("Article journal does not expose verified reads")
	}
	facts, err := article.NewPostgresReplayFactReader(b.pool)
	if err != nil {
		return nil, errors.New("create Article replay fact reader")
	}
	source, err := article.NewProductionDeletionReplaySource(
		verified,
		reader,
		finalizer,
		facts,
		facts,
		func() time.Time { return b.prepared.clock.Now() },
	)
	if err != nil {
		return nil, errors.New("create Article deletion replay source")
	}
	if slot := b.composition.articleReplaySource; slot != nil {
		slot.source = source
	}
	store := b.composition.articleEvidence
	if store == nil {
		store, err = article.NewPostgresReplayEvidenceStore(b.pool)
		if err != nil {
			return nil, errors.New("create Article deletion replay store")
		}
	}
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{
		Store:       store,
		Source:      source,
		EvidenceIDs: b.prepared.ids,
		Tokens:      b.prepared.ids,
		Clock:       b.prepared.clock,
	})
	if err != nil {
		return nil, errors.New("create Article deletion replay provider")
	}
	return provider, nil
}

func (b *productionBusinessBuilder) newArticleFinalizer(
	repository article.Repository,
) (*article.DeletionFinalizer, error) {
	observeArticleSharedDependency(b.composition, "identity_lifecycle", "article_finalizer", b.articleIdentity)
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions: b.pool, Repository: repository,
		Journal: b.composition.articleJournal, Auditor: b.auditStore,
		AuditIDs: b.prepared.ids, Clock: b.prepared.clock,
		WorkerID:      b.composition.articleDeletion.workerID,
		LeaseDuration: b.composition.articleDeletion.lease,
		Identity:      b.articleIdentity,
	})
	if err != nil {
		return nil, errors.New("create Article deletion finalizer")
	}
	return finalizer, nil
}

func (b *productionBusinessBuilder) newArticleWorker(
	finalizer *article.DeletionFinalizer,
) (*article.DeletionWorker, error) {
	discoverer, err := article.NewPostgresDeletionOperationDiscoverer(b.pool)
	if err != nil {
		return nil, errors.New("create Article deletion discovery")
	}
	worker, err := article.NewDeletionWorker(article.DeletionWorkerDependencies{
		Discoverer: discoverer,
		Converger:  finalizer,
		Now: func() time.Time {
			return b.prepared.clock.Now()
		},
		NewTraceID: platform.NewTraceID,
	}, b.composition.articleDeletion.worker)
	if err != nil {
		return nil, errors.New("create Article deletion worker")
	}
	return worker, nil
}

func (b *productionBusinessBuilder) newArticleCoordinator(
	repository article.Repository,
	finalizer *article.DeletionFinalizer,
) (*article.PermanentDeletionCoordinator, error) {
	coordinator, err := article.NewPermanentDeletionCoordinator(
		article.PermanentDeletionDependencies{
			Transactions: b.pool, Repository: repository,
			Journal: b.composition.articleJournal, Finalizer: finalizer,
			OperationIDs: b.prepared.ids, Clock: b.prepared.clock,
		},
	)
	if err != nil {
		return nil, errors.New("create Article deletion coordinator")
	}
	return coordinator, nil
}

func (b *productionBusinessBuilder) newArticleHandler(
	repository article.Repository,
	coordinator *article.PermanentDeletionCoordinator,
	worker *article.DeletionWorker,
) (*article.Handler, error) {
	observeArticleSharedDependency(b.composition, "identity_lifecycle", "article_command", b.articleIdentity)
	responseLogger, err := article.NewSlogArticleResponseWriteLogger(
		b.composition.logger,
		serviceName,
	)
	if err != nil {
		return nil, errors.New("create Article response logger")
	}
	deleter := wakingArticleDeleter{delegate: coordinator, worker: worker}
	service, err := article.NewCommandService(article.CommandDependencies{
		Transactions: b.pool, Repository: repository,
		IDs: b.prepared.ids, Clock: b.prepared.clock, Deletion: deleter,
		Moderation: &article.ModerationDependencies{
			Auditor:      b.auditStore,
			AuthorStates: moderationAuthorStates,
		},
		Disposition: &article.OrphanDispositionDependencies{Identity: b.articleIdentity,
			Auditor: b.auditStore, RecipientMaxAttempts: b.composition.articleTransferNoticeMaxAttempts},
	})
	if err != nil {
		return nil, errors.New("create Article command service")
	}
	handler, err := article.NewHandler(service, responseLogger, service)
	if err != nil {
		return nil, errors.New("create Article HTTP handler")
	}
	observeArticleSharedDependency(b.composition, "article_command_service", "article_creator", service)
	observeArticleSharedDependency(b.composition, "article_command_service", "article_content_replacer", service)
	observeArticleSharedDependency(b.composition, "article_response_logger", "article_command_handler", responseLogger)
	observeArticleSharedDependency(b.composition, "logger", "article_response_logger", b.composition.logger)
	gate, err := b.newProcessingGate()
	if err != nil {
		return nil, err
	}
	return article.NewOrphanCommandHandler(handler, service, gate)
}

func moderationAuthorStates(ctx context.Context, transaction pgx.Tx, authorIDs []string) (map[string]string, error) {
	authorReader, err := identity.NewArticleIdentityReader(transaction)
	if err != nil {
		return nil, err
	}
	states, err := authorReader.GetAuthorStates(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	normalized := make(map[string]string, len(states))
	for authorID, state := range states {
		normalized[authorID] = string(state)
	}
	return normalized, nil
}

func (b *productionBusinessBuilder) protectArticleHandlers(
	authorizer platform.HTTPAuthorizer,
	handler *article.Handler,
) []http.Handler {
	routeHandlers := [9]http.Handler{
		http.HandlerFunc(handler.DeleteOrdinary),
		http.HandlerFunc(handler.DeleteViolating),
		http.HandlerFunc(handler.DeleteOrphan),
		http.HandlerFunc(handler.MarkViolation),
		http.HandlerFunc(handler.RestoreArticle),
		http.HandlerFunc(handler.PublishOrphan),
		http.HandlerFunc(handler.TransferOrphan),
		http.HandlerFunc(handler.Create),
		http.HandlerFunc(handler.UpdateArticle),
	}
	observeArticleSharedDependency(b.composition, "article_command_handler", "create", handler)
	observeArticleSharedDependency(b.composition, "article_command_handler", "update", handler)
	if b.composition.articleTestSeam != nil {
		if injected := b.composition.articleTestSeam.createHandler; injected != nil {
			routeHandlers[7] = injected
		}
		if injected := b.composition.articleTestSeam.updateHandler; injected != nil {
			routeHandlers[8] = injected
		}
		for index, injected := range b.composition.articleTestSeam.handlers {
			if injected != nil {
				routeHandlers[index] = injected
			}
		}
		for index := 2; index < 4; index++ {
			if injected := b.composition.articleTestSeam.orphanHandlers[index]; injected != nil {
				routeHandlers[index+3] = injected
			}
		}
	}
	return b.protectArticleMutationRoutes(authorizer, routeHandlers[:])
}

func (b *productionBusinessBuilder) protectArticleMutationRoutes(
	authorizer platform.HTTPAuthorizer,
	routeHandlers []http.Handler,
) []http.Handler {
	protectOrigin := b.composition.originGuard.Protect
	if b.composition.articleTestSeam != nil &&
		b.composition.articleTestSeam.protectOrigin != nil {
		protectOrigin = b.composition.articleTestSeam.protectOrigin
	}
	handlers := make([]http.Handler, len(routeHandlers))
	for index, consumer := range []string{"ordinary", "violating", "orphan", "violation", "restore", "orphan_publish", "orphan_transfer", "create", "update"} {
		handlers[index] = protectOrigin(authorizer.Protect(
			session.AccessGeneral,
			observeArticleAuthorization(b.composition, consumer, routeHandlers[index]),
		))
		observeArticleSharedDependency(
			b.composition,
			"origin_guard",
			consumer,
			b.composition.originGuard,
		)
	}
	return handlers
}

func newLoginBusinessRouter(login http.Handler, password ...http.Handler) http.Handler {
	return newLoginBusinessRouterWithTrace(login, nil, password...)
}

func newLoginBusinessRouterWithTrace(
	login http.Handler,
	trace func(*http.Request),
	password ...http.Handler,
) http.Handler {
	return newLoginBusinessRouterWithTraceAndRecovery(login, trace, nil, password...)
}

func newLoginBusinessRouterWithTraceAndRecovery(
	login http.Handler,
	trace func(*http.Request),
	recoveryHandle func() recovery.Runtime,
	password ...http.Handler,
) http.Handler {
	var handlers [23]http.Handler
	for index := range handlers {
		if index < len(password) {
			handlers[index] = password[index]
		}
	}
	router := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var handler http.Handler
		var allow string
		methodAllowed := false
		probe := false
		readiness := false
		switch {
		case request.URL.Path == "/livez":
			probe = true
			allow = http.MethodGet
			methodAllowed = request.Method == http.MethodGet
		case request.URL.Path == "/readyz":
			probe = true
			readiness = true
			allow = http.MethodGet
			methodAllowed = request.Method == http.MethodGet
		case request.URL.Path == "/api/v1/sessions":
			handler = login
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case request.URL.Path == "/api/v1/sessions/logout":
			handler = handlers[22]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case request.URL.Path == "/api/v1/users/me/password":
			handler = handlers[0]
			allow = http.MethodPut
			methodAllowed = request.Method == http.MethodPut
		case request.URL.Path == "/api/v1/admin/administrator-information-provision-notices/current":
			handler = handlers[1]
			allow = http.MethodGet
			methodAllowed = request.Method == http.MethodGet
		case request.URL.Path == "/api/v1/admin/administrators":
			handler = handlers[2]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case request.URL.Path == "/api/v1/users/me/privacy-acknowledgements":
			handler = handlers[3]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case exactPasswordResetPath(
			request.URL.Path,
			"/api/v1/admin/users/",
		), exactPasswordResetPath(
			request.URL.Path,
			"/api/v1/admin/administrators/",
		):
			handler = handlers[7]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case exactTemporaryCredentialResendPath(request.URL.Path):
			handler = handlers[8]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case exactArticleModerationPath(request.URL.Path, "violation"):
			handler = handlers[11]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case exactArticleModerationPath(request.URL.Path, "restore"):
			handler = handlers[12]
			allow = http.MethodPost
			methodAllowed = request.Method == http.MethodPost
		case request.URL.Path == "/api/v1/admin/orphan-articles":
			handler, allow, methodAllowed = handlers[13], http.MethodGet, request.Method == http.MethodGet
		case exactOrphanActionPath(request.URL.Path, "publish"):
			handler, allow, methodAllowed = handlers[15], http.MethodPost, request.Method == http.MethodPost
		case exactOrphanActionPath(request.URL.Path, "transfer"):
			handler, allow, methodAllowed = handlers[16], http.MethodPost, request.Method == http.MethodPost
		case request.URL.Path == "/api/v1/articles":
			allow = http.MethodGet + ", " + http.MethodPost
			switch request.Method {
			case http.MethodGet:
				handler, methodAllowed = handlers[9], true
			case http.MethodPost:
				handler, methodAllowed = handlers[17], true
			}
		case request.URL.Path == "/api/v1/admin/articles":
			handler, allow, methodAllowed = handlers[20], http.MethodGet, request.Method == http.MethodGet
		case exactArticleDeletePath(request.URL.Path, "/api/v1/users/me/articles/"):
			handler = handlers[18]
			allow = http.MethodGet
			methodAllowed = request.Method == http.MethodGet
		case exactArticleDeletePath(request.URL.Path, "/api/v1/articles/"):
			allow = http.MethodGet + ", " + http.MethodPut + ", " + http.MethodDelete
			switch request.Method {
			case http.MethodGet:
				handler = handlers[10]
				methodAllowed = true
			case http.MethodDelete:
				handler = handlers[4]
				methodAllowed = true
			case http.MethodPut:
				handler, methodAllowed = handlers[19], true
			}
		case exactArticleDeletePath(request.URL.Path, "/api/v1/admin/articles/"):
			allow = http.MethodGet + ", " + http.MethodDelete
			switch request.Method {
			case http.MethodGet:
				handler, methodAllowed = handlers[21], true
			case http.MethodDelete:
				handler, methodAllowed = handlers[5], true
			}
		case exactArticleDeletePath(request.URL.Path, "/api/v1/admin/orphan-articles/"):
			allow = http.MethodGet + ", " + http.MethodDelete
			switch request.Method {
			case http.MethodGet:
				handler, methodAllowed = handlers[14], true
			case http.MethodDelete:
				handler, methodAllowed = handlers[6], true
			}
		default:
			response.Header().Set("X-Content-Type-Options", "nosniff")
			_ = platform.WriteError(response, request, platform.NewPublicError(
				platform.CodeRequestRouteNotFound,
				errors.New("route not found"),
			))
			return
		}
		if !methodAllowed {
			response.Header().Set("Allow", allow)
			response.Header().Set("X-Content-Type-Options", "nosniff")
			_ = platform.WriteError(response, request, platform.NewPublicError(
				platform.CodeRequestMethodNotAllowed,
				errors.New("method not allowed"),
			))
			return
		}
		if probe {
			guard, _ := recoveryGuardFromRuntimeHandle(recoveryHandle)
			if readiness {
				guard.Readiness(response, request)
			} else {
				guard.Liveness(response, request)
			}
			return
		}
		if handler == nil {
			response.Header().Set("X-Content-Type-Options", "nosniff")
			_ = platform.WriteError(response, request, platform.NewPublicError(
				platform.CodeRequestRouteNotFound,
				errors.New("route not registered"),
			))
			return
		}
		if guard, guarded := recoveryGuardFromRuntimeHandle(recoveryHandle); guarded {
			guard.Protect(handler).ServeHTTP(response, request)
			return
		}
		handler.ServeHTTP(response, request)
	})
	if trace != nil {
		next := router
		router = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			trace(request)
			next.ServeHTTP(response, request)
		})
	}
	return platform.TraceMiddleware(router)
}

func recoveryGuardFromRuntimeHandle(
	handle func() recovery.Runtime,
) (platform.RecoveryGuard, bool) {
	if handle == nil {
		return platform.RecoveryGuard{}, false
	}
	runtime := handle()
	if runtime == nil {
		return platform.RecoveryGuard{}, false
	}
	guard, err := platform.NewRecoveryGuard(runtime)
	if err != nil {
		return platform.RecoveryGuard{}, true
	}
	return guard, true
}

func observeArticleAuthorization(
	composition businessComposition,
	route string,
	next http.Handler,
) http.Handler {
	if composition.articleTestSeam == nil ||
		composition.articleTestSeam.authorized == nil {
		return next
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		composition.articleTestSeam.authorized(route)
		next.ServeHTTP(response, request)
	})
}

func observeArticleSharedDependency(
	composition businessComposition,
	kind string,
	consumer string,
	instance any,
) {
	if composition.articleTestSeam == nil || composition.articleTestSeam.shared == nil {
		return
	}
	composition.articleTestSeam.shared(articleSharedDependency{
		Kind:     kind,
		Consumer: consumer,
		Instance: instance,
	})
}

func exactArticleDeletePath(path, prefix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	articleID := strings.TrimPrefix(path, prefix)
	return articleID != "" && !strings.Contains(articleID, "/")
}

func exactArticleModerationPath(path, action string) bool {
	suffix := "/" + action
	return strings.HasSuffix(path, suffix) &&
		exactArticleDeletePath(strings.TrimSuffix(path, suffix), "/api/v1/admin/articles/")
}

func exactOrphanActionPath(path, action string) bool {
	suffix := "/" + action
	return strings.HasSuffix(path, suffix) && exactArticleDeletePath(strings.TrimSuffix(path, suffix), "/api/v1/admin/orphan-articles/")
}

type articleIdentityLifecycleAdapter struct {
	audit identity.CanonicalDeleteAuditAppender
	ids   identity.RegistrationIDGenerator
	clock identity.RegistrationClock
}

func newArticleIdentityLifecycleAdapter(appender identity.CanonicalDeleteAuditAppender,
	ids identity.RegistrationIDGenerator, clock identity.RegistrationClock,
) (*articleIdentityLifecycleAdapter, error) {
	if appender == nil || ids == nil || clock == nil {
		return nil, errors.New("article identity adapter dependencies are required")
	}
	return &articleIdentityLifecycleAdapter{audit: appender, ids: ids, clock: clock}, nil
}

func (adapter *articleIdentityLifecycleAdapter) LockTransferParties(
	ctx context.Context, transaction pgx.Tx, targetID *string, sourceID string,
) (article.DispositionParties, error) {
	parties, err := (&identity.ArticleIdentityLifecycle{}).LockTransferParties(ctx, transaction, targetID, sourceID)
	return article.DispositionParties{SourceID: parties.SourceID, SourceEmail: parties.SourceEmail,
		TargetID: parties.TargetID, TargetEmail: parties.TargetEmail}, articleIdentityBoundaryError(err)
}

func (adapter *articleIdentityLifecycleAdapter) LockDeletionCandidate(
	ctx context.Context, transaction pgx.Tx, userID string,
) (article.DispositionDeletionCandidate, error) {
	candidate, err := (&identity.ArticleIdentityLifecycle{}).LockDeletionCandidate(ctx, transaction, userID)
	return article.DispositionDeletionCandidate{UserID: candidate.UserID, Deleted: candidate.Deleted}, articleIdentityBoundaryError(err)
}

func (adapter *articleIdentityLifecycleAdapter) ClearRetainedEmailIfEligible(
	ctx context.Context, transaction pgx.Tx, sourceID string, proof *article.DispositionCompletionProof,
) (bool, error) {
	lifecycle, reference, err := adapter.bindCompletion(ctx, transaction, proof)
	if err != nil {
		return false, err
	}
	cleared, err := lifecycle.ClearRetainedEmailIfEligible(ctx, transaction, sourceID, reference)
	return cleared, articleIdentityBoundaryError(err)
}

func (adapter *articleIdentityLifecycleAdapter) bindCompletion(
	ctx context.Context, transaction pgx.Tx, proof *article.DispositionCompletionProof,
) (*identity.ArticleIdentityLifecycle, identity.ArticleCompletionReference, error) {
	traceID, present := platform.TraceIDFromContext(ctx)
	if adapter == nil || proof == nil || transaction == nil || !present {
		return nil, identity.ArticleCompletionReference{}, errors.New("article completion binding is required")
	}
	reference := articleIdentityReference(proof.Reference())
	if reference.Transaction != transaction {
		return nil, reference, errors.New("article completion exact transaction mismatch")
	}
	verifier := &articleBoundCompletionVerifier{proof: proof, reference: reference, traceID: traceID}
	lifecycle, err := identity.NewArticleIdentityLifecycle(verifier, adapter.audit, adapter.ids, adapter.clock)
	return lifecycle, reference, err
}

type articleBoundCompletionVerifier struct {
	proof     *article.DispositionCompletionProof
	reference identity.ArticleCompletionReference
	traceID   string
}

func (verifier *articleBoundCompletionVerifier) VerifyCompletion(
	ctx context.Context, transaction pgx.Tx, reference identity.ArticleCompletionReference,
) (identity.VerifiedArticleCompletion, error) {
	traceID, present := platform.TraceIDFromContext(ctx)
	if verifier == nil || verifier.proof == nil || !present || traceID != verifier.traceID ||
		transaction == nil || transaction != verifier.reference.Transaction || reference.Transaction != transaction ||
		!reflect.DeepEqual(reference, articleIdentityReference(verifier.proof.Reference())) {
		return identity.VerifiedArticleCompletion{}, errors.New("article completion reference binding mismatch")
	}
	facts, err := verifier.proof.VerifyCompletion(ctx, transaction)
	if err != nil {
		return identity.VerifiedArticleCompletion{}, articleIdentityBoundaryError(err)
	}
	return articleIdentityCompletionFacts(facts), nil
}

func articleIdentityReference(reference article.DispositionCompletionReference) identity.ArticleCompletionReference {
	return identity.ArticleCompletionReference{Transaction: reference.Transaction, ExpectedArticleID: reference.ExpectedArticleID,
		SourceID: reference.SourceID, BeforeVersion: reference.BeforeVersion, Action: identity.ArticleDispositionAction(reference.Action),
		NoticeID: reference.NoticeID, NewAuthorID: reference.NewAuthorID, OperationID: reference.OperationID,
		AuditEventID: reference.AuditEventID, BeforeArticle: articleIdentityState(reference.BeforeArticle)}
}

func articleIdentityState(state article.DispositionArticleState) identity.ArticleDispositionState {
	var category *string
	if state.Category != nil {
		value := string(*state.Category)
		category = &value
	}
	return identity.ArticleDispositionState{Exists: state.Exists, ID: state.ID, Version: state.Version,
		AuthorID: state.AuthorID, SourceID: state.SourceID, Publication: string(state.Publication),
		Violation: string(state.Violation), Category: category, Deletion: string(state.Deletion), OperationID: state.OperationID}
}

func articleIdentityCompletionFacts(facts article.DispositionCompletionFacts) identity.VerifiedArticleCompletion {
	result := identity.VerifiedArticleCompletion{AfterArticle: articleIdentityState(facts.AfterArticle),
		RemainingActiveCount: facts.RemainingActiveCount, RemainingPendingCount: facts.RemainingPendingCount,
		Audit: identity.ArticleDispositionAudit{ID: facts.Audit.ID, ArticleID: facts.Audit.TargetID,
			Action: facts.Audit.Action, Outcome: facts.Audit.Outcome, TraceID: facts.Audit.TraceID}}
	if facts.Notice != nil {
		result.Notice = &identity.ArticleDispositionNotice{ID: facts.Notice.ID, ArticleID: facts.Notice.ArticleRef,
			NewAuthorID: facts.Notice.NewAuthorID, FormerDeletedUserID: facts.Notice.FormerDeletedUserID}
	}
	for _, recipient := range facts.Recipients {
		result.Recipients = append(result.Recipients, identity.ArticleDispositionRecipient{
			Kind: string(recipient.Kind), EmailSnapshot: recipient.EmailSnapshot, Status: string(recipient.Status)})
	}
	if operation := facts.CompletedOperation; operation != nil {
		mapped := &identity.ArticleDispositionOperation{ID: operation.ID, ArticleID: operation.ArticleID,
			ArticleVersion: operation.ArticleVersion, DeleteKind: string(operation.DeleteKind),
			SourceID: operation.OrphanedFromUserID, Status: string(operation.Status)}
		if operation.AuditEventID != nil {
			mapped.AuditEventID = *operation.AuditEventID
		}
		result.CompletedOperation = mapped
	}
	return result
}

func articleIdentityBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, identity.ErrArticleTransferTargetInvalid) {
		return errors.Join(article.ErrTransferTargetInvalid, err)
	}
	var postgresError *pgconn.PgError
	var networkError net.Error
	if errors.As(err, &postgresError) || errors.As(err, &networkError) || errors.Is(err, io.EOF) ||
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return platform.NewPublicError(platform.CodePlatformDatabaseUnavailable, err)
	}
	return err
}

func exactPasswordResetPath(path, prefix string) bool {
	const suffix = "/password-reset"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(path, prefix)
	if !strings.HasSuffix(remainder, suffix) {
		return false
	}
	targetID := strings.TrimSuffix(remainder, suffix)
	return targetID != "" && !strings.Contains(targetID, "/")
}

func exactTemporaryCredentialResendPath(path string) bool {
	for _, route := range []struct {
		prefix string
		suffix string
	}{
		{
			prefix: "/api/v1/admin/users/",
			suffix: "/password-reset/resend",
		},
		{
			prefix: "/api/v1/admin/administrators/",
			suffix: "/temporary-password/resend",
		},
	} {
		if !strings.HasPrefix(path, route.prefix) ||
			!strings.HasSuffix(path, route.suffix) {
			continue
		}
		targetID := strings.TrimSuffix(
			strings.TrimPrefix(path, route.prefix),
			route.suffix,
		)
		if targetID != "" && !strings.Contains(targetID, "/") {
			return true
		}
	}
	return false
}

type wakingArticleDeleter struct {
	delegate article.PermanentArticleDeleter
	worker   *article.DeletionWorker
}

func (deleter wakingArticleDeleter) DeletePermanent(
	ctx context.Context,
	principal session.Principal,
	request article.PermanentDeletionRequest,
) error {
	if deleter.delegate == nil || deleter.worker == nil {
		return errors.New("Article deletion composition is incomplete")
	}
	err := deleter.delegate.DeletePermanent(ctx, principal, request)
	deleter.worker.Wake()
	return err
}

func productionDependencies() startupDependencies {
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGTERM, syscall.SIGINT)
	return startupDependencies{
		getenv:            os.Getenv,
		lookupEnvironment: os.LookupEnv,
		parseDatabase: func(dsn string) (*pgxpool.Config, error) {
			config, err := parseDatabasePoolConfig(dsn, os.Getenv)
			if err != nil {
				return nil, err
			}
			if err := validateProductionMigrationConfiguration(config, os.Getenv); err != nil {
				return nil, err
			}
			return config, nil
		},
		connect: func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
			return pgxpool.NewWithConfig(ctx, config)
		},
		ping: func(ctx context.Context, pool *pgxpool.Pool) error {
			return pool.Ping(ctx)
		},
		migrate: func(ctx context.Context, pool *pgxpool.Pool) error {
			return productionMigration(ctx, pool, os.Getenv)
		},
		bootstrapInitialAdmin: func(
			ctx context.Context,
			pool *pgxpool.Pool,
			prepared *preparedLoginDependencies,
		) error {
			if prepared == nil {
				return errors.New("prepared identity dependencies are required")
			}
			manager, err := identity.NewInitialAdminTrustManager(
				identity.InitialAdminDependencies{
					Transactions: pool,
					Password: func(context.Context) (string, error) {
						value := os.Getenv(initialAdminPasswordEnvironment)
						if value == "" {
							return "", errors.New("initial admin password is required")
						}
						return value, nil
					},
					PasswordPolicy: security.PasswordPolicy{},
					PasswordHasher: prepared.passwords,
					Repository:     identity.Repository{},
					Audit:          audit.Store{},
					IDs:            prepared.ids,
					Clock:          prepared.clock,
				},
			)
			if err != nil {
				return err
			}
			return manager.EnsureInitialAdmin(ctx)
		},
		listen: net.Listen,
		serve: func(listener net.Listener) error {
			server := &http.Server{
				Handler: http.NewServeMux(),
			}
			return server.Serve(listener)
		},
		closePool: func(pool *pgxpool.Pool) {
			pool.Close()
		},
		composeBusiness:        newProductionBusinessHandler,
		composeRecoveryControl: newProductionRecoveryControlHandler,
		prepareBusiness: func(config startupConfig) (*preparedLoginDependencies, error) {
			return prepareProductionLoginDependencies(config, rand.Reader)
		},
		prepareArticle:             prepareArticleDeletion,
		requirePasswordResetWorker: true,
		requireNoticeWorker:        true,
		openArticle:                openArticleDeletionJournal,
		newServer: func(_ string, handler http.Handler) lifecycleServer {
			return &http.Server{Handler: handler}
		},
		signals:    shutdownSignals,
		coordinate: runLifecycle,
		logOutput:  os.Stdout,
	}
}

func closeHTTPListener(listener net.Listener) error {
	err := listener.Close()
	if err == nil || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return fmt.Errorf("close HTTP listener: %w", err)
}

const (
	maintenanceTrustEnvironment     = "HAN_PASSWORD_RESET_MAINTENANCE_TRUST"
	maintenanceTTLEnvironment       = "HAN_PASSWORD_RESET_MAINTENANCE_MAX_TTL"
	maintenanceOperationEnvironment = "HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET"
	maintenanceEnvelopeLimit        = 16 * 1024
	maintenanceReadChunk            = 4096
	maintenancePollInterval         = 10 * time.Millisecond
)

type maintenanceFailure struct {
	code  string
	cause error
}

func (failure *maintenanceFailure) Error() string { return failure.code }
func (failure *maintenanceFailure) Unwrap() error { return failure.cause }

func maintenanceFail(code string, cause error) error {
	return &maintenanceFailure{code: code, cause: cause}
}

func selectMaintenanceMode(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) != 3 || args[0] != "maintenance" || args[1] != "password-reset" || args[2] != "retry" {
		return true, maintenanceFail("maintenance.invalid_request", nil)
	}
	return true, nil
}

type maintenanceConfig struct {
	trust            []byte
	maxTTL           time.Duration
	operationBudget  time.Duration
	shutdownBudget   time.Duration
	connectionBudget int
}

func loadMaintenanceConfig(getenv func(string) string) (maintenanceConfig, error) {
	var config maintenanceConfig
	config.trust = []byte(getenv(maintenanceTrustEnvironment))
	if len(config.trust) == 0 || len(config.trust) > maintenanceEnvelopeLimit {
		return config, maintenanceFail("maintenance.invalid_config", nil)
	}
	ttl, err := parsePositiveDuration(getenv(maintenanceTTLEnvironment), maintenanceTTLEnvironment)
	if err != nil {
		return config, maintenanceFail("maintenance.invalid_config", err)
	}
	operation, err := parsePositiveDuration(getenv(maintenanceOperationEnvironment), maintenanceOperationEnvironment)
	if err != nil {
		return config, maintenanceFail("maintenance.invalid_config", err)
	}
	budget, err := parsePositiveDecimal(getenv(passwordResetWorkerConnectionBudgetEnvironment), passwordResetWorkerConnectionBudgetEnvironment, 32)
	if err != nil {
		return config, maintenanceFail("maintenance.invalid_config", err)
	}
	shutdown, err := parseRequiredDuration(getenv(shutdownBudgetEnvironment), shutdownBudgetEnvironment, time.Second, 60*time.Second)
	if err != nil {
		return config, maintenanceFail("maintenance.invalid_config", err)
	}
	if getenv(recoveryModeEnvironment) != "normal" {
		return config, maintenanceFail("maintenance.mode_conflict", nil)
	}
	config.maxTTL, config.operationBudget = ttl, operation
	config.shutdownBudget, config.connectionBudget = shutdown, int(budget)
	return config, nil
}

func maintenanceAddBudget(start time.Time, budget time.Duration) (time.Time, error) {
	end := start.Add(budget)
	if budget <= 0 || !end.After(start) || end.Sub(start) != budget {
		return time.Time{}, maintenanceFail("maintenance.invalid_config", nil)
	}
	return end, nil
}

func maintenanceCleanupContext(operation context.Context, deadline time.Time, budget time.Duration) (context.Context, context.CancelFunc) {
	cutoff := deadline.Add(budget)
	if local := time.Now().Add(budget); local.Before(cutoff) {
		cutoff = local
	}
	return context.WithDeadline(context.WithoutCancel(operation), cutoff)
}

type maintenanceInputOps interface {
	Peek() (available uint32, eof bool, err error)
	ReadAvailable([]byte) (int, error)
	Close() error
}

type maintenanceInputOwner struct {
	file     *os.File
	ops      maintenanceInputOps
	once     sync.Once
	closeErr error
}

func (owner *maintenanceInputOwner) Close() error {
	owner.once.Do(func() {
		if owner.ops != nil {
			owner.closeErr = owner.ops.Close()
		} else if owner.file != nil {
			owner.closeErr = owner.file.Close()
		}
	})
	return owner.closeErr
}

func (owner *maintenanceInputOwner) Peek() (uint32, bool, error) { return owner.ops.Peek() }
func (owner *maintenanceInputOwner) ReadAvailable(buffer []byte) (int, error) {
	return owner.ops.ReadAvailable(buffer)
}

func (owner *maintenanceInputOwner) read(ctx context.Context) ([]byte, error) {
	defer owner.Close()
	ops, err := openMaintenanceInput(owner.file)
	if err != nil {
		return nil, err
	}
	owner.ops = ops
	return readMaintenanceInput(ctx, owner)
}

func readMaintenanceEnvelope(ctx context.Context, input *os.File) ([]byte, error) {
	return (&maintenanceInputOwner{file: input}).read(ctx)
}

func readMaintenanceInput(ctx context.Context, input maintenanceInputOps) (result []byte, err error) {
	defer func() {
		if closeErr := input.Close(); err == nil && closeErr != nil {
			result, err = nil, maintenanceFail("maintenance.failed", closeErr)
		}
	}()
	if ctx == nil {
		return nil, maintenanceFail("maintenance.failed", nil)
	}
	for {
		if err = ctx.Err(); err != nil {
			return nil, maintenanceFail("maintenance.failed", err)
		}
		available, eof, peekErr := input.Peek()
		if err = ctx.Err(); err != nil {
			return nil, maintenanceFail("maintenance.failed", err)
		}
		if peekErr != nil {
			return nil, maintenanceFail("maintenance.failed", peekErr)
		}
		if available > 0 {
			result, err = readMaintenanceAvailable(ctx, input, result, available)
			if err != nil {
				return nil, err
			}
			continue
		}
		if eof {
			if err = ctx.Err(); err != nil {
				return nil, maintenanceFail("maintenance.failed", err)
			}
			if len(result) == 0 {
				return nil, maintenanceFail("maintenance.invalid_request", nil)
			}
			return result, nil
		}
		if err = waitMaintenancePoll(ctx); err != nil {
			return nil, maintenanceFail("maintenance.failed", err)
		}
	}
}

func readMaintenanceAvailable(ctx context.Context, input maintenanceInputOps, result []byte, available uint32) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, maintenanceFail("maintenance.failed", err)
	}
	size := min(int64(available), int64(maintenanceEnvelopeLimit+1-len(result)), int64(maintenanceReadChunk))
	buffer := make([]byte, int(size))
	n, err := input.ReadAvailable(buffer)
	if cancel := ctx.Err(); cancel != nil {
		return nil, maintenanceFail("maintenance.failed", cancel)
	}
	if n <= 0 || n > len(buffer) || err != nil {
		return nil, maintenanceFail("maintenance.failed", err)
	}
	result = append(result, buffer[:n]...)
	if len(result) > maintenanceEnvelopeLimit {
		return nil, maintenanceFail("maintenance.invalid_request", nil)
	}
	return result, nil
}

func waitMaintenancePoll(ctx context.Context) error {
	interval := maintenancePollInterval
	if deadline, ok := ctx.Deadline(); ok {
		interval = min(interval, time.Until(deadline))
	}
	if interval <= 0 {
		return context.DeadlineExceeded
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

type maintenanceOutput struct {
	RequestID   string `json:"request_id"`
	OperationID string `json:"operation_id"`
	Outcome     string `json:"outcome"`
	AuditID     string `json:"audit_id"`
}

func writeMaintenanceOutput(output io.Writer, result maintenanceOutput) error {
	if result.Outcome != "applied" && result.Outcome != "already_applied" {
		return maintenanceFail("maintenance.failed", nil)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return maintenanceFail("maintenance.failed", err)
	}
	encoded = append(encoded, '\n')
	n, err := output.Write(encoded)
	if err == nil && n != len(encoded) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return maintenanceFail("maintenance.failed", err)
	}
	return nil
}

type maintenanceAttempt struct {
	handed  bool
	closing bool
}

type maintenanceStage struct {
	attempt *maintenanceAttempt
	cancel  context.CancelFunc
}

type maintenanceConnectionOwner struct {
	operation     context.Context
	deadline      time.Time
	budget        time.Duration
	mu            sync.Mutex
	quiescing     bool
	nextStage     uint64
	stages        map[uint64]maintenanceStage
	attempts      map[*maintenanceAttempt]struct{}
	sockets       map[*maintenanceTransport]*maintenanceAttempt
	connections   map[*pgx.Conn]*maintenanceCloseEntry
	stopWatch     func()
	transportErr  error
	cleanupCutoff time.Time
	cleanupErr    error
}

func newMaintenanceConnectionOwner(ctx context.Context, budget time.Duration) *maintenanceConnectionOwner {
	deadline, _ := ctx.Deadline()
	owner := &maintenanceConnectionOwner{operation: ctx, deadline: deadline, budget: budget,
		stages: make(map[uint64]maintenanceStage), attempts: make(map[*maintenanceAttempt]struct{}),
		sockets: make(map[*maintenanceTransport]*maintenanceAttempt), connections: make(map[*pgx.Conn]*maintenanceCloseEntry)}
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(finished); owner.quiesce(true) })
	var stopOnce sync.Once
	owner.stopWatch = func() {
		stopOnce.Do(func() {
			if !stop() {
				<-finished
			}
		})
	}
	return owner
}

func (owner *maintenanceConnectionOwner) stage(parent context.Context, attempt *maintenanceAttempt) (context.Context, func(), error) {
	ctx, cancel := context.WithCancel(owner.operation)
	if d, ok := parent.Deadline(); ok && d.Before(owner.deadline) {
		cancel()
		ctx, cancel = context.WithDeadline(owner.operation, d)
	}
	owner.mu.Lock()
	if owner.quiescing || attempt.closing || maintenanceContextError(owner.operation) != nil || maintenanceContextError(parent) != nil {
		owner.mu.Unlock()
		cancel()
		return nil, nil, maintenanceFail("maintenance.failed", context.Canceled)
	}
	owner.nextStage++
	id := owner.nextStage
	owner.stages[id] = maintenanceStage{attempt: attempt, cancel: cancel}
	owner.mu.Unlock()
	finished := make(chan struct{})
	stop := context.AfterFunc(parent, func() { defer close(finished); cancel() })
	var once sync.Once
	done := func() {
		once.Do(func() {
			if !stop() {
				<-finished
			}
			cancel()
			owner.mu.Lock()
			delete(owner.stages, id)
			owner.mu.Unlock()
		})
	}
	return ctx, done, nil
}

func (owner *maintenanceConnectionOwner) beforeConnect(ctx context.Context, config *pgx.ConnConfig) error {
	attempt := &maintenanceAttempt{}
	owner.mu.Lock()
	if owner.quiescing || maintenanceContextError(owner.operation) != nil {
		owner.mu.Unlock()
		return maintenanceFail("maintenance.failed", context.Canceled)
	}
	owner.attempts[attempt] = struct{}{}
	owner.mu.Unlock()
	dial := config.DialFunc
	config.LookupFunc = func(ctx context.Context, host string) ([]string, error) {
		stage, finish, err := owner.stage(ctx, attempt)
		if err != nil {
			return nil, err
		}
		defer finish()
		return maintenanceLiteralLookup(stage, host)
	}
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		return owner.dial(ctx, attempt, dial, network, address)
	}
	config.Tracer = &maintenanceConnectTracer{owner: owner, attempt: attempt}
	return nil
}

func (owner *maintenanceConnectionOwner) dial(parent context.Context, attempt *maintenanceAttempt, dial pgconn.DialFunc, network, address string) (net.Conn, error) {
	ctx, finish, err := owner.stage(parent, attempt)
	if err != nil {
		return nil, err
	}
	defer finish()
	host, port, addressErr := net.SplitHostPort(address)
	_, portErr := strconv.ParseUint(port, 10, 16)
	if addressErr != nil || net.ParseIP(host) == nil || port == "" ||
		portErr != nil ||
		(network != "tcp" && network != "tcp4" && network != "tcp6") {
		return nil, maintenanceFail("maintenance.invalid_config", nil)
	}
	connection, err := dial(ctx, network, address)
	if err != nil {
		if connection != nil {
			_ = connection.Close()
		}
		return nil, err
	}
	transport := &maintenanceTransport{Conn: connection, owner: owner}
	owner.mu.Lock()
	reject := owner.quiescing || attempt.closing || maintenanceContextError(ctx) != nil
	if !reject {
		owner.sockets[transport] = attempt
	}
	owner.mu.Unlock()
	if reject {
		_ = transport.Close()
		return nil, maintenanceFail("maintenance.failed", context.Canceled)
	}
	return transport, nil
}

type maintenanceTransport struct {
	net.Conn
	owner         *maintenanceConnectionOwner
	once          sync.Once
	err           error
	deadlineMu    sync.Mutex
	cleanupCutoff time.Time
}

func (transport *maintenanceTransport) clampDeadline(deadline time.Time) time.Time {
	if !transport.cleanupCutoff.IsZero() && (deadline.IsZero() || deadline.After(transport.cleanupCutoff)) {
		return transport.cleanupCutoff
	}
	return deadline
}

func (transport *maintenanceTransport) limitCleanupDeadline(cutoff time.Time) error {
	transport.deadlineMu.Lock()
	defer transport.deadlineMu.Unlock()
	if transport.cleanupCutoff.IsZero() || cutoff.Before(transport.cleanupCutoff) {
		transport.cleanupCutoff = cutoff
	}
	return transport.Conn.SetDeadline(transport.cleanupCutoff)
}

func (transport *maintenanceTransport) SetDeadline(deadline time.Time) error {
	transport.deadlineMu.Lock()
	defer transport.deadlineMu.Unlock()
	return transport.Conn.SetDeadline(transport.clampDeadline(deadline))
}

func (transport *maintenanceTransport) SetReadDeadline(deadline time.Time) error {
	transport.deadlineMu.Lock()
	defer transport.deadlineMu.Unlock()
	return transport.Conn.SetReadDeadline(transport.clampDeadline(deadline))
}

func (transport *maintenanceTransport) SetWriteDeadline(deadline time.Time) error {
	transport.deadlineMu.Lock()
	defer transport.deadlineMu.Unlock()
	return transport.Conn.SetWriteDeadline(transport.clampDeadline(deadline))
}

func (transport *maintenanceTransport) Close() error {
	transport.once.Do(func() {
		transport.err = transport.Conn.Close()
		transport.owner.mu.Lock()
		transport.owner.transportErr = errors.Join(transport.owner.transportErr, transport.err)
		delete(transport.owner.sockets, transport)
		transport.owner.mu.Unlock()
	})
	return transport.err
}

type maintenanceConnectTracer struct {
	owner   *maintenanceConnectionOwner
	attempt *maintenanceAttempt
	finish  func()
}

func (tracer *maintenanceConnectTracer) TraceConnectStart(ctx context.Context, _ pgx.TraceConnectStartData) context.Context {
	stage, finish, err := tracer.owner.stage(ctx, tracer.attempt)
	if err == nil {
		tracer.finish = finish
		return stage
	}
	canceled, cancel := context.WithCancel(tracer.owner.operation)
	cancel()
	return canceled
}

func (tracer *maintenanceConnectTracer) TraceConnectEnd(_ context.Context, data pgx.TraceConnectEndData) {
	if data.Err == nil && data.Conn != nil {
		tracer.owner.register(data.Conn, tracer.attempt)
	}
	if tracer.finish != nil {
		tracer.finish()
	}
}

func (*maintenanceConnectTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return ctx
}
func (*maintenanceConnectTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

type maintenanceCloseEntry struct {
	attempt     *maintenanceAttempt
	closeSDK    func(context.Context) error
	isClosed    func() bool
	cleanupDone <-chan struct{}
	mu          sync.Mutex
	started     bool
	finished    chan struct{}
	err         error
}

func (owner *maintenanceConnectionOwner) register(connection *pgx.Conn, attempt *maintenanceAttempt) {
	entry := &maintenanceCloseEntry{attempt: attempt, closeSDK: connection.Close,
		isClosed: connection.IsClosed, cleanupDone: connection.PgConn().CleanupDone(), finished: make(chan struct{})}
	owner.mu.Lock()
	owner.connections[connection] = entry
	owner.mu.Unlock()
}

func (owner *maintenanceConnectionOwner) afterConnect(_ context.Context, connection *pgx.Conn) error {
	owner.mu.Lock()
	entry := owner.connections[connection]
	reject := owner.quiescing || maintenanceContextError(owner.operation) != nil
	if entry != nil && !reject {
		entry.attempt.handed = true
	}
	owner.mu.Unlock()
	if entry == nil {
		return maintenanceFail("maintenance.failed", nil)
	}
	if !reject {
		return nil
	}
	ctx, cancel := maintenanceCleanupContext(owner.operation, owner.deadline, owner.budget)
	defer cancel()
	return errors.Join(maintenanceFail("maintenance.failed", context.Canceled), owner.closeEntry(ctx, entry))
}

func (owner *maintenanceConnectionOwner) closeMaintenanceConnection(ctx context.Context, connection *pgx.Conn) error {
	owner.mu.Lock()
	entry := owner.connections[connection]
	owner.mu.Unlock()
	if entry == nil {
		return maintenanceFail("maintenance.failed", nil)
	}
	return owner.closeEntry(ctx, entry)
}

func (owner *maintenanceConnectionOwner) closeEntry(ctx context.Context, entry *maintenanceCloseEntry) error {
	entry.mu.Lock()
	if entry.started {
		finished := entry.finished
		entry.mu.Unlock()
		<-finished
		entry.mu.Lock()
		err := entry.err
		entry.mu.Unlock()
		return err
	}
	entry.started = true
	entry.mu.Unlock()
	owner.mu.Lock()
	entry.attempt.closing = true
	owner.mu.Unlock()
	err := owner.finishConnectionClose(ctx, entry)
	entry.mu.Lock()
	entry.err = err
	close(entry.finished)
	entry.mu.Unlock()
	return err
}

func (owner *maintenanceConnectionOwner) finishConnectionClose(ctx context.Context, entry *maintenanceCloseEntry) (err error) {
	cleanup, cancel := owner.connectionCloseContext(ctx)
	defer cancel()
	cutoff, _ := cleanup.Deadline()
	err = owner.limitAttemptDeadline(entry.attempt, cutoff)
	stopInterrupt := watchMaintenanceCleanup(cleanup, func() { owner.interrupt(entry.attempt) })
	defer func() {
		stopInterrupt()
		err = owner.finishCleanup(cleanup, err, time.Now())
	}()
	if entry.isClosed() {
		owner.interrupt(entry.attempt)
	} else {
		err = errors.Join(err, entry.closeSDK(cleanup))
	}
	select {
	case <-entry.cleanupDone:
	case <-cleanup.Done():
		owner.interrupt(entry.attempt)
		// Interrupt first, then join the existing SDK task; never abandon it.
		<-entry.cleanupDone
		err = errors.Join(err, cleanup.Err())
	}
	return err
}

func (owner *maintenanceConnectionOwner) connectionCloseContext(parent context.Context) (context.Context, context.CancelFunc) {
	cutoff := owner.deadline.Add(owner.budget)
	if local := time.Now().Add(owner.budget); local.Before(cutoff) {
		cutoff = local
	}
	if deadline, ok := parent.Deadline(); ok && deadline.Before(cutoff) {
		cutoff = deadline
	}
	owner.mu.Lock()
	if owner.cleanupCutoff.IsZero() || cutoff.Before(owner.cleanupCutoff) {
		owner.cleanupCutoff = cutoff
	}
	cutoff = owner.cleanupCutoff
	owner.mu.Unlock()
	return context.WithDeadline(parent, cutoff)
}

// Record before self-cancel, and check monotonic completion even if a timer was late.
// Once observed, a cleanup failure survives later callbacks and drained counters.
func (owner *maintenanceConnectionOwner) finishCleanup(ctx context.Context, err error, end time.Time) error {
	err = errors.Join(err, ctx.Err())
	if cutoff, ok := ctx.Deadline(); ok && end.After(cutoff) {
		err = errors.Join(err, context.DeadlineExceeded)
	}
	owner.mu.Lock()
	owner.cleanupErr = errors.Join(owner.cleanupErr, err)
	err = owner.cleanupErr
	owner.mu.Unlock()
	return err
}

func (owner *maintenanceConnectionOwner) limitAttemptDeadline(attempt *maintenanceAttempt, cutoff time.Time) error {
	owner.mu.Lock()
	var sockets []*maintenanceTransport
	for socket, registered := range owner.sockets {
		if registered == attempt {
			sockets = append(sockets, socket)
		}
	}
	owner.mu.Unlock()
	var err error
	for _, socket := range sockets {
		err = errors.Join(err, socket.limitCleanupDeadline(cutoff))
	}
	return err
}

func watchMaintenanceCleanup(ctx context.Context, interrupt func()) func() {
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(finished); interrupt() })
	var once sync.Once
	return func() {
		once.Do(func() {
			if !stop() {
				<-finished
			}
		})
	}
}

func (owner *maintenanceConnectionOwner) interrupt(attempt *maintenanceAttempt) {
	owner.mu.Lock()
	var cancels []context.CancelFunc
	var sockets []*maintenanceTransport
	for _, stage := range owner.stages {
		if attempt == nil || stage.attempt == attempt {
			cancels = append(cancels, stage.cancel)
		}
	}
	for socket, registered := range owner.sockets {
		if attempt == nil || registered == attempt {
			sockets = append(sockets, socket)
		}
	}
	owner.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	for _, socket := range sockets {
		_ = socket.Close()
	}
}

func (owner *maintenanceConnectionOwner) quiesce(interruptAll bool) {
	owner.mu.Lock()
	owner.quiescing = true
	var cancels []context.CancelFunc
	var sockets []*maintenanceTransport
	for _, stage := range owner.stages {
		cancels = append(cancels, stage.cancel)
	}
	for socket, attempt := range owner.sockets {
		if interruptAll || !attempt.handed {
			sockets = append(sockets, socket)
		}
	}
	owner.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	for _, socket := range sockets {
		_ = socket.Close()
	}
}

func (owner *maintenanceConnectionOwner) configure(config *pgxpool.Config) {
	config.MinConns, config.MinIdleConns = 0, 0
	config.AfterRelease = nil
	config.BeforeConnect, config.AfterConnect = owner.beforeConnect, owner.afterConnect
	config.BeforeClose = func(connection *pgx.Conn) {
		ctx, cancel := maintenanceCleanupContext(owner.operation, owner.deadline, owner.budget)
		defer cancel()
		_ = owner.closeMaintenanceConnection(ctx, connection)
	}
}

func (owner *maintenanceConnectionOwner) drained(state func() (int32, int32)) bool {
	constructing, acquired := state()
	owner.mu.Lock()
	stages := len(owner.stages)
	owner.mu.Unlock()
	return constructing == 0 && acquired == 0 && stages == 0
}

func (owner *maintenanceConnectionOwner) waitDrained(ctx context.Context, state func() (int32, int32)) error {
	var deadlineErr error
	for !owner.drained(state) {
		if ctx.Err() != nil {
			deadlineErr = ctx.Err()
			owner.interrupt(nil)
		}
		// Constructors already hold the original canceled context and registered transports.
		// Keep joining after an interrupt instead of leaking a detached constructor.
		timer := time.NewTimer(time.Millisecond)
		<-timer.C
	}
	owner.mu.Lock()
	clear(owner.attempts)
	owner.mu.Unlock()
	return deadlineErr
}

func (owner *maintenanceConnectionOwner) cleanup(state func() (int32, int32), closePool func()) (err error) {
	ctx, cancel := owner.connectionCloseContext(context.WithoutCancel(owner.operation))
	defer cancel()
	defer func() { err = owner.finishCleanup(ctx, err, time.Now()) }()
	owner.stopWatch()
	owner.quiesce(false)
	err = owner.waitDrained(ctx, state)
	owner.mu.Lock()
	entries := make([]*maintenanceCloseEntry, 0, len(owner.connections))
	for _, entry := range owner.connections {
		entries = append(entries, entry)
	}
	owner.mu.Unlock()
	for _, entry := range entries {
		err = errors.Join(err, owner.closeEntry(ctx, entry))
	}
	owner.interrupt(nil)
	closePool()
	owner.mu.Lock()
	err = errors.Join(err, owner.transportErr)
	owner.mu.Unlock()
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	return err
}

type maintenanceExecute func(context.Context, *pgxpool.Pool, int) (maintenanceOutput, error)
type maintenanceValidate func(context.Context, []byte) (maintenanceExecute, error)

type maintenanceRootDependencies struct {
	getenv    func(string) string
	prepare   func(maintenanceConfig) (maintenanceValidate, error)
	parse     func(string) (*pgxpool.Config, error)
	connect   func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error)
	ping      func(context.Context, *pgxpool.Pool) error
	state     func(*pgxpool.Pool) (int32, int32)
	closePool func(*pgxpool.Pool)
}

func runMaintenance(ctx context.Context, input *os.File, dependencies maintenanceRootDependencies) (result maintenanceOutput, err error) {
	stdin := &maintenanceInputOwner{file: input}
	operationOwnsInput := false
	defer func() {
		if recovered := recover(); recovered != nil {
			err = maintenanceFail("maintenance.failed", nil)
		}
		if !operationOwnsInput {
			err = maintenanceFinalError(err, stdin.Close(), ctx)
		}
	}()
	started := time.Now()
	config, err := loadMaintenanceConfig(dependencies.getenv)
	if err != nil {
		return result, err
	}
	deadline, err := maintenanceAddBudget(started, config.operationBudget)
	if err != nil {
		return result, err
	}
	if _, err = maintenanceAddBudget(deadline, config.shutdownBudget); err != nil {
		return result, err
	}
	operation, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	// LIFO: release input and check the live operation before self-cancel.
	defer func() { err = maintenanceFinalError(err, stdin.Close(), operation) }()
	operationOwnsInput = true
	if err := maintenanceContextError(operation); err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	validate, err := dependencies.prepare(config)
	if err != nil {
		return result, err
	}
	envelope, err := stdin.read(operation)
	if err != nil {
		return result, err
	}
	execute, err := validate(operation, envelope)
	if err != nil {
		return result, err
	}
	return executeMaintenanceDatabase(operation, config, execute, dependencies)
}

func executeMaintenanceDatabase(ctx context.Context, config maintenanceConfig, execute maintenanceExecute, dependencies maintenanceRootDependencies) (result maintenanceOutput, err error) {
	if err := maintenanceContextError(ctx); err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	dsn := dependencies.getenv(databaseDSNEnvironment)
	if strings.TrimSpace(dsn) == "" {
		return result, maintenanceFail("maintenance.invalid_config", nil)
	}
	poolConfig, err := dependencies.parse(dsn)
	if err != nil {
		return result, maintenanceFail("maintenance.invalid_config", err)
	}
	if err := validateMaintenanceEndpoints(poolConfig); err != nil {
		return result, err
	}
	if config.connectionBudget > int(poolConfig.MaxConns) {
		return result, maintenanceFail("maintenance.invalid_config", nil)
	}
	owner := newMaintenanceConnectionOwner(ctx, config.shutdownBudget)
	owner.configure(poolConfig)
	var pool *pgxpool.Pool
	defer func() {
		cleanupErr := owner.cleanup(func() (int32, int32) {
			if pool == nil {
				return 0, 0
			}
			return dependencies.state(pool)
		}, func() {
			if pool != nil {
				dependencies.closePool(pool)
			}
		})
		err = maintenanceFinalError(err, cleanupErr, ctx)
	}()
	if err = maintenanceContextError(ctx); err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	pool, err = dependencies.connect(ctx, poolConfig)
	if err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	if err = maintenanceContextError(ctx); err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	if err = dependencies.ping(ctx, pool); err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	if err = maintenanceContextError(ctx); err != nil {
		return result, maintenanceFail("maintenance.failed", err)
	}
	return execute(ctx, pool, config.connectionBudget)
}

func maintenanceContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func maintenanceFinalError(primary, cleanup error, operation context.Context) error {
	cause := errors.Join(cleanup, maintenanceContextError(operation))
	if cause == nil {
		return primary
	}
	if primary != nil {
		return errors.Join(primary, cause)
	}
	return maintenanceFail("maintenance.failed", cause)
}

func maintenanceLiteralLookup(ctx context.Context, host string) ([]string, error) {
	if err := maintenanceContextError(ctx); err != nil {
		return nil, err
	}
	if net.ParseIP(host) == nil {
		return nil, maintenanceFail("maintenance.invalid_config", nil)
	}
	return []string{host}, nil
}

func validateMaintenanceEndpoints(config *pgxpool.Config) error {
	if config == nil || config.ConnConfig == nil || net.ParseIP(config.ConnConfig.Host) == nil {
		return maintenanceFail("maintenance.invalid_config", nil)
	}
	for _, fallback := range config.ConnConfig.Fallbacks {
		if fallback == nil || net.ParseIP(fallback.Host) == nil {
			return maintenanceFail("maintenance.invalid_config", nil)
		}
	}
	return nil
}

type maintenanceClock struct{}

func (maintenanceClock) Now() time.Time { return time.Now() }

func prepareMaintenanceOperator(config maintenanceConfig) (maintenanceValidate, error) {
	operator, err := identity.NewPasswordResetMaintenanceOperator(identity.PasswordResetMaintenanceConfig{
		TrustJSON: config.trust, MaxTTL: config.maxTTL, OperationBudget: config.operationBudget,
		ShutdownBudget: config.shutdownBudget,
	}, maintenanceClock{})
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, envelope []byte) (maintenanceExecute, error) {
		request, err := operator.ValidateRequeue(ctx, envelope)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, pool *pgxpool.Pool, budget int) (maintenanceOutput, error) {
			permit, err := identity.NewPasswordResetConnectionPermit(budget)
			if err != nil {
				return maintenanceOutput{}, maintenanceFail("maintenance.invalid_config", err)
			}
			result, err := operator.Requeue(ctx, request, identity.PasswordResetRequeueDependencies{
				Transactions: pool, Store: &identity.PasswordResetStore{}, Audit: audit.Store{}, Permit: permit,
			})
			if err != nil {
				return maintenanceOutput{}, err
			}
			return maintenanceOutput{RequestID: result.Receipt.RequestID, OperationID: result.Receipt.OperationID,
				Outcome: string(result.Disposition), AuditID: result.Receipt.AuditID}, nil
		}, nil
	}, nil
}

func productionMaintenanceDependencies() maintenanceRootDependencies {
	return maintenanceRootDependencies{
		getenv: os.Getenv, prepare: prepareMaintenanceOperator,
		parse:   func(dsn string) (*pgxpool.Config, error) { return parseDatabasePoolConfig(dsn, os.Getenv) },
		connect: pgxpool.NewWithConfig,
		ping:    func(ctx context.Context, pool *pgxpool.Pool) error { return pool.Ping(ctx) },
		state: func(pool *pgxpool.Pool) (int32, int32) {
			stat := pool.Stat()
			return stat.ConstructingConns(), stat.AcquiredConns()
		},
		closePool: func(pool *pgxpool.Pool) { pool.Close() },
	}
}

func maintenanceErrorCode(err error) (string, int) {
	var domain *identity.PasswordResetMaintenanceError
	var root *maintenanceFailure
	code := "maintenance.failed"
	if errors.As(err, &domain) {
		code = string(domain.Code)
	} else if errors.As(err, &root) {
		code = root.code
	}
	switch code {
	case "maintenance.invalid_request", "maintenance.invalid_config", "maintenance.mode_conflict":
		return code, 2
	case "maintenance.unsupported_platform", "maintenance.authority_unavailable", "maintenance.forbidden":
		return code, 3
	case "maintenance.state_conflict":
		return code, 4
	case "maintenance.outcome_unknown":
		return code, 5
	case "maintenance.failed":
		return code, 6
	default:
		return "maintenance.failed", 6
	}
}

func finishMaintenanceCommand(stdout, stderr io.Writer, result maintenanceOutput, err error) int {
	if err == nil {
		err = writeMaintenanceOutput(stdout, result)
	}
	if err == nil {
		return 0
	}
	code, exit := maintenanceErrorCode(err)
	if _, writeErr := io.WriteString(stderr, code+"\n"); writeErr != nil && exit != 5 {
		return 6
	}
	return exit
}

func maintenanceMain(routeErr error, input *os.File, stdout, stderr io.Writer) (code int) {
	if routeErr != nil {
		if input != nil {
			_ = input.Close()
		}
		return finishMaintenanceCommand(stdout, stderr, maintenanceOutput{}, routeErr)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	result, err := runMaintenance(ctx, input, productionMaintenanceDependencies())
	stop()
	return finishMaintenanceCommand(stdout, stderr, result, err)
}
