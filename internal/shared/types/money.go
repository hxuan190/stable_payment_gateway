package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// Money represents a monetary value with currency
// Uses decimal.Decimal to avoid floating-point precision issues
type Money struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// NewMoney creates a new Money value
func NewMoney(amount decimal.Decimal, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

// NewMoneyFromFloat creates Money from a float64 (use with caution)
func NewMoneyFromFloat(amount float64, currency string) Money {
	return Money{
		Amount:   decimal.NewFromFloat(amount),
		Currency: currency,
	}
}

// NewMoneyFromInt creates Money from an integer
func NewMoneyFromInt(amount int64, currency string) Money {
	return Money{
		Amount:   decimal.NewFromInt(amount),
		Currency: currency,
	}
}

// NewMoneyFromString creates Money from a string
func NewMoneyFromString(amount string, currency string) (Money, error) {
	dec, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount: %w", err)
	}
	return Money{
		Amount:   dec,
		Currency: currency,
	}, nil
}

// Add adds two Money values (must have same currency)
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot add different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{
		Amount:   m.Amount.Add(other.Amount),
		Currency: m.Currency,
	}, nil
}

// Sub subtracts two Money values (must have same currency)
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot subtract different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{
		Amount:   m.Amount.Sub(other.Amount),
		Currency: m.Currency,
	}, nil
}

// Mul multiplies Money by a scalar
func (m Money) Mul(multiplier decimal.Decimal) Money {
	return Money{
		Amount:   m.Amount.Mul(multiplier),
		Currency: m.Currency,
	}
}

// Div divides Money by a scalar
func (m Money) Div(divisor decimal.Decimal) Money {
	return Money{
		Amount:   m.Amount.Div(divisor),
		Currency: m.Currency,
	}
}

// IsPositive returns true if amount is greater than zero
func (m Money) IsPositive() bool {
	return m.Amount.GreaterThan(decimal.Zero)
}

// IsZero returns true if amount is zero
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// IsNegative returns true if amount is less than zero
func (m Money) IsNegative() bool {
	return m.Amount.LessThan(decimal.Zero)
}

// GreaterThan returns true if this Money is greater than other
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.Currency != other.Currency {
		return false, fmt.Errorf("cannot compare different currencies")
	}
	return m.Amount.GreaterThan(other.Amount), nil
}

// LessThan returns true if this Money is less than other
func (m Money) LessThan(other Money) (bool, error) {
	if m.Currency != other.Currency {
		return false, fmt.Errorf("cannot compare different currencies")
	}
	return m.Amount.LessThan(other.Amount), nil
}

// Equal returns true if amounts and currencies are equal
func (m Money) Equal(other Money) bool {
	return m.Currency == other.Currency && m.Amount.Equal(other.Amount)
}

// String returns a string representation
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.Amount.StringFixed(2), m.Currency)
}

// MarshalJSON implements json.Marshaler
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}{
		Amount:   m.Amount.String(),
		Currency: m.Currency,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (m *Money) UnmarshalJSON(data []byte) error {
	var v struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	amount, err := decimal.NewFromString(v.Amount)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	m.Amount = amount
	m.Currency = v.Currency
	return nil
}

// Value implements driver.Valuer for database storage
func (m Money) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner for database retrieval
func (m *Money) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Money value: %v", value)
	}
	return json.Unmarshal(bytes, m)
}

// VND creates a Money value in Vietnamese Dong
func VND(amount decimal.Decimal) Money {
	return NewMoney(amount, "VND")
}

// USDT creates a Money value in USDT
func USDT(amount decimal.Decimal) Money {
	return NewMoney(amount, "USDT")
}

// USDC creates a Money value in USDC
func USDC(amount decimal.Decimal) Money {
	return NewMoney(amount, "USDC")
}
