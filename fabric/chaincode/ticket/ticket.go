package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Status constants for applications and tickets
const (
	StatusPending  = "PENDING"
	StatusWon      = "WON"
	StatusLost     = "LOST"
	StatusClaimed  = "CLAIMED"
	StatusUsed     = "USED"
	StatusRefunded = "REFUNDED"
)

// Application represents a ticket application
type Application struct {
	DocType   string  `json:"docType"`
	EventID   string  `json:"eventID"`
	UserID    string  `json:"userID"`
	Status    string  `json:"status"`
	Priority  float64 `json:"priority"`
	Timestamp string  `json:"timestamp"`
}

// Ticket represents an issued ticket
type Ticket struct {
	DocType    string `json:"docType"`
	TicketID   string `json:"ticketID"`
	EventID    string `json:"eventID"`
	OwnerID    string `json:"ownerID"`
	Status     string `json:"status"`
	ClaimHash  string `json:"claimHash"`
	VerifiedAt string `json:"verifiedAt"`
}

// LotteryResult is returned after running the lottery
type LotteryResult struct {
	EventID    string             `json:"eventID"`
	Winners    []string           `json:"winners"`
	Losers     []string           `json:"losers"`
	Priorities map[string]float64 `json:"priorities"`
}

// UserScoreResponse matches the structure returned by prediction-cc.GetUserScore
type UserScoreResponse struct {
	UserID       string  `json:"userID"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

// applicantPriority is used for sorting during lottery
type applicantPriority struct {
	UserID   string
	Priority float64
}

// TicketContract implements the ticket management chaincode
type TicketContract struct {
	contractapi.Contract
}

// ApplyTicket submits a ticket application for an event
func (tc *TicketContract) ApplyTicket(ctx contractapi.TransactionContextInterface, eventID string, userID string) error {
	if eventID == "" || userID == "" {
		return fmt.Errorf("event ID and user ID must not be empty")
	}

	key := fmt.Sprintf("application:%s:%s", eventID, userID)

	// Check for existing application
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("application already exists for user %s on event %s", userID, eventID)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	app := Application{
		DocType:   "application",
		EventID:   eventID,
		UserID:    userID,
		Status:    StatusPending,
		Priority:  0,
		Timestamp: timeStr,
	}

	appJSON, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("failed to marshal application: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, appJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// RunLottery executes the weighted lottery for an event
// Priority = 0.6 * accuracy + 0.4 * pseudorandom
// Pseudorandom seed = SHA256(txID + eventID + userID) for deterministic cross-peer consensus
func (tc *TicketContract) RunLottery(ctx contractapi.TransactionContextInterface, eventID string, ticketCountStr string) (*LotteryResult, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	ticketCount, err := strconv.Atoi(ticketCountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket count: %s", err.Error())
	}
	if ticketCount <= 0 {
		return nil, fmt.Errorf("ticket count must be positive")
	}

	// Check if lottery already ran (any non-PENDING applications)
	startKey := fmt.Sprintf("application:%s:", eventID)
	endKey := fmt.Sprintf("application:%s:~", eventID)
	iterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %s", err.Error())
	}
	defer iterator.Close()

	txID := ctx.GetStub().GetTxID()
	var applicants []applicantPriority
	priorities := make(map[string]float64)

	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %s", err.Error())
		}

		var app Application
		err = json.Unmarshal(queryResponse.Value, &app)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal application: %s", err.Error())
		}

		// If any application is not PENDING, lottery already ran
		if app.Status != StatusPending {
			return nil, fmt.Errorf("lottery already executed for event %s", eventID)
		}

		// Get user's prediction accuracy via cross-chaincode call
		scoreArgs := [][]byte{
			[]byte("PredictionContract:GetUserScore"),
			[]byte(app.UserID),
		}
		scoreResp := ctx.GetStub().InvokeChaincode("prediction-cc", scoreArgs, "eventchain")

		var accuracyRate float64
		if scoreResp.Status == 200 && scoreResp.Payload != nil {
			var score UserScoreResponse
			err = json.Unmarshal(scoreResp.Payload, &score)
			if err == nil {
				accuracyRate = score.AccuracyRate
			}
		}

		// Calculate pseudo-random component
		// seed = SHA256(txID + eventID + userID) for deterministic consensus
		seedData := txID + eventID + app.UserID
		hash := sha256.Sum256([]byte(seedData))
		// Use first 8 bytes of hash as a uint64, then normalize to [0, 1)
		var hashVal uint64
		for i := 0; i < 8; i++ {
			hashVal = (hashVal << 8) | uint64(hash[i])
		}
		pseudoRandom := float64(hashVal) / float64(^uint64(0))

		// priority = 0.6 * accuracy + 0.4 * pseudorandom
		priority := 0.6*accuracyRate + 0.4*pseudoRandom

		applicants = append(applicants, applicantPriority{
			UserID:   app.UserID,
			Priority: priority,
		})
		priorities[app.UserID] = priority
	}

	// Sort by priority descending
	sort.Slice(applicants, func(i, j int) bool {
		return applicants[i].Priority > applicants[j].Priority
	})

	// Determine winners (top N = ticketCount)
	winnerCount := ticketCount
	if winnerCount > len(applicants) {
		winnerCount = len(applicants)
	}

	var winners []string
	var losers []string

	for i, ap := range applicants {
		appKey := fmt.Sprintf("application:%s:%s", eventID, ap.UserID)
		appBytes, err := ctx.GetStub().GetState(appKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read application: %s", err.Error())
		}

		var app Application
		err = json.Unmarshal(appBytes, &app)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal application: %s", err.Error())
		}

		if i < winnerCount {
			app.Status = StatusWon
			app.Priority = ap.Priority
			winners = append(winners, ap.UserID)
		} else {
			app.Status = StatusLost
			app.Priority = ap.Priority
			losers = append(losers, ap.UserID)
		}

		appJSON, err := json.Marshal(app)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal application: %s", err.Error())
		}

		err = ctx.GetStub().PutState(appKey, appJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to update application: %s", err.Error())
		}
	}

	if winners == nil {
		winners = []string{}
	}
	if losers == nil {
		losers = []string{}
	}

	result := &LotteryResult{
		EventID:    eventID,
		Winners:    winners,
		Losers:     losers,
		Priorities: priorities,
	}

	return result, nil
}

// ClaimTicket allows a lottery winner to claim their ticket, generating a SHA256 hash for QR
func (tc *TicketContract) ClaimTicket(ctx contractapi.TransactionContextInterface, eventID string, userID string) (*Ticket, error) {
	if eventID == "" || userID == "" {
		return nil, fmt.Errorf("event ID and user ID must not be empty")
	}

	// Verify user won the lottery
	appKey := fmt.Sprintf("application:%s:%s", eventID, userID)
	appBytes, err := ctx.GetStub().GetState(appKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read application: %s", err.Error())
	}
	if appBytes == nil {
		return nil, fmt.Errorf("no application found for user %s on event %s", userID, eventID)
	}

	var app Application
	err = json.Unmarshal(appBytes, &app)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal application: %s", err.Error())
	}

	if app.Status != StatusWon {
		return nil, fmt.Errorf("user %s did not win the lottery (status: %s)", userID, app.Status)
	}

	// Generate ticket ID and claim hash
	txID := ctx.GetStub().GetTxID()
	ticketID := fmt.Sprintf("%s-%s-%s", eventID, userID, txID)

	// Generate SHA256 hash for QR code
	hashInput := fmt.Sprintf("%s:%s:%s:%s", ticketID, eventID, userID, txID)
	hash := sha256.Sum256([]byte(hashInput))
	claimHash := hex.EncodeToString(hash[:])

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	ticket := Ticket{
		DocType:    "ticket",
		TicketID:   ticketID,
		EventID:    eventID,
		OwnerID:    userID,
		Status:     StatusClaimed,
		ClaimHash:  claimHash,
		VerifiedAt: "",
	}

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ticket: %s", err.Error())
	}

	ticketKey := "ticket:" + ticketID
	err = ctx.GetStub().PutState(ticketKey, ticketJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to store ticket: %s", err.Error())
	}

	// Update application status to CLAIMED
	app.Status = StatusClaimed
	app.Timestamp = timeStr
	appJSON, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application: %s", err.Error())
	}

	err = ctx.GetStub().PutState(appKey, appJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update application: %s", err.Error())
	}

	return &ticket, nil
}

// VerifyTicket checks the claim hash and marks the ticket as USED
func (tc *TicketContract) VerifyTicket(ctx contractapi.TransactionContextInterface, ticketID string, hash string) (*Ticket, error) {
	if ticketID == "" || hash == "" {
		return nil, fmt.Errorf("ticket ID and hash must not be empty")
	}

	ticketKey := "ticket:" + ticketID
	ticketBytes, err := ctx.GetStub().GetState(ticketKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticket: %s", err.Error())
	}
	if ticketBytes == nil {
		return nil, fmt.Errorf("ticket %s not found", ticketID)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ticket: %s", err.Error())
	}

	// Check if already used
	if ticket.Status == StatusUsed {
		return nil, fmt.Errorf("ticket %s has already been used", ticketID)
	}

	// Check if ticket is in CLAIMED status
	if ticket.Status != StatusClaimed {
		return nil, fmt.Errorf("ticket %s is not in CLAIMED status (current: %s)", ticketID, ticket.Status)
	}

	// Verify hash
	if ticket.ClaimHash != hash {
		return nil, fmt.Errorf("hash mismatch for ticket %s", ticketID)
	}

	// Mark as used
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	ticket.Status = StatusUsed
	ticket.VerifiedAt = timeStr

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ticket: %s", err.Error())
	}

	err = ctx.GetStub().PutState(ticketKey, ticketJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket: %s", err.Error())
	}

	return &ticket, nil
}

// RefundTicket allows the ticket owner to refund before the event
func (tc *TicketContract) RefundTicket(ctx contractapi.TransactionContextInterface, ticketID string, userID string) error {
	if ticketID == "" || userID == "" {
		return fmt.Errorf("ticket ID and user ID must not be empty")
	}

	ticketKey := "ticket:" + ticketID
	ticketBytes, err := ctx.GetStub().GetState(ticketKey)
	if err != nil {
		return fmt.Errorf("failed to read ticket: %s", err.Error())
	}
	if ticketBytes == nil {
		return fmt.Errorf("ticket %s not found", ticketID)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ticket: %s", err.Error())
	}

	// Verify owner
	if ticket.OwnerID != userID {
		return fmt.Errorf("user %s is not the owner of ticket %s", userID, ticketID)
	}

	// Can only refund CLAIMED tickets (not USED or already REFUNDED)
	if ticket.Status != StatusClaimed {
		return fmt.Errorf("can only refund CLAIMED tickets, current status: %s", ticket.Status)
	}

	// Mark ticket as refunded
	ticket.Status = StatusRefunded

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		return fmt.Errorf("failed to marshal ticket: %s", err.Error())
	}

	err = ctx.GetStub().PutState(ticketKey, ticketJSON)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %s", err.Error())
	}

	// Update application status to REFUNDED
	appKey := fmt.Sprintf("application:%s:%s", ticket.EventID, userID)
	appBytes, err := ctx.GetStub().GetState(appKey)
	if err != nil {
		return fmt.Errorf("failed to read application: %s", err.Error())
	}

	if appBytes != nil {
		var app Application
		err = json.Unmarshal(appBytes, &app)
		if err != nil {
			return fmt.Errorf("failed to unmarshal application: %s", err.Error())
		}

		app.Status = StatusRefunded

		appJSON, err := json.Marshal(app)
		if err != nil {
			return fmt.Errorf("failed to marshal application: %s", err.Error())
		}

		err = ctx.GetStub().PutState(appKey, appJSON)
		if err != nil {
			return fmt.Errorf("failed to update application: %s", err.Error())
		}
	}

	return nil
}

// GetUserTickets returns all tickets owned by a user
func (tc *TicketContract) GetUserTickets(ctx contractapi.TransactionContextInterface, userID string) ([]*Ticket, error) {
	if userID == "" {
		return []*Ticket{}, nil
	}

	iterator, err := ctx.GetStub().GetStateByRange("ticket:", "ticket:~")
	if err != nil {
		return []*Ticket{}, nil
	}
	defer iterator.Close()

	var tickets []*Ticket
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			break
		}
		var ticket Ticket
		if json.Unmarshal(queryResponse.Value, &ticket) == nil && ticket.OwnerID == userID {
			tickets = append(tickets, &ticket)
		}
	}
	if tickets == nil {
		tickets = []*Ticket{}
	}
	return tickets, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&TicketContract{})
	if err != nil {
		log.Fatalf("Error creating chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Fatalf("Error starting chaincode: %v", err)
	}
}
