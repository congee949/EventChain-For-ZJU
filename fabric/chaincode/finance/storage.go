package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func publicKey(ctx contractapi.TransactionContextInterface, objectType string, attrs ...string) (string, error) {
	return ctx.GetStub().CreateCompositeKey(objectType, attrs)
}

func getPublic(ctx contractapi.TransactionContextInterface, key string, target any) (bool, error) {
	raw, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, fmt.Errorf("read public state %q: %w", key, err)
	}
	if raw == nil {
		return false, nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false, fmt.Errorf("decode public state %q: %w", key, err)
	}
	return true, nil
}

func putPublic(ctx contractapi.TransactionContextInterface, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode public state %q: %w", key, err)
	}
	if err := ctx.GetStub().PutState(key, raw); err != nil {
		return fmt.Errorf("write public state %q: %w", key, err)
	}
	return nil
}

func getPrivate(ctx contractapi.TransactionContextInterface, key string, target any) (bool, error) {
	raw, err := ctx.GetStub().GetPrivateData(financeCollection, key)
	if err != nil {
		return false, fmt.Errorf("read private state %q: %w", key, err)
	}
	if raw == nil {
		return false, nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false, fmt.Errorf("decode private state %q: %w", key, err)
	}
	return true, nil
}

func putPrivate(ctx contractapi.TransactionContextInterface, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode private state %q: %w", key, err)
	}
	if err := ctx.GetStub().PutPrivateData(financeCollection, key, raw); err != nil {
		return fmt.Errorf("write private state %q: %w", key, err)
	}
	return nil
}

func getConfig(ctx contractapi.TransactionContextInterface) (*SystemConfig, error) {
	key, err := publicKey(ctx, "meta", "config")
	if err != nil {
		return nil, err
	}
	var config SystemConfig
	found, err := getPublic(ctx, key, &config)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("finance V2 is not initialized")
	}
	return &config, nil
}

func walletKey(ctx contractapi.TransactionContextInterface, accountID string) (string, error) {
	return publicKey(ctx, "wallet", accountID)
}

func loadWallet(ctx contractapi.TransactionContextInterface, accountID, now string) (*Wallet, error) {
	key, err := walletKey(ctx, accountID)
	if err != nil {
		return nil, err
	}
	wallet := &Wallet{
		DocType:     "wallet",
		AccountID:   accountID,
		Categories:  map[string]*CategoryWallet{},
		RiskGroupID: accountID,
		UpdatedAt:   now,
	}
	found, err := getPrivate(ctx, key, wallet)
	if err != nil {
		return nil, err
	}
	if found && wallet.Categories == nil {
		wallet.Categories = map[string]*CategoryWallet{}
	}
	return wallet, nil
}

func saveWallet(ctx contractapi.TransactionContextInterface, wallet *Wallet) error {
	key, err := walletKey(ctx, wallet.AccountID)
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, wallet)
}

func categoryWallet(wallet *Wallet, categoryID string) *CategoryWallet {
	value := wallet.Categories[categoryID]
	if value == nil {
		value = &CategoryWallet{}
		wallet.Categories[categoryID] = value
	}
	return value
}

func receiptKey(ctx contractapi.TransactionContextInterface, refID string) (string, error) {
	return publicKey(ctx, "receipt", refID)
}
