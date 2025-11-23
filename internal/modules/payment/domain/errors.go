package domain

import "errors"

var (
	// ErrPaymentNotFound is returned when a payment is not found
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrPaymentAlreadyExists is returned when a payment with the same payment reference already exists
	ErrPaymentAlreadyExists = errors.New("payment with this reference already exists")
	// ErrInvalidPaymentID is returned when the payment ID is invalid
	ErrInvalidPaymentID = errors.New("invalid payment ID")
	// ErrInvalidPaymentStatus is returned when the payment status is invalid
	ErrInvalidPaymentStatus = errors.New("invalid payment status")

	// ErrInvalidAmount is returned when payment amount is invalid
	ErrInvalidAmount = errors.New("invalid payment amount")
	// ErrPaymentExpired is returned when payment has expired
	ErrPaymentExpired = errors.New("payment has expired")
	// ErrPaymentAlreadyCompleted is returned when payment is already completed
	ErrPaymentAlreadyCompleted = errors.New("payment is already completed")
	// ErrInvalidPaymentState is returned when payment is in invalid state for operation
	ErrInvalidPaymentState = errors.New("payment is in invalid state for this operation")
	// ErrAmountMismatch is returned when actual amount doesn't match expected amount
	ErrAmountMismatch = errors.New("payment amount mismatch")
	// ErrInvalidChain is returned when chain is not supported
	ErrInvalidChain = errors.New("invalid or unsupported blockchain chain")
	// ErrInvalidSignature is returned when wallet signature verification fails
	ErrInvalidSignature = errors.New("invalid wallet signature: proof of ownership failed")
	// ErrMerchantNotFound is returned when merchant is not found
	ErrMerchantNotFound = errors.New("merchant not found")
	// ErrMerchantNotApproved is returned when merchant is not approved
	ErrMerchantNotApproved = errors.New("merchant is not approved")
)
