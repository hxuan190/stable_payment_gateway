package bsc

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

// Wallet represents a BSC wallet for managing transactions
type Wallet struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	address    common.Address
	client     *Client
}

// WalletConfig holds configuration for wallet creation
type WalletConfig struct {
	PrivateKey string
	Client     *Client
}

// LoadWallet loads a wallet from a private key hex string
func LoadWallet(privateKeyHex string, rpcURL string) (*Wallet, error) {
	// Create client
	client, err := NewClientWithURL(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Remove 0x prefix if present
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	// Load private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Derive public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	// Derive address
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	wallet := &Wallet{
		privateKey: privateKey,
		publicKey:  publicKeyECDSA,
		address:    address,
		client:     client,
	}

	return wallet, nil
}

// LoadWalletWithClient loads a wallet from a private key with an existing client
func LoadWalletWithClient(privateKeyHex string, client *Client) (*Wallet, error) {
	// Remove 0x prefix if present
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	// Load private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Derive public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	// Derive address
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	wallet := &Wallet{
		privateKey: privateKey,
		publicKey:  publicKeyECDSA,
		address:    address,
		client:     client,
	}

	return wallet, nil
}

// GetAddress returns the wallet address as a hex string
func (w *Wallet) GetAddress() string {
	return w.address.Hex()
}

// GetCommonAddress returns the wallet address as common.Address
func (w *Wallet) GetCommonAddress() common.Address {
	return w.address
}

// GetPrivateKey returns the private key (use with caution!)
func (w *Wallet) GetPrivateKey() *ecdsa.PrivateKey {
	return w.privateKey
}

// GetPublicKey returns the public key
func (w *Wallet) GetPublicKey() *ecdsa.PublicKey {
	return w.publicKey
}

// GetClient returns the RPC client
func (w *Wallet) GetClient() *Client {
	return w.client
}

// GetBalance retrieves the BNB balance of the wallet
func (w *Wallet) GetBalance(ctx context.Context) (decimal.Decimal, error) {
	return w.client.GetBalance(ctx, w.address)
}

// GetBEP20Balance retrieves the BEP20 token balance of the wallet
// This requires calling the balanceOf function on the token contract
func (w *Wallet) GetBEP20Balance(ctx context.Context, tokenContract common.Address) (decimal.Decimal, uint8, error) {
	// Call balanceOf(address) function
	// Method ID: 0x70a08231
	data := []byte{0x70, 0xa0, 0x82, 0x31}
	// Append address (padded to 32 bytes)
	addressBytes := w.address.Bytes()
	paddedAddress := common.LeftPadBytes(addressBytes, 32)
	data = append(data, paddedAddress...)

	// Create call message
	msg := common.Address{}
	result, err := w.client.GetEthClient().CallContract(ctx, callMsg{
		To:   &tokenContract,
		Data: data,
	}, nil)

	if err != nil {
		return decimal.Zero, 0, fmt.Errorf("failed to call balanceOf: %w", err)
	}

	if len(result) != 32 {
		return decimal.Zero, 0, fmt.Errorf("invalid balanceOf result length: %d", len(result))
	}

	balance := new(big.Int).SetBytes(result)

	// Get token decimals
	decimals, err := w.GetBEP20Decimals(ctx, tokenContract)
	if err != nil {
		return decimal.Zero, 0, fmt.Errorf("failed to get token decimals: %w", err)
	}

	// Convert to decimal
	divisor := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(decimals)))
	balanceDecimal := decimal.NewFromBigInt(balance, 0).Div(divisor)

	return balanceDecimal, decimals, nil
}

// GetBEP20Decimals retrieves the decimals of a BEP20 token
func (w *Wallet) GetBEP20Decimals(ctx context.Context, tokenContract common.Address) (uint8, error) {
	// Call decimals() function
	// Method ID: 0x313ce567
	data := []byte{0x31, 0x3c, 0xe5, 0x67}

	result, err := w.client.GetEthClient().CallContract(ctx, callMsg{
		To:   &tokenContract,
		Data: data,
	}, nil)

	if err != nil {
		return 0, fmt.Errorf("failed to call decimals: %w", err)
	}

	if len(result) == 0 {
		return 0, fmt.Errorf("empty decimals result")
	}

	// Decimals is usually returned as uint8 (1 byte) but padded to 32 bytes
	if len(result) == 32 {
		return uint8(result[31]), nil
	}

	return uint8(result[0]), nil
}

// HealthCheck performs a basic health check on the wallet
// Verifies connection and that the wallet address is valid
func (w *Wallet) HealthCheck(ctx context.Context) error {
	// Check client health
	if err := w.client.HealthCheck(ctx); err != nil {
		return fmt.Errorf("client health check failed: %w", err)
	}

	// Try to get balance to verify wallet is accessible
	_, err := w.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %w", err)
	}

	return nil
}

// ValidateAddress checks if an address is valid
func ValidateAddress(address string) bool {
	return common.IsHexAddress(address)
}

// ParseAddress parses a hex string to common.Address
func ParseAddress(address string) (common.Address, error) {
	if !common.IsHexAddress(address) {
		return common.Address{}, fmt.Errorf("invalid address: %s", address)
	}
	return common.HexToAddress(address), nil
}

// callMsg is a helper struct for calling contracts
type callMsg struct {
	To   *common.Address
	Data []byte
}

// Implement the ethereum.CallMsg interface
func (m callMsg) From() common.Address         { return common.Address{} }
func (m callMsg) To() *common.Address          { return m.To }
func (m callMsg) Gas() uint64                  { return 0 }
func (m callMsg) GasPrice() *big.Int           { return nil }
func (m callMsg) GasFeeCap() *big.Int          { return nil }
func (m callMsg) GasTipCap() *big.Int          { return nil }
func (m callMsg) Value() *big.Int              { return big.NewInt(0) }
func (m callMsg) Data() []byte                 { return m.Data }
func (m callMsg) AccessList() types.AccessList { return nil }

// Note: For sending transactions (not needed for listener, but for future implementation)
// You would implement SendBEP20Transfer, SendBNB, etc. methods here
// These would use the private key to sign transactions and broadcast them
