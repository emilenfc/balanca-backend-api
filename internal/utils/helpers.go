package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func ParseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

func FormatCurrency(amount int64) string {
	// Convert cents to dollars/pounds/euros
	dollars := float64(amount) / 100
	return fmt.Sprintf("%.2f", dollars)
}

func ParseCurrency(amountStr string) (int64, error) {
	// Remove any currency symbols and whitespace
	amountStr = strings.TrimSpace(amountStr)
	amountStr = strings.ReplaceAll(amountStr, "$", "")
	amountStr = strings.ReplaceAll(amountStr, "€", "")
	amountStr = strings.ReplaceAll(amountStr, "£", "")
	amountStr = strings.ReplaceAll(amountStr, ",", "")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid currency format: %w", err)
	}

	// Convert to cents
	return int64(amount * 100), nil
}

func BeginningOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func EndOfMonth(t time.Time) time.Time {
	return BeginningOfMonth(t).AddDate(0, 1, -1)
}


func GenerateTransactionID() string {
	return fmt.Sprintf("TX-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
}