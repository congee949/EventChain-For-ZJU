package main

import (
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

func setupPredictionChaincode(t *testing.T) (*shimtest.MockStub, *PredictionContract) {
	pc := new(PredictionContract)
	cc, err := contractapi.NewChaincode(pc)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("prediction", cc)
	require.NotNil(t, stub)
	return stub, pc
}

// mockTokenTransferSuccess creates a mock token chaincode that always returns success
func mockTokenTransferSuccess() *shimtest.MockStub {
	// Create a minimal mock stub for the token chaincode
	// The mock will return OK for Transfer calls
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	return tokenStub
}

func initializePool(t *testing.T, stub *shimtest.MockStub, eventID string) {
	// Set up mock for token chaincode
	tokenStub := mockTokenTransferSuccess()
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx-init-"+eventID, [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte(eventID),
		[]byte("A"),
		[]byte("B"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
}

func TestInitializePool(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := mockTokenTransferSuccess()
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte("evt001"),
		[]byte("A"),
		[]byte("B"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify pool state
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("PredictionContract:GetPool"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var pool Pool
	err := json.Unmarshal(resp.Payload, &pool)
	require.NoError(t, err)
	assert.Equal(t, "evt001", pool.EventID)
	assert.Equal(t, int64(10000), pool.PoolA)
	assert.Equal(t, int64(10000), pool.PoolB)
	assert.Equal(t, int64(100000000), pool.K)
	assert.Equal(t, int64(0), pool.TotalVolume)
	assert.Equal(t, "A", pool.OptionA)
	assert.Equal(t, "B", pool.OptionB)
}

func TestInitializePoolDuplicate(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := mockTokenTransferSuccess()
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte("evt001"),
		[]byte("A"),
		[]byte("B"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second init should fail
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte("evt001"),
		[]byte("A"),
		[]byte("B"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestPlaceBet(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Set up mock token chaincode that returns success for Transfer
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	tokenStub.MockInvokeWithSignedProposal("mock", [][]byte{[]byte("init")}, nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place a bet on option A with 100 tokens
	resp := stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Parse the bet result
	var betResult BetResult
	err := json.Unmarshal(resp.Payload, &betResult)
	require.NoError(t, err)
	assert.Greater(t, betResult.Shares, int64(0))
	assert.Equal(t, "A", betResult.Option)

	// Verify pool state changed
	resp = stub.MockInvoke("tx-pool1", [][]byte{
		[]byte("PredictionContract:GetPool"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var pool Pool
	err = json.Unmarshal(resp.Payload, &pool)
	require.NoError(t, err)

	// After betting on A: poolB += 100 = 10100, newPoolA = 100000000/10100 = 9900 (approx)
	assert.Equal(t, int64(10100), pool.PoolB)
	assert.Equal(t, int64(100), pool.TotalVolume)
	// Verify k invariant: newPoolA * newPoolB should be close to k
	assert.InDelta(t, float64(pool.K), float64(pool.PoolA*pool.PoolB), float64(pool.K)*0.01)
}

func TestPlaceBetInvalidOption(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place bet with invalid option
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("C"),
		[]byte("100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestPlaceBetInvalidAmount(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place bet with zero amount
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("0"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestPlaceBetNoPool(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	// Bet on event with no pool
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt999"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestGetOdds(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Initial odds should be 50/50
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:GetOdds"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var odds Odds
	err := json.Unmarshal(resp.Payload, &odds)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, odds.ProbA, 0.001)
	assert.InDelta(t, 0.5, odds.ProbB, 0.001)

	// Place a bet on A, odds should shift
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("1000"),
	})

	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("PredictionContract:GetOdds"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &odds)
	require.NoError(t, err)
	// After betting on A, P(A) should increase
	assert.Greater(t, odds.ProbA, 0.5)
	assert.Less(t, odds.ProbB, 0.5)
	assert.InDelta(t, 1.0, odds.ProbA+odds.ProbB, 0.001)
}

func TestAMMConstantProduct(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place multiple bets and verify k invariant holds
	bets := []struct {
		user   string
		option string
		amount string
	}{
		{"user1", "A", "500"},
		{"user2", "B", "300"},
		{"user3", "A", "1000"},
		{"user4", "B", "200"},
		{"user5", "A", "100"},
	}

	for i, bet := range bets {
		txID := fmt.Sprintf("tx-bet-%d", i)
		resp := stub.MockInvoke(txID, [][]byte{
			[]byte("PredictionContract:PlaceBet"),
			[]byte("evt001"),
			[]byte(bet.user),
			[]byte(bet.option),
			[]byte(bet.amount),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

		// Check pool after each bet
		poolResp := stub.MockInvoke(txID+"-pool", [][]byte{
			[]byte("PredictionContract:GetPool"),
			[]byte("evt001"),
		})
		var pool Pool
		json.Unmarshal(poolResp.Payload, &pool)

		// k should remain constant (within integer rounding)
		product := pool.PoolA * pool.PoolB
		assert.InDelta(t, float64(pool.K), float64(product), float64(pool.K)*0.01,
			"k invariant violated after bet %d: poolA=%d, poolB=%d, product=%d, k=%d",
			i, pool.PoolA, pool.PoolB, product, pool.K)
	}
}

func TestSettle(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Create a mock token chaincode
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place bets
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("500"),
	})
	stub.MockInvoke("tx-bet2", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user2"),
		[]byte("B"),
		[]byte("300"),
	})
	stub.MockInvoke("tx-bet3", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user3"),
		[]byte("A"),
		[]byte("200"),
	})

	// Settle with result A
	resp := stub.MockInvoke("tx-settle", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result SettlementResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.WinnerCount)
	assert.Equal(t, int64(1000), result.TotalPool) // 500 + 300 + 200
	assert.Equal(t, true, result.Settled)

	// Verify scores updated
	resp = stub.MockInvoke("tx-score1", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var score UserScore
	err = json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, 1, score.TotalBets)
	assert.Equal(t, 1, score.CorrectBets)
	assert.InDelta(t, 1.0, score.AccuracyRate, 0.001)

	// User2 (loser) should have 0 correct
	resp = stub.MockInvoke("tx-score2", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("user2"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, 1, score.TotalBets)
	assert.Equal(t, 0, score.CorrectBets)
	assert.InDelta(t, 0.0, score.AccuracyRate, 0.001)
}

func TestSettleNoPool(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt999"),
		[]byte("A"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestSettleAlreadySettled(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})

	// First settle
	resp := stub.MockInvoke("tx-settle1", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second settle should fail
	resp = stub.MockInvoke("tx-settle2", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestGetUserScore(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Query score for user with no bets
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("newuser"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var score UserScore
	err := json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, "newuser", score.UserID)
	assert.Equal(t, 0, score.TotalBets)
	assert.Equal(t, 0, score.CorrectBets)
	assert.InDelta(t, 0.0, score.AccuracyRate, 0.001)
}

func TestGetUserBets(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")
	initializePool(t, stub, "evt002")

	// User1 bets on two events
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-bet2", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt002"),
		[]byte("user1"),
		[]byte("B"),
		[]byte("200"),
	})

	// Get all bets for user1
	resp := stub.MockInvoke("tx-bets1", [][]byte{
		[]byte("PredictionContract:GetUserBets"),
		[]byte("user1"),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var bets []Bet
	err := json.Unmarshal(resp.Payload, &bets)
	require.NoError(t, err)
	assert.Equal(t, 2, len(bets))

	// Get bets for user1 on evt001 only
	resp = stub.MockInvoke("tx-bets2", [][]byte{
		[]byte("PredictionContract:GetUserBets"),
		[]byte("user1"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &bets)
	require.NoError(t, err)
	assert.Equal(t, 1, len(bets))
	assert.Equal(t, "evt001", bets[0].EventID)
}

func TestMultipleSettlementsScoreAccumulation(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	// Event 1: user1 wins
	initializePool(t, stub, "evt001")
	stub.MockInvoke("tx-bet1a", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-settle1", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})

	// Event 2: user1 loses
	initializePool(t, stub, "evt002")
	stub.MockInvoke("tx-bet2a", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt002"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-settle2", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt002"),
		[]byte("B"),
	})

	// Event 3: user1 wins
	initializePool(t, stub, "evt003")
	stub.MockInvoke("tx-bet3a", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt003"),
		[]byte("user1"),
		[]byte("B"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-settle3", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt003"),
		[]byte("B"),
	})

	// Check user1 score: 3 total, 2 correct, 66.7% accuracy
	resp := stub.MockInvoke("tx-score", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var score UserScore
	err := json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, 3, score.TotalBets)
	assert.Equal(t, 2, score.CorrectBets)
	assert.InDelta(t, 2.0/3.0, score.AccuracyRate, 0.01)
}

func TestCrossChaincodeMockInvoke(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Set up a mock token chaincode that returns a specific response
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// The InvokeChaincode call should go through to the mock
	// Place a bet which internally calls token-cc.Transfer
	resp := stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	// Should succeed because mock returns OK by default
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
}

func TestSettlementPayoutDistribution(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// User1 bets 600 on A, user2 bets 400 on A, user3 bets 1000 on B
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("600"),
	})
	stub.MockInvoke("tx-bet2", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user2"),
		[]byte("A"),
		[]byte("400"),
	})
	stub.MockInvoke("tx-bet3", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user3"),
		[]byte("B"),
		[]byte("1000"),
	})

	// Settle with A winning
	resp := stub.MockInvoke("tx-settle", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result SettlementResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.WinnerCount)
	assert.Equal(t, int64(2000), result.TotalPool) // 600 + 400 + 1000

	// Total payouts should equal total pool (minus rounding dust)
	var totalPayout int64
	for _, p := range result.Payouts {
		totalPayout += p.Payout
	}
	assert.InDelta(t, float64(result.TotalPool), float64(totalPayout), 2.0, "total payouts should approximately equal total pool")
}

// Mock chaincode for testing InvokeChaincode with controlled responses
type MockTokenCC struct {
}

func (m *MockTokenCC) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (m *MockTokenCC) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, _ := stub.GetFunctionAndParameters()
	if fn == "TokenContract:Transfer" {
		return shim.Success(nil)
	}
	return shim.Error("unknown function: " + fn)
}
