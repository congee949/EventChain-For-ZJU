package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Event status constants
const (
	StatusCreated        = "CREATED"
	StatusPredictionOpen = "PREDICTION_OPEN"
	StatusTicketOpen     = "TICKET_OPEN"
	StatusOngoing        = "ONGOING"
	StatusSettled        = "SETTLED"
)

// validTransitions defines the allowed state machine transitions
var validTransitions = map[string]string{
	StatusCreated:        StatusPredictionOpen,
	StatusPredictionOpen: StatusTicketOpen,
	StatusTicketOpen:     StatusOngoing,
	StatusOngoing:        StatusSettled,
}

// Event represents a campus event stored in world state
type Event struct {
	DocType           string   `json:"docType"`
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Type              string   `json:"type"`
	Teams             []string `json:"teams"`
	TicketTotal       int      `json:"ticketTotal"`
	Status            string   `json:"status"`
	PredictionOptions []string `json:"predictionOptions"`
	Result            string   `json:"result"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

// EventContract implements the event chaincode
type EventContract struct {
	contractapi.Contract
}

// CreateEvent creates a new event
func (ec *EventContract) CreateEvent(ctx contractapi.TransactionContextInterface, eventID string, title string, eventType string, teamsJSON string, ticketTotalStr string, optionsJSON string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}
	if title == "" {
		return fmt.Errorf("title must not be empty")
	}

	// Check if event already exists
	key := "event:" + eventID
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("event %s already exists", eventID)
	}

	// Parse teams
	var teams []string
	err = json.Unmarshal([]byte(teamsJSON), &teams)
	if err != nil {
		return fmt.Errorf("failed to parse teams: %s", err.Error())
	}
	if len(teams) < 2 {
		return fmt.Errorf("at least 2 teams required")
	}

	// Parse ticket total
	ticketTotal, err := strconv.Atoi(ticketTotalStr)
	if err != nil {
		return fmt.Errorf("invalid ticket total: %s", err.Error())
	}
	if ticketTotal <= 0 {
		return fmt.Errorf("ticket total must be positive")
	}

	// Parse prediction options
	var options []string
	err = json.Unmarshal([]byte(optionsJSON), &options)
	if err != nil {
		return fmt.Errorf("failed to parse prediction options: %s", err.Error())
	}
	if len(options) < 2 {
		return fmt.Errorf("at least 2 prediction options required")
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	evt := Event{
		DocType:           "event",
		ID:                eventID,
		Title:             title,
		Type:              eventType,
		Teams:             teams,
		TicketTotal:       ticketTotal,
		Status:            StatusCreated,
		PredictionOptions: options,
		Result:            "",
		CreatedAt:         timeStr,
		UpdatedAt:         timeStr,
	}

	evtJSON, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, evtJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// QueryEvent returns a single event by ID
func (ec *EventContract) QueryEvent(ctx contractapi.TransactionContextInterface, eventID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	key := "event:" + eventID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %s", err.Error())
	}
	if data == nil {
		return nil, fmt.Errorf("event %s not found", eventID)
	}

	var evt Event
	err = json.Unmarshal(data, &evt)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %s", err.Error())
	}

	return &evt, nil
}

// ListEvents returns events filtered by status and/or type using key range query
func (ec *EventContract) ListEvents(ctx contractapi.TransactionContextInterface, status string, eventType string) ([]*Event, error) {
	// Use GetStateByRange to get all events
	resultsIterator, err := ctx.GetStub().GetStateByRange("event:", "event:~")
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %s", err.Error())
	}
	defer resultsIterator.Close()

	var events []*Event
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %s", err.Error())
		}

		var evt Event
		err = json.Unmarshal(queryResponse.Value, &evt)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %s", err.Error())
		}

		// Apply filters
		if status != "" && evt.Status != status {
			continue
		}
		if eventType != "" && evt.Type != eventType {
			continue
		}

		events = append(events, &evt)
	}

	if events == nil {
		events = []*Event{}
	}

	return events, nil
}

// UpdateStatus advances the event state machine
func (ec *EventContract) UpdateStatus(ctx contractapi.TransactionContextInterface, eventID string, newStatus string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}

	key := "event:" + eventID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if data == nil {
		return fmt.Errorf("event %s not found", eventID)
	}

	var evt Event
	err = json.Unmarshal(data, &evt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %s", err.Error())
	}

	// Validate state transition
	allowedNext, exists := validTransitions[evt.Status]
	if !exists {
		return fmt.Errorf("event %s is in terminal state %s", eventID, evt.Status)
	}
	if newStatus != allowedNext {
		return fmt.Errorf("invalid transition: %s -> %s (allowed: %s -> %s)", evt.Status, newStatus, evt.Status, allowedNext)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	evt.Status = newStatus
	evt.UpdatedAt = timeStr

	evtJSON, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, evtJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// UpdateResult records the event outcome and sets status to SETTLED
func (ec *EventContract) UpdateResult(ctx contractapi.TransactionContextInterface, eventID string, outcome string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}
	if outcome == "" {
		return fmt.Errorf("outcome must not be empty")
	}

	key := "event:" + eventID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if data == nil {
		return fmt.Errorf("event %s not found", eventID)
	}

	var evt Event
	err = json.Unmarshal(data, &evt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %s", err.Error())
	}

	// Can only update result when event is ONGOING
	if evt.Status != StatusOngoing {
		return fmt.Errorf("can only update result for ONGOING events, current status: %s", evt.Status)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	evt.Result = outcome
	evt.Status = StatusSettled
	evt.UpdatedAt = timeStr

	evtJSON, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, evtJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&EventContract{})
	if err != nil {
		log.Fatalf("Error creating chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Fatalf("Error starting chaincode: %v", err)
	}
}
