package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	pbtimestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/require"
)

type fakeIdentity struct {
	mspID string
	attrs map[string]string
}

func (f *fakeIdentity) GetID() (string, error)    { return f.attrs["eventchain.accountID"], nil }
func (f *fakeIdentity) GetMSPID() (string, error) { return f.mspID, nil }
func (f *fakeIdentity) GetAttributeValue(name string) (string, bool, error) {
	value, found := f.attrs[name]
	return value, found, nil
}
func (f *fakeIdentity) AssertAttributeValue(name, value string) error {
	actual, found := f.attrs[name]
	if !found || actual != value {
		return fmt.Errorf("attribute mismatch")
	}
	return nil
}
func (f *fakeIdentity) GetX509Certificate() (*x509.Certificate, error) { return nil, nil }

var _ cid.ClientIdentity = (*fakeIdentity)(nil)

type financeHarness struct {
	t        *testing.T
	contract *FinanceContract
	stub     *shimtest.MockStub
	ctx      *contractapi.TransactionContext
	now      time.Time
}

func newHarness(t *testing.T) *financeHarness {
	stub := shimtest.NewMockStub("finance", nil)
	ctx := new(contractapi.TransactionContext)
	ctx.SetStub(stub)
	return &financeHarness{t: t, contract: new(FinanceContract), stub: stub, ctx: ctx, now: time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)}
}

func identity(mspID, accountID, role string) *fakeIdentity {
	return &fakeIdentity{mspID: mspID, attrs: map[string]string{"eventchain.accountID": accountID, "eventchain.role": role}}
}

func (h *financeHarness) begin(txID string, who *fakeIdentity, at time.Time) {
	h.stub.MockTransactionStart(txID)
	h.stub.TxTimestamp = &pbtimestamp.Timestamp{Seconds: at.Unix(), Nanos: int32(at.Nanosecond())}
	h.ctx.SetClientIdentity(who)
	h.stub.TransientMap = nil
}

func (h *financeHarness) end() { h.stub.MockTransactionEnd(h.stub.TxID) }

func (h *financeHarness) tx(txID string, who *fakeIdentity, at time.Time, run func()) {
	h.begin(txID, who, at)
	defer h.end()
	run()
}

func TestAmountParsingAndPayoutMath(t *testing.T) {
	amount, err := parseAmount("123.456789")
	require.NoError(t, err)
	require.Equal(t, int64(123456789), amount)
	_, err = parseAmount("1.0000001")
	require.Error(t, err)
	_, err = parseAmount("1e3")
	require.Error(t, err)
	payout, err := proportionalAmount(198*amountScale, 100*amountScale, 100*amountScale)
	require.NoError(t, err)
	require.Equal(t, int64(198*amountScale), payout)
}

func TestBonusPariMutuelPendingClaimLifecycle(t *testing.T) {
	h := newHarness(t)
	admin := identity("PlatformMSP", "admin01", "admin")
	operator := identity("PlatformMSP", "operator01", "operator")
	verifier := identity("PlatformMSP", "verifier01", "verifier")
	organizer := identity("OrganizerMSP", "organizer01", "organizer")
	student1 := identity("StudentMSP", "student01", "student")
	student2 := identity("StudentMSP", "student02", "student")

	h.tx("init", admin, h.now, func() {
		config, err := h.contract.InitializeV2(h.ctx)
		require.NoError(t, err)
		require.Equal(t, int64(7), config.ClaimPendingDays)
	})
	h.tx("category", admin, h.now.Add(time.Second), func() {
		_, err := h.contract.CreateCategory(h.ctx, "basketball", "篮球")
		require.NoError(t, err)
	})
	credits := []struct {
		account string
		amount  string
		ref     string
	}{{"organizer01", "1000", "credit-org"}, {"student01", "1000", "credit-s1"}, {"student02", "1000", "credit-s2"}}
	for index, item := range credits {
		h.tx(fmt.Sprintf("credit-%d", index), admin, h.now.Add(time.Duration(index+2)*time.Second), func() {
			_, err := h.contract.CreditDemoA(h.ctx, item.account, item.amount, "seed", item.ref)
			require.NoError(t, err)
		})
	}
	for index, item := range []struct{ account, ref string }{{"student01", "bonus-s1"}, {"student02", "bonus-s2"}} {
		h.tx(fmt.Sprintf("bonus-%d", index), admin, h.now.Add(time.Duration(index+6)*time.Second), func() {
			_, err := h.contract.CreditDemoBonus(h.ctx, item.account, "basketball", "500", item.ref)
			require.NoError(t, err)
		})
	}

	closeAt := h.now.Add(time.Hour)
	outcomes, _ := json.Marshal([]Outcome{{ID: "home", Label: "主队"}, {ID: "away", Label: "客队"}, {ID: "draw", Label: "平局"}})
	h.tx("create-market", organizer, h.now.Add(10*time.Second), func() {
		market, err := h.contract.CreateMarket(h.ctx, "market01", "event01", "basketball", string(outcomes), timestamp(closeAt), "BONUS", "1000")
		require.NoError(t, err)
		require.Equal(t, "DRAFT", market.Status)
		require.Equal(t, int64(200*amountScale), market.ListingBond)
	})
	h.tx("open-market", organizer, h.now.Add(11*time.Second), func() {
		_, err := h.contract.OpenMarket(h.ctx, "market01")
		require.NoError(t, err)
	})

	place := func(txID string, who *fakeIdentity, outcome, ref string) {
		h.tx(txID, who, h.now.Add(20*time.Second), func() {
			order, _ := json.Marshal(PlacePositionRequest{MarketID: "market01", OutcomeID: outcome, Amount: "50", RefID: ref})
			h.stub.TransientMap = map[string][]byte{"order": order}
			receipt, err := h.contract.PlacePosition(h.ctx)
			require.NoError(t, err)
			require.Equal(t, int64(50*amountScale), receipt.Amount)
		})
	}
	place("position-s1", student1, "home", "order-s1")
	place("position-s2", student2, "away", "order-s2")

	h.tx("lock", operator, closeAt.Add(time.Second), func() {
		_, err := h.contract.LockMarket(h.ctx, "market01")
		require.NoError(t, err)
	})
	evidence := fmt.Sprintf("%064x", 42)
	h.tx("propose", organizer, closeAt.Add(2*time.Second), func() {
		_, err := h.contract.ProposeResult(h.ctx, "market01", "home", evidence)
		require.NoError(t, err)
	})
	h.tx("confirm", verifier, closeAt.Add(3*time.Second), func() {
		market, err := h.contract.ConfirmResult(h.ctx, "market01")
		require.NoError(t, err)
		require.Equal(t, "CHALLENGE_WINDOW", market.Status)
	})
	finalizedAt := closeAt.Add(25 * time.Hour)
	h.tx("finalize", operator, finalizedAt, func() {
		settlement, err := h.contract.FinalizeMarket(h.ctx, "market01")
		require.NoError(t, err)
		require.Equal(t, int64(1*amountScale), settlement.Fee)
		require.Equal(t, int64(99*amountScale), settlement.NetPool)
		require.Equal(t, int64(1), settlement.ExpectedClaims)
	})
	h.tx("claim", student1, finalizedAt.Add(time.Minute), func() {
		claim, err := h.contract.ClaimOne(h.ctx, "market01", "1", "student01")
		require.NoError(t, err)
		require.Equal(t, "PENDING", claim.Status)
		require.Equal(t, int64(99*amountScale), claim.Amount)
	})
	h.tx("activate-too-early", operator, finalizedAt.Add(6*24*time.Hour), func() {
		_, err := h.contract.ActivateSettlement(h.ctx, "market01", "1")
		require.Error(t, err)
	})
	maturity := finalizedAt.Add(7*24*time.Hour + time.Second)
	h.tx("activate", operator, maturity, func() {
		settlement, err := h.contract.ActivateSettlement(h.ctx, "market01", "1")
		require.NoError(t, err)
		require.True(t, settlement.FeesActivated)
	})
	h.tx("mature", student1, maturity.Add(time.Second), func() {
		claim, err := h.contract.MatureClaimOne(h.ctx, "market01", "1", "student01")
		require.NoError(t, err)
		require.Equal(t, "MATURED", claim.Status)
	})
	h.tx("close", operator, maturity.Add(2*time.Second), func() {
		settlement, err := h.contract.CloseSettlement(h.ctx, "market01", "1")
		require.NoError(t, err)
		require.Equal(t, "CLOSED", settlement.Status)
	})
	h.tx("wallet", student1, maturity.Add(3*time.Second), func() {
		wallet, err := h.contract.GetMyWallet(h.ctx)
		require.NoError(t, err)
		require.Equal(t, int64(549*amountScale), wallet.Categories["basketball"].Bonus.Available)
	})
}

func TestTwoOfThreeEmergencyCorrectionCreatesNewEpoch(t *testing.T) {
	h := newHarness(t)
	admin := identity("PlatformMSP", "admin01", "admin")
	operator := identity("PlatformMSP", "operator01", "operator")
	approver2 := identity("PlatformMSP", "approver02", "emergency")
	approver3 := identity("PlatformMSP", "approver03", "emergency")
	h.tx("init-correction", admin, h.now, func() {
		_, err := h.contract.InitializeV2(h.ctx)
		require.NoError(t, err)
	})
	for index, accountID := range []string{"admin01", "approver02", "approver03"} {
		h.tx(fmt.Sprintf("add-approver-%d", index), admin, h.now.Add(time.Duration(index+1)*time.Second), func() {
			require.NoError(t, h.contract.AddEmergencyApprover(h.ctx, accountID))
		})
	}
	market := &Market{
		DocType: "market", MarketID: "correct01", EventID: "event01", CategoryID: "basketball",
		OrganizerAccountID: "organizer01", Outcomes: []Outcome{{ID: "home", Label: "主队"}, {ID: "away", Label: "客队"}},
		StakeBucket: "BONUS", Status: "PROVISIONAL_FINALIZED", SettlementEpoch: 1,
	}
	ledger := &MarketLedger{
		DocType: "marketLedger", MarketID: "correct01", TotalPool: 200 * amountScale,
		OutcomeTotals:           map[string]int64{"home": 100 * amountScale, "away": 100 * amountScale},
		OutcomeParticipantCount: map[string]int64{"home": 1, "away": 1}, ParticipantCount: 2,
	}
	old := &Settlement{
		DocType: "settlement", MarketID: "correct01", Epoch: 1, Status: "PENDING",
		WinningOutcomeID: "home", GrossPool: 200 * amountScale, WinningStake: 100 * amountScale,
		Fee: 2 * amountScale, NetPool: 198 * amountScale, FinalizedAt: timestamp(h.now),
		CorrectionDeadline: timestamp(h.now.Add(7 * 24 * time.Hour)), AvailableAt: timestamp(h.now.Add(7 * 24 * time.Hour)), ExpectedClaims: 1,
	}
	h.tx("seed-correction-state", admin, h.now.Add(5*time.Second), func() {
		require.NoError(t, saveMarket(h.ctx, market))
		require.NoError(t, saveMarketLedger(h.ctx, ledger))
		require.NoError(t, saveSettlement(h.ctx, old))
	})
	reason := fmt.Sprintf("%064x", 99)
	payloadRaw, _ := json.Marshal(EmergencyCorrectionPayload{MarketID: "correct01", BaseEpoch: 1, NewOutcomeID: "away", ReasonHash: reason})
	var payloadHash string
	h.tx("propose-correction", admin, h.now.Add(6*time.Second), func() {
		action, err := h.contract.ProposeEmergency(h.ctx, "correction01", "CORRECT_SETTLEMENT", string(payloadRaw), "nonce01", timestamp(h.now.Add(2*time.Hour)))
		require.NoError(t, err)
		payloadHash = action.PayloadHash
		require.Len(t, action.Approvals, 1)
	})
	h.tx("approve-correction", approver2, h.now.Add(7*time.Second), func() {
		action, err := h.contract.ApproveEmergency(h.ctx, "correction01", payloadHash)
		require.NoError(t, err)
		require.Len(t, action.Approvals, 2)
	})
	h.tx("execute-correction", operator, h.now.Add(8*time.Second), func() {
		action, err := h.contract.ExecuteEmergency(h.ctx, "correction01")
		require.NoError(t, err)
		require.True(t, action.Executed)
		updated, err := loadMarket(h.ctx, "correct01")
		require.NoError(t, err)
		require.Equal(t, int64(2), updated.SettlementEpoch)
		oldEpoch, err := loadSettlement(h.ctx, "correct01", 1)
		require.NoError(t, err)
		require.Equal(t, "SUPERSEDED", oldEpoch.Status)
		newEpoch, err := loadSettlement(h.ctx, "correct01", 2)
		require.NoError(t, err)
		require.Equal(t, "away", newEpoch.WinningOutcomeID)
	})
	_ = approver3
}

func TestServiceFaultCancellationPaysOnlyPrefundedOnePointFiveX(t *testing.T) {
	h := newHarness(t)
	admin := identity("PlatformMSP", "admin01", "admin")
	organizer := identity("OrganizerMSP", "organizer01", "organizer")
	student := identity("StudentMSP", "student01", "student")
	h.tx("service-init", admin, h.now, func() {
		_, err := h.contract.InitializeV2(h.ctx)
		require.NoError(t, err)
		_, err = h.contract.CreateCategory(h.ctx, "basketball", "篮球")
		require.NoError(t, err)
	})
	h.tx("service-credit-org", admin, h.now.Add(time.Second), func() {
		_, err := h.contract.CreditDemoA(h.ctx, "organizer01", "1000", "seed", "svc-credit-org")
		require.NoError(t, err)
		_, err = h.contract.CreditDemoBonus(h.ctx, "organizer01", "basketball", "100", "svc-bonus-org")
		require.NoError(t, err)
	})
	h.tx("service-credit-user", admin, h.now.Add(2*time.Second), func() {
		_, err := h.contract.CreditDemoA(h.ctx, "student01", "1000", "seed", "svc-credit-user")
		require.NoError(t, err)
	})
	h.tx("service-convert", student, h.now.Add(3*time.Second), func() {
		_, err := h.contract.ConvertAToPaidB(h.ctx, "basketball", "100", "svc-convert")
		require.NoError(t, err)
	})
	h.tx("service-offer", organizer, h.now.Add(4*time.Second), func() {
		offer, err := h.contract.CreateServiceOffer(h.ctx, "venue01", "basketball", "VENUE", "篮球馆一小时", "100", "15000", "50", "20", "1")
		require.NoError(t, err)
		require.Equal(t, int64(50*amountScale), offer.GuaranteeAvailableA)
	})
	h.tx("service-order", student, h.now.Add(5*time.Second), func() {
		request, _ := json.Marshal(CreateServiceOrderRequest{OfferID: "venue01", ScheduledAt: timestamp(h.now.Add(24 * time.Hour)), RefID: "service-order01"})
		h.stub.TransientMap = map[string][]byte{"order": request}
		order, err := h.contract.CreateServiceOrder(h.ctx)
		require.NoError(t, err)
		require.Equal(t, int64(50*amountScale), order.GuaranteeReservedA)
	})
	h.tx("service-cancel", organizer, h.now.Add(6*time.Second), func() {
		order, err := h.contract.CancelServiceOrder(h.ctx, "service-order01", "ORGANIZER_FAULT")
		require.NoError(t, err)
		require.Equal(t, int64(150*amountScale), order.RefundedA)
	})
	h.tx("service-user-wallet", student, h.now.Add(7*time.Second), func() {
		wallet, err := h.contract.GetMyWallet(h.ctx)
		require.NoError(t, err)
		require.Equal(t, int64(1050*amountScale), wallet.AAvailable)
		require.Equal(t, int64(0), wallet.Categories["basketball"].Paid.Available)
	})
}
