package domain

import "time"

type PayoutRequestedEvent struct {
	PayoutID    string
	MerchantID  string
	Amount      string
	RequestedAt time.Time
}

func (e PayoutRequestedEvent) Name() string {
	return "payout.requested"
}

type PayoutApprovedEvent struct {
	PayoutID   string
	ApproverID string
	ApprovedAt time.Time
}

func (e PayoutApprovedEvent) Name() string {
	return "payout.approved"
}

type PayoutRejectedEvent struct {
	PayoutID   string
	Reason     string
	RejectedAt time.Time
}

func (e PayoutRejectedEvent) Name() string {
	return "payout.rejected"
}

type PayoutCompletedEvent struct {
	PayoutID    string
	TxHash      string
	CompletedAt time.Time
}

func (e PayoutCompletedEvent) Name() string {
	return "payout.completed"
}

type PayoutFailedEvent struct {
	PayoutID string
	Reason   string
	FailedAt time.Time
}

func (e PayoutFailedEvent) Name() string {
	return "payout.failed"
}
