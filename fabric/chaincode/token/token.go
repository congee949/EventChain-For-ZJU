package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func main() {
	chaincode, err := contractapi.NewChaincode(&TokenContract{})
	if err != nil {
		log.Fatalf("Error creating chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Fatalf("Error starting chaincode: %v", err)
	}
}
