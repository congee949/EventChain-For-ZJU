package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (fc *FinanceContract) CreateMarket(ctx contractapi.TransactionContextInterface, marketID, eventID, categoryID, outcomesJSON, closeAtText, stakeBucket, marketCapText string) (*Market, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "OrganizerMSP" || identity.Role != "organizer" {
		return nil, fmt.Errorf("organizer identity is required")
	}
	if err := validateID("market ID", marketID); err != nil {
		return nil, err
	}
	if err := validateID("event ID", eventID); err != nil {
		return nil, err
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	stakeBucket = strings.ToUpper(strings.TrimSpace(stakeBucket))
	if stakeBucket == "" {
		stakeBucket = "BONUS"
	}
	if stakeBucket != "BONUS" && stakeBucket != "PAID" {
		return nil, fmt.Errorf("stake bucket must be BONUS or PAID")
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if stakeBucket == "PAID" && !config.DemoMode {
		return nil, fmt.Errorf("redeemable paid-point prediction is disabled outside demo mode")
	}
	var outcomes []Outcome
	if err := decodeJSONStrict(outcomesJSON, &outcomes); err != nil {
		return nil, err
	}
	if len(outcomes) < 2 || len(outcomes) > 8 {
		return nil, fmt.Errorf("market requires 2-8 outcomes")
	}
	seen := map[string]bool{}
	for index := range outcomes {
		if err := validateID("outcome ID", outcomes[index].ID); err != nil {
			return nil, err
		}
		outcomes[index].Label = strings.TrimSpace(outcomes[index].Label)
		if outcomes[index].Label == "" || len([]rune(outcomes[index].Label)) > 64 {
			return nil, fmt.Errorf("outcome label must contain 1-64 characters")
		}
		if seen[outcomes[index].ID] {
			return nil, fmt.Errorf("duplicate outcome ID %q", outcomes[index].ID)
		}
		seen[outcomes[index].ID] = true
	}
	closeAt, err := parseTimestamp(closeAtText)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if !closeAt.After(now.Add(time.Minute)) {
		return nil, fmt.Errorf("closeAt must be at least one minute in the future")
	}
	marketCap := config.DefaultMarketCap
	if strings.TrimSpace(marketCapText) != "" {
		marketCap, err = parseAmount(marketCapText)
		if err != nil {
			return nil, err
		}
	}
	listingBond, err := mulBPS(marketCap, 1000, true)
	if err != nil {
		return nil, err
	}
	if listingBond < config.MinimumListingBond {
		listingBond = config.MinimumListingBond
	}
	marketKey, err := publicKey(ctx, "market", marketID)
	if err != nil {
		return nil, err
	}
	var existing Market
	found, err := getPublic(ctx, marketKey, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("market already exists")
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if wallet.AAvailable < listingBond {
		return nil, fmt.Errorf("insufficient A for listing bond")
	}
	wallet.AAvailable -= listingBond
	wallet.AReserved, err = safeAdd(wallet.AReserved, listingBond)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedAt = timestamp(now)
	probability := map[string]int64{}
	for _, outcome := range outcomes {
		probability[outcome.ID] = 0
	}
	market := &Market{
		DocType: "market", MarketID: marketID, EventID: eventID, CategoryID: categoryID,
		OrganizerAccountID: identity.AccountID, Outcomes: outcomes, StakeBucket: stakeBucket,
		Status: "DRAFT", MarketCap: marketCap, OutcomeProbability: probability,
		CloseAt: timestamp(closeAt), CreatedAt: timestamp(now), ListingBond: listingBond,
		ResultProposal: &ResultProposal{}, Dispute: &Dispute{Votes: map[string]string{}},
	}
	ledger := &MarketLedger{
		DocType: "marketLedger", MarketID: marketID, OutcomeTotals: map[string]int64{},
		OutcomeParticipantCount: map[string]int64{},
	}
	for _, outcome := range outcomes {
		ledger.OutcomeTotals[outcome.ID] = 0
		ledger.OutcomeParticipantCount[outcome.ID] = 0
	}
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := putPublic(ctx, marketKey, market); err != nil {
		return nil, err
	}
	if err := saveMarketLedger(ctx, ledger); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) GetMarket(ctx contractapi.TransactionContextInterface, marketID string) (*Market, error) {
	return loadMarket(ctx, marketID)
}

func (fc *FinanceContract) ListMarkets(ctx contractapi.TransactionContextInterface) ([]*Market, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("market", []string{})
	if err != nil {
		return nil, fmt.Errorf("list markets: %w", err)
	}
	defer iterator.Close()
	markets := make([]*Market, 0)
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("iterate markets: %w", err)
		}
		var market Market
		if err := decodeJSONStrict(string(item.Value), &market); err != nil {
			return nil, fmt.Errorf("decode market: %w", err)
		}
		markets = append(markets, &market)
	}
	return markets, nil
}

func (fc *FinanceContract) GetSettlement(ctx contractapi.TransactionContextInterface, marketID, epochText string) (*Settlement, error) {
	epoch, err := strconv.ParseInt(epochText, 10, 64)
	if err != nil || epoch <= 0 {
		return nil, fmt.Errorf("invalid settlement epoch")
	}
	return loadSettlement(ctx, marketID, epoch)
}

func (fc *FinanceContract) GetClaim(ctx contractapi.TransactionContextInterface, marketID, epochText, accountID string) (*Claim, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.AccountID != accountID && !(identity.MSPID == "PlatformMSP" && (identity.Role == "operator" || identity.Role == "admin")) {
		return nil, fmt.Errorf("claim access denied")
	}
	if _, err := strconv.ParseInt(epochText, 10, 64); err != nil {
		return nil, fmt.Errorf("invalid settlement epoch")
	}
	key, err := publicKey(ctx, "claim", marketID, epochText, accountID)
	if err != nil {
		return nil, err
	}
	var claim Claim
	found, err := getPrivate(ctx, key, &claim)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("claim not found")
	}
	return &claim, nil
}

func (fc *FinanceContract) OpenMarket(ctx contractapi.TransactionContextInterface, marketID string) (*Market, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if identity.AccountID != market.OrganizerAccountID && !(identity.MSPID == "PlatformMSP" && identity.Role == "admin") {
		return nil, fmt.Errorf("market owner or platform admin is required")
	}
	if market.Status != "DRAFT" {
		return nil, fmt.Errorf("market must be DRAFT")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	closeAt, _ := parseTimestamp(market.CloseAt)
	if !closeAt.After(now) {
		return nil, fmt.Errorf("market close time has passed")
	}
	market.Status = "OPEN"
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) LockMarket(ctx contractapi.TransactionContextInterface, marketID string) (*Market, error) {
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.Status == "LOCKED" {
		return market, nil
	}
	if market.Status != "OPEN" {
		return nil, fmt.Errorf("market must be OPEN")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	closeAt, _ := parseTimestamp(market.CloseAt)
	if now.Before(closeAt) {
		identity, err := caller(ctx)
		if err != nil {
			return nil, err
		}
		if identity.MSPID != "PlatformMSP" || identity.Role != "admin" {
			return nil, fmt.Errorf("only platform admin can lock a market early")
		}
	}
	market.Status = "LOCKED"
	market.LockedAt = timestamp(now)
	ledger, err := loadMarketLedger(ctx, marketID)
	if err != nil {
		return nil, err
	}
	updateMarketSnapshot(market, ledger, now, true)
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) PlacePosition(ctx contractapi.TransactionContextInterface) (*PositionReceipt, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "StudentMSP" || identity.Role != "student" {
		return nil, fmt.Errorf("student identity is required")
	}
	transient, err := ctx.GetStub().GetTransient()
	if err != nil {
		return nil, fmt.Errorf("read transient data: %w", err)
	}
	raw, ok := transient["order"]
	if !ok || len(raw) == 0 || len(raw) > 4096 {
		return nil, fmt.Errorf("transient order is required and must not exceed 4096 bytes")
	}
	var request PlacePositionRequest
	if err := decodeJSONStrict(string(raw), &request); err != nil {
		return nil, err
	}
	if err := validateID("reference ID", request.RefID); err != nil {
		return nil, err
	}
	if receipt, found, err := loadPositionReceipt(ctx, request.RefID); err != nil {
		return nil, err
	} else if found {
		if receipt.AccountID != identity.AccountID || receipt.MarketID != request.MarketID || receipt.OutcomeID != request.OutcomeID || receipt.AmountText() != canonicalAmount(request.Amount) {
			return nil, fmt.Errorf("idempotency key was already used for a different order")
		}
		return receipt, nil
	}
	market, err := loadMarket(ctx, request.MarketID)
	if err != nil {
		return nil, err
	}
	if market.Status != "OPEN" || market.Paused {
		return nil, fmt.Errorf("market is not accepting positions")
	}
	if identity.AccountID == market.OrganizerAccountID {
		return nil, fmt.Errorf("market organizer is prohibited from participating")
	}
	if !marketHasOutcome(market, request.OutcomeID) {
		return nil, fmt.Errorf("unknown outcome")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	closeAt, _ := parseTimestamp(market.CloseAt)
	if !now.Before(closeAt) {
		return nil, fmt.Errorf("market is closed")
	}
	amount, err := parseAmount(request.Amount)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	ledger, err := loadMarketLedger(ctx, market.MarketID)
	if err != nil {
		return nil, err
	}
	if ledger.TotalPool > market.MarketCap-amount {
		return nil, fmt.Errorf("market cap exceeded")
	}
	currentExposure, err := accountMarketExposure(ctx, market, identity.AccountID)
	if err != nil {
		return nil, err
	}
	userCap, err := mulBPS(market.MarketCap, config.MaxUserExposureBPS, false)
	if err != nil {
		return nil, err
	}
	absoluteCap := 200 * amountScale
	if userCap > absoluteCap {
		userCap = absoluteCap
	}
	if currentExposure > userCap-amount {
		return nil, fmt.Errorf("per-user market exposure limit exceeded")
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	riskGroup := wallet.RiskGroupID
	if riskGroup == "" {
		riskGroup = identity.AccountID
	}
	exposure, exposureKey, err := loadGroupExposure(ctx, market.MarketID, riskGroup)
	if err != nil {
		return nil, err
	}
	groupCap, err := mulBPS(market.MarketCap, config.MaxRiskGroupExposureBPS, false)
	if err != nil {
		return nil, err
	}
	if exposure.Amount > groupCap-amount {
		return nil, fmt.Errorf("related-account exposure limit exceeded")
	}
	categoryBalance := categoryWallet(wallet, market.CategoryID)
	var balance *BucketBalance
	if market.StakeBucket == "PAID" {
		balance = &categoryBalance.Paid
	} else {
		balance = &categoryBalance.Bonus
	}
	if balance.Available < amount {
		return nil, fmt.Errorf("insufficient B_%s balance", strings.ToLower(market.StakeBucket))
	}
	if _, err := consumeLots(ctx, identity.AccountID, market.CategoryID, market.StakeBucket, amount, now, true); err != nil {
		return nil, err
	}
	balance.Available -= amount
	wallet.UpdatedAt = timestamp(now)
	position, isNew, err := loadPosition(ctx, market.MarketID, identity.AccountID, request.OutcomeID, market.StakeBucket, timestamp(now))
	if err != nil {
		return nil, err
	}
	position.Stake, err = safeAdd(position.Stake, amount)
	if err != nil {
		return nil, err
	}
	position.UpdatedAt = timestamp(now)
	ledger.TotalPool, err = safeAdd(ledger.TotalPool, amount)
	if err != nil {
		return nil, err
	}
	ledger.OutcomeTotals[request.OutcomeID], err = safeAdd(ledger.OutcomeTotals[request.OutcomeID], amount)
	if err != nil {
		return nil, err
	}
	if isNew {
		ledger.OutcomeParticipantCount[request.OutcomeID]++
	}
	participantKey, err := publicKey(ctx, "participant", market.MarketID, identity.AccountID)
	if err != nil {
		return nil, err
	}
	var participantMarker struct {
		Exists bool `json:"exists"`
	}
	participantFound, err := getPrivate(ctx, participantKey, &participantMarker)
	if err != nil {
		return nil, err
	}
	if !participantFound {
		ledger.ParticipantCount++
		participantMarker.Exists = true
		if err := putPrivate(ctx, participantKey, &participantMarker); err != nil {
			return nil, err
		}
	}
	exposure.Amount, err = safeAdd(exposure.Amount, amount)
	if err != nil {
		return nil, err
	}
	updateMarketSnapshot(market, ledger, now, false)
	receipt := &PositionReceipt{
		DocType: "positionReceipt", RefID: request.RefID, MarketID: market.MarketID,
		OutcomeID: request.OutcomeID, AccountID: identity.AccountID, Bucket: market.StakeBucket,
		Amount: amount, TxID: ctx.GetStub().GetTxID(), CreatedAt: timestamp(now),
	}
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := savePosition(ctx, position); err != nil {
		return nil, err
	}
	if err := saveMarketLedger(ctx, ledger); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, exposureKey, exposure); err != nil {
		return nil, err
	}
	if err := savePositionReceipt(ctx, receipt); err != nil {
		return nil, err
	}
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return receipt, nil
}

func (fc *FinanceContract) ProposeResult(ctx contractapi.TransactionContextInterface, marketID, outcomeID, evidenceHash string) (*Market, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "OrganizerMSP" || identity.Role != "organizer" || identity.AccountID != market.OrganizerAccountID {
		return nil, fmt.Errorf("market organizer is required")
	}
	if !marketHasOutcome(market, outcomeID) {
		return nil, fmt.Errorf("unknown outcome")
	}
	if err := validateHash("evidence hash", evidenceHash); err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if market.Status == "OPEN" {
		closeAt, _ := parseTimestamp(market.CloseAt)
		if now.Before(closeAt) {
			return nil, fmt.Errorf("result cannot be proposed before market close")
		}
		market.Status = "LOCKED"
		market.LockedAt = timestamp(now)
	}
	if market.Status != "LOCKED" {
		return nil, fmt.Errorf("market must be LOCKED")
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	ledger, err := loadMarketLedger(ctx, marketID)
	if err != nil {
		return nil, err
	}
	bond, err := mulBPS(ledger.TotalPool, 500, true)
	if err != nil {
		return nil, err
	}
	if bond < config.MinimumResultBond {
		bond = config.MinimumResultBond
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if wallet.AAvailable < bond {
		return nil, fmt.Errorf("insufficient A for result bond")
	}
	wallet.AAvailable -= bond
	wallet.AReserved, err = safeAdd(wallet.AReserved, bond)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedAt = timestamp(now)
	market.ResultProposal = &ResultProposal{
		OutcomeID: outcomeID, ProposerAccountID: identity.AccountID, EvidenceHash: evidenceHash,
		ProposedAt: timestamp(now), ProposalBond: bond,
	}
	market.Status = "RESULT_PROPOSED"
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) ConfirmResult(ctx contractapi.TransactionContextInterface, marketID string) (*Market, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.Role != "verifier" && identity.Role != "admin" {
		return nil, fmt.Errorf("independent verifier is required")
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.Status != "RESULT_PROPOSED" || market.ResultProposal == nil || market.ResultProposal.OutcomeID == "" {
		return nil, fmt.Errorf("market has no result proposal awaiting confirmation")
	}
	if identity.AccountID == market.ResultProposal.ProposerAccountID || identity.AccountID == market.OrganizerAccountID {
		return nil, fmt.Errorf("result must be confirmed by a different identity")
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	market.ResultProposal.ConfirmedBy = identity.AccountID
	market.ResultProposal.ConfirmedAt = timestamp(now)
	market.ResultProposal.ChallengeDeadline = timestamp(now.Add(time.Duration(config.ChallengeHours) * time.Hour))
	market.Status = "CHALLENGE_WINDOW"
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) ChallengeResult(ctx contractapi.TransactionContextInterface, marketID, reasonHash string) (*Market, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "StudentMSP" || identity.Role != "student" {
		return nil, fmt.Errorf("student identity is required")
	}
	if err := validateHash("challenge reason hash", reasonHash); err != nil {
		return nil, err
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.Status != "CHALLENGE_WINDOW" || market.ResultProposal == nil || market.ResultProposal.OutcomeID == "" {
		return nil, fmt.Errorf("market is not challengeable")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	deadline, _ := parseTimestamp(market.ResultProposal.ChallengeDeadline)
	if !now.Before(deadline) {
		return nil, fmt.Errorf("challenge window has closed")
	}
	bond := market.ResultProposal.ProposalBond
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if wallet.AAvailable < bond {
		return nil, fmt.Errorf("insufficient A for challenge bond")
	}
	wallet.AAvailable -= bond
	wallet.AReserved, err = safeAdd(wallet.AReserved, bond)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedAt = timestamp(now)
	market.ResultProposal.ChallengerAccountID = identity.AccountID
	market.ResultProposal.ChallengeReasonHash = reasonHash
	market.ResultProposal.ChallengeBond = bond
	market.Dispute = &Dispute{Votes: map[string]string{}}
	market.Status = "REVIEW"
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) AddArbitrator(ctx contractapi.TransactionContextInterface, accountID string) error {
	identity, err := caller(ctx)
	if err != nil {
		return err
	}
	if identity.MSPID != "PlatformMSP" || identity.Role != "admin" {
		return fmt.Errorf("platform admin is required")
	}
	if err := validateID("arbitrator account ID", accountID); err != nil {
		return err
	}
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("arbitrator", []string{})
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
	if count >= 5 {
		return fmt.Errorf("arbitrator roster already contains five accounts")
	}
	key, err := publicKey(ctx, "arbitrator", accountID)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte(`{"active":true}`))
}

func (fc *FinanceContract) VoteDispute(ctx contractapi.TransactionContextInterface, marketID, outcomeID string) (*Market, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.Role != "arbitrator" {
		return nil, fmt.Errorf("arbitrator role is required")
	}
	rosterKey, _ := publicKey(ctx, "arbitrator", identity.AccountID)
	roster, err := ctx.GetStub().GetState(rosterKey)
	if err != nil || roster == nil {
		return nil, fmt.Errorf("caller is not on the arbitrator roster")
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.Status != "REVIEW" || market.Dispute == nil || market.ResultProposal == nil || market.ResultProposal.OutcomeID == "" {
		return nil, fmt.Errorf("market is not under review")
	}
	if outcomeID != "void" && !marketHasOutcome(market, outcomeID) {
		return nil, fmt.Errorf("unknown outcome")
	}
	if identity.AccountID == market.OrganizerAccountID || identity.AccountID == market.ResultProposal.ProposerAccountID || identity.AccountID == market.ResultProposal.ChallengerAccountID {
		return nil, fmt.Errorf("conflicted identity cannot arbitrate")
	}
	if _, exists := market.Dispute.Votes[identity.AccountID]; exists {
		return nil, fmt.Errorf("arbitrator has already voted")
	}
	market.Dispute.Votes[identity.AccountID] = outcomeID
	counts := map[string]int{}
	for _, vote := range market.Dispute.Votes {
		counts[vote]++
	}
	resolved := ""
	for candidate, count := range counts {
		if count >= 3 {
			resolved = candidate
			break
		}
	}
	if resolved == "" && len(market.Dispute.Votes) == 5 {
		resolved = "void"
	}
	if resolved != "" {
		now, err := txTime(ctx)
		if err != nil {
			return nil, err
		}
		market.Dispute.ResolvedOutcomeID = resolved
		market.Dispute.ResolvedAt = timestamp(now)
		market.Status = "RESOLVED"
	}
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return market, nil
}

func (fc *FinanceContract) FinalizeMarket(ctx contractapi.TransactionContextInterface, marketID string) (*Settlement, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || (identity.Role != "operator" && identity.Role != "admin") {
		return nil, fmt.Errorf("platform operator is required")
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.Status == "PROVISIONAL_FINALIZED" {
		return loadSettlement(ctx, marketID, market.SettlementEpoch)
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	winner := ""
	if market.Status == "CHALLENGE_WINDOW" && market.ResultProposal != nil {
		deadline, _ := parseTimestamp(market.ResultProposal.ChallengeDeadline)
		if now.Before(deadline) {
			return nil, fmt.Errorf("challenge window is still open")
		}
		winner = market.ResultProposal.OutcomeID
	} else if market.Status == "RESOLVED" && market.Dispute != nil {
		winner = market.Dispute.ResolvedOutcomeID
	} else {
		return nil, fmt.Errorf("market is not ready to finalize")
	}
	ledger, err := loadMarketLedger(ctx, marketID)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	status := "PENDING"
	winningStake := int64(0)
	expectedClaims := ledger.ParticipantCount
	fee := int64(0)
	netPool := ledger.TotalPool
	if winner != "void" {
		winningStake = ledger.OutcomeTotals[winner]
		if winningStake > 0 {
			fee, err = mulBPS(ledger.TotalPool, config.MarketFeeBPS, false)
			if err != nil {
				return nil, err
			}
			netPool = ledger.TotalPool - fee
			expectedClaims = ledger.OutcomeParticipantCount[winner]
		} else {
			status = "VOID_REFUND"
		}
	} else {
		status = "VOID_REFUND"
	}
	market.SettlementEpoch++
	availableAt := now.Add(time.Duration(config.ClaimPendingDays) * 24 * time.Hour)
	settlement := &Settlement{
		DocType: "settlement", MarketID: marketID, Epoch: market.SettlementEpoch, Status: status,
		WinningOutcomeID: winner, GrossPool: ledger.TotalPool, WinningStake: winningStake,
		Fee: fee, NetPool: netPool, FinalizedAt: timestamp(now),
		CorrectionDeadline: timestamp(now.Add(time.Duration(config.CorrectionWindowDays) * 24 * time.Hour)),
		AvailableAt:        timestamp(availableAt), ExpectedClaims: expectedClaims,
	}
	if err := settleBonds(ctx, market, winner, now); err != nil {
		return nil, err
	}
	market.Status = "PROVISIONAL_FINALIZED"
	if err := saveSettlement(ctx, settlement); err != nil {
		return nil, err
	}
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return settlement, nil
}

func (fc *FinanceContract) ClaimOne(ctx contractapi.TransactionContextInterface, marketID, epochText, accountID string) (*Claim, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.AccountID != accountID && !(identity.MSPID == "PlatformMSP" && (identity.Role == "operator" || identity.Role == "admin")) {
		return nil, fmt.Errorf("claim access denied")
	}
	epoch, err := strconv.ParseInt(epochText, 10, 64)
	if err != nil || epoch <= 0 {
		return nil, fmt.Errorf("invalid settlement epoch")
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.SettlementEpoch != epoch || market.Status != "PROVISIONAL_FINALIZED" {
		return nil, fmt.Errorf("settlement epoch is not current")
	}
	claimKey, err := publicKey(ctx, "claim", marketID, epochText, accountID)
	if err != nil {
		return nil, err
	}
	var existing Claim
	found, err := getPrivate(ctx, claimKey, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return &existing, nil
	}
	settlement, err := loadSettlement(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	amount := int64(0)
	if settlement.Status == "VOID_REFUND" {
		for _, outcome := range market.Outcomes {
			position, found, err := findPosition(ctx, marketID, accountID, outcome.ID, market.StakeBucket)
			if err != nil {
				return nil, err
			}
			if found {
				amount, err = safeAdd(amount, position.Stake)
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		position, found, err := findPosition(ctx, marketID, accountID, settlement.WinningOutcomeID, market.StakeBucket)
		if err != nil {
			return nil, err
		}
		if found {
			amount, err = proportionalAmount(settlement.NetPool, position.Stake, settlement.WinningStake)
			if err != nil {
				return nil, err
			}
		}
	}
	if amount <= 0 {
		return nil, fmt.Errorf("account has no claimable position")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	shard := shardFor(accountID)
	shardState, shardKey, err := loadEscrowShard(ctx, marketID, epoch, shard)
	if err != nil {
		return nil, err
	}
	shardState.Allocated, err = safeAdd(shardState.Allocated, amount)
	if err != nil {
		return nil, err
	}
	shardState.ClaimCount++
	claim := &Claim{
		DocType: "claim", ClaimID: fmt.Sprintf("%s:%d:%s", marketID, epoch, accountID),
		MarketID: marketID, Epoch: epoch, AccountID: accountID, Bucket: market.StakeBucket,
		Amount: amount, Shard: shard, Status: "PENDING", AvailableAt: settlement.AvailableAt,
		PreparedAt: timestamp(now), TxID: ctx.GetStub().GetTxID(),
	}
	if err := putPrivate(ctx, shardKey, shardState); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, claimKey, claim); err != nil {
		return nil, err
	}
	return claim, nil
}

func (fc *FinanceContract) ActivateSettlement(ctx contractapi.TransactionContextInterface, marketID, epochText string) (*Settlement, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || (identity.Role != "operator" && identity.Role != "admin") {
		return nil, fmt.Errorf("platform operator is required")
	}
	epoch, err := strconv.ParseInt(epochText, 10, 64)
	if err != nil || epoch <= 0 {
		return nil, fmt.Errorf("invalid settlement epoch")
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.SettlementEpoch != epoch {
		return nil, fmt.Errorf("settlement epoch is not current")
	}
	settlement, err := loadSettlement(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	if settlement.FeesActivated {
		return settlement, nil
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	availableAt, _ := parseTimestamp(settlement.AvailableAt)
	if now.Before(availableAt) {
		return nil, fmt.Errorf("claims are still in the reversible pending period")
	}
	allocated, claims, err := sumEscrowShards(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	if claims != settlement.ExpectedClaims {
		return nil, fmt.Errorf("prepared claim count %d does not match expected %d", claims, settlement.ExpectedClaims)
	}
	if allocated > settlement.NetPool {
		return nil, fmt.Errorf("prepared claims exceed settlement pool")
	}
	dust := settlement.NetPool - allocated
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	organizerFee, err := mulBPS(settlement.Fee, config.OrganizerFeeShareBPS, false)
	if err != nil {
		return nil, err
	}
	riskFee, err := mulBPS(settlement.Fee, config.RiskFeeShareBPS, false)
	if err != nil {
		return nil, err
	}
	burn := settlement.Fee - organizerFee - riskFee
	riskFee, err = safeAdd(riskFee, dust)
	if err != nil {
		return nil, err
	}
	if organizerFee > 0 {
		if err := creditB(ctx, market.OrganizerAccountID, market.CategoryID, market.StakeBucket, organizerFee, "MARKET_FEE", now, config); err != nil {
			return nil, err
		}
	}
	if riskFee > 0 {
		if err := creditB(ctx, "risk_reserve", market.CategoryID, market.StakeBucket, riskFee, "MARKET_RISK_RESERVE", now, config); err != nil {
			return nil, err
		}
	}
	if burn > 0 && market.StakeBucket == "PAID" {
		vault, err := loadVault(ctx, market.CategoryID, timestamp(now))
		if err != nil {
			return nil, err
		}
		if vault.LockedA < burn {
			return nil, fmt.Errorf("backing invariant violated while burning fee")
		}
		vault.LockedA -= burn
		vault.UpdatedAt = timestamp(now)
		supply, err := loadSupply(ctx)
		if err != nil {
			return nil, err
		}
		supply.BurnedA, err = safeAdd(supply.BurnedA, burn)
		if err != nil {
			return nil, err
		}
		if err := saveVault(ctx, vault); err != nil {
			return nil, err
		}
		if err := saveSupply(ctx, supply); err != nil {
			return nil, err
		}
	}
	settlement.PreparedClaims = claims
	settlement.PreparedAmount = allocated
	settlement.FeesActivated = true
	settlement.Status = "MATURED_READY"
	if err := saveSettlement(ctx, settlement); err != nil {
		return nil, err
	}
	return settlement, nil
}

func (fc *FinanceContract) MatureClaimOne(ctx contractapi.TransactionContextInterface, marketID, epochText, accountID string) (*Claim, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.AccountID != accountID && !(identity.MSPID == "PlatformMSP" && (identity.Role == "operator" || identity.Role == "admin")) {
		return nil, fmt.Errorf("claim access denied")
	}
	epoch, err := strconv.ParseInt(epochText, 10, 64)
	if err != nil || epoch <= 0 {
		return nil, fmt.Errorf("invalid settlement epoch")
	}
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if market.SettlementEpoch != epoch {
		return nil, fmt.Errorf("settlement epoch was superseded")
	}
	settlement, err := loadSettlement(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	if !settlement.FeesActivated {
		return nil, fmt.Errorf("settlement has not been activated")
	}
	claimKey, err := publicKey(ctx, "claim", marketID, epochText, accountID)
	if err != nil {
		return nil, err
	}
	var claim Claim
	found, err := getPrivate(ctx, claimKey, &claim)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("claim not found")
	}
	if claim.Status == "MATURED" {
		return &claim, nil
	}
	if claim.Status != "PENDING" {
		return nil, fmt.Errorf("claim is not pending")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	availableAt, _ := parseTimestamp(claim.AvailableAt)
	if now.Before(availableAt) {
		return nil, fmt.Errorf("claim is still pending")
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := creditB(ctx, accountID, market.CategoryID, claim.Bucket, claim.Amount, "MARKET_CLAIM", now, config); err != nil {
		return nil, err
	}
	shard, shardKey, err := loadEscrowShard(ctx, marketID, epoch, claim.Shard)
	if err != nil {
		return nil, err
	}
	if shard.Matured > shard.Allocated-claim.Amount {
		return nil, fmt.Errorf("escrow shard invariant violated")
	}
	shard.Matured += claim.Amount
	shard.MaturedCount++
	claim.Status = "MATURED"
	claim.MaturedAt = timestamp(now)
	if err := putPrivate(ctx, shardKey, shard); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, claimKey, &claim); err != nil {
		return nil, err
	}
	return &claim, nil
}

func (fc *FinanceContract) CloseSettlement(ctx contractapi.TransactionContextInterface, marketID, epochText string) (*Settlement, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || (identity.Role != "operator" && identity.Role != "admin") {
		return nil, fmt.Errorf("platform operator is required")
	}
	epoch, err := strconv.ParseInt(epochText, 10, 64)
	if err != nil || epoch <= 0 {
		return nil, fmt.Errorf("invalid settlement epoch")
	}
	settlement, err := loadSettlement(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	if settlement.Status == "CLOSED" {
		return settlement, nil
	}
	allocated, claims, matured, maturedClaims, err := sumEscrowShardMaturity(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	if !settlement.FeesActivated || matured != allocated || maturedClaims != claims || claims != settlement.ExpectedClaims {
		return nil, fmt.Errorf("not all claims are matured")
	}
	settlement.MaturedAmount = matured
	settlement.MaturedClaims = maturedClaims
	settlement.Status = "CLOSED"
	market, err := loadMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	market.Status = "CLOSED"
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	organizer, err := loadWallet(ctx, market.OrganizerAccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if organizer.AReserved < market.ListingBond {
		return nil, fmt.Errorf("listing bond invariant violated")
	}
	organizer.AReserved -= market.ListingBond
	organizer.AAvailable, err = safeAdd(organizer.AAvailable, market.ListingBond)
	if err != nil {
		return nil, err
	}
	if err := saveWallet(ctx, organizer); err != nil {
		return nil, err
	}
	if err := saveSettlement(ctx, settlement); err != nil {
		return nil, err
	}
	if err := saveMarket(ctx, market); err != nil {
		return nil, err
	}
	return settlement, nil
}

type groupExposure struct {
	DocType     string `json:"docType"`
	MarketID    string `json:"marketId"`
	RiskGroupID string `json:"riskGroupId"`
	Amount      int64  `json:"amount"`
}

func loadMarket(ctx contractapi.TransactionContextInterface, marketID string) (*Market, error) {
	if err := validateID("market ID", marketID); err != nil {
		return nil, err
	}
	key, err := publicKey(ctx, "market", marketID)
	if err != nil {
		return nil, err
	}
	var market Market
	found, err := getPublic(ctx, key, &market)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("market not found")
	}
	return &market, nil
}

func saveMarket(ctx contractapi.TransactionContextInterface, market *Market) error {
	key, err := publicKey(ctx, "market", market.MarketID)
	if err != nil {
		return err
	}
	return putPublic(ctx, key, market)
}

func marketLedgerKey(ctx contractapi.TransactionContextInterface, marketID string) (string, error) {
	return publicKey(ctx, "marketLedger", marketID)
}

func loadMarketLedger(ctx contractapi.TransactionContextInterface, marketID string) (*MarketLedger, error) {
	key, err := marketLedgerKey(ctx, marketID)
	if err != nil {
		return nil, err
	}
	var ledger MarketLedger
	found, err := getPrivate(ctx, key, &ledger)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("market ledger not found")
	}
	return &ledger, nil
}

func saveMarketLedger(ctx contractapi.TransactionContextInterface, ledger *MarketLedger) error {
	key, err := marketLedgerKey(ctx, ledger.MarketID)
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, ledger)
}

func marketHasOutcome(market *Market, outcomeID string) bool {
	for _, outcome := range market.Outcomes {
		if outcome.ID == outcomeID {
			return true
		}
	}
	return false
}

func positionKey(ctx contractapi.TransactionContextInterface, marketID, accountID, outcomeID, bucket string) (string, error) {
	return publicKey(ctx, "position", marketID, accountID, outcomeID, bucket)
}

func findPosition(ctx contractapi.TransactionContextInterface, marketID, accountID, outcomeID, bucket string) (*Position, bool, error) {
	key, err := positionKey(ctx, marketID, accountID, outcomeID, bucket)
	if err != nil {
		return nil, false, err
	}
	var position Position
	found, err := getPrivate(ctx, key, &position)
	return &position, found, err
}

func loadPosition(ctx contractapi.TransactionContextInterface, marketID, accountID, outcomeID, bucket, now string) (*Position, bool, error) {
	position, found, err := findPosition(ctx, marketID, accountID, outcomeID, bucket)
	if err != nil {
		return nil, false, err
	}
	if !found {
		position = &Position{DocType: "position", MarketID: marketID, AccountID: accountID, OutcomeID: outcomeID, Bucket: bucket, OpenedAt: now, UpdatedAt: now}
	}
	return position, !found, nil
}

func savePosition(ctx contractapi.TransactionContextInterface, position *Position) error {
	key, err := positionKey(ctx, position.MarketID, position.AccountID, position.OutcomeID, position.Bucket)
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, position)
}

func accountMarketExposure(ctx contractapi.TransactionContextInterface, market *Market, accountID string) (int64, error) {
	total := int64(0)
	for _, outcome := range market.Outcomes {
		position, found, err := findPosition(ctx, market.MarketID, accountID, outcome.ID, market.StakeBucket)
		if err != nil {
			return 0, err
		}
		if found {
			total, err = safeAdd(total, position.Stake)
			if err != nil {
				return 0, err
			}
		}
	}
	return total, nil
}

func loadGroupExposure(ctx contractapi.TransactionContextInterface, marketID, groupID string) (*groupExposure, string, error) {
	key, err := publicKey(ctx, "groupExposure", marketID, groupID)
	if err != nil {
		return nil, "", err
	}
	exposure := &groupExposure{DocType: "groupExposure", MarketID: marketID, RiskGroupID: groupID}
	_, err = getPrivate(ctx, key, exposure)
	return exposure, key, err
}

func updateMarketSnapshot(market *Market, ledger *MarketLedger, now time.Time, force bool) {
	if !force && market.SnapshotAt != "" {
		last, err := parseTimestamp(market.SnapshotAt)
		if err == nil && now.Sub(last) < 5*time.Minute {
			return
		}
	}
	market.DisplayedPool = (ledger.TotalPool / amountScale) * amountScale
	market.OutcomeProbability = map[string]int64{}
	for _, outcome := range market.Outcomes {
		value := int64(0)
		if ledger.TotalPool > 0 {
			value = (ledger.OutcomeTotals[outcome.ID] * bpsScale) / ledger.TotalPool
			value = ((value + 25) / 50) * 50
		}
		market.OutcomeProbability[outcome.ID] = value
	}
	market.SnapshotAt = timestamp(now)
}

func positionReceiptKey(ctx contractapi.TransactionContextInterface, refID string) (string, error) {
	return publicKey(ctx, "positionReceipt", refID)
}

func loadPositionReceipt(ctx contractapi.TransactionContextInterface, refID string) (*PositionReceipt, bool, error) {
	key, err := positionReceiptKey(ctx, refID)
	if err != nil {
		return nil, false, err
	}
	var receipt PositionReceipt
	found, err := getPrivate(ctx, key, &receipt)
	return &receipt, found, err
}

func savePositionReceipt(ctx contractapi.TransactionContextInterface, receipt *PositionReceipt) error {
	key, err := positionReceiptKey(ctx, receipt.RefID)
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, receipt)
}

func canonicalAmount(value string) string {
	amount, err := parseAmount(value)
	if err != nil {
		return ""
	}
	return strconv.FormatInt(amount, 10)
}

func (receipt *PositionReceipt) AmountText() string {
	return strconv.FormatInt(receipt.Amount, 10)
}

func proportionalAmount(pool, stake, totalStake int64) (int64, error) {
	if pool < 0 || stake < 0 || totalStake <= 0 {
		return 0, fmt.Errorf("invalid payout operands")
	}
	numerator := new(big.Int).Mul(big.NewInt(pool), big.NewInt(stake))
	result := new(big.Int).Div(numerator, big.NewInt(totalStake))
	if !result.IsInt64() {
		return 0, fmt.Errorf("payout overflow")
	}
	return result.Int64(), nil
}

func settlementKey(ctx contractapi.TransactionContextInterface, marketID string, epoch int64) (string, error) {
	return publicKey(ctx, "settlement", marketID, strconv.FormatInt(epoch, 10))
}

func saveSettlement(ctx contractapi.TransactionContextInterface, settlement *Settlement) error {
	key, err := settlementKey(ctx, settlement.MarketID, settlement.Epoch)
	if err != nil {
		return err
	}
	return putPublic(ctx, key, settlement)
}

func loadSettlement(ctx contractapi.TransactionContextInterface, marketID string, epoch int64) (*Settlement, error) {
	key, err := settlementKey(ctx, marketID, epoch)
	if err != nil {
		return nil, err
	}
	var settlement Settlement
	found, err := getPublic(ctx, key, &settlement)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("settlement not found")
	}
	return &settlement, nil
}

func loadEscrowShard(ctx contractapi.TransactionContextInterface, marketID string, epoch int64, shard int) (*EscrowShard, string, error) {
	key, err := publicKey(ctx, "escrow", marketID, strconv.FormatInt(epoch, 10), fmt.Sprintf("%02x", shard))
	if err != nil {
		return nil, "", err
	}
	state := &EscrowShard{DocType: "escrowShard", MarketID: marketID, Epoch: epoch, Shard: shard}
	_, err = getPrivate(ctx, key, state)
	return state, key, err
}

func sumEscrowShards(ctx contractapi.TransactionContextInterface, marketID string, epoch int64) (int64, int64, error) {
	allocated := int64(0)
	claims := int64(0)
	for shard := 0; shard < 16; shard++ {
		state, _, err := loadEscrowShard(ctx, marketID, epoch, shard)
		if err != nil {
			return 0, 0, err
		}
		allocated, err = safeAdd(allocated, state.Allocated)
		if err != nil {
			return 0, 0, err
		}
		claims += state.ClaimCount
	}
	return allocated, claims, nil
}

func sumEscrowShardMaturity(ctx contractapi.TransactionContextInterface, marketID string, epoch int64) (int64, int64, int64, int64, error) {
	allocated, claims, err := sumEscrowShards(ctx, marketID, epoch)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	matured := int64(0)
	maturedClaims := int64(0)
	for shard := 0; shard < 16; shard++ {
		state, _, err := loadEscrowShard(ctx, marketID, epoch, shard)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		matured, err = safeAdd(matured, state.Matured)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		maturedClaims += state.MaturedCount
	}
	return allocated, claims, matured, maturedClaims, nil
}

func creditB(ctx contractapi.TransactionContextInterface, accountID, categoryID, bucket string, amount int64, source string, now time.Time, config *SystemConfig) error {
	wallet, err := loadWallet(ctx, accountID, timestamp(now))
	if err != nil {
		return err
	}
	category := categoryWallet(wallet, categoryID)
	var balance *BucketBalance
	expiryDays := config.BonusExpiryDays
	if bucket == "PAID" {
		balance = &category.Paid
		expiryDays = config.PaidExpiryDays
	} else if bucket == "BONUS" {
		balance = &category.Bonus
	} else {
		return fmt.Errorf("unknown B bucket")
	}
	balance.Available, err = safeAdd(balance.Available, amount)
	if err != nil {
		return err
	}
	wallet.UpdatedAt = timestamp(now)
	index, _, err := loadLotIndex(ctx, accountID, categoryID, bucket)
	if err != nil {
		return err
	}
	lotIDSeed := fmt.Sprintf("%s:%s:%s:%s:%s:%d:%d", ctx.GetStub().GetTxID(), accountID, categoryID, bucket, source, amount, len(index.LotIDs))
	digest := sha256.Sum256([]byte(lotIDSeed))
	lot := &AssetLot{
		DocType: "assetLot", LotID: hex.EncodeToString(digest[:12]), AccountID: accountID,
		CategoryID: categoryID, Bucket: bucket, Available: amount, IssuedAt: timestamp(now),
		ExpiresAt:      timestamp(now.Add(time.Duration(expiryDays) * 24 * time.Hour)),
		TransferableAt: timestamp(now), Source: source,
	}
	if bucket == "PAID" {
		lot.BackingVaultID = categoryID
	}
	if err := saveWallet(ctx, wallet); err != nil {
		return err
	}
	return putLot(ctx, lot)
}

func settleBonds(ctx contractapi.TransactionContextInterface, market *Market, winner string, now time.Time) error {
	proposal := market.ResultProposal
	if proposal == nil {
		return fmt.Errorf("result proposal is missing")
	}
	proposer, err := loadWallet(ctx, proposal.ProposerAccountID, timestamp(now))
	if err != nil {
		return err
	}
	if proposer.AReserved < proposal.ProposalBond {
		return fmt.Errorf("proposal bond invariant violated")
	}
	proposer.AReserved -= proposal.ProposalBond
	if proposal.ChallengerAccountID == "" {
		proposer.AAvailable += proposal.ProposalBond
		proposer.UpdatedAt = timestamp(now)
		return saveWallet(ctx, proposer)
	}
	challenger, err := loadWallet(ctx, proposal.ChallengerAccountID, timestamp(now))
	if err != nil {
		return err
	}
	if challenger.AReserved < proposal.ChallengeBond {
		return fmt.Errorf("challenge bond invariant violated")
	}
	challenger.AReserved -= proposal.ChallengeBond
	risk, err := loadWallet(ctx, "risk_reserve", timestamp(now))
	if err != nil {
		return err
	}
	proposerWon := winner == proposal.OutcomeID
	if winner == "void" {
		proposer.AAvailable += proposal.ProposalBond
		challenger.AAvailable += proposal.ChallengeBond
	} else if proposerWon {
		proposer.AAvailable += proposal.ProposalBond
		award := (proposal.ChallengeBond * 7000) / bpsScale
		proposer.AAvailable += award
		risk.AAvailable += proposal.ChallengeBond - award
	} else {
		challenger.AAvailable += proposal.ChallengeBond
		award := (proposal.ProposalBond * 7000) / bpsScale
		challenger.AAvailable += award
		risk.AAvailable += proposal.ProposalBond - award
	}
	proposer.UpdatedAt = timestamp(now)
	challenger.UpdatedAt = timestamp(now)
	risk.UpdatedAt = timestamp(now)
	if err := saveWallet(ctx, proposer); err != nil {
		return err
	}
	if err := saveWallet(ctx, challenger); err != nil {
		return err
	}
	return saveWallet(ctx, risk)
}

func (fc *FinanceContract) ListParticipants(ctx contractapi.TransactionContextInterface, marketID, startAfter string, limitText string) ([]string, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || (identity.Role != "operator" && identity.Role != "admin") {
		return nil, fmt.Errorf("platform operator is required")
	}
	limit, err := strconv.Atoi(limitText)
	if err != nil || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be 1-100")
	}
	iterator, err := ctx.GetStub().GetPrivateDataByPartialCompositeKey(financeCollection, "participant", []string{marketID})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	result := []string{}
	for iterator.HasNext() && len(result) < limit {
		item, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		_, attrs, err := ctx.GetStub().SplitCompositeKey(item.Key)
		if err != nil || len(attrs) != 2 {
			continue
		}
		accountID := attrs[1]
		if startAfter != "" && accountID <= startAfter {
			continue
		}
		result = append(result, accountID)
	}
	sort.Strings(result)
	return result, nil
}
