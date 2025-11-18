package types

import (
	"fmt"
	"strings"
)

// BlockchainAddress represents a blockchain wallet address
type BlockchainAddress struct {
	Address string `json:"address"`
	Chain   string `json:"chain"`
}

// NewBlockchainAddress creates a new blockchain address
func NewBlockchainAddress(address, chain string) BlockchainAddress {
	return BlockchainAddress{
		Address: strings.TrimSpace(address),
		Chain:   strings.ToUpper(chain),
	}
}

// Validate validates the blockchain address
func (a BlockchainAddress) Validate() error {
	if a.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if a.Chain == "" {
		return fmt.Errorf("chain cannot be empty")
	}

	// Basic validation based on chain
	switch a.Chain {
	case "SOLANA":
		if len(a.Address) < 32 || len(a.Address) > 44 {
			return fmt.Errorf("invalid Solana address length")
		}
	case "BSC", "ETHEREUM":
		if !strings.HasPrefix(a.Address, "0x") || len(a.Address) != 42 {
			return fmt.Errorf("invalid EVM address format")
		}
	default:
		// Other chains - basic check
		if len(a.Address) < 10 {
			return fmt.Errorf("invalid address length")
		}
	}

	return nil
}

// String returns a string representation
func (a BlockchainAddress) String() string {
	return fmt.Sprintf("%s:%s", a.Chain, a.Address)
}

// Equal checks if two addresses are equal
func (a BlockchainAddress) Equal(other BlockchainAddress) bool {
	return a.Chain == other.Chain && a.Address == other.Address
}
