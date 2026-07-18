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

type activityHarness struct {
	contract *ActivityContract
	stub     *shimtest.MockStub
	ctx      *contractapi.TransactionContext
}

func newActivityHarness() *activityHarness {
	stub := shimtest.NewMockStub("activity", nil)
	ctx := new(contractapi.TransactionContext)
	ctx.SetStub(stub)
	return &activityHarness{contract: new(ActivityContract), stub: stub, ctx: ctx}
}

func activityIdentity(mspID, accountID, role string) *fakeIdentity {
	return &fakeIdentity{mspID: mspID, attrs: map[string]string{"eventchain.accountID": accountID, "eventchain.role": role}}
}

func (h *activityHarness) tx(id string, who *fakeIdentity, at time.Time, run func()) {
	h.stub.MockTransactionStart(id)
	h.stub.TxTimestamp = &pbtimestamp.Timestamp{Seconds: at.Unix(), Nanos: int32(at.Nanosecond())}
	h.stub.TransientMap = nil
	h.ctx.SetClientIdentity(who)
	defer h.stub.MockTransactionEnd(id)
	run()
}

func TestCommittedLotteryClaimAndCheckIn(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	organizer := activityIdentity("OrganizerMSP", "organizer01", "organizer")
	verifier := activityIdentity("PlatformMSP", "verifier01", "verifier")
	operator := activityIdentity("PlatformMSP", "operator01", "operator")
	student1 := activityIdentity("StudentMSP", "student01", "student")
	student2 := activityIdentity("StudentMSP", "student02", "student")
	seed := "independent-verifier-seed-0000000001"

	h.tx("create", organizer, base, func() {
		activity, err := h.contract.CreateActivity(h.ctx, "event01", "basketball", "Final", "1",
			timestamp(base.Add(time.Hour)), timestamp(base.Add(2*time.Hour)), timestamp(base.Add(4*time.Hour)))
		require.NoError(t, err)
		require.Equal(t, "DRAFT", activity.Status)
	})
	h.tx("commit", verifier, base.Add(time.Minute), func() {
		activity, err := h.contract.SetDrawCommitment(h.ctx, "event01", digest(seed))
		require.NoError(t, err)
		require.Equal(t, "APPLICATION_OPEN", activity.Status)
	})
	for index, student := range []*fakeIdentity{student1, student2} {
		h.tx(fmt.Sprintf("apply-%d", index), student, base.Add(time.Duration(index+2)*time.Minute), func() {
			_, err := h.contract.Apply(h.ctx, "event01")
			require.NoError(t, err)
		})
	}
	var winner string
	h.tx("draw", organizer, base.Add(61*time.Minute), func() {
		result, err := h.contract.RunDraw(h.ctx, "event01", seed)
		require.NoError(t, err)
		require.Len(t, result.Winners, 1)
		require.Len(t, result.Losers, 1)
		winner = result.Winners[0]
	})
	winnerIdentity := student1
	if winner == "student02" {
		winnerIdentity = student2
	}
	secret := "student-held-ticket-secret-000000001"
	var ticket *Ticket
	h.tx("claim", winnerIdentity, base.Add(70*time.Minute), func() {
		raw, _ := json.Marshal(ClaimRequest{Secret: secret, RefID: "claim-event01"})
		h.stub.TransientMap = map[string][]byte{"claim": raw}
		var err error
		ticket, err = h.contract.ClaimTicket(h.ctx, "event01")
		require.NoError(t, err)
		require.NotEqual(t, secret, ticket.ClaimHash)
	})
	h.tx("bad-checkin", operator, base.Add(2*time.Hour), func() {
		raw, _ := json.Marshal(CheckInRequest{TicketID: ticket.TicketID, Secret: "wrong-secret-wrong-secret-wrong", TimeSlice: base.Add(2*time.Hour).Unix() / 30})
		h.stub.TransientMap = map[string][]byte{"checkIn": raw}
		_, err := h.contract.CheckIn(h.ctx)
		require.ErrorContains(t, err, "invalid ticket proof")
	})
	h.tx("checkin", operator, base.Add(2*time.Hour), func() {
		raw, _ := json.Marshal(CheckInRequest{TicketID: ticket.TicketID, Secret: secret, TimeSlice: base.Add(2*time.Hour).Unix() / 30})
		h.stub.TransientMap = map[string][]byte{"checkIn": raw}
		receipt, err := h.contract.CheckIn(h.ctx)
		require.NoError(t, err)
		require.Equal(t, winner, receipt.AccountID)
		require.Equal(t, "checkin:event01:"+winner, receipt.RefID)
	})
	h.tx("replay", operator, base.Add(2*time.Hour+time.Second), func() {
		raw, _ := json.Marshal(CheckInRequest{TicketID: ticket.TicketID, Secret: secret, TimeSlice: base.Add(2*time.Hour).Unix() / 30})
		h.stub.TransientMap = map[string][]byte{"checkIn": raw}
		receipt, err := h.contract.CheckIn(h.ctx)
		require.NoError(t, err)
		require.Equal(t, "checkin:event01:"+winner, receipt.RefID)
	})
}

func TestOrganizerCannotSelfCommitDraw(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	organizer := activityIdentity("OrganizerMSP", "organizer01", "organizer")
	h.tx("create", organizer, base, func() {
		_, err := h.contract.CreateActivity(h.ctx, "event02", "football", "Cup", "10",
			timestamp(base.Add(time.Hour)), timestamp(base.Add(2*time.Hour)), timestamp(base.Add(3*time.Hour)))
		require.NoError(t, err)
	})
	h.tx("bad-commit", organizer, base.Add(time.Minute), func() {
		_, err := h.contract.SetDrawCommitment(h.ctx, "event02", digest("seed-seed-seed-seed-seed-seed-0001"))
		require.ErrorContains(t, err, "verifier")
	})
}
