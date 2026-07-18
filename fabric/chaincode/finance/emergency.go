package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (fc *FinanceContract) AddEmergencyApprover(ctx contractapi.TransactionContextInterface, accountID string) error {
	identity, err := caller(ctx)
	if err != nil {
		return err
	}
	if identity.MSPID != "PlatformMSP" || identity.Role != "admin" {
		return fmt.Errorf("platform admin is required")
	}
	if err := validateID("approver account ID", accountID); err != nil {
		return err
	}
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("emergencyApprover", []string{})
	if err != nil {
		return err
	}
	defer iterator.Close()
	count := 0
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return err
		}
		_, attrs, _ := ctx.GetStub().SplitCompositeKey(item.Key)
		if len(attrs) == 1 && attrs[0] == accountID {
			return nil
		}
		count++
	}
	if count >= 3 {
		return fmt.Errorf("emergency approver roster already contains three accounts")
	}
	key, err := publicKey(ctx, "emergencyApprover", accountID)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte(`{"active":true}`))
}

func (fc *FinanceContract) ProposeEmergency(ctx contractapi.TransactionContextInterface, actionID, actionType, payload, nonce, expiresAtText string) (*EmergencyAction, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireEmergencyApprover(ctx, identity.AccountID); err != nil {
		return nil, err
	}
	if err := validateID("action ID", actionID); err != nil {
		return nil, err
	}
	if err := validateID("nonce", nonce); err != nil {
		return nil, err
	}
	if !allowedEmergencyAction(actionType) {
		return nil, fmt.Errorf("unsupported emergency action")
	}
	if len(payload) == 0 || len(payload) > 8192 {
		return nil, fmt.Errorf("payload is required and must not exceed 8192 bytes")
	}
	expiresAt, err := parseTimestamp(expiresAtText)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if !expiresAt.After(now) || expiresAt.After(now.Add(24*time.Hour)) {
		return nil, fmt.Errorf("emergency action must expire within 24 hours")
	}
	key, err := publicKey(ctx, "emergency", actionID)
	if err != nil {
		return nil, err
	}
	var existing EmergencyAction
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("emergency action already exists")
	}
	digest := emergencyDigest(actionType, payload, nonce, timestamp(expiresAt))
	action := &EmergencyAction{
		DocType: "emergencyAction", ActionID: actionID, ActionType: actionType,
		PayloadHash: digest, Payload: payload, Nonce: nonce, ExpiresAt: timestamp(expiresAt),
		Approvals: map[string]string{identity.AccountID: ctx.GetStub().GetTxID()}, CreatedAt: timestamp(now),
	}
	if err := putPublic(ctx, key, action); err != nil {
		return nil, err
	}
	return action, nil
}

func (fc *FinanceContract) ApproveEmergency(ctx contractapi.TransactionContextInterface, actionID, expectedPayloadHash string) (*EmergencyAction, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireEmergencyApprover(ctx, identity.AccountID); err != nil {
		return nil, err
	}
	action, key, err := loadEmergency(ctx, actionID)
	if err != nil {
		return nil, err
	}
	if action.Executed {
		return action, nil
	}
	if action.PayloadHash != expectedPayloadHash {
		return nil, fmt.Errorf("payload hash mismatch")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	expiresAt, _ := parseTimestamp(action.ExpiresAt)
	if !now.Before(expiresAt) {
		return nil, fmt.Errorf("emergency action expired")
	}
	if _, exists := action.Approvals[identity.AccountID]; !exists {
		action.Approvals[identity.AccountID] = ctx.GetStub().GetTxID()
	}
	if err := putPublic(ctx, key, action); err != nil {
		return nil, err
	}
	return action, nil
}

func (fc *FinanceContract) ExecuteEmergency(ctx contractapi.TransactionContextInterface, actionID string) (*EmergencyAction, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || (identity.Role != "admin" && identity.Role != "operator") {
		return nil, fmt.Errorf("platform operator is required")
	}
	action, key, err := loadEmergency(ctx, actionID)
	if err != nil {
		return nil, err
	}
	if action.Executed {
		return action, nil
	}
	if len(action.Approvals) < 2 {
		return nil, fmt.Errorf("two distinct emergency approvals are required")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	expiresAt, _ := parseTimestamp(action.ExpiresAt)
	if !now.Before(expiresAt) {
		return nil, fmt.Errorf("emergency action expired")
	}
	if action.PayloadHash != emergencyDigest(action.ActionType, action.Payload, action.Nonce, action.ExpiresAt) {
		return nil, fmt.Errorf("emergency payload integrity check failed")
	}
	switch action.ActionType {
	case "CORRECT_SETTLEMENT":
		var payload EmergencyCorrectionPayload
		if err := decodeJSONStrict(action.Payload, &payload); err != nil {
			return nil, err
		}
		if err := fc.correctSettlement(ctx, &payload, now); err != nil {
			return nil, err
		}
	case "PAUSE_MARKET", "UNPAUSE_MARKET":
		var payload EmergencyMarketPayload
		if err := decodeJSONStrict(action.Payload, &payload); err != nil {
			return nil, err
		}
		market, err := loadMarket(ctx, payload.MarketID)
		if err != nil {
			return nil, err
		}
		if action.ActionType == "PAUSE_MARKET" {
			payload.Reason = strings.TrimSpace(payload.Reason)
			if payload.Reason == "" || len(payload.Reason) > 256 {
				return nil, fmt.Errorf("pause reason is required")
			}
			market.Paused = true
			market.PauseReason = payload.Reason
		} else {
			market.Paused = false
			market.PauseReason = ""
		}
		if err := saveMarket(ctx, market); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported emergency action")
	}
	action.Executed = true
	action.ExecutedAt = timestamp(now)
	if err := putPublic(ctx, key, action); err != nil {
		return nil, err
	}
	return action, nil
}

func (fc *FinanceContract) correctSettlement(ctx contractapi.TransactionContextInterface, payload *EmergencyCorrectionPayload, now time.Time) error {
	if err := validateID("market ID", payload.MarketID); err != nil {
		return err
	}
	if err := validateHash("correction reason hash", payload.ReasonHash); err != nil {
		return err
	}
	market, err := loadMarket(ctx, payload.MarketID)
	if err != nil {
		return err
	}
	if market.Status != "PROVISIONAL_FINALIZED" || market.SettlementEpoch != payload.BaseEpoch {
		return fmt.Errorf("base settlement is not the current provisional epoch")
	}
	if payload.NewOutcomeID != "void" && !marketHasOutcome(market, payload.NewOutcomeID) {
		return fmt.Errorf("unknown corrected outcome")
	}
	oldSettlement, err := loadSettlement(ctx, market.MarketID, payload.BaseEpoch)
	if err != nil {
		return err
	}
	if oldSettlement.FeesActivated {
		return fmt.Errorf("matured settlement cannot be financially corrected")
	}
	deadline, _ := parseTimestamp(oldSettlement.CorrectionDeadline)
	if !now.Before(deadline) {
		return fmt.Errorf("correction window has closed")
	}
	ledger, err := loadMarketLedger(ctx, market.MarketID)
	if err != nil {
		return err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return err
	}
	status := "PENDING"
	winningStake := int64(0)
	expectedClaims := ledger.ParticipantCount
	fee := int64(0)
	netPool := ledger.TotalPool
	if payload.NewOutcomeID != "void" {
		winningStake = ledger.OutcomeTotals[payload.NewOutcomeID]
		if winningStake > 0 {
			fee, err = mulBPS(ledger.TotalPool, config.MarketFeeBPS, false)
			if err != nil {
				return err
			}
			netPool = ledger.TotalPool - fee
			expectedClaims = ledger.OutcomeParticipantCount[payload.NewOutcomeID]
		} else {
			status = "VOID_REFUND"
		}
	} else {
		status = "VOID_REFUND"
	}
	oldSettlement.Status = "SUPERSEDED"
	if err := saveSettlement(ctx, oldSettlement); err != nil {
		return err
	}
	market.SettlementEpoch++
	newSettlement := &Settlement{
		DocType: "settlement", MarketID: market.MarketID, Epoch: market.SettlementEpoch,
		Status: status, WinningOutcomeID: payload.NewOutcomeID, GrossPool: ledger.TotalPool,
		WinningStake: winningStake, Fee: fee, NetPool: netPool, FinalizedAt: timestamp(now),
		CorrectionDeadline: timestamp(now.Add(time.Duration(config.CorrectionWindowDays) * 24 * time.Hour)),
		AvailableAt:        timestamp(now.Add(time.Duration(config.ClaimPendingDays) * 24 * time.Hour)),
		ExpectedClaims:     expectedClaims, Reason: payload.ReasonHash,
	}
	if err := saveSettlement(ctx, newSettlement); err != nil {
		return err
	}
	return saveMarket(ctx, market)
}

func requireEmergencyApprover(ctx contractapi.TransactionContextInterface, accountID string) error {
	key, err := publicKey(ctx, "emergencyApprover", accountID)
	if err != nil {
		return err
	}
	raw, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("caller is not an emergency approver")
	}
	return nil
}

func allowedEmergencyAction(actionType string) bool {
	switch actionType {
	case "CORRECT_SETTLEMENT", "PAUSE_MARKET", "UNPAUSE_MARKET":
		return true
	default:
		return false
	}
}

func emergencyDigest(actionType, payload, nonce, expiresAt string) string {
	sum := sha256.Sum256([]byte("eventchain:v2:emergency:" + actionType + ":" + payload + ":" + nonce + ":" + expiresAt))
	return hex.EncodeToString(sum[:])
}

func loadEmergency(ctx contractapi.TransactionContextInterface, actionID string) (*EmergencyAction, string, error) {
	if err := validateID("action ID", actionID); err != nil {
		return nil, "", err
	}
	key, err := publicKey(ctx, "emergency", actionID)
	if err != nil {
		return nil, "", err
	}
	var action EmergencyAction
	found, err := getPublic(ctx, key, &action)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", fmt.Errorf("emergency action not found")
	}
	return &action, key, nil
}
