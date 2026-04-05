package event

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventChaincode(t *testing.T) (*shimtest.MockStub, *EventContract) {
	ec := new(EventContract)
	cc, err := contractapi.NewChaincode(ec)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("event", cc)
	require.NotNil(t, stub)
	return stub, ec
}

func createTestEvent(t *testing.T, stub *shimtest.MockStub, eventID string) {
	teamsJSON, _ := json.Marshal([]string{"TeamA", "TeamB"})
	optionsJSON, _ := json.Marshal([]string{"TeamA wins", "TeamB wins"})

	resp := stub.MockInvoke("tx-create-"+eventID, [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte(eventID),
		[]byte("Test Basketball Game"),
		[]byte("basketball"),
		teamsJSON,
		[]byte("100"),
		optionsJSON,
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
}

func TestCreateEvent(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	teamsJSON, _ := json.Marshal([]string{"ZJU Eagles", "ZJU Lions"})
	optionsJSON, _ := json.Marshal([]string{"Eagles win", "Lions win"})

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte("evt001"),
		[]byte("Basketball Finals"),
		[]byte("basketball"),
		teamsJSON,
		[]byte("200"),
		optionsJSON,
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Query the created event
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var evt Event
	err := json.Unmarshal(resp.Payload, &evt)
	require.NoError(t, err)
	assert.Equal(t, "evt001", evt.ID)
	assert.Equal(t, "Basketball Finals", evt.Title)
	assert.Equal(t, "basketball", evt.Type)
	assert.Equal(t, []string{"ZJU Eagles", "ZJU Lions"}, evt.Teams)
	assert.Equal(t, 200, evt.TicketTotal)
	assert.Equal(t, StatusCreated, evt.Status)
	assert.Equal(t, []string{"Eagles win", "Lions win"}, evt.PredictionOptions)
	assert.Equal(t, "", evt.Result)
}

func TestCreateEventDuplicate(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// Try to create again with same ID
	teamsJSON, _ := json.Marshal([]string{"A", "B"})
	optionsJSON, _ := json.Marshal([]string{"A wins", "B wins"})
	resp := stub.MockInvoke("tx-dup", [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte("evt001"),
		[]byte("Duplicate"),
		[]byte("football"),
		teamsJSON,
		[]byte("50"),
		optionsJSON,
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestCreateEventInvalidTicketTotal(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	teamsJSON, _ := json.Marshal([]string{"A", "B"})
	optionsJSON, _ := json.Marshal([]string{"A wins", "B wins"})

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte("evt001"),
		[]byte("Test"),
		[]byte("basketball"),
		teamsJSON,
		[]byte("0"),
		optionsJSON,
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestQueryEventNotFound(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("nonexistent"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestUpdateStatusValidTransitions(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// CREATED -> PREDICTION_OPEN
	resp := stub.MockInvoke("tx-s1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("PREDICTION_OPEN"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify status
	resp = stub.MockInvoke("tx-q1", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	var evt Event
	json.Unmarshal(resp.Payload, &evt)
	assert.Equal(t, StatusPredictionOpen, evt.Status)

	// PREDICTION_OPEN -> TICKET_OPEN
	resp = stub.MockInvoke("tx-s2", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("TICKET_OPEN"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// TICKET_OPEN -> ONGOING
	resp = stub.MockInvoke("tx-s3", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("ONGOING"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// ONGOING -> SETTLED
	resp = stub.MockInvoke("tx-s4", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("SETTLED"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify final status
	resp = stub.MockInvoke("tx-q2", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	json.Unmarshal(resp.Payload, &evt)
	assert.Equal(t, StatusSettled, evt.Status)
}

func TestUpdateStatusInvalidTransition(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// CREATED -> ONGOING should fail (must go through PREDICTION_OPEN first)
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("ONGOING"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)

	// CREATED -> SETTLED should fail
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("SETTLED"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestUpdateResult(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// Advance to ONGOING
	stub.MockInvoke("tx-s1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("PREDICTION_OPEN"),
	})
	stub.MockInvoke("tx-s2", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("TICKET_OPEN"),
	})
	stub.MockInvoke("tx-s3", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("ONGOING"),
	})

	// Update result
	resp := stub.MockInvoke("tx-r1", [][]byte{
		[]byte("EventContract:UpdateResult"),
		[]byte("evt001"),
		[]byte("TeamA wins"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify result is stored and status is SETTLED
	resp = stub.MockInvoke("tx-q1", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	var evt Event
	json.Unmarshal(resp.Payload, &evt)
	assert.Equal(t, "TeamA wins", evt.Result)
	assert.Equal(t, StatusSettled, evt.Status)
}

func TestUpdateResultWrongStatus(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// Try to update result on CREATED event (should fail, must be ONGOING)
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:UpdateResult"),
		[]byte("evt001"),
		[]byte("TeamA wins"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestListEvents(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	// Create multiple events
	createTestEvent(t, stub, "evt001")
	createTestEvent(t, stub, "evt002")
	createTestEvent(t, stub, "evt003")

	// Advance evt002 to PREDICTION_OPEN
	stub.MockInvoke("tx-s1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt002"),
		[]byte("PREDICTION_OPEN"),
	})

	// List all events (empty filter returns all)
	resp := stub.MockInvoke("tx-list1", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte(""),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var events []Event
	err := json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 3, len(events))

	// List by status
	resp = stub.MockInvoke("tx-list2", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte("CREATED"),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 2, len(events))

	// List by status PREDICTION_OPEN
	resp = stub.MockInvoke("tx-list3", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte("PREDICTION_OPEN"),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, "evt002", events[0].ID)

	// List by type
	resp = stub.MockInvoke("tx-list4", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte(""),
		[]byte("basketball"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 3, len(events))
}
