package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Pool represents the AMM liquidity pool for an event
type Pool struct {
	DocType     string `json:"docType"`
	EventID     string `json:"eventID"`
	OptionA     string `json:"optionA"`
	OptionB     string `json:"optionB"`
	PoolA       int64  `json:"poolA"`
	PoolB       int64  `json:"poolB"`
	K           int64  `json:"k"`
	TotalVolume int64  `json:"totalVolume"`
	Settled     bool   `json:"settled"`
}

// Bet represents a single user bet
type Bet struct {
	DocType   string `json:"docType"`
	EventID   string `json:"eventID"`
	UserID    string `json:"userID"`
	BetID     string `json:"betID"`
	Option    string `json:"option"`
	Amount    int64  `json:"amount"`
	Shares    int64  `json:"shares"`
	Timestamp string `json:"timestamp"`
}

// UserScore tracks a user's prediction accuracy.
//
// Field semantics (these are NOT all the same thing):
//   - PlacedBets: # of PlaceBet calls the user made (incremented in PlaceBet)
//   - TotalBets:  # of unique events the user participated in that have been settled
//     (incremented in Settle, deduplicated per event)
//   - CorrectBets: # of those settled events where at least one of the user's bets won
//   - AccuracyRate: CorrectBets / TotalBets
//
// PlacedBets is the user-facing "总下注" counter. TotalBets is the denominator
// for accuracy and only makes sense after settlement.
type UserScore struct {
	DocType      string  `json:"docType"`
	UserID       string  `json:"userID"`
	PlacedBets   int     `json:"placedBets"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

// BetResult is returned after placing a bet
type BetResult struct {
	BetID  string  `json:"betID"`
	Option string  `json:"option"`
	Amount int64   `json:"amount"`
	Shares int64   `json:"shares"`
	ProbA  float64 `json:"probA"`
	ProbB  float64 `json:"probB"`
}

// Odds represents current probabilities
type Odds struct {
	EventID string  `json:"eventID"`
	ProbA   float64 `json:"probA"`
	ProbB   float64 `json:"probB"`
	OptionA string  `json:"optionA"`
	OptionB string  `json:"optionB"`
}

// PayoutRecord represents a single winner's payout
type PayoutRecord struct {
	UserID string `json:"userID"`
	Shares int64  `json:"shares"`
	Payout int64  `json:"payout"`
}

// SettlementResult is returned after settling a pool
type SettlementResult struct {
	Settled     bool           `json:"settled"`
	WinnerCount int            `json:"winnerCount"`
	TotalPool   int64          `json:"totalPool"`
	Payouts     []PayoutRecord `json:"payouts"`
}

// PredictionContract implements the prediction market chaincode
type PredictionContract struct {
	contractapi.Contract
}

// InitializePool creates a new AMM pool for an event with initial liquidity
func (pc *PredictionContract) InitializePool(ctx contractapi.TransactionContextInterface, eventID string, optionA string, optionB string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}
	if optionA == "" || optionB == "" {
		return fmt.Errorf("options must not be empty")
	}

	key := "pool:" + eventID
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("pool for event %s already exists", eventID)
	}

	pool := Pool{
		DocType:     "pool",
		EventID:     eventID,
		OptionA:     optionA,
		OptionB:     optionB,
		PoolA:       10000,
		PoolB:       10000,
		K:           100000000, // 10000 * 10000
		TotalVolume: 0,
		Settled:     false,
	}

	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return fmt.Errorf("failed to marshal pool: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, poolJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// PlaceBet places a bet on an event option using the AMM
func (pc *PredictionContract) PlaceBet(ctx contractapi.TransactionContextInterface, eventID string, userID string, option string, amountStr string) (*BetResult, error) {
	if eventID == "" || userID == "" {
		return nil, fmt.Errorf("event ID and user ID must not be empty")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", err.Error())
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Read pool
	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	if pool.Settled {
		return nil, fmt.Errorf("pool for event %s is already settled", eventID)
	}

	// Validate option
	if option != pool.OptionA && option != pool.OptionB {
		return nil, fmt.Errorf("invalid option %s, must be %s or %s", option, pool.OptionA, pool.OptionB)
	}

	// Cross-chaincode call: transfer tokens from user to pool account
	poolAccountID := "pool:" + eventID
	transferArgs := [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte(userID),
		[]byte(poolAccountID),
		[]byte(amountStr),
	}
	transferResp := ctx.GetStub().InvokeChaincode("token", transferArgs, "eventchain")
	if transferResp.Status != 200 {
		return nil, fmt.Errorf("token transfer failed: %s", transferResp.Message)
	}

	// AMM calculation
	var shares int64
	if option == pool.OptionA {
		// Buy option A: poolB += amount, newPoolA = k / poolB, shares = poolA - newPoolA
		pool.PoolB += amount
		newPoolA := pool.K / pool.PoolB
		shares = pool.PoolA - newPoolA
		pool.PoolA = newPoolA
	} else {
		// Buy option B: poolA += amount, newPoolB = k / poolA, shares = poolB - newPoolB
		pool.PoolA += amount
		newPoolB := pool.K / pool.PoolA
		shares = pool.PoolB - newPoolB
		pool.PoolB = newPoolB
	}

	if shares <= 0 {
		return nil, fmt.Errorf("calculated shares must be positive, got %d", shares)
	}

	pool.TotalVolume += amount

	// Save updated pool
	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pool: %s", err.Error())
	}
	err = ctx.GetStub().PutState(poolKey, poolJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update pool: %s", err.Error())
	}

	// Create bet record
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)
	txID := ctx.GetStub().GetTxID()

	bet := Bet{
		DocType:   "bet",
		EventID:   eventID,
		UserID:    userID,
		BetID:     txID,
		Option:    option,
		Amount:    amount,
		Shares:    shares,
		Timestamp: timeStr,
	}

	betJSON, err := json.Marshal(bet)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bet: %s", err.Error())
	}

	betKey := fmt.Sprintf("bet:%s:%s:%s", eventID, userID, txID)
	err = ctx.GetStub().PutState(betKey, betJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to store bet: %s", err.Error())
	}

	// Increment the user's PlacedBets counter so the UI's "总下注" label
	// reflects every PlaceBet call (not just settled ones — TotalBets is
	// the post-settlement counter used for accuracy).
	scoreKey := "score:" + userID
	scoreBytes, scoreErr := ctx.GetStub().GetState(scoreKey)
	if scoreErr != nil {
		return nil, fmt.Errorf("failed to read score: %s", scoreErr.Error())
	}
	var score UserScore
	if scoreBytes != nil {
		if err := json.Unmarshal(scoreBytes, &score); err != nil {
			return nil, fmt.Errorf("failed to unmarshal score: %s", err.Error())
		}
	} else {
		score = UserScore{DocType: "score", UserID: userID}
	}
	score.PlacedBets++
	scoreJSON, err := json.Marshal(&score)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal score: %s", err.Error())
	}
	if err := ctx.GetStub().PutState(scoreKey, scoreJSON); err != nil {
		return nil, fmt.Errorf("failed to update score: %s", err.Error())
	}

	// Calculate new probabilities
	totalPool := float64(pool.PoolA + pool.PoolB)
	probA := float64(pool.PoolB) / totalPool
	probB := float64(pool.PoolA) / totalPool

	result := &BetResult{
		BetID:  txID,
		Option: option,
		Amount: amount,
		Shares: shares,
		ProbA:  probA,
		ProbB:  probB,
	}

	return result, nil
}

// GetOdds returns the current probabilities for an event
func (pc *PredictionContract) GetOdds(ctx contractapi.TransactionContextInterface, eventID string) (*Odds, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	totalPool := float64(pool.PoolA + pool.PoolB)
	odds := &Odds{
		EventID: eventID,
		ProbA:   float64(pool.PoolB) / totalPool,
		ProbB:   float64(pool.PoolA) / totalPool,
		OptionA: pool.OptionA,
		OptionB: pool.OptionB,
	}

	return odds, nil
}

// GetPool returns the full pool details for an event
func (pc *PredictionContract) GetPool(ctx contractapi.TransactionContextInterface, eventID string) (*Pool, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	return &pool, nil
}

// Settle distributes the pool to winners proportionally by shares
func (pc *PredictionContract) Settle(ctx contractapi.TransactionContextInterface, eventID string, winningOption string) (*SettlementResult, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}
	if winningOption == "" {
		return nil, fmt.Errorf("winning option must not be empty")
	}

	// Read pool
	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	if pool.Settled {
		return nil, fmt.Errorf("pool for event %s is already settled", eventID)
	}

	if winningOption != pool.OptionA && winningOption != pool.OptionB {
		return nil, fmt.Errorf("invalid winning option %s", winningOption)
	}

	// Get all bets for this event
	betStartKey := fmt.Sprintf("bet:%s:", eventID)
	betEndKey := fmt.Sprintf("bet:%s:~", eventID)
	iterator, err := ctx.GetStub().GetStateByRange(betStartKey, betEndKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get bets: %s", err.Error())
	}
	defer iterator.Close()

	var allBets []Bet
	var winningBets []Bet
	var totalWinningShares int64

	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate bets: %s", err.Error())
		}

		var bet Bet
		err = json.Unmarshal(queryResponse.Value, &bet)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal bet: %s", err.Error())
		}

		allBets = append(allBets, bet)

		if bet.Option == winningOption {
			winningBets = append(winningBets, bet)
			totalWinningShares += bet.Shares
		}
	}

	totalPool := pool.TotalVolume
	poolAccountID := "pool:" + eventID

	// Distribute payouts to winners
	var payouts []PayoutRecord

	if totalWinningShares > 0 && totalPool > 0 {
		var distributedTotal int64
		for i, bet := range winningBets {
			var payout int64
			if i == len(winningBets)-1 {
				// Last winner gets remainder to avoid rounding dust loss
				payout = totalPool - distributedTotal
			} else {
				payout = (bet.Shares * totalPool) / totalWinningShares
			}
			distributedTotal += payout

			if payout > 0 {
				// Cross-chaincode call: transfer payout from pool to winner
				transferArgs := [][]byte{
					[]byte("TokenContract:Transfer"),
					[]byte(poolAccountID),
					[]byte(bet.UserID),
					[]byte(strconv.FormatInt(payout, 10)),
				}
				transferResp := ctx.GetStub().InvokeChaincode("token", transferArgs, "eventchain")
				if transferResp.Status != 200 {
					return nil, fmt.Errorf("failed to transfer payout to %s: %s", bet.UserID, transferResp.Message)
				}
			}

			payouts = append(payouts, PayoutRecord{
				UserID: bet.UserID,
				Shares: bet.Shares,
				Payout: payout,
			})
		}
	}

	// Update scores for all participants
	// Track unique users to avoid double-counting
	userBetMap := make(map[string]bool) // userID -> won at least one bet
	for _, bet := range allBets {
		if _, exists := userBetMap[bet.UserID]; !exists {
			userBetMap[bet.UserID] = false
		}
		if bet.Option == winningOption {
			userBetMap[bet.UserID] = true
		}
	}

	for userID, won := range userBetMap {
		scoreKey := "score:" + userID
		scoreBytes, err := ctx.GetStub().GetState(scoreKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read score for %s: %s", userID, err.Error())
		}

		var score UserScore
		if scoreBytes != nil {
			err = json.Unmarshal(scoreBytes, &score)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal score for %s: %s", userID, err.Error())
			}
		} else {
			score = UserScore{
				DocType: "score",
				UserID:  userID,
			}
		}

		score.TotalBets++
		if won {
			score.CorrectBets++
		}

		if score.TotalBets > 0 {
			score.AccuracyRate = float64(score.CorrectBets) / float64(score.TotalBets)
		}

		scoreJSON, err := json.Marshal(score)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal score for %s: %s", userID, err.Error())
		}

		err = ctx.GetStub().PutState(scoreKey, scoreJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to update score for %s: %s", userID, err.Error())
		}
	}

	// Mark pool as settled
	pool.Settled = true
	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pool: %s", err.Error())
	}
	err = ctx.GetStub().PutState(poolKey, poolJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update pool: %s", err.Error())
	}

	if payouts == nil {
		payouts = []PayoutRecord{}
	}

	result := &SettlementResult{
		Settled:     true,
		WinnerCount: len(winningBets),
		TotalPool:   totalPool,
		Payouts:     payouts,
	}

	return result, nil
}

// GetUserScore returns a user's prediction accuracy score
func (pc *PredictionContract) GetUserScore(ctx contractapi.TransactionContextInterface, userID string) (*UserScore, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	scoreKey := "score:" + userID
	scoreBytes, err := ctx.GetStub().GetState(scoreKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read score: %s", err.Error())
	}

	if scoreBytes == nil {
		return &UserScore{
			DocType:      "score",
			UserID:       userID,
			TotalBets:    0,
			CorrectBets:  0,
			AccuracyRate: 0.0,
		}, nil
	}

	var score UserScore
	err = json.Unmarshal(scoreBytes, &score)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal score: %s", err.Error())
	}

	return &score, nil
}

// GetUserBets returns all bets for a user, optionally filtered by event
func (pc *PredictionContract) GetUserBets(ctx contractapi.TransactionContextInterface, userID string, eventID string) ([]*Bet, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	var bets []*Bet

	if eventID != "" {
		// Query bets for specific event and user
		startKey := fmt.Sprintf("bet:%s:%s:", eventID, userID)
		endKey := fmt.Sprintf("bet:%s:%s:~", eventID, userID)
		iterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get bets: %s", err.Error())
		}
		defer iterator.Close()

		for iterator.HasNext() {
			queryResponse, err := iterator.Next()
			if err != nil {
				return nil, fmt.Errorf("failed to iterate: %s", err.Error())
			}

			var bet Bet
			err = json.Unmarshal(queryResponse.Value, &bet)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal bet: %s", err.Error())
			}

			bets = append(bets, &bet)
		}
	} else {
		// Query all bets across all events for this user
		// Scan all bets and filter by userID
		iterator, err := ctx.GetStub().GetStateByRange("bet:", "bet:~")
		if err != nil {
			return nil, fmt.Errorf("failed to get bets: %s", err.Error())
		}
		defer iterator.Close()

		for iterator.HasNext() {
			queryResponse, err := iterator.Next()
			if err != nil {
				return nil, fmt.Errorf("failed to iterate: %s", err.Error())
			}

			var bet Bet
			err = json.Unmarshal(queryResponse.Value, &bet)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal bet: %s", err.Error())
			}

			if bet.UserID == userID {
				bets = append(bets, &bet)
			}
		}
	}

	if bets == nil {
		bets = []*Bet{}
	}

	return bets, nil
}

// GetLeaderboard returns top users sorted by accuracy rate (CouchDB rich query)
func (pc *PredictionContract) GetLeaderboard(ctx contractapi.TransactionContextInterface) ([]*UserScore, error) {
	queryString := `{"selector":{"docType":"score","totalBets":{"$gt":0}},"sort":[{"accuracyRate":"desc"}],"limit":20}`

	iterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		// Fallback: scan all scores by key range if rich query fails
		iterator2, err2 := ctx.GetStub().GetStateByRange("score:", "score:~")
		if err2 != nil {
			return []*UserScore{}, nil
		}
		defer iterator2.Close()

		var scores []*UserScore
		for iterator2.HasNext() {
			queryResponse, err3 := iterator2.Next()
			if err3 != nil {
				break
			}
			var score UserScore
			if json.Unmarshal(queryResponse.Value, &score) == nil && score.TotalBets > 0 {
				scores = append(scores, &score)
			}
		}

		// Sort by accuracy descending
		sort.Slice(scores, func(i, j int) bool {
			return scores[i].AccuracyRate > scores[j].AccuracyRate
		})

		if len(scores) > 20 {
			scores = scores[:20]
		}
		if scores == nil {
			scores = []*UserScore{}
		}
		return scores, nil
	}
	defer iterator.Close()

	var scores []*UserScore
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			break
		}
		var score UserScore
		if json.Unmarshal(queryResponse.Value, &score) == nil {
			scores = append(scores, &score)
		}
	}

	if scores == nil {
		scores = []*UserScore{}
	}
	return scores, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&PredictionContract{})
	if err != nil {
		log.Fatalf("Error creating chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Fatalf("Error starting chaincode: %v", err)
	}
}
