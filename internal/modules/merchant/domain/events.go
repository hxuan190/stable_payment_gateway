package domain

import "time"

type MerchantRegisteredEvent struct {
	MerchantID   string
	Email        string
	BusinessName string
	RegisteredAt time.Time
}

func (e MerchantRegisteredEvent) Name() string {
	return "merchant.registered"
}

type MerchantKYCSubmittedEvent struct {
	MerchantID  string
	SubmittedAt time.Time
}

func (e MerchantKYCSubmittedEvent) Name() string {
	return "merchant.kyc_submitted"
}

type MerchantKYCApprovedEvent struct {
	MerchantID string
	KYCTier    int
	ApprovedAt time.Time
}

func (e MerchantKYCApprovedEvent) Name() string {
	return "merchant.kyc_approved"
}

type MerchantKYCRejectedEvent struct {
	MerchantID string
	Reason     string
	RejectedAt time.Time
}

func (e MerchantKYCRejectedEvent) Name() string {
	return "merchant.kyc_rejected"
}
