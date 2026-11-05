// Package privacy owns ProcessingGate deployment evidence and evaluation.
package privacy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/platform"

	"github.com/jackc/pgx/v5"
)

// ProcessingRecord is one append-only deployment processing registration.
type ProcessingRecord struct {
	ID               int64
	RegionRef        string
	ControllerRef    string
	LegalBasisRef    string
	SMTPProcessorRef string
	EffectiveAt      time.Time
	RecordedBy       string
}

// QueryRower is the minimum PostgreSQL query contract consumed by Store.
type QueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Store reads append-only privacy deployment evidence.
type Store struct {
	queryer QueryRower
}

// NewStore creates a privacy evidence store.
func NewStore(queryer QueryRower) (*Store, error) {
	if queryer == nil {
		return nil, errors.New("privacy queryer is required")
	}
	return &Store{queryer: queryer}, nil
}

// CurrentProcessingRecord returns the latest effective ProcessingGate registration.
func (s *Store) CurrentProcessingRecord(ctx context.Context) (ProcessingRecord, error) {
	if s == nil || s.queryer == nil {
		return ProcessingRecord{}, errors.New("privacy store is not initialized")
	}

	var record ProcessingRecord
	err := s.queryer.QueryRow(ctx, `
		SELECT id, region_ref, controller_ref, legal_basis_ref, smtp_processor_ref, effective_at, recorded_by
		FROM processing_gate_records
		WHERE effective_at <= now()
		ORDER BY effective_at DESC, id DESC
		LIMIT 1
	`).Scan(
		&record.ID,
		&record.RegionRef,
		&record.ControllerRef,
		&record.LegalBasisRef,
		&record.SMTPProcessorRef,
		&record.EffectiveAt,
		&record.RecordedBy,
	)
	if err != nil {
		return ProcessingRecord{}, fmt.Errorf("query current processing record: %w", err)
	}
	return record, nil
}

// ProcessingRecordReader reads current ProcessingGate evidence.
type ProcessingRecordReader interface {
	CurrentProcessingRecord(context.Context) (ProcessingRecord, error)
}

// ProcessingGate compares required environment references with registered evidence.
type ProcessingGate struct {
	config  platform.ProcessingGateConfig
	records ProcessingRecordReader
}

// NewProcessingGate creates a fail-closed ProcessingGate.
func NewProcessingGate(config platform.ProcessingGateConfig, records ProcessingRecordReader) (*ProcessingGate, error) {
	if records == nil {
		return nil, errors.New("processing record reader is required")
	}
	return &ProcessingGate{config: config, records: records}, nil
}

// RequireOpen rejects capabilities when required processing evidence is incomplete or mismatched.
func (g *ProcessingGate) RequireOpen(ctx context.Context) error {
	if g == nil || g.records == nil {
		return closedError(errors.New("processing gate is not initialized"))
	}
	if err := g.config.Validate(); err != nil {
		return closedError(err)
	}

	record, err := g.records.CurrentProcessingRecord(ctx)
	if err != nil {
		return closedError(err)
	}
	if err := validateProcessingRecord(record); err != nil {
		return closedError(err)
	}
	if record.RegionRef != g.config.RegionRef ||
		record.ControllerRef != g.config.ControllerRef ||
		record.LegalBasisRef != g.config.LegalBasisRef ||
		record.SMTPProcessorRef != g.config.SMTPProcessorRef {
		return closedError(errors.New("processing evidence does not match deployment configuration"))
	}
	return nil
}

func validateProcessingRecord(record ProcessingRecord) error {
	if record.ID <= 0 {
		return errors.New("processing evidence ID is invalid")
	}
	if record.EffectiveAt.IsZero() {
		return errors.New("processing evidence effective time is required")
	}
	required := []struct {
		name  string
		value string
	}{
		{name: "region reference", value: record.RegionRef},
		{name: "controller reference", value: record.ControllerRef},
		{name: "legal basis reference", value: record.LegalBasisRef},
		{name: "SMTP processor reference", value: record.SMTPProcessorRef},
		{name: "recorded by", value: record.RecordedBy},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("processing evidence %s is required", item.name)
		}
	}
	return nil
}

func closedError(cause error) error {
	return platform.NewPublicError(platform.CodePlatformProcessingGateClosed, cause)
}
