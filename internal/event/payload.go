package event

import (
	"encoding/json"
	"time"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// StreamName is the Redis stream used by the publisher worker.
const StreamName = "wallet_events"

// TransactionEventPayload is the stream payload contract.
type TransactionEventPayload struct {
	EventID              uuid.UUID  `json:"event_id"`
	EventType            string     `json:"event_type"`
	TransactionID        uuid.UUID  `json:"transaction_id"`
	ReferenceCode        string     `json:"reference_code"`
	SourceWalletID       *uuid.UUID `json:"source_wallet_id,omitempty"`
	DestinationWalletID  *uuid.UUID `json:"destination_wallet_id,omitempty"`
	AmountMinor          int64      `json:"amount_minor"`
	Currency             string     `json:"currency"`
	OccurredAt           time.Time  `json:"occurred_at"`
	RelatedTransactionID *uuid.UUID `json:"related_transaction_id,omitempty"`
}

// NewOutboxEvent converts a transaction into an outbox event.
func NewOutboxEvent(eventType string, txModel model.Transaction) (*model.OutboxEvent, error) {
	payload := TransactionEventPayload{
		EventID:              uuid.New(),
		EventType:            eventType,
		TransactionID:        txModel.ID,
		ReferenceCode:        txModel.ReferenceCode,
		SourceWalletID:       txModel.SourceWalletID,
		DestinationWalletID:  txModel.DestinationWalletID,
		AmountMinor:          txModel.AmountMinor,
		Currency:             txModel.Currency,
		OccurredAt:           txModel.CreatedAt,
		RelatedTransactionID: txModel.RelatedTransactionID,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &model.OutboxEvent{
		AggregateType: "transaction",
		AggregateID:   txModel.ID,
		EventType:     eventType,
		PayloadJSON:   datatypes.JSON(raw),
		Status:        model.OutboxStatusPending,
	}, nil
}

// ParsePayload decodes an outbox payload.
func ParsePayload(raw []byte) (TransactionEventPayload, error) {
	var payload TransactionEventPayload
	err := json.Unmarshal(raw, &payload)
	return payload, err
}

// ToStreamValues converts the payload into Redis stream fields.
func (p TransactionEventPayload) ToStreamValues() map[string]any {
	values := map[string]any{
		"event_id":       p.EventID.String(),
		"event_type":     p.EventType,
		"transaction_id": p.TransactionID.String(),
		"reference_code": p.ReferenceCode,
		"amount_minor":   p.AmountMinor,
		"currency":       p.Currency,
		"occurred_at":    p.OccurredAt.UTC().Format(time.RFC3339Nano),
	}

	if p.SourceWalletID != nil {
		values["source_wallet_id"] = p.SourceWalletID.String()
	}
	if p.DestinationWalletID != nil {
		values["destination_wallet_id"] = p.DestinationWalletID.String()
	}
	if p.RelatedTransactionID != nil {
		values["related_transaction_id"] = p.RelatedTransactionID.String()
	}

	return values
}
