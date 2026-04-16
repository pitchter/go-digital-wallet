package event_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"go-digital-wallet/internal/event"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/testutil"

	"gorm.io/datatypes"
)

type fakePublisher struct {
	publishErr error
	calls      int
	stream     string
	values     map[string]any
}

func (p *fakePublisher) Publish(_ context.Context, stream string, values map[string]any) error {
	p.calls++
	p.stream = stream
	p.values = values
	return p.publishErr
}

func (p *fakePublisher) Ping(context.Context) error {
	return nil
}

func TestWorkerPublishPendingSuccessMarksPublished(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "worker-success@example.com")
	wallet := env.SeedWallet(t, user.ID, 0)

	transaction := model.Transaction{
		ReferenceCode:       "TEST-WORKER-SUCCESS",
		Type:                model.TransactionTypeTopUp,
		Status:              model.TransactionStatusCompleted,
		SourceWalletID:      &model.SystemWalletID,
		DestinationWalletID: &wallet.ID,
		AmountMinor:         1000,
		Currency:            model.CurrencyTHB,
	}
	if err := env.TransactionRepo.Create(context.Background(), &transaction); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	outboxEvent, err := event.NewOutboxEvent("transaction.completed", transaction)
	if err != nil {
		t.Fatalf("new outbox event: %v", err)
	}
	if err := env.OutboxRepo.Create(context.Background(), outboxEvent); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	publisher := &fakePublisher{}
	worker := event.NewWorker(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		env.OutboxRepo,
		publisher,
		10,
		event.StreamName,
	)

	if err := worker.PublishPending(context.Background()); err != nil {
		t.Fatalf("publish pending: %v", err)
	}

	var stored model.OutboxEvent
	if err := env.DB.First(&stored, "id = ?", outboxEvent.ID).Error; err != nil {
		t.Fatalf("load outbox event: %v", err)
	}
	if stored.Status != model.OutboxStatusPublished {
		t.Fatalf("expected published status, got %s", stored.Status)
	}
	if stored.PublishedAt == nil {
		t.Fatal("expected published_at to be set")
	}
	if publisher.calls != 1 {
		t.Fatalf("expected publisher to be called once, got %d", publisher.calls)
	}
	if publisher.stream != event.StreamName {
		t.Fatalf("expected stream %s, got %s", event.StreamName, publisher.stream)
	}
}

func TestWorkerPublishPendingRecordsFailureForRetry(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "worker@example.com")
	wallet := env.SeedWallet(t, user.ID, 0)

	transaction := model.Transaction{
		ReferenceCode:       "TEST-WORKER",
		Type:                model.TransactionTypeTopUp,
		Status:              model.TransactionStatusCompleted,
		SourceWalletID:      &model.SystemWalletID,
		DestinationWalletID: &wallet.ID,
		AmountMinor:         1000,
		Currency:            model.CurrencyTHB,
	}
	if err := env.TransactionRepo.Create(context.Background(), &transaction); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	outboxEvent, err := event.NewOutboxEvent("transaction.completed", transaction)
	if err != nil {
		t.Fatalf("new outbox event: %v", err)
	}
	if err := env.OutboxRepo.Create(context.Background(), outboxEvent); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	worker := event.NewWorker(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		env.OutboxRepo,
		&fakePublisher{publishErr: errors.New("redis unavailable")},
		10,
		event.StreamName,
	)

	if err := worker.PublishPending(context.Background()); err != nil {
		t.Fatalf("publish pending: %v", err)
	}

	var stored model.OutboxEvent
	if err := env.DB.First(&stored, "id = ?", outboxEvent.ID).Error; err != nil {
		t.Fatalf("load outbox event: %v", err)
	}
	if stored.RetryCount != 1 {
		t.Fatalf("expected retry_count 1, got %d", stored.RetryCount)
	}
	if stored.Status != model.OutboxStatusPending {
		t.Fatalf("expected status pending, got %s", stored.Status)
	}
	if stored.LastError == nil || *stored.LastError == "" {
		t.Fatal("expected last_error to be recorded")
	}
}

func TestWorkerPublishPendingInvalidPayloadRecordsFailure(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	outboxEvent := model.OutboxEvent{
		AggregateType: "transaction",
		AggregateID:   model.SystemWalletID,
		EventType:     "transaction.completed",
		PayloadJSON:   datatypes.JSON([]byte("{invalid")),
		Status:        model.OutboxStatusPending,
	}
	if err := env.OutboxRepo.Create(context.Background(), &outboxEvent); err != nil {
		t.Fatalf("create invalid outbox event: %v", err)
	}

	publisher := &fakePublisher{}
	worker := event.NewWorker(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		env.OutboxRepo,
		publisher,
		10,
		event.StreamName,
	)

	if err := worker.PublishPending(context.Background()); err != nil {
		t.Fatalf("publish pending: %v", err)
	}

	var stored model.OutboxEvent
	if err := env.DB.First(&stored, "id = ?", outboxEvent.ID).Error; err != nil {
		t.Fatalf("load outbox event: %v", err)
	}
	if stored.RetryCount != 1 {
		t.Fatalf("expected retry_count 1, got %d", stored.RetryCount)
	}
	if stored.Status != model.OutboxStatusPending {
		t.Fatalf("expected status pending, got %s", stored.Status)
	}
	if stored.LastError == nil || *stored.LastError == "" {
		t.Fatal("expected last_error to be recorded")
	}
	if publisher.calls != 0 {
		t.Fatalf("expected publisher not to be called, got %d calls", publisher.calls)
	}
}

func TestWorkerPublishPendingEmptyQueue(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	publisher := &fakePublisher{}
	worker := event.NewWorker(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		env.OutboxRepo,
		publisher,
		10,
		event.StreamName,
	)

	if err := worker.PublishPending(context.Background()); err != nil {
		t.Fatalf("publish pending: %v", err)
	}
	if publisher.calls != 0 {
		t.Fatalf("expected no publish calls, got %d", publisher.calls)
	}

	var count int64
	if err := env.DB.Model(&model.OutboxEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox events: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no outbox rows, got %d", count)
	}
}
