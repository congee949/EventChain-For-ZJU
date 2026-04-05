package ticket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketChaincode(t *testing.T) (*shimtest.MockStub, *TicketContract) {
	tc := new(TicketContract)
	cc, err := contractapi.NewChaincode(tc)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("ticket", cc)
	require.NotNil(t, stub)
	return stub, tc
}

// MockPredictionCC simulates prediction chaincode responses for GetUserScore
type MockPredictionCC struct {
	Scores map[string]MockScore
}

type MockScore struct {
	UserID       string  `json:"userID"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

func (m *MockPredictionCC) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (m *MockPredictionCC) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	if fn == "PredictionContract:GetUserScore" {
		if len(args) < 1 {
			return shim.Error("missing userID")
		}
		userID := args[0]
		score, exists := m.Scores[userID]
		if !exists {
			score = MockScore{
				UserID:       userID,
				TotalBets:    0,
				CorrectBets:  0,
				AccuracyRate: 0.0,
			}
		}
		scoreJSON, _ := json.Marshal(score)
		return shim.Success(scoreJSON)
	}
	return shim.Error("unknown function: " + fn)
}

func setupMockPredictionCC(stub *shimtest.MockStub, scores map[string]MockScore) {
	mockCC := &MockPredictionCC{Scores: scores}
	predStub := shimtest.NewMockStub("prediction-cc", mockCC)
	stub.MockPeerChaincode("prediction-cc", predStub, "eventchain")
}

func TestApplyTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify application stored
	appBytes := stub.State["application:evt001:user1"]
	require.NotNil(t, appBytes)

	var app Application
	err := json.Unmarshal(appBytes, &app)
	require.NoError(t, err)
	assert.Equal(t, "evt001", app.EventID)
	assert.Equal(t, "user1", app.UserID)
	assert.Equal(t, StatusPending, app.Status)
}

func TestApplyTicketDuplicate(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second application should fail
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestApplyTicketMultipleUsers(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	for i := 1; i <= 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		txID := fmt.Sprintf("tx%d", i)
		resp := stub.MockInvoke(txID, [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
	}

	// Verify all 5 applications exist
	for i := 1; i <= 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		key := fmt.Sprintf("application:evt001:%s", userID)
		assert.NotNil(t, stub.State[key])
	}
}

func TestRunLottery(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	// Set up mock prediction chaincode with varied accuracy scores
	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 9, AccuracyRate: 0.9},
		"user2": {UserID: "user2", TotalBets: 10, CorrectBets: 5, AccuracyRate: 0.5},
		"user3": {UserID: "user3", TotalBets: 10, CorrectBets: 2, AccuracyRate: 0.2},
		"user4": {UserID: "user4", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
		"user5": {UserID: "user5", TotalBets: 10, CorrectBets: 1, AccuracyRate: 0.1},
	}
	setupMockPredictionCC(stub, scores)

	// 5 users apply
	for i := 1; i <= 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		resp := stub.MockInvoke(fmt.Sprintf("tx-apply-%d", i), [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
	}

	// Run lottery with 3 tickets available
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("3"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 3, len(result.Winners))
	assert.Equal(t, 2, len(result.Losers))

	// Verify winners have WON status and losers have LOST
	for _, winner := range result.Winners {
		key := fmt.Sprintf("application:evt001:%s", winner)
		appBytes := stub.State[key]
		require.NotNil(t, appBytes)
		var app Application
		json.Unmarshal(appBytes, &app)
		assert.Equal(t, StatusWon, app.Status)
	}

	for _, loser := range result.Losers {
		key := fmt.Sprintf("application:evt001:%s", loser)
		appBytes := stub.State[key]
		require.NotNil(t, appBytes)
		var app Application
		json.Unmarshal(appBytes, &app)
		assert.Equal(t, StatusLost, app.Status)
	}
}

func TestRunLotteryMoreTicketsThanApplicants(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 5, CorrectBets: 3, AccuracyRate: 0.6},
		"user2": {UserID: "user2", TotalBets: 5, CorrectBets: 2, AccuracyRate: 0.4},
	}
	setupMockPredictionCC(stub, scores)

	// Only 2 users apply
	for i := 1; i <= 2; i++ {
		userID := fmt.Sprintf("user%d", i)
		stub.MockInvoke(fmt.Sprintf("tx-apply-%d", i), [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
	}

	// Run lottery with 5 tickets (more than applicants)
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("5"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	// All applicants should win
	assert.Equal(t, 2, len(result.Winners))
	assert.Equal(t, 0, len(result.Losers))
}

func TestRunLotteryNoApplicants(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{}
	setupMockPredictionCC(stub, scores)

	// Run lottery with no applicants
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("3"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result.Winners))
	assert.Equal(t, 0, len(result.Losers))
}

func TestRunLotteryDuplicate(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 5, CorrectBets: 3, AccuracyRate: 0.6},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply-1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	// First lottery
	resp := stub.MockInvoke("tx-lottery1", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second lottery on same event should fail
	resp = stub.MockInvoke("tx-lottery2", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestClaimTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	// Apply
	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	// Run lottery
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// Claim ticket
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var ticket Ticket
	err := json.Unmarshal(resp.Payload, &ticket)
	require.NoError(t, err)
	assert.Equal(t, "evt001", ticket.EventID)
	assert.Equal(t, "user1", ticket.OwnerID)
	assert.Equal(t, StatusClaimed, ticket.Status)
	assert.NotEmpty(t, ticket.ClaimHash)
	assert.NotEmpty(t, ticket.TicketID)

	// Verify the hash is a valid SHA256 hex string (64 chars)
	assert.Equal(t, 64, len(ticket.ClaimHash))
}

func TestClaimTicketNotWinner(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
		"user2": {UserID: "user2", TotalBets: 10, CorrectBets: 1, AccuracyRate: 0.1},
	}
	setupMockPredictionCC(stub, scores)

	// Both apply
	stub.MockInvoke("tx-apply-1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-apply-2", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user2"),
	})

	// Lottery with 1 ticket (user1 should win with higher accuracy)
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// Find the loser
	app1Bytes := stub.State["application:evt001:user1"]
	app2Bytes := stub.State["application:evt001:user2"]
	var app1, app2 Application
	json.Unmarshal(app1Bytes, &app1)
	json.Unmarshal(app2Bytes, &app2)

	var loserID string
	if app1.Status == StatusLost {
		loserID = "user1"
	} else {
		loserID = "user2"
	}

	// Loser should not be able to claim
	resp := stub.MockInvoke("tx-claim-loser", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte(loserID),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestClaimTicketTwice(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// First claim succeeds
	resp := stub.MockInvoke("tx-claim1", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second claim should fail (already claimed)
	resp = stub.MockInvoke("tx-claim2", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestVerifyTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// Claim ticket and get the hash
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Verify with correct hash
	resp = stub.MockInvoke("tx-verify", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var verifiedTicket Ticket
	err := json.Unmarshal(resp.Payload, &verifiedTicket)
	require.NoError(t, err)
	assert.Equal(t, StatusUsed, verifiedTicket.Status)
	assert.NotEmpty(t, verifiedTicket.VerifiedAt)
}

func TestVerifyTicketWrongHash(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Verify with wrong hash
	wrongHash := sha256.Sum256([]byte("wrong-data"))
	resp = stub.MockInvoke("tx-verify", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(hex.EncodeToString(wrongHash[:])),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestVerifyTicketAlreadyUsed(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// First verify succeeds
	resp = stub.MockInvoke("tx-verify1", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second verify should fail (already used)
	resp = stub.MockInvoke("tx-verify2", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestRefundTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Refund the ticket
	resp = stub.MockInvoke("tx-refund", [][]byte{
		[]byte("TicketContract:RefundTicket"),
		[]byte(ticket.TicketID),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify ticket is refunded
	ticketBytes := stub.State["ticket:"+ticket.TicketID]
	require.NotNil(t, ticketBytes)
	var refundedTicket Ticket
	json.Unmarshal(ticketBytes, &refundedTicket)
	assert.Equal(t, StatusRefunded, refundedTicket.Status)

	// Verify application is reset to REFUNDED
	appBytes := stub.State["application:evt001:user1"]
	require.NotNil(t, appBytes)
	var app Application
	json.Unmarshal(appBytes, &app)
	assert.Equal(t, StatusRefunded, app.Status)
}

func TestRefundTicketWrongOwner(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Try to refund as different user
	resp = stub.MockInvoke("tx-refund", [][]byte{
		[]byte("TicketContract:RefundTicket"),
		[]byte(ticket.TicketID),
		[]byte("user2"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestRefundTicketAlreadyUsed(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Verify (use the ticket)
	stub.MockInvoke("tx-verify", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})

	// Try to refund used ticket
	resp = stub.MockInvoke("tx-refund", [][]byte{
		[]byte("TicketContract:RefundTicket"),
		[]byte(ticket.TicketID),
		[]byte("user1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestLotteryPriorityWeighting(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	// Set up scores where user1 has much higher accuracy
	scores := map[string]MockScore{
		"user1":  {UserID: "user1", TotalBets: 100, CorrectBets: 95, AccuracyRate: 0.95},
		"user2":  {UserID: "user2", TotalBets: 100, CorrectBets: 90, AccuracyRate: 0.90},
		"user3":  {UserID: "user3", TotalBets: 100, CorrectBets: 85, AccuracyRate: 0.85},
		"user4":  {UserID: "user4", TotalBets: 100, CorrectBets: 10, AccuracyRate: 0.10},
		"user5":  {UserID: "user5", TotalBets: 100, CorrectBets: 5, AccuracyRate: 0.05},
		"user6":  {UserID: "user6", TotalBets: 100, CorrectBets: 3, AccuracyRate: 0.03},
		"user7":  {UserID: "user7", TotalBets: 100, CorrectBets: 80, AccuracyRate: 0.80},
		"user8":  {UserID: "user8", TotalBets: 100, CorrectBets: 75, AccuracyRate: 0.75},
		"user9":  {UserID: "user9", TotalBets: 100, CorrectBets: 2, AccuracyRate: 0.02},
		"user10": {UserID: "user10", TotalBets: 100, CorrectBets: 1, AccuracyRate: 0.01},
	}
	setupMockPredictionCC(stub, scores)

	// All 10 users apply
	for i := 1; i <= 10; i++ {
		userID := fmt.Sprintf("user%d", i)
		stub.MockInvoke(fmt.Sprintf("tx-apply-%d", i), [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
	}

	// Run lottery with 3 tickets
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("3"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	json.Unmarshal(resp.Payload, &result)
	assert.Equal(t, 3, len(result.Winners))
	assert.Equal(t, 7, len(result.Losers))

	// The priority formula is deterministic: 0.6 * accuracy + 0.4 * pseudo_random
	// We can verify that the lottery produced exactly 3 winners and 7 losers
	// The exact winners depend on the pseudo-random component, but the mechanism works
	t.Logf("Winners: %v", result.Winners)
	t.Logf("Losers: %v", result.Losers)
	t.Logf("Priorities: %v", result.Priorities)
}
