package tron

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
)

// Wallet represents a TRON wallet with signing capabilities
type Wallet struct {
	privateKey string
	publicKey  string
	address    string
	grpcClient *client.GrpcClient
	isMainnet  bool
}

// TRC20TokenInfo contains information about TRC20 token balances
type TRC20TokenInfo struct {
	ContractAddress string          `json:"contract_address"`
	Symbol          string          `json:"symbol"`
	Name            string          `json:"name"`
	Balance         decimal.Decimal `json:"balance"`
	Decimals        uint8           `json:"decimals"`
}

// WalletBalance contains comprehensive balance information
type WalletBalance struct {
	TRX    decimal.Decimal           `json:"trx"`
	Tokens map[string]TRC20TokenInfo `json:"tokens"`
}

// LoadWallet creates a wallet instance from a private key (hex string)
// The private key should be in hex format (64 characters)
func LoadWallet(privateKeyHex string, rpcURL string, isMainnet bool) (*Wallet, error) {
	if privateKeyHex == "" {
		return nil, fmt.Errorf("private key cannot be empty")
	}

	if rpcURL == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	// Validate private key format
	if len(privateKeyHex) != 64 {
		return nil, fmt.Errorf("invalid private key length: expected 64 hex characters, got %d", len(privateKeyHex))
	}

	// Create gRPC client
	grpcClient := client.NewGrpcClient(rpcURL)
	if grpcClient == nil {
		return nil, fmt.Errorf("failed to create TRON gRPC client")
	}

	err := grpcClient.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start TRON gRPC client: %w", err)
	}

	// Derive address from private key
	addr, err := address.PubKeyToAddress(common.FromHex(privateKeyHex))
	if err != nil {
		return nil, fmt.Errorf("failed to derive address from private key: %w", err)
	}

	wallet := &Wallet{
		privateKey: privateKeyHex,
		address:    addr.String(),
		grpcClient: grpcClient,
		isMainnet:  isMainnet,
	}

	return wallet, nil
}

// GenerateWallet creates a new TRON wallet with a random private key
func GenerateWallet(rpcURL string, isMainnet bool) (*Wallet, error) {
	if rpcURL == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	// Generate new key pair
	key, err := keystore.NewKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Create gRPC client
	grpcClient := client.NewGrpcClient(rpcURL)
	if grpcClient == nil {
		return nil, fmt.Errorf("failed to create TRON gRPC client")
	}

	err = grpcClient.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start TRON gRPC client: %w", err)
	}

	wallet := &Wallet{
		privateKey: common.Bytes2Hex(key.PrivateKey.D.Bytes()),
		address:    key.Address.String(),
		grpcClient: grpcClient,
		isMainnet:  isMainnet,
	}

	return wallet, nil
}

// GetAddress returns the wallet's TRON address (base58 encoded)
func (w *Wallet) GetAddress() string {
	return w.address
}

// GetPrivateKey returns the wallet's private key in hex format
// WARNING: This should only be used for transaction signing
// Never log or expose the private key
func (w *Wallet) GetPrivateKey() string {
	return w.privateKey
}

// GetTRXBalance returns the wallet's TRX balance in TRX (not SUN)
// 1 TRX = 1,000,000 SUN
func (w *Wallet) GetTRXBalance(ctx context.Context) (decimal.Decimal, error) {
	// Get account info
	account, err := w.grpcClient.GetAccount(w.address)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		// Account doesn't exist yet, return zero balance
		return decimal.Zero, nil
	}

	// Convert SUN to TRX
	sunBalance := decimal.NewFromInt(account.Balance)
	trxBalance := sunBalance.Div(decimal.NewFromInt(1_000_000))

	return trxBalance, nil
}

// GetTRC20Balance returns the wallet's TRC20 token balance
// contractAddress is the TRC20 token contract address (e.g., USDT: TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t)
func (w *Wallet) GetTRC20Balance(ctx context.Context, contractAddress string) (decimal.Decimal, uint8, error) {
	// Call TRC20 balanceOf function
	result, err := w.grpcClient.TRC20GetBalance(w.address, contractAddress)
	if err != nil {
		return decimal.Zero, 0, fmt.Errorf("failed to get TRC20 balance: %w", err)
	}

	// Get token decimals
	decimals, err := w.grpcClient.TRC20GetDecimals(contractAddress)
	if err != nil {
		// Default to 6 decimals (USDT standard)
		decimals = 6
	}

	// Parse balance
	balance := decimal.NewFromBigInt(result, 0)

	return balance, uint8(decimals), nil
}

// GetFullBalance returns comprehensive balance information including TRX and all TRC20 tokens
func (w *Wallet) GetFullBalance(ctx context.Context, tokenContracts []string) (*WalletBalance, error) {
	balance := &WalletBalance{
		Tokens: make(map[string]TRC20TokenInfo),
	}

	// Get TRX balance
	trxBalance, err := w.GetTRXBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get TRX balance: %w", err)
	}
	balance.TRX = trxBalance

	// Get TRC20 token balances
	for _, contractAddr := range tokenContracts {
		tokenBalance, decimals, err := w.GetTRC20Balance(ctx, contractAddr)
		if err != nil {
			// Skip tokens that fail to fetch
			continue
		}

		// Get token name and symbol
		name, _ := w.grpcClient.TRC20GetName(contractAddr)
		symbol, _ := w.grpcClient.TRC20GetSymbol(contractAddr)

		balance.Tokens[contractAddr] = TRC20TokenInfo{
			ContractAddress: contractAddr,
			Symbol:          symbol,
			Name:            name,
			Balance:         tokenBalance,
			Decimals:        decimals,
		}
	}

	return balance, nil
}

// TransferTRX sends TRX from this wallet to another address
// amount is in TRX (not SUN)
func (w *Wallet) TransferTRX(ctx context.Context, toAddress string, amount decimal.Decimal) (string, error) {
	// Convert TRX to SUN
	sunAmount := amount.Mul(decimal.NewFromInt(1_000_000)).BigInt().Int64()

	// Create transfer transaction
	tx, err := w.grpcClient.Transfer(w.address, toAddress, sunAmount)
	if err != nil {
		return "", fmt.Errorf("failed to create transfer transaction: %w", err)
	}

	// Sign transaction
	signedTx, err := w.signTransaction(tx)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Broadcast transaction
	result, err := w.grpcClient.Broadcast(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	if !result.Result {
		return "", fmt.Errorf("broadcast failed: %s", string(result.Message))
	}

	// Get transaction ID
	txID := fmt.Sprintf("%x", signedTx.GetTxid())

	return txID, nil
}

// TransferTRC20 sends TRC20 tokens from this wallet to another address
// amount should be in the token's base unit (e.g., for USDT with 6 decimals, 1 USDT = 1,000,000)
func (w *Wallet) TransferTRC20(ctx context.Context, contractAddress string, toAddress string, amount decimal.Decimal) (string, error) {
	// Get token decimals
	decimals, err := w.grpcClient.TRC20GetDecimals(contractAddress)
	if err != nil {
		// Default to 6 decimals (USDT standard)
		decimals = 6
	}

	// Convert amount to base unit
	baseAmount := amount.Shift(int32(decimals)).BigInt()

	// Create TRC20 transfer transaction
	tx, err := w.grpcClient.TRC20Send(w.address, toAddress, contractAddress, baseAmount, 50_000_000) // 50 TRX fee limit
	if err != nil {
		return "", fmt.Errorf("failed to create TRC20 transfer transaction: %w", err)
	}

	// Sign transaction
	signedTx, err := w.signTransaction(tx)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Broadcast transaction
	result, err := w.grpcClient.Broadcast(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	if !result.Result {
		return "", fmt.Errorf("broadcast failed: %s", string(result.Message))
	}

	// Get transaction ID
	txID := fmt.Sprintf("%x", signedTx.GetTxid())

	return txID, nil
}

// signTransaction signs a transaction with the wallet's private key
func (w *Wallet) signTransaction(tx *core.Transaction) (*core.Transaction, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	// Get raw data hash
	rawData, err := tx.GetRawData().XXX_Marshal(nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw data: %w", err)
	}

	hash := sha256.Sum256(rawData)

	// Sign with private key
	privateKeyBytes := common.FromHex(w.privateKey)
	signature, err := keystore.SignHash(hash[:], privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Add signature to transaction
	tx.Signature = append(tx.Signature, signature)

	return tx, nil
}

// ValidateAddress checks if a TRON address is valid
func (w *Wallet) ValidateAddress(addr string) (bool, error) {
	result, err := w.grpcClient.ValidateAddress(addr)
	if err != nil {
		return false, fmt.Errorf("failed to validate address: %w", err)
	}

	return result.Result, nil
}

// GetAccountResource retrieves account resource information (bandwidth, energy)
func (w *Wallet) GetAccountResource(ctx context.Context) (int64, int64, error) {
	resource, err := w.grpcClient.GetAccountResource(w.address)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get account resource: %w", err)
	}

	bandwidth := resource.GetFreeNetLimit()
	energy := resource.GetEnergyLimit()

	return bandwidth, energy, nil
}

// WaitForTransaction waits for a transaction to be confirmed
func (w *Wallet) WaitForTransaction(ctx context.Context, txID string, minConfirmations int64, timeout time.Duration) error {
	if minConfirmations <= 0 {
		// TRON standard: 19 confirmations for solid block
		minConfirmations = 19
	}

	deadline := time.Now().Add(timeout)
	retryDelay := 3 * time.Second // TRON block time is ~3 seconds

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get transaction info
		txInfo, err := w.grpcClient.GetTransactionInfoByID(txID)
		if err != nil {
			// Transaction might not be confirmed yet
			time.Sleep(retryDelay)
			continue
		}

		// Check if transaction is confirmed
		if txInfo != nil && txInfo.BlockNumber > 0 {
			// Get current block number
			currentBlock, err := w.grpcClient.GetNowBlock()
			if err == nil && currentBlock != nil && currentBlock.BlockHeader != nil {
				confirmations := currentBlock.BlockHeader.RawData.Number - txInfo.BlockNumber
				if confirmations >= minConfirmations {
					// Check if transaction was successful
					if txInfo.Result != core.TransactionInfo_SUCCESS {
						return fmt.Errorf("transaction failed with result: %s", txInfo.Result.String())
					}
					return nil
				}
			}
		}

		time.Sleep(retryDelay)
	}

	return fmt.Errorf("transaction not confirmed within timeout")
}

// Close closes the gRPC connection
func (w *Wallet) Close() error {
	if w.grpcClient != nil {
		return w.grpcClient.Stop()
	}
	return nil
}

// GetGrpcClient returns the underlying gRPC client for advanced operations
func (w *Wallet) GetGrpcClient() *client.GrpcClient {
	return w.grpcClient
}

// IsMainnet returns true if wallet is configured for mainnet
func (w *Wallet) IsMainnet() bool {
	return w.isMainnet
}
