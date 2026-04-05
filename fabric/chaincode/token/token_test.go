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
