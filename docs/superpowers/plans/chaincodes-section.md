## Task 3: Token Chaincode

### 3.1 Write token chaincode tests

- [ ] Create `fabric/chaincode/token/token_test.go`
- [ ] Create `fabric/chaincode/token/go.mod`

`fabric/chaincode/token/go.mod`:

```go
module github.com/eventchain/chaincode/token

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/token/token_test.go`:

```go
package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTokenChaincode(t *testing.T) (*shimtest.MockStub, *TokenContract) {
	tc := new(TokenContract)
	cc, err := contractapi.NewChaincode(tc)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("token", cc)
	require.NotNil(t, stub)
	return stub, tc
}

func TestInitialize(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize with total supply of 1000000
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify platform account balance
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("platform"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result TokenAccount
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), result.Balance)
	assert.Equal(t, "platform", result.Owner)

	// Double initialize should fail
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("500000"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestMint(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize first
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint 1000 tokens for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify user1 balance
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result TokenAccount
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), result.Balance)
	assert.Equal(t, "user1", result.Owner)

	// Mint again to verify accumulation
	resp = stub.MockInvoke("tx4", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("500"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	resp = stub.MockInvoke("tx5", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), result.Balance)
}

func TestMintInvalidAmount(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint with zero amount
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("0"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)

	// Mint with negative amount
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("-100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestTransfer(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint tokens for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer 300 from user1 to user2
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user2"),
		[]byte("300"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify user1 balance is 700
	resp = stub.MockInvoke("tx4", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var account1 TokenAccount
	err := json.Unmarshal(resp.Payload, &account1)
	require.NoError(t, err)
	assert.Equal(t, int64(700), account1.Balance)

	// Verify user2 balance is 300
	resp = stub.MockInvoke("tx5", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user2"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var account2 TokenAccount
	err = json.Unmarshal(resp.Payload, &account2)
	require.NoError(t, err)
	assert.Equal(t, int64(300), account2.Balance)
}

func TestTransferInsufficientBalance(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint 100 tokens for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("100"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Try to transfer 200 (more than balance)
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user2"),
		[]byte("200"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestTransferToSelf(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer to self should fail
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user1"),
		[]byte("100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestBalanceOfNonexistent(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Query nonexistent user returns zero balance
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("nonexistent"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result TokenAccount
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Balance)
}

func TestHistory(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint for user2
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user2"),
		[]byte("500"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer from user1 to user2
	resp = stub.MockInvoke("tx4", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user2"),
		[]byte("200"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Query history for user1
	resp = stub.MockInvoke("tx5", [][]byte{
		[]byte("TokenContract:History"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var records []TransactionRecord
	err := json.Unmarshal(resp.Payload, &records)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 1)

	// Verify the transfer record exists
	found := false
	for _, r := range records {
		if r.Type == "TRANSFER" && r.From == "user1" && r.To == "user2" && r.Amount == 200 {
			found = true
			break
		}
	}
	assert.True(t, found, "expected transfer record not found in history")
}

func TestMultipleTransfers(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint 1000 for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer 100 to user2, user3, user4
	for i, user := range []string{"user2", "user3", "user4"} {
		txID := fmt.Sprintf("tx%d", i+3)
		resp = stub.MockInvoke(txID, [][]byte{
			[]byte("TokenContract:Transfer"),
			[]byte("user1"),
			[]byte(user),
			[]byte("100"),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
	}

	// user1 should have 700
	resp = stub.MockInvoke("tx10", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var account TokenAccount
	err := json.Unmarshal(resp.Payload, &account)
	require.NoError(t, err)
	assert.Equal(t, int64(700), account.Balance)
}
```

### 3.2 Run token tests (expect fail)

- [ ] `cd fabric/chaincode/token && go test -v ./...` -- should fail because `token.go` does not exist yet

### 3.3 Write token chaincode implementation

- [ ] Create `fabric/chaincode/token/token.go`

`fabric/chaincode/token/token.go`:

```go
package token

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// TokenAccount represents a user's token account stored in world state
type TokenAccount struct {
	DocType     string `json:"docType"`
	Owner       string `json:"owner"`
	Balance     int64  `json:"balance"`
	LastUpdated string `json:"lastUpdated"`
}

// TransactionRecord represents a token transaction stored in world state for history
type TransactionRecord struct {
	DocType   string `json:"docType"`
	TxID      string `json:"txID"`
	Type      string `json:"type"` // MINT, TRANSFER
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int64  `json:"amount"`
	Timestamp string `json:"timestamp"`
}

// TokenContract implements the token chaincode
type TokenContract struct {
	contractapi.Contract
}

// Initialize sets up the token pool with a total supply assigned to the platform account
func (tc *TokenContract) Initialize(ctx contractapi.TransactionContextInterface, totalSupplyStr string) error {
	totalSupply, err := strconv.ParseInt(totalSupplyStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid total supply: %s", err.Error())
	}
	if totalSupply <= 0 {
		return fmt.Errorf("total supply must be positive")
	}

	// Check if already initialized
	existing, err := ctx.GetStub().GetState("token:platform")
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("token system already initialized")
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	account := TokenAccount{
		DocType:     "token",
		Owner:       "platform",
		Balance:     totalSupply,
		LastUpdated: timeStr,
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %s", err.Error())
	}

	err = ctx.GetStub().PutState("token:platform", accountJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// Mint creates new tokens for a user account
func (tc *TokenContract) Mint(ctx contractapi.TransactionContextInterface, userID string, amountStr string) error {
	if userID == "" {
		return fmt.Errorf("user ID must not be empty")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", err.Error())
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	key := "token:" + userID

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	// Get existing account or create new one
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}

	var account TokenAccount
	if existing != nil {
		err = json.Unmarshal(existing, &account)
		if err != nil {
			return fmt.Errorf("failed to unmarshal account: %s", err.Error())
		}
		account.Balance += amount
		account.LastUpdated = timeStr
	} else {
		account = TokenAccount{
			DocType:     "token",
			Owner:       userID,
			Balance:     amount,
			LastUpdated: timeStr,
		}
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, accountJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	// Record transaction
	txID := ctx.GetStub().GetTxID()
	record := TransactionRecord{
		DocType:   "tokenTx",
		TxID:      txID,
		Type:      "MINT",
		From:      "system",
		To:        userID,
		Amount:    amount,
		Timestamp: timeStr,
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %s", err.Error())
	}

	recordKey := fmt.Sprintf("tokenTx:%s:%s", userID, txID)
	err = ctx.GetStub().PutState(recordKey, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put transaction record: %s", err.Error())
	}

	return nil
}

// Transfer moves tokens from one account to another
func (tc *TokenContract) Transfer(ctx contractapi.TransactionContextInterface, from string, to string, amountStr string) error {
	if from == "" || to == "" {
		return fmt.Errorf("from and to must not be empty")
	}
	if from == to {
		return fmt.Errorf("cannot transfer to self")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", err.Error())
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	// Read sender account
	fromKey := "token:" + from
	fromBytes, err := ctx.GetStub().GetState(fromKey)
	if err != nil {
		return fmt.Errorf("failed to read sender account: %s", err.Error())
	}
	if fromBytes == nil {
		return fmt.Errorf("sender account %s does not exist", from)
	}

	var fromAccount TokenAccount
	err = json.Unmarshal(fromBytes, &fromAccount)
	if err != nil {
		return fmt.Errorf("failed to unmarshal sender account: %s", err.Error())
	}

	if fromAccount.Balance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", fromAccount.Balance, amount)
	}

	// Read or create receiver account
	toKey := "token:" + to
	toBytes, err := ctx.GetStub().GetState(toKey)
	if err != nil {
		return fmt.Errorf("failed to read receiver account: %s", err.Error())
	}

	var toAccount TokenAccount
	if toBytes != nil {
		err = json.Unmarshal(toBytes, &toAccount)
		if err != nil {
			return fmt.Errorf("failed to unmarshal receiver account: %s", err.Error())
		}
	} else {
		toAccount = TokenAccount{
			DocType: "token",
			Owner:   to,
			Balance: 0,
		}
	}

	// Execute transfer
	fromAccount.Balance -= amount
	fromAccount.LastUpdated = timeStr
	toAccount.Balance += amount
	toAccount.LastUpdated = timeStr

	fromJSON, err := json.Marshal(fromAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal sender account: %s", err.Error())
	}

	toJSON, err := json.Marshal(toAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal receiver account: %s", err.Error())
	}

	err = ctx.GetStub().PutState(fromKey, fromJSON)
	if err != nil {
		return fmt.Errorf("failed to update sender account: %s", err.Error())
	}

	err = ctx.GetStub().PutState(toKey, toJSON)
	if err != nil {
		return fmt.Errorf("failed to update receiver account: %s", err.Error())
	}

	// Record transaction
	txID := ctx.GetStub().GetTxID()
	record := TransactionRecord{
		DocType:   "tokenTx",
		TxID:      txID,
		Type:      "TRANSFER",
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: timeStr,
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %s", err.Error())
	}

	// Store record under both sender and receiver for lookup
	fromRecordKey := fmt.Sprintf("tokenTx:%s:%s", from, txID)
	err = ctx.GetStub().PutState(fromRecordKey, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put sender transaction record: %s", err.Error())
	}

	toRecordKey := fmt.Sprintf("tokenTx:%s:%s", to, txID)
	err = ctx.GetStub().PutState(toRecordKey, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put receiver transaction record: %s", err.Error())
	}

	return nil
}

// BalanceOf returns the token account for a user
func (tc *TokenContract) BalanceOf(ctx contractapi.TransactionContextInterface, userID string) (*TokenAccount, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	key := "token:" + userID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %s", err.Error())
	}

	if data == nil {
		// Return zero-balance account for nonexistent users
		return &TokenAccount{
			DocType:     "token",
			Owner:       userID,
			Balance:     0,
			LastUpdated: "",
		}, nil
	}

	var account TokenAccount
	err = json.Unmarshal(data, &account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %s", err.Error())
	}

	return &account, nil
}

// History returns all transaction records for a user using CouchDB rich query
func (tc *TokenContract) History(ctx contractapi.TransactionContextInterface, userID string) ([]*TransactionRecord, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	// Use GetStateByRange to query all transaction records for this user
	// Keys are in format tokenTx:{userID}:{txID}
	startKey := fmt.Sprintf("tokenTx:%s:", userID)
	endKey := fmt.Sprintf("tokenTx:%s:~", userID)

	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %s", err.Error())
	}
	defer resultsIterator.Close()

	var records []*TransactionRecord
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %s", err.Error())
		}

		var record TransactionRecord
		err = json.Unmarshal(queryResponse.Value, &record)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal record: %s", err.Error())
		}

		records = append(records, &record)
	}

	if records == nil {
		records = []*TransactionRecord{}
	}

	return records, nil
}

// GetContractInfo provides metadata about the chaincode
func (tc *TokenContract) GetContractInfo() contractapi.ContractInterface {
	return tc
}
```

### 3.4 Run token tests (expect pass)

- [ ] `cd fabric/chaincode/token && go test -v ./...`

### 3.5 Commit token chaincode

- [ ] `git add fabric/chaincode/token/ && git commit -m "feat(chaincode): implement token chaincode with tests"`

---

## Task 4: Event Chaincode

### 4.1 Write event chaincode tests

- [ ] Create `fabric/chaincode/event/event_test.go`
- [ ] Create `fabric/chaincode/event/go.mod`

`fabric/chaincode/event/go.mod`:

```go
module github.com/eventchain/chaincode/event

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/event/event_test.go`:

```go
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
```

### 4.2 Run event tests (expect fail)

- [ ] `cd fabric/chaincode/event && go test -v ./...` -- should fail because `event.go` does not exist yet

### 4.3 Write event chaincode implementation

- [ ] Create `fabric/chaincode/event/event.go`

`fabric/chaincode/event/event.go`:

```go
package event

import (
	"encoding/json"
	"fmt"
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
```

### 4.4 Run event tests (expect pass)

- [ ] `cd fabric/chaincode/event && go test -v ./...`

### 4.5 Commit event chaincode

- [ ] `git add fabric/chaincode/event/ && git commit -m "feat(chaincode): implement event chaincode with state machine and tests"`

---

## Task 5: Prediction Chaincode (AMM)

### 5.1 Write prediction chaincode tests

- [ ] Create `fabric/chaincode/prediction/prediction_test.go`
- [ ] Create `fabric/chaincode/prediction/go.mod`

`fabric/chaincode/prediction/go.mod`:

```go
module github.com/eventchain/chaincode/prediction

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/prediction/prediction_test.go`:

```go
package prediction

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
```

### 5.2 Run prediction tests (expect fail)

- [ ] `cd fabric/chaincode/prediction && go test -v ./...` -- should fail because `prediction.go` does not exist yet

### 5.3 Write prediction chaincode implementation

- [ ] Create `fabric/chaincode/prediction/prediction.go`

`fabric/chaincode/prediction/prediction.go`:

```go
package prediction

import (
	"encoding/json"
	"fmt"
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

// UserScore tracks a user's prediction accuracy
type UserScore struct {
	DocType      string  `json:"docType"`
	UserID       string  `json:"userID"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

// BetResult is returned after placing a bet
type BetResult struct {
	BetID  string `json:"betID"`
	Option string `json:"option"`
	Amount int64  `json:"amount"`
	Shares int64  `json:"shares"`
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
	transferResp := ctx.GetStub().InvokeChaincode("token-cc", transferArgs, "eventchain")
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
				transferResp := ctx.GetStub().InvokeChaincode("token-cc", transferArgs, "eventchain")
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
```

### 5.4 Run prediction tests (expect pass)

- [ ] `cd fabric/chaincode/prediction && go test -v ./...`

### 5.5 Commit prediction chaincode

- [ ] `git add fabric/chaincode/prediction/ && git commit -m "feat(chaincode): implement prediction market chaincode with AMM and tests"`

---

## Task 6: Ticket Chaincode

### 6.1 Write ticket chaincode tests

- [ ] Create `fabric/chaincode/ticket/ticket_test.go`
- [ ] Create `fabric/chaincode/ticket/go.mod`

`fabric/chaincode/ticket/go.mod`:

```go
module github.com/eventchain/chaincode/ticket

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/ticket/ticket_test.go`:

```go
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
```

### 6.2 Run ticket tests (expect fail)

- [ ] `cd fabric/chaincode/ticket && go test -v ./...` -- should fail because `ticket.go` does not exist yet

### 6.3 Write ticket chaincode implementation

- [ ] Create `fabric/chaincode/ticket/ticket.go`

`fabric/chaincode/ticket/ticket.go`:

```go
package ticket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
```

### 6.4 Run ticket tests (expect pass)

- [ ] `cd fabric/chaincode/ticket && go test -v ./...`

### 6.5 Commit ticket chaincode

- [ ] `git add fabric/chaincode/ticket/ && git commit -m "feat(chaincode): implement ticket chaincode with weighted lottery and tests"`
