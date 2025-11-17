package middleware

import (
	"fmt"
	"html"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

// ValidationError represents a single field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationErrorResponse represents the response for validation errors
type ValidationErrorResponse struct {
	Error  string            `json:"error"`
	Errors []ValidationError `json:"errors"`
}

// Custom validators
const (
	// Transaction limits (VND)
	MinPaymentAmount = 10000      // 10,000 VND (~$0.40)
	MaxPaymentAmount = 500000000  // 500M VND (~$20,000)
	MinPayoutAmount  = 1000000    // 1M VND (~$40)
	MaxPayoutAmount  = 1000000000 // 1B VND (~$40,000)
)

var (
	// Validator instance
	validate *validator.Validate

	// UUID regex pattern (v4)
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	// Vietnamese phone number regex (basic)
	phoneRegex = regexp.MustCompile(`^(\+84|0)[0-9]{9,10}$`)

	// Bank account number regex (Vietnamese banks: 6-20 digits)
	bankAccountRegex = regexp.MustCompile(`^[0-9]{6,20}$`)
)

// InitValidator initializes the validator with custom validations
func InitValidator() {
	validate = validator.New()

	// Register custom validators
	validate.RegisterValidation("vnd_payment_amount", validatePaymentAmount)
	validate.RegisterValidation("vnd_payout_amount", validatePayoutAmount)
	validate.RegisterValidation("uuid_v4", validateUUIDv4)
	validate.RegisterValidation("vn_phone", validateVietnamesePhone)
	validate.RegisterValidation("vn_bank_account", validateBankAccount)
	validate.RegisterValidation("no_html", validateNoHTML)
	validate.RegisterValidation("decimal_positive", validateDecimalPositive)

	// Register custom tag name function to use json tags as field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// GetValidator returns the validator instance
func GetValidator() *validator.Validate {
	if validate == nil {
		InitValidator()
	}
	return validate
}

// ValidateRequest is a middleware that validates request payloads
func ValidateRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Initialize validator if not already done
		if validate == nil {
			InitValidator()
		}

		c.Next()
	}
}

// ValidateStruct validates a struct and returns structured error response
func ValidateStruct(c *gin.Context, s interface{}) bool {
	if validate == nil {
		InitValidator()
	}

	err := validate.Struct(s)
	if err != nil {
		validationErrors := []ValidationError{}

		// Type assertion for ValidationErrors
		if errs, ok := err.(validator.ValidationErrors); ok {
			for _, e := range errs {
				validationErrors = append(validationErrors, ValidationError{
					Field:   e.Field(),
					Message: getErrorMessage(e),
					Tag:     e.Tag(),
					Value:   e.Value(),
				})
			}
		}

		c.JSON(http.StatusBadRequest, ValidationErrorResponse{
			Error:  "Validation failed",
			Errors: validationErrors,
		})
		return false
	}

	return true
}

// getErrorMessage returns a human-readable error message for validation errors
func getErrorMessage(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "uuid_v4":
		return fmt.Sprintf("%s must be a valid UUID v4", field)
	case "vnd_payment_amount":
		return fmt.Sprintf("%s must be between %d and %d VND", field, MinPaymentAmount, MaxPaymentAmount)
	case "vnd_payout_amount":
		return fmt.Sprintf("%s must be between %d and %d VND", field, MinPayoutAmount, MaxPayoutAmount)
	case "vn_phone":
		return fmt.Sprintf("%s must be a valid Vietnamese phone number", field)
	case "vn_bank_account":
		return fmt.Sprintf("%s must be a valid bank account number (6-20 digits)", field)
	case "no_html":
		return fmt.Sprintf("%s must not contain HTML tags", field)
	case "decimal_positive":
		return fmt.Sprintf("%s must be a positive number", field)
	default:
		return fmt.Sprintf("%s failed validation for %s", field, tag)
	}
}

// Custom validation functions

// validatePaymentAmount validates payment amount is within allowed range
func validatePaymentAmount(fl validator.FieldLevel) bool {
	value := fl.Field().Float()
	return value >= float64(MinPaymentAmount) && value <= float64(MaxPaymentAmount)
}

// validatePayoutAmount validates payout amount is within allowed range
func validatePayoutAmount(fl validator.FieldLevel) bool {
	value := fl.Field().Float()
	return value >= float64(MinPayoutAmount) && value <= float64(MaxPayoutAmount)
}

// validateUUIDv4 validates UUID v4 format
func validateUUIDv4(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return uuidRegex.MatchString(strings.ToLower(value))
}

// validateVietnamesePhone validates Vietnamese phone number format
func validateVietnamesePhone(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return phoneRegex.MatchString(value)
}

// validateBankAccount validates Vietnamese bank account number
func validateBankAccount(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return bankAccountRegex.MatchString(value)
}

// validateNoHTML validates that string doesn't contain HTML tags
func validateNoHTML(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	// Check if string contains HTML tags
	return !strings.Contains(value, "<") && !strings.Contains(value, ">")
}

// validateDecimalPositive validates that decimal is positive
func validateDecimalPositive(fl validator.FieldLevel) bool {
	switch v := fl.Field().Interface().(type) {
	case decimal.Decimal:
		return v.GreaterThan(decimal.Zero)
	case *decimal.Decimal:
		if v == nil {
			return false
		}
		return v.GreaterThan(decimal.Zero)
	default:
		return false
	}
}

// SanitizeString removes HTML tags and trims whitespace
func SanitizeString(s string) string {
	// Escape HTML entities
	s = html.EscapeString(s)
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

// SanitizeInput is a middleware that sanitizes string inputs to prevent XSS
func SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For JSON requests, sanitization happens at the handler level
		// This middleware can be extended for form data or query params

		// Sanitize query parameters
		queryParams := c.Request.URL.Query()
		for key, values := range queryParams {
			for i, value := range values {
				queryParams[key][i] = SanitizeString(value)
			}
		}
		c.Request.URL.RawQuery = queryParams.Encode()

		c.Next()
	}
}

// ValidateMoneyAmount validates money amounts with specific rules
func ValidateMoneyAmount(amount float64, minAmount, maxAmount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	if amount < float64(minAmount) {
		return fmt.Errorf("amount must be at least %d VND", minAmount)
	}

	if amount > float64(maxAmount) {
		return fmt.Errorf("amount must not exceed %d VND", maxAmount)
	}

	// Check for excessive decimal places (should be whole numbers for VND)
	if amount != float64(int64(amount)) {
		return fmt.Errorf("VND amounts must be whole numbers (no decimals)")
	}

	return nil
}

// ValidateDecimalAmount validates decimal amounts
func ValidateDecimalAmount(amount decimal.Decimal, min, max decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("amount must be greater than 0")
	}

	if amount.LessThan(min) {
		return fmt.Errorf("amount must be at least %s", min.String())
	}

	if amount.GreaterThan(max) {
		return fmt.Errorf("amount must not exceed %s", max.String())
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if validate == nil {
		InitValidator()
	}

	err := validate.Var(email, "required,email,max=255")
	if err != nil {
		return fmt.Errorf("invalid email format")
	}

	// Additional check: no spaces allowed
	if strings.Contains(email, " ") {
		return fmt.Errorf("email must not contain spaces")
	}

	return nil
}

// ValidateURL validates URL format
func ValidateURL(url string) error {
	if validate == nil {
		InitValidator()
	}

	err := validate.Var(url, "required,url,max=500")
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	// Must be HTTPS in production
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://localhost") {
		return fmt.Errorf("URL must use HTTPS")
	}

	return nil
}

// ValidateUUID validates UUID v4 format
func ValidateUUID(id string) error {
	if !uuidRegex.MatchString(strings.ToLower(id)) {
		return fmt.Errorf("invalid UUID format")
	}
	return nil
}

// ValidateBankAccountNumber validates Vietnamese bank account number
func ValidateBankAccountNumber(accountNumber string) error {
	if !bankAccountRegex.MatchString(accountNumber) {
		return fmt.Errorf("invalid bank account number (must be 6-20 digits)")
	}
	return nil
}

// ValidatePhoneNumber validates Vietnamese phone number
func ValidatePhoneNumber(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid Vietnamese phone number format")
	}
	return nil
}
