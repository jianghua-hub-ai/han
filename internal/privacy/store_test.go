package privacy

import (
	"context"
	"errors"
	"testing"
	"time"

	"han/internal/platform"

	"github.com/jackc/pgx/v5"
)

func TestProcessingGate_MissingConfiguration_FailsClosedWithoutReadingEvidence(t *testing.T) {
	reader := &recordReaderStub{
		record: completeProcessingRecord(),
	}
	gate, err := NewProcessingGate(platform.ProcessingGateConfig{}, reader)
	if err != nil {
		t.Fatalf("NewProcessingGate() error = %v", err)
	}

	err = gate.RequireOpen(context.Background())
	if err == nil {
		t.Fatal("RequireOpen() error = nil")
	}
	if code := platform.PublicCode(err); code != platform.CodePlatformProcessingGateClosed {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodePlatformProcessingGateClosed)
	}
	if reader.calls != 0 {
		t.Fatalf("CurrentProcessingRecord() calls = %d, want 0", reader.calls)
	}
}

func TestProcessingGate_RegisteredEvidenceMustExactlyMatchConfiguration(t *testing.T) {
	config := completeProcessingConfig()
	tests := []struct {
		name   string
		record ProcessingRecord
		err    error
		open   bool
	}{
		{name: "matching current evidence", record: completeProcessingRecord(), open: true},
		{name: "missing evidence", err: pgx.ErrNoRows},
		{name: "mismatched processor", record: func() ProcessingRecord {
			record := completeProcessingRecord()
			record.SMTPProcessorRef = "different-processor"
			return record
		}()},
		{name: "incomplete evidence", record: func() ProcessingRecord {
			record := completeProcessingRecord()
			record.RecordedBy = ""
			return record
		}()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := &recordReaderStub{record: test.record, err: test.err}
			gate, err := NewProcessingGate(config, reader)
			if err != nil {
				t.Fatalf("NewProcessingGate() error = %v", err)
			}

			err = gate.RequireOpen(context.Background())
			if test.open {
				if err != nil {
					t.Fatalf("RequireOpen() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("RequireOpen() error = nil")
			}
			if code := platform.PublicCode(err); code != platform.CodePlatformProcessingGateClosed {
				t.Fatalf("PublicCode() = %q, want %q", code, platform.CodePlatformProcessingGateClosed)
			}
		})
	}
}

func TestStore_CurrentProcessingRecord_ReturnsLatestRegisteredEvidence(t *testing.T) {
	want := completeProcessingRecord()
	queryer := queryRowerStub{row: processingRow{record: want}}
	store, err := NewStore(queryer)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	got, err := store.CurrentProcessingRecord(context.Background())
	if err != nil {
		t.Fatalf("CurrentProcessingRecord() error = %v", err)
	}
	if got != want {
		t.Fatalf("CurrentProcessingRecord() = %#v, want %#v", got, want)
	}
}

func completeProcessingConfig() platform.ProcessingGateConfig {
	return platform.ProcessingGateConfig{
		RegionRef:        "region-record",
		ControllerRef:    "controller-record",
		LegalBasisRef:    "basis-record",
		SMTPProcessorRef: "smtp-record",
	}
}

func completeProcessingRecord() ProcessingRecord {
	return ProcessingRecord{
		ID:               7,
		RegionRef:        "region-record",
		ControllerRef:    "controller-record",
		LegalBasisRef:    "basis-record",
		SMTPProcessorRef: "smtp-record",
		EffectiveAt:      time.Date(2026, time.October, 21, 9, 0, 0, 0, time.UTC),
		RecordedBy:       "deployment-owner",
	}
}

type recordReaderStub struct {
	record ProcessingRecord
	err    error
	calls  int
}

func (s *recordReaderStub) CurrentProcessingRecord(context.Context) (ProcessingRecord, error) {
	s.calls++
	return s.record, s.err
}

type queryRowerStub struct {
	row pgx.Row
}

func (s queryRowerStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return s.row
}

type processingRow struct {
	record ProcessingRecord
	err    error
}

func (r processingRow) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(destinations) != 7 {
		return errors.New("unexpected scan destination count")
	}
	*destinations[0].(*int64) = r.record.ID
	*destinations[1].(*string) = r.record.RegionRef
	*destinations[2].(*string) = r.record.ControllerRef
	*destinations[3].(*string) = r.record.LegalBasisRef
	*destinations[4].(*string) = r.record.SMTPProcessorRef
	*destinations[5].(*time.Time) = r.record.EffectiveAt
	*destinations[6].(*string) = r.record.RecordedBy
	return nil
}
