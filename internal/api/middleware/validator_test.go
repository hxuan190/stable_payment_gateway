package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitValidator(t *testing.T) {
	InitValidator()
	assert.NotNil(t, validate)

	// Test that custom validators are registered
	v := GetValidator()
	assert.NotNil(t, v)
}

func TestValidatePaymentAmount(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "valid payment amount - minimum",
			amount:  10000,
			wantErr: false,
		},
		{
			name:    "valid payment amount - middle range",
			amount:  1000000,
			wantErr: false,
		},
		{
			name:    "valid payment amount - maximum",
			amount:  500000000,
			wantErr: false,
		},
		{
			name:    "invalid payment amount - too small",
			amount:  9999,
			wantErr: true,
		},
		{
			name:    "invalid payment amount - zero",
			amount:  0,
			wantErr: true,
		},
		{
			name:    "invalid payment amount - negative",
			amount:  -1000,
			wantErr: true,
		},
		{
			name:    "invalid payment amount - too large",
			amount:  500000001,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Amount float64 `validate:"vnd_payment_amount"`
			}

			s := TestStruct{Amount: tt.amount}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePayoutAmount(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "valid payout amount - minimum",
			amount:  1000000,
			wantErr: false,
		},
		{
			name:    "valid payout amount - middle range",
			amount:  50000000,
			wantErr: false,
		},
		{
			name:    "valid payout amount - maximum",
			amount:  1000000000,
			wantErr: false,
		},
		{
			name:    "invalid payout amount - too small",
			amount:  999999,
			wantErr: true,
		},
		{
			name:    "invalid payout amount - zero",
			amount:  0,
			wantErr: true,
		},
		{
			name:    "invalid payout amount - too large",
			amount:  1000000001,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Amount float64 `validate:"vnd_payout_amount"`
			}

			s := TestStruct{Amount: tt.amount}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUIDv4(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		uuid    string
		wantErr bool
	}{
		{
			name:    "valid UUID v4",
			uuid:    "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "valid UUID v4 uppercase",
			uuid:    "550E8400-E29B-41D4-A716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID - wrong format",
			uuid:    "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "invalid UUID - missing dashes",
			uuid:    "550e8400e29b41d4a716446655440000",
			wantErr: true,
		},
		{
			name:    "invalid UUID - empty",
			uuid:    "",
			wantErr: true,
		},
		{
			name:    "invalid UUID - UUID v1",
			uuid:    "550e8400-e29b-11d4-a716-446655440000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				ID string `validate:"uuid_v4"`
			}

			s := TestStruct{ID: tt.uuid}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVietnamesePhone(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{
			name:    "valid phone - starts with 0",
			phone:   "0901234567",
			wantErr: false,
		},
		{
			name:    "valid phone - starts with +84",
			phone:   "+84901234567",
			wantErr: false,
		},
		{
			name:    "valid phone - 10 digits",
			phone:   "0123456789",
			wantErr: false,
		},
		{
			name:    "invalid phone - too short",
			phone:   "090123456",
			wantErr: true,
		},
		{
			name:    "invalid phone - too long",
			phone:   "090123456789",
			wantErr: true,
		},
		{
			name:    "invalid phone - contains letters",
			phone:   "090123456a",
			wantErr: true,
		},
		{
			name:    "invalid phone - wrong prefix",
			phone:   "190123456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Phone string `validate:"vn_phone"`
			}

			s := TestStruct{Phone: tt.phone}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBankAccount(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		account string
		wantErr bool
	}{
		{
			name:    "valid account - 6 digits",
			account: "123456",
			wantErr: false,
		},
		{
			name:    "valid account - 10 digits",
			account: "1234567890",
			wantErr: false,
		},
		{
			name:    "valid account - 20 digits",
			account: "12345678901234567890",
			wantErr: false,
		},
		{
			name:    "invalid account - too short",
			account: "12345",
			wantErr: true,
		},
		{
			name:    "invalid account - too long",
			account: "123456789012345678901",
			wantErr: true,
		},
		{
			name:    "invalid account - contains letters",
			account: "12345a",
			wantErr: true,
		},
		{
			name:    "invalid account - contains spaces",
			account: "123 456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Account string `validate:"vn_bank_account"`
			}

			s := TestStruct{Account: tt.account}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNoHTML(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{
			name:    "valid text - no HTML",
			text:    "This is plain text",
			wantErr: false,
		},
		{
			name:    "valid text - with special chars",
			text:    "Amount: $100.50",
			wantErr: false,
		},
		{
			name:    "invalid text - HTML tag",
			text:    "Hello <script>alert('xss')</script>",
			wantErr: true,
		},
		{
			name:    "invalid text - simple tag",
			text:    "<b>Bold text</b>",
			wantErr: true,
		},
		{
			name:    "invalid text - unclosed tag",
			text:    "Text <div",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Text string `validate:"no_html"`
			}

			s := TestStruct{Text: tt.text}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDecimalPositive(t *testing.T) {
	InitValidator()

	tests := []struct {
		name    string
		amount  decimal.Decimal
		wantErr bool
	}{
		{
			name:    "valid decimal - positive",
			amount:  decimal.NewFromFloat(100.50),
			wantErr: false,
		},
		{
			name:    "valid decimal - small positive",
			amount:  decimal.NewFromFloat(0.01),
			wantErr: false,
		},
		{
			name:    "invalid decimal - zero",
			amount:  decimal.Zero,
			wantErr: true,
		},
		{
			name:    "invalid decimal - negative",
			amount:  decimal.NewFromFloat(-100),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Amount decimal.Decimal `validate:"decimal_positive"`
			}

			s := TestStruct{Amount: tt.amount}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type TestRequest struct {
		Email  string  `json:"email" validate:"required,email"`
		Amount float64 `json:"amount" validate:"required,vnd_payment_amount"`
		URL    string  `json:"url" validate:"omitempty,url"`
	}

	tests := []struct {
		name           string
		request        TestRequest
		wantValid      bool
		wantStatusCode int
	}{
		{
			name: "valid request",
			request: TestRequest{
				Email:  "test@example.com",
				Amount: 100000,
				URL:    "https://example.com",
			},
			wantValid:      true,
			wantStatusCode: 0,
		},
		{
			name: "invalid email",
			request: TestRequest{
				Email:  "invalid-email",
				Amount: 100000,
			},
			wantValid:      false,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalid amount - too small",
			request: TestRequest{
				Email:  "test@example.com",
				Amount: 1000,
			},
			wantValid:      false,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalid URL",
			request: TestRequest{
				Email:  "test@example.com",
				Amount: 100000,
				URL:    "not-a-url",
			},
			wantValid:      false,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			valid := ValidateStruct(c, &tt.request)

			assert.Equal(t, tt.wantValid, valid)
			if !tt.wantValid {
				assert.Equal(t, tt.wantStatusCode, w.Code)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text - no change",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "trim whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "escape HTML entities",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "escape ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "escape quotes",
			input:    `He said "Hello"`,
			expected: "He said &#34;Hello&#34;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMoneyAmount(t *testing.T) {
	tests := []struct {
		name      string
		amount    float64
		minAmount int
		maxAmount int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid amount",
			amount:    100000,
			minAmount: MinPaymentAmount,
			maxAmount: MaxPaymentAmount,
			wantErr:   false,
		},
		{
			name:      "zero amount",
			amount:    0,
			minAmount: MinPaymentAmount,
			maxAmount: MaxPaymentAmount,
			wantErr:   true,
			errMsg:    "amount must be greater than 0",
		},
		{
			name:      "negative amount",
			amount:    -1000,
			minAmount: MinPaymentAmount,
			maxAmount: MaxPaymentAmount,
			wantErr:   true,
			errMsg:    "amount must be greater than 0",
		},
		{
			name:      "below minimum",
			amount:    5000,
			minAmount: MinPaymentAmount,
			maxAmount: MaxPaymentAmount,
			wantErr:   true,
			errMsg:    "amount must be at least",
		},
		{
			name:      "above maximum",
			amount:    600000000,
			minAmount: MinPaymentAmount,
			maxAmount: MaxPaymentAmount,
			wantErr:   true,
			errMsg:    "amount must not exceed",
		},
		{
			name:      "decimal amount - not allowed for VND",
			amount:    100000.50,
			minAmount: MinPaymentAmount,
			maxAmount: MaxPaymentAmount,
			wantErr:   true,
			errMsg:    "VND amounts must be whole numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMoneyAmount(tt.amount, tt.minAmount, tt.maxAmount)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDecimalAmount(t *testing.T) {
	min := decimal.NewFromFloat(10000)
	max := decimal.NewFromFloat(500000000)

	tests := []struct {
		name    string
		amount  decimal.Decimal
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid amount",
			amount:  decimal.NewFromFloat(100000),
			wantErr: false,
		},
		{
			name:    "zero amount",
			amount:  decimal.Zero,
			wantErr: true,
			errMsg:  "amount must be greater than 0",
		},
		{
			name:    "negative amount",
			amount:  decimal.NewFromFloat(-1000),
			wantErr: true,
			errMsg:  "amount must be greater than 0",
		},
		{
			name:    "below minimum",
			amount:  decimal.NewFromFloat(5000),
			wantErr: true,
			errMsg:  "amount must be at least",
		},
		{
			name:    "above maximum",
			amount:  decimal.NewFromFloat(600000000),
			wantErr: true,
			errMsg:  "amount must not exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDecimalAmount(tt.amount, min, max)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "invalid email - no @",
			email:   "invalid.email.com",
			wantErr: true,
		},
		{
			name:    "invalid email - spaces",
			email:   "test @example.com",
			wantErr: true,
		},
		{
			name:    "invalid email - empty",
			email:   "",
			wantErr: true,
		},
		{
			name:    "invalid email - missing domain",
			email:   "test@",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with path",
			url:     "https://example.com/webhook",
			wantErr: false,
		},
		{
			name:    "valid localhost HTTP (for development)",
			url:     "http://localhost:3000/webhook",
			wantErr: false,
		},
		{
			name:    "invalid URL - HTTP not localhost",
			url:     "http://example.com",
			wantErr: true,
		},
		{
			name:    "invalid URL - malformed",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "invalid URL - empty",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		uuid    string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			uuid:    "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID - wrong format",
			uuid:    "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "invalid UUID - empty",
			uuid:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.uuid)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBankAccountNumber(t *testing.T) {
	tests := []struct {
		name    string
		account string
		wantErr bool
	}{
		{
			name:    "valid account",
			account: "1234567890",
			wantErr: false,
		},
		{
			name:    "invalid account - too short",
			account: "12345",
			wantErr: true,
		},
		{
			name:    "invalid account - contains letters",
			account: "12345a7890",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBankAccountNumber(tt.account)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{
			name:    "valid phone",
			phone:   "0901234567",
			wantErr: false,
		},
		{
			name:    "invalid phone - too short",
			phone:   "090123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneNumber(tt.phone)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	InitValidator()

	// Create a test struct to trigger validation errors
	type TestStruct struct {
		Email  string  `json:"email" validate:"required,email"`
		Amount float64 `json:"amount" validate:"min=100,max=1000"`
		Status string  `json:"status" validate:"oneof=active inactive"`
	}

	// Test required field error
	s := TestStruct{Email: "", Amount: 150, Status: "active"}
	err := validate.Struct(s)
	require.Error(t, err)

	errs, ok := err.(validator.ValidationErrors)
	require.True(t, ok)
	require.NotEmpty(t, errs)

	msg := getErrorMessage(errs[0])
	assert.Contains(t, msg, "required")

	// Test email format error
	s = TestStruct{Email: "invalid", Amount: 150, Status: "active"}
	err = validate.Struct(s)
	require.Error(t, err)

	errs = err.(validator.ValidationErrors)
	msg = getErrorMessage(errs[0])
	assert.Contains(t, msg, "valid email")
}

func TestSanitizeInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		queryParams   map[string]string
		expectedQuery string
	}{
		{
			name:          "sanitize query params",
			queryParams:   map[string]string{"name": "  John Doe  ", "email": "test@example.com"},
			expectedQuery: "email=test%40example.com&name=John+Doe",
		},
		{
			name:          "sanitize HTML in query params",
			queryParams:   map[string]string{"text": "<script>alert('xss')</script>"},
			expectedQuery: "text=%26lt%3Bscript%26gt%3Balert%28%26%2339%3Bxss%26%2339%3B%29%26lt%3B%2Fscript%26gt%3B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Build request with query params
			req, _ := http.NewRequest("GET", "http://example.com/test", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			// Apply middleware
			handler := SanitizeInput()
			handler(c)

			// Check that query params are sanitized
			assert.Contains(t, c.Request.URL.RawQuery, "=")
		})
	}
}
