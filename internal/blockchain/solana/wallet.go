package solana

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
)

// Wallet represents a Solana wallet with signing capabilities
type Wallet struct {
	privateKey solana.PrivateKey
	publicKey  solana.PublicKey
	rpcClient  *rpc.Client
}

// TokenInfo contains information about SPL token balances
type TokenInfo struct {
	Mint     solana.PublicKey `json:"mint"`
	Symbol   string           `json:"symbol"`
	Balance  decimal.Decimal  `json:"balance"`
	Decimals uint8            `json:"decimals"`
}

// WalletBalance contains comprehensive balance information
type WalletBalance struct {
	SOL   decimal.Decimal          `json:"sol"`
	Tokens map[string]TokenInfo    `json:"tokens"`
}

// LoadWallet creates a wallet instance from a base58-encoded private key
// The private key should be in the format exported by Solana CLI or Phantom wallet
func LoadWallet(privateKeyBase58 string, rpcURL string) (*Wallet, error) {
	if privateKeyBase58 == "" {
		return nil, fmt.Errorf("private key cannot be empty")
	}

	if rpcURL == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	// Decode base58-encoded private key
	privateKeyBytes, err := base58.Decode(privateKeyBase58)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Validate private key length (should be 64 bytes for ed25519)
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key length: expected %d bytes, got %d bytes",
			ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	// Create Solana private key
	privateKey := solana.PrivateKey(privateKeyBytes)
	publicKey := privateKey.PublicKey()

	// Create RPC client
	rpcClient := rpc.New(rpcURL)

	wallet := &Wallet{
		privateKey: privateKey,
		publicKey:  publicKey,
		rpcClient:  rpcClient,
	}

	return wallet, nil
}

// GetAddress returns the wallet's public address as a base58 string
func (w *Wallet) GetAddress() string {
	return w.publicKey.String()
}

// GetPublicKey returns the wallet's public key
func (w *Wallet) GetPublicKey() solana.PublicKey {
	return w.publicKey
}

// GetPrivateKey returns the wallet's private key
// WARNING: This should only be used for transaction signing
// Never log or expose the private key
func (w *Wallet) GetPrivateKey() solana.PrivateKey {
	return w.privateKey
}

// GetSOLBalance returns the wallet's SOL balance in SOL (not lamports)
func (w *Wallet) GetSOLBalance(ctx context.Context) (decimal.Decimal, error) {
	// Get balance in lamports
	balance, err := w.rpcClient.GetBalance(
		ctx,
		w.publicKey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get SOL balance: %w", err)
	}

	// Convert lamports to SOL (1 SOL = 1,000,000,000 lamports)
	lamportsDecimal := decimal.NewFromUint64(balance.Value)
	solBalance := lamportsDecimal.Div(decimal.NewFromInt(1_000_000_000))

	return solBalance, nil
}

// GetTokenBalance returns the balance of a specific SPL token
// tokenMint is the token's mint address (e.g., USDT, USDC)
func (w *Wallet) GetTokenBalance(ctx context.Context, tokenMint string) (decimal.Decimal, error) {
	if tokenMint == "" {
		return decimal.Zero, fmt.Errorf("token mint cannot be empty")
	}

	// Parse token mint address
	mintPubkey, err := solana.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid token mint address: %w", err)
	}

	// Get the associated token account for this wallet and mint
	ata, _, err := solana.FindAssociatedTokenAddress(
		w.publicKey,
		mintPubkey,
	)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to find associated token account: %w", err)
	}

	// Get account info
	accountInfo, err := w.rpcClient.GetAccountInfo(ctx, ata)
	if err != nil {
		// Account doesn't exist, return zero balance
		return decimal.Zero, nil
	}

	if accountInfo == nil || accountInfo.Value == nil {
		// Account doesn't exist, return zero balance
		return decimal.Zero, nil
	}

	// Parse token account data
	var tokenAccount token.Account
	err = tokenAccount.UnmarshalWithDecoder(solana.NewBinDecoder(accountInfo.Value.Data.GetBinary()))
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse token account: %w", err)
	}

	// Get mint info to determine decimals
	mintInfo, err := w.GetTokenMintInfo(ctx, tokenMint)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get mint info: %w", err)
	}

	// Convert raw amount to decimal based on token decimals
	rawAmount := decimal.NewFromBigInt(tokenAccount.Amount.BigInt(), 0)
	divisor := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(mintInfo.Decimals)))
	balance := rawAmount.Div(divisor)

	return balance, nil
}

// GetTokenMintInfo retrieves information about a token mint
func (w *Wallet) GetTokenMintInfo(ctx context.Context, tokenMint string) (*token.Mint, error) {
	mintPubkey, err := solana.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return nil, fmt.Errorf("invalid token mint address: %w", err)
	}

	accountInfo, err := w.rpcClient.GetAccountInfo(ctx, mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get mint account info: %w", err)
	}

	if accountInfo == nil || accountInfo.Value == nil {
		return nil, fmt.Errorf("mint account not found")
	}

	var mintInfo token.Mint
	err = mintInfo.UnmarshalWithDecoder(solana.NewBinDecoder(accountInfo.Value.Data.GetBinary()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse mint info: %w", err)
	}

	return &mintInfo, nil
}

// GetUSDTBalance is a convenience method to get USDT balance
func (w *Wallet) GetUSDTBalance(ctx context.Context, usdtMint string) (decimal.Decimal, error) {
	return w.GetTokenBalance(ctx, usdtMint)
}

// GetUSDCBalance is a convenience method to get USDC balance
func (w *Wallet) GetUSDCBalance(ctx context.Context, usdcMint string) (decimal.Decimal, error) {
	return w.GetTokenBalance(ctx, usdcMint)
}

// GetWalletBalance returns comprehensive wallet balance including SOL and all SPL tokens
func (w *Wallet) GetWalletBalance(ctx context.Context, tokenMints map[string]string) (*WalletBalance, error) {
	balance := &WalletBalance{
		Tokens: make(map[string]TokenInfo),
	}

	// Get SOL balance
	solBalance, err := w.GetSOLBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SOL balance: %w", err)
	}
	balance.SOL = solBalance

	// Get token balances
	for symbol, mintAddress := range tokenMints {
		tokenBalance, err := w.GetTokenBalance(ctx, mintAddress)
		if err != nil {
			// Log error but continue with other tokens
			continue
		}

		mintInfo, err := w.GetTokenMintInfo(ctx, mintAddress)
		if err != nil {
			continue
		}

		mintPubkey, _ := solana.PublicKeyFromBase58(mintAddress)
		balance.Tokens[symbol] = TokenInfo{
			Mint:     mintPubkey,
			Symbol:   symbol,
			Balance:  tokenBalance,
			Decimals: mintInfo.Decimals,
		}
	}

	return balance, nil
}

// SignTransaction signs a Solana transaction with the wallet's private key
func (w *Wallet) SignTransaction(tx *solana.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	// Create list of signers
	signers := []solana.PrivateKey{w.privateKey}

	// Sign the transaction
	_, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		for _, signer := range signers {
			if signer.PublicKey() == key {
				return &signer
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	return nil
}

// SignAndSendTransaction signs and sends a transaction to the Solana network
func (w *Wallet) SignAndSendTransaction(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	// Get recent blockhash
	recent, err := w.rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	tx.Message.RecentBlockhash = recent.Value.Blockhash

	// Sign transaction
	if err := w.SignTransaction(tx); err != nil {
		return solana.Signature{}, err
	}

	// Send transaction
	sig, err := w.rpcClient.SendTransactionWithOpts(
		ctx,
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig, nil
}

// VerifyTransaction verifies that a transaction has been finalized
func (w *Wallet) VerifyTransaction(ctx context.Context, signature solana.Signature, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 30 // Default to 30 retries
	}

	for i := 0; i < maxRetries; i++ {
		status, err := w.rpcClient.GetSignatureStatuses(ctx, true, signature)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if status == nil || len(status.Value) == 0 || status.Value[0] == nil {
			time.Sleep(2 * time.Second)
			continue
		}

		txStatus := status.Value[0]

		// Check if transaction failed
		if txStatus.Err != nil {
			return fmt.Errorf("transaction failed: %v", txStatus.Err)
		}

		// Check if transaction is finalized
		if txStatus.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("transaction not finalized after %d retries", maxRetries)
}

// CreateTransferInstruction creates a SOL transfer instruction
func (w *Wallet) CreateTransferInstruction(to solana.PublicKey, lamports uint64) solana.Instruction {
	return solana.NewTransferInstruction(
		lamports,
		w.publicKey,
		to,
	).Build()
}

// HealthCheck verifies the wallet can connect to the RPC endpoint
func (w *Wallet) HealthCheck(ctx context.Context) error {
	// Try to get wallet balance as a health check
	_, err := w.rpcClient.GetBalance(ctx, w.publicKey, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("wallet health check failed: %w", err)
	}
	return nil
}

// GetRPCClient returns the underlying RPC client for advanced operations
func (w *Wallet) GetRPCClient() *rpc.Client {
	return w.rpcClient
}
