package web3

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

var (
	// ErrInvalidSignature is returned when signature verification fails
	ErrInvalidSignature = errors.New("invalid signature: verification failed")
	// ErrInvalidWalletAddress is returned when wallet address format is invalid
	ErrInvalidWalletAddress = errors.New("invalid wallet address format")
	// ErrInvalidMessageFormat is returned when message format is invalid
	ErrInvalidMessageFormat = errors.New("invalid message format")
)

// VerifySolanaSignature verifies that a message was signed by the claimed wallet address
// This implements "Proof of Ownership" for unhosted wallets (PRD v2.2 requirement)
//
// Parameters:
//   - walletAddress: Base58-encoded Solana public key (e.g., "5FHwkr...")
//   - message: Plain text message that was signed (e.g., "I own this wallet: 5FHwkr...")
//   - signatureBase64: Base64-encoded signature bytes
//
// Returns:
//   - true if signature is valid and matches the wallet address
//   - false if signature is invalid or doesn't match
//   - error for invalid inputs or verification errors
//
// Security Notes:
//   - ALWAYS verify signatures before accepting unhosted wallet transactions > $1000
//   - Prevents wallet address spoofing and unauthorized transactions
//   - Message should include payment_id or nonce to prevent replay attacks
func VerifySolanaSignature(walletAddress string, message string, signatureBase64 string) (bool, error) {
	// Validate inputs
	if walletAddress == "" {
		return false, ErrInvalidWalletAddress
	}
	if message == "" {
		return false, ErrInvalidMessageFormat
	}
	if signatureBase64 == "" {
		return false, errors.New("signature cannot be empty")
	}

	// Parse wallet address (Base58 -> PublicKey)
	publicKey, err := solana.PublicKeyFromBase58(walletAddress)
	if err != nil {
		return false, fmt.Errorf("%w: failed to parse wallet address: %v", ErrInvalidWalletAddress, err)
	}

	// Decode signature from Base64
	signatureBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature from base64: %w", err)
	}

	// Parse signature bytes into Solana Signature type
	if len(signatureBytes) != 64 {
		return false, fmt.Errorf("invalid signature length: expected 64 bytes, got %d bytes", len(signatureBytes))
	}

	var signature solana.Signature
	copy(signature[:], signatureBytes)

	// Convert message to bytes
	messageBytes := []byte(message)

	// Verify signature against message and public key
	// Solana uses Ed25519 signature scheme
	isValid := signature.Verify(publicKey, messageBytes)

	if !isValid {
		return false, ErrInvalidSignature
	}

	return true, nil
}

// GenerateChallengeMessage generates a challenge message for wallet verification
// The message should be unique per payment to prevent replay attacks
//
// Format: "Sign this message to prove ownership of wallet {walletAddress} for payment {paymentID}"
//
// Parameters:
//   - walletAddress: Solana wallet address
//   - paymentID: Unique payment identifier (prevents replay attacks)
//
// Returns:
//   - Challenge message string that the user must sign with their private key
func GenerateChallengeMessage(walletAddress string, paymentID string) string {
	return fmt.Sprintf(
		"Sign this message to prove ownership of wallet %s for payment %s",
		walletAddress,
		paymentID,
	)
}

// VerifyPaymentSignature is a convenience wrapper that verifies a signature for a specific payment
// It generates the expected challenge message and verifies the signature
//
// Parameters:
//   - walletAddress: Base58-encoded Solana public key
//   - paymentID: Payment identifier
//   - signatureBase64: Base64-encoded signature
//
// Returns:
//   - true if signature is valid for this payment
//   - false if signature is invalid
//   - error for invalid inputs
func VerifyPaymentSignature(walletAddress string, paymentID string, signatureBase64 string) (bool, error) {
	// Generate expected message
	expectedMessage := GenerateChallengeMessage(walletAddress, paymentID)

	// Verify signature
	return VerifySolanaSignature(walletAddress, expectedMessage, signatureBase64)
}
