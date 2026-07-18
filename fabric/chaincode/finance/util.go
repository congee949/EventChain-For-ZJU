package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type callerIdentity struct {
	MSPID     string
	AccountID string
	Role      string
}

func caller(ctx contractapi.TransactionContextInterface) (*callerIdentity, error) {
	clientIdentity := ctx.GetClientIdentity()
	if clientIdentity == nil {
		return nil, fmt.Errorf("client identity is missing")
	}
	mspID, err := clientIdentity.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("cannot read caller MSP: %w", err)
	}
	accountID, found, err := clientIdentity.GetAttributeValue("eventchain.accountID")
	if err != nil {
		return nil, fmt.Errorf("cannot read account identity: %w", err)
	}
	if !found || !idPattern.MatchString(accountID) {
		return nil, fmt.Errorf("certificate is missing a valid eventchain.accountID")
	}
	role, found, err := clientIdentity.GetAttributeValue("eventchain.role")
	if err != nil {
		return nil, fmt.Errorf("cannot read role: %w", err)
	}
	if !found || role == "" {
		return nil, fmt.Errorf("certificate is missing eventchain.role")
	}
	return &callerIdentity{MSPID: mspID, AccountID: accountID, Role: role}, nil
}

func requirePlatformMSP(ctx contractapi.TransactionContextInterface) error {
	clientIdentity := ctx.GetClientIdentity()
	if clientIdentity == nil {
		return fmt.Errorf("client identity is missing")
	}
	mspID, err := clientIdentity.GetMSPID()
	if err != nil {
		return fmt.Errorf("cannot read caller MSP: %w", err)
	}
	if mspID != "PlatformMSP" {
		return fmt.Errorf("PlatformMSP is required")
	}
	return nil
}

func requireRole(identity *callerIdentity, roles ...string) error {
	for _, role := range roles {
		if identity.Role == role {
			return nil
		}
	}
	return fmt.Errorf("role %q is not authorized", identity.Role)
}

func txTime(ctx contractapi.TransactionContextInterface) (time.Time, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot read transaction timestamp: %w", err)
	}
	if ts == nil {
		return time.Time{}, fmt.Errorf("transaction timestamp is missing")
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos)).UTC(), nil
}

func timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTimestamp(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q", value)
	}
	return t.UTC(), nil
}

func parseAmount(value string) (int64, error) {
	return parseDecimal(value, false)
}

func parseSignedAmount(value string) (int64, error) {
	return parseDecimal(value, true)
}

func parseDecimal(value string, signed bool) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("amount is required")
	}
	negative := strings.HasPrefix(value, "-")
	if negative {
		if !signed {
			return 0, fmt.Errorf("amount must be positive")
		}
		value = strings.TrimPrefix(value, "-")
	}
	if strings.HasPrefix(value, "+") || strings.ContainsAny(value, "eE") {
		return 0, fmt.Errorf("amount must use plain decimal notation")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid amount")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 || whole > math.MaxInt64/amountScale {
		return 0, fmt.Errorf("amount is out of range")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 6 {
			return 0, fmt.Errorf("amount supports at most 6 decimal places")
		}
	}
	for len(fraction) < 6 {
		fraction += "0"
	}
	fractionValue := int64(0)
	if fraction != "" {
		fractionValue, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount")
		}
	}
	if whole > (math.MaxInt64-fractionValue)/amountScale {
		return 0, fmt.Errorf("amount is out of range")
	}
	result := whole*amountScale + fractionValue
	if result == 0 {
		return 0, fmt.Errorf("amount must not be zero")
	}
	if negative {
		result = -result
	}
	return result, nil
}

func safeAdd(a, b int64) (int64, error) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, fmt.Errorf("integer overflow")
	}
	return a + b, nil
}

func safeSub(a, b int64) (int64, error) {
	if b == math.MinInt64 {
		return 0, fmt.Errorf("integer overflow")
	}
	return safeAdd(a, -b)
}

func mulBPS(amount, bps int64, roundUp bool) (int64, error) {
	if amount < 0 || bps < 0 || bps > bpsScale*10 {
		return 0, fmt.Errorf("invalid basis-point calculation")
	}
	if amount != 0 && bps > math.MaxInt64/amount {
		return 0, fmt.Errorf("integer overflow")
	}
	product := amount * bps
	if roundUp && product > 0 {
		return (product + bpsScale - 1) / bpsScale, nil
	}
	return product / bpsScale, nil
}

func validateID(kind, value string) error {
	if !idPattern.MatchString(value) {
		return fmt.Errorf("%s must match %s", kind, idPattern.String())
	}
	return nil
}

func validateHash(kind, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s must be a lowercase SHA-256 hex digest", kind)
	}
	_, err := hex.DecodeString(value)
	if err != nil || value != strings.ToLower(value) {
		return fmt.Errorf("%s must be a lowercase SHA-256 hex digest", kind)
	}
	return nil
}

func shardFor(accountID string) int {
	sum := sha256.Sum256([]byte(accountID))
	return int(sum[0] & 0x0f)
}

func decodeJSONStrict(raw string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("invalid JSON: trailing data")
	}
	return nil
}
