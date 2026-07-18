package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type FinanceContract struct {
	contractapi.Contract
}

type lotSlice struct {
	Amount         int64
	ExpiresAt      string
	TransferableAt string
	Source         string
	BackingVaultID string
}

type transferCounter struct {
	DocType    string          `json:"docType"`
	AccountID  string          `json:"accountId"`
	Day        string          `json:"day"`
	Amount     int64           `json:"amount"`
	Recipients map[string]bool `json:"recipients"`
}

func (fc *FinanceContract) InitializeV2(ctx contractapi.TransactionContextInterface) (*SystemConfig, error) {
	if err := requirePlatformMSP(ctx); err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	key, err := publicKey(ctx, "meta", "config")
	if err != nil {
		return nil, err
	}
	var existing SystemConfig
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		if existing.SchemaVersion != 2 {
			return nil, fmt.Errorf("unsupported existing schema version %d", existing.SchemaVersion)
		}
		return &existing, nil
	}
	config := defaultSystemConfig(timestamp(now))
	if err := putPublic(ctx, key, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (fc *FinanceContract) GetConfig(ctx contractapi.TransactionContextInterface) (*SystemConfig, error) {
	return getConfig(ctx)
}

func (fc *FinanceContract) CreateCategory(ctx contractapi.TransactionContextInterface, categoryID, name string) (*Category, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" {
		return nil, fmt.Errorf("PlatformMSP is required")
	}
	if err := requireRole(identity, "admin"); err != nil {
		return nil, err
	}
	if err := validateID("category ID", categoryID); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, fmt.Errorf("category name must contain 1-64 characters")
	}
	if _, err := getConfig(ctx); err != nil {
		return nil, err
	}
	key, err := publicKey(ctx, "category", categoryID)
	if err != nil {
		return nil, err
	}
	var existing Category
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("category already exists")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	category := &Category{DocType: "category", ID: categoryID, Name: name, Active: true, CreatedAt: timestamp(now)}
	if err := putPublic(ctx, key, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (fc *FinanceContract) GetCategory(ctx contractapi.TransactionContextInterface, categoryID string) (*Category, error) {
	if err := validateID("category ID", categoryID); err != nil {
		return nil, err
	}
	key, err := publicKey(ctx, "category", categoryID)
	if err != nil {
		return nil, err
	}
	var category Category
	found, err := getPublic(ctx, key, &category)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("category not found")
	}
	return &category, nil
}

func (fc *FinanceContract) ListCategories(ctx contractapi.TransactionContextInterface) ([]*Category, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("category", []string{})
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer iterator.Close()
	result := []*Category{}
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("iterate categories: %w", err)
		}
		var category Category
		if err := json.Unmarshal(item.Value, &category); err != nil {
			return nil, fmt.Errorf("decode category: %w", err)
		}
		result = append(result, &category)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (fc *FinanceContract) CreditDemoA(ctx contractapi.TransactionContextInterface, accountID, amountText, reason, refID string) (*OperationReceipt, error) {
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !config.DemoMode {
		return nil, fmt.Errorf("demo faucet is disabled")
	}
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || identity.Role != "admin" {
		return nil, fmt.Errorf("platform admin is required")
	}
	if err := validateID("account ID", accountID); err != nil {
		return nil, err
	}
	amount, err := parseAmount(amountText)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 256 {
		return nil, fmt.Errorf("reason is required and must not exceed 256 bytes")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if receipt, found, err := loadReceipt(ctx, refID); err != nil {
		return nil, err
	} else if found {
		return receipt, nil
	}
	wallet, err := loadWallet(ctx, accountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	wallet.AAvailable, err = safeAdd(wallet.AAvailable, amount)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedAt = timestamp(now)
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	supply, err := loadSupply(ctx)
	if err != nil {
		return nil, err
	}
	supply.IssuedA, err = safeAdd(supply.IssuedA, amount)
	if err != nil {
		return nil, err
	}
	if err := saveSupply(ctx, supply); err != nil {
		return nil, err
	}
	return saveReceipt(ctx, refID, "CREDIT_DEMO_A", accountID, amount, 0, now)
}

func (fc *FinanceContract) GetMyWallet(ctx contractapi.TransactionContextInterface) (*Wallet, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	return loadWallet(ctx, identity.AccountID, timestamp(now))
}

func (fc *FinanceContract) GetWalletForAccount(ctx contractapi.TransactionContextInterface, accountID string) (*Wallet, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.AccountID != accountID && !(identity.MSPID == "PlatformMSP" && (identity.Role == "admin" || identity.Role == "operator")) {
		return nil, fmt.Errorf("wallet access denied")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	return loadWallet(ctx, accountID, timestamp(now))
}

func (fc *FinanceContract) ConvertAToPaidB(ctx contractapi.TransactionContextInterface, categoryID, amountText, refID string) (*OperationReceipt, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.Role != "student" && identity.Role != "organizer" {
		return nil, fmt.Errorf("only students and organizers can convert A")
	}
	category, err := fc.GetCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if !category.Active {
		return nil, fmt.Errorf("category is inactive")
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := parseAmount(amountText)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if receipt, found, err := loadReceipt(ctx, refID); err != nil {
		return nil, err
	} else if found {
		return receipt, nil
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if wallet.AAvailable < amount {
		return nil, fmt.Errorf("insufficient A balance")
	}
	wallet.AAvailable -= amount
	categoryBalance := categoryWallet(wallet, categoryID)
	categoryBalance.Paid.Available, err = safeAdd(categoryBalance.Paid.Available, amount)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedAt = timestamp(now)
	vault, err := loadVault(ctx, categoryID, timestamp(now))
	if err != nil {
		return nil, err
	}
	vault.LockedA, err = safeAdd(vault.LockedA, amount)
	if err != nil {
		return nil, err
	}
	vault.UpdatedAt = timestamp(now)
	lot := &AssetLot{
		DocType: "assetLot", LotID: ctx.GetStub().GetTxID(), AccountID: identity.AccountID,
		CategoryID: categoryID, Bucket: "PAID", Available: amount, IssuedAt: timestamp(now),
		ExpiresAt:      timestamp(now.Add(time.Duration(config.PaidExpiryDays) * 24 * time.Hour)),
		TransferableAt: timestamp(now), Source: "A_CONVERSION", BackingVaultID: categoryID,
	}
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := saveVault(ctx, vault); err != nil {
		return nil, err
	}
	if err := putLot(ctx, lot); err != nil {
		return nil, err
	}
	return saveReceipt(ctx, refID, "A_TO_B_PAID", identity.AccountID, amount, 0, now)
}

func (fc *FinanceContract) RedeemPaidBToA(ctx contractapi.TransactionContextInterface, categoryID, amountText, refID string) (*OperationReceipt, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	amount, err := parseAmount(amountText)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if receipt, found, err := loadReceipt(ctx, refID); err != nil {
		return nil, err
	} else if found {
		return receipt, nil
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	categoryBalance := categoryWallet(wallet, categoryID)
	if categoryBalance.Paid.Available < amount {
		return nil, fmt.Errorf("insufficient B_paid balance")
	}
	if _, err := consumeLots(ctx, identity.AccountID, categoryID, "PAID", amount, now, false); err != nil {
		return nil, err
	}
	fee, err := mulBPS(amount, config.RedemptionFeeBPS, true)
	if err != nil {
		return nil, err
	}
	if fee >= amount {
		return nil, fmt.Errorf("amount is too small after fee")
	}
	net := amount - fee
	vault, err := loadVault(ctx, categoryID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if vault.LockedA < amount {
		return nil, fmt.Errorf("backing invariant violated: category vault is undercollateralized")
	}
	vault.LockedA -= amount
	vault.UpdatedAt = timestamp(now)
	categoryBalance.Paid.Available -= amount
	wallet.AAvailable, err = safeAdd(wallet.AAvailable, net)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedAt = timestamp(now)
	riskWallet, err := loadWallet(ctx, "risk_reserve", timestamp(now))
	if err != nil {
		return nil, err
	}
	riskWallet.AAvailable, err = safeAdd(riskWallet.AAvailable, fee)
	if err != nil {
		return nil, err
	}
	riskWallet.UpdatedAt = timestamp(now)
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := saveWallet(ctx, riskWallet); err != nil {
		return nil, err
	}
	if err := saveVault(ctx, vault); err != nil {
		return nil, err
	}
	return saveReceipt(ctx, refID, "B_PAID_TO_A", identity.AccountID, amount, fee, now)
}

func (fc *FinanceContract) TransferPaidB(ctx contractapi.TransactionContextInterface, categoryID, toAccountID, amountText, refID string) (*OperationReceipt, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.Role != "student" {
		return nil, fmt.Errorf("only student accounts can transfer B_paid")
	}
	if err := validateID("recipient account ID", toAccountID); err != nil {
		return nil, err
	}
	if identity.AccountID == toAccountID {
		return nil, fmt.Errorf("self transfer is not allowed")
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := parseAmount(amountText)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if receipt, found, err := loadReceipt(ctx, refID); err != nil {
		return nil, err
	} else if found {
		return receipt, nil
	}
	counter, counterKey, err := loadTransferCounter(ctx, identity.AccountID, now)
	if err != nil {
		return nil, err
	}
	newTotal, err := safeAdd(counter.Amount, amount)
	if err != nil || newTotal > config.TransferDailyLimit {
		return nil, fmt.Errorf("daily transfer amount limit exceeded")
	}
	if !counter.Recipients[toAccountID] && len(counter.Recipients) >= config.TransferDailyRecipients {
		return nil, fmt.Errorf("daily recipient limit exceeded")
	}
	fromWallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	fromCategory := categoryWallet(fromWallet, categoryID)
	if fromCategory.Paid.Available < amount {
		return nil, fmt.Errorf("insufficient B_paid balance")
	}
	segments, err := consumeLots(ctx, identity.AccountID, categoryID, "PAID", amount, now, true)
	if err != nil {
		return nil, err
	}
	toWallet, err := loadWallet(ctx, toAccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	toCategory := categoryWallet(toWallet, categoryID)
	fromCategory.Paid.Available -= amount
	toCategory.Paid.Available, err = safeAdd(toCategory.Paid.Available, amount)
	if err != nil {
		return nil, err
	}
	fromWallet.UpdatedAt = timestamp(now)
	toWallet.UpdatedAt = timestamp(now)
	for index, segment := range segments {
		lot := &AssetLot{
			DocType: "assetLot", LotID: fmt.Sprintf("%s-%02d", ctx.GetStub().GetTxID(), index),
			AccountID: toAccountID, CategoryID: categoryID, Bucket: "PAID", Available: segment.Amount,
			IssuedAt: timestamp(now), ExpiresAt: segment.ExpiresAt,
			TransferableAt: timestamp(now.Add(time.Duration(config.TransferCooldownHours) * time.Hour)),
			Source:         "TRANSFER", BackingVaultID: segment.BackingVaultID,
		}
		if err := putLot(ctx, lot); err != nil {
			return nil, err
		}
	}
	counter.Amount = newTotal
	counter.Recipients[toAccountID] = true
	if err := saveWallet(ctx, fromWallet); err != nil {
		return nil, err
	}
	if err := saveWallet(ctx, toWallet); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, counterKey, counter); err != nil {
		return nil, err
	}
	return saveReceipt(ctx, refID, "TRANSFER_B_PAID", identity.AccountID, amount, 0, now)
}

func (fc *FinanceContract) CreditDemoBonus(ctx contractapi.TransactionContextInterface, accountID, categoryID, amountText, refID string) (*OperationReceipt, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || identity.Role != "admin" {
		return nil, fmt.Errorf("platform admin is required")
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !config.DemoMode {
		return nil, fmt.Errorf("demo bonus credit is disabled")
	}
	if err := validateID("account ID", accountID); err != nil {
		return nil, err
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	amount, err := parseAmount(amountText)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if receipt, found, err := loadReceipt(ctx, refID); err != nil {
		return nil, err
	} else if found {
		return receipt, nil
	}
	if err := creditB(ctx, accountID, categoryID, "BONUS", amount, "DEMO_SEED", now, config); err != nil {
		return nil, err
	}
	return saveReceipt(ctx, refID, "CREDIT_DEMO_BONUS", accountID, amount, 0, now)
}

type checkInCounter struct {
	DocType       string `json:"docType"`
	AccountID     string `json:"accountId"`
	Day           string `json:"day"`
	Month         string `json:"month"`
	DailyAmount   int64  `json:"dailyAmount"`
	MonthlyAmount int64  `json:"monthlyAmount"`
}

func (fc *FinanceContract) GrantCheckInBonus(ctx contractapi.TransactionContextInterface, accountID, categoryID, eventID, refID string) (*OperationReceipt, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "PlatformMSP" || (identity.Role != "operator" && identity.Role != "admin") {
		return nil, fmt.Errorf("platform operator is required")
	}
	if err := validateID("account ID", accountID); err != nil {
		return nil, err
	}
	if err := validateID("event ID", eventID); err != nil {
		return nil, err
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if receipt, found, err := loadReceipt(ctx, refID); err != nil {
		return nil, err
	} else if found {
		return receipt, nil
	}
	checkInKey, err := publicKey(ctx, "checkIn", eventID, accountID)
	if err != nil {
		return nil, err
	}
	var marker struct {
		TxID string `json:"txId"`
	}
	if found, err := getPrivate(ctx, checkInKey, &marker); err != nil {
		return nil, err
	} else if found {
		return nil, fmt.Errorf("account has already checked in to this event")
	}
	local := now.In(time.FixedZone("UTC+8", 8*60*60))
	day := local.Format("2006-01-02")
	month := local.Format("2006-01")
	monthKey, err := publicKey(ctx, "checkInCounter", accountID, month)
	if err != nil {
		return nil, err
	}
	counter := &checkInCounter{DocType: "checkInCounter", AccountID: accountID, Day: day, Month: month}
	found, err := getPrivate(ctx, monthKey, counter)
	if err != nil {
		return nil, err
	}
	if found && counter.Day != day {
		counter.Day = day
		counter.DailyAmount = 0
	}
	reward := config.CheckInDailyBonus
	if counter.DailyAmount > config.CheckInDailyBonus-reward {
		return nil, fmt.Errorf("daily check-in bonus limit exceeded")
	}
	if counter.MonthlyAmount > config.CheckInMonthlyBonus-reward {
		return nil, fmt.Errorf("monthly check-in bonus limit exceeded")
	}
	counter.DailyAmount += reward
	counter.MonthlyAmount += reward
	if err := creditB(ctx, accountID, categoryID, "BONUS", reward, "CHECK_IN", now, config); err != nil {
		return nil, err
	}
	marker.TxID = ctx.GetStub().GetTxID()
	if err := putPrivate(ctx, monthKey, counter); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, checkInKey, &marker); err != nil {
		return nil, err
	}
	return saveReceipt(ctx, refID, "CHECK_IN_BONUS", accountID, reward, 0, now)
}

func (fc *FinanceContract) SweepExpiredLots(ctx contractapi.TransactionContextInterface, accountID, categoryID, limitText string) (*ExpirySweepResult, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.AccountID != accountID && !(identity.MSPID == "PlatformMSP" && (identity.Role == "operator" || identity.Role == "admin")) {
		return nil, fmt.Errorf("expiry sweep access denied")
	}
	limit, err := strconv.Atoi(limitText)
	if err != nil || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be 1-100")
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	wallet, err := loadWallet(ctx, accountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	result := &ExpirySweepResult{AccountID: accountID, CategoryID: categoryID}
	for _, bucket := range []string{"PAID", "BONUS"} {
		index, indexKey, err := loadLotIndex(ctx, accountID, categoryID, bucket)
		if err != nil {
			return nil, err
		}
		remainingIDs := make([]string, 0, len(index.LotIDs))
		for position, lotID := range index.LotIDs {
			if result.ProcessedLots >= limit {
				remainingIDs = append(remainingIDs, index.LotIDs[position:]...)
				break
			}
			key, err := publicKey(ctx, "lot", accountID, categoryID, bucket, lotID)
			if err != nil {
				return nil, err
			}
			var lot AssetLot
			found, err := getPrivate(ctx, key, &lot)
			if err != nil {
				return nil, err
			}
			if !found {
				continue
			}
			expiresAt, err := parseTimestamp(lot.ExpiresAt)
			if err != nil {
				return nil, err
			}
			if expiresAt.After(now) || lot.Locked > 0 || lot.Available <= 0 {
				remainingIDs = append(remainingIDs, lotID)
				continue
			}
			category := categoryWallet(wallet, categoryID)
			if bucket == "PAID" {
				if category.Paid.Available < lot.Available {
					return nil, fmt.Errorf("paid balance invariant violated")
				}
				vault, err := loadVault(ctx, categoryID, timestamp(now))
				if err != nil {
					return nil, err
				}
				if vault.LockedA < lot.Available {
					return nil, fmt.Errorf("backing invariant violated")
				}
				category.Paid.Available -= lot.Available
				wallet.AAvailable, err = safeAdd(wallet.AAvailable, lot.Available)
				if err != nil {
					return nil, err
				}
				vault.LockedA -= lot.Available
				vault.UpdatedAt = timestamp(now)
				result.RefundedA += lot.Available
				if err := saveVault(ctx, vault); err != nil {
					return nil, err
				}
			} else {
				if category.Bonus.Available < lot.Available {
					return nil, fmt.Errorf("bonus balance invariant violated")
				}
				category.Bonus.Available -= lot.Available
				result.BurnedBonus += lot.Available
			}
			if err := ctx.GetStub().PutPrivateData(financeCollection, key, nil); err != nil {
				return nil, err
			}
			result.ProcessedLots++
		}
		index.LotIDs = remainingIDs
		if err := putPrivate(ctx, indexKey, index); err != nil {
			return nil, err
		}
		if result.ProcessedLots >= limit {
			break
		}
	}
	wallet.UpdatedAt = timestamp(now)
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	return result, nil
}

func loadReceipt(ctx contractapi.TransactionContextInterface, refID string) (*OperationReceipt, bool, error) {
	if err := validateID("reference ID", refID); err != nil {
		return nil, false, err
	}
	key, err := receiptKey(ctx, refID)
	if err != nil {
		return nil, false, err
	}
	var receipt OperationReceipt
	found, err := getPrivate(ctx, key, &receipt)
	return &receipt, found, err
}

func saveReceipt(ctx contractapi.TransactionContextInterface, refID, operation, accountID string, amount, fee int64, now time.Time) (*OperationReceipt, error) {
	key, err := receiptKey(ctx, refID)
	if err != nil {
		return nil, err
	}
	receipt := &OperationReceipt{
		DocType: "operationReceipt", RefID: refID, Operation: operation, AccountID: accountID,
		Amount: amount, Fee: fee, TxID: ctx.GetStub().GetTxID(), CreatedAt: timestamp(now),
	}
	if err := putPrivate(ctx, key, receipt); err != nil {
		return nil, err
	}
	return receipt, nil
}

func loadVault(ctx contractapi.TransactionContextInterface, categoryID, now string) (*Vault, error) {
	key, err := publicKey(ctx, "vault", categoryID)
	if err != nil {
		return nil, err
	}
	vault := &Vault{DocType: "vault", VaultID: categoryID, CategoryID: categoryID, UpdatedAt: now}
	_, err = getPrivate(ctx, key, vault)
	return vault, err
}

func saveVault(ctx contractapi.TransactionContextInterface, vault *Vault) error {
	key, err := publicKey(ctx, "vault", vault.CategoryID)
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, vault)
}

func loadSupply(ctx contractapi.TransactionContextInterface) (*Supply, error) {
	key, err := publicKey(ctx, "meta", "supply")
	if err != nil {
		return nil, err
	}
	supply := &Supply{DocType: "supply"}
	_, err = getPrivate(ctx, key, supply)
	return supply, err
}

func saveSupply(ctx contractapi.TransactionContextInterface, supply *Supply) error {
	key, err := publicKey(ctx, "meta", "supply")
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, supply)
}

func lotKey(ctx contractapi.TransactionContextInterface, lot *AssetLot) (string, error) {
	return publicKey(ctx, "lot", lot.AccountID, lot.CategoryID, lot.Bucket, lot.LotID)
}

func putLot(ctx contractapi.TransactionContextInterface, lot *AssetLot) error {
	key, err := lotKey(ctx, lot)
	if err != nil {
		return err
	}
	if err := putPrivate(ctx, key, lot); err != nil {
		return err
	}
	index, indexKey, err := loadLotIndex(ctx, lot.AccountID, lot.CategoryID, lot.Bucket)
	if err != nil {
		return err
	}
	for _, existing := range index.LotIDs {
		if existing == lot.LotID {
			return nil
		}
	}
	index.LotIDs = append(index.LotIDs, lot.LotID)
	return putPrivate(ctx, indexKey, index)
}

func consumeLots(ctx contractapi.TransactionContextInterface, accountID, categoryID, bucket string, amount int64, now time.Time, enforceTransferable bool) ([]lotSlice, error) {
	index, indexKey, err := loadLotIndex(ctx, accountID, categoryID, bucket)
	if err != nil {
		return nil, err
	}
	type keyedLot struct {
		key string
		lot AssetLot
	}
	lots := []keyedLot{}
	for _, lotID := range index.LotIDs {
		key, err := publicKey(ctx, "lot", accountID, categoryID, bucket, lotID)
		if err != nil {
			return nil, err
		}
		var lot AssetLot
		found, err := getPrivate(ctx, key, &lot)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		expiresAt, err := parseTimestamp(lot.ExpiresAt)
		if err != nil {
			return nil, err
		}
		if !expiresAt.After(now) || lot.Available <= 0 {
			continue
		}
		if enforceTransferable {
			transferableAt, err := parseTimestamp(lot.TransferableAt)
			if err != nil {
				return nil, err
			}
			if transferableAt.After(now) {
				continue
			}
		}
		lots = append(lots, keyedLot{key: key, lot: lot})
	}
	sort.Slice(lots, func(i, j int) bool {
		if lots[i].lot.ExpiresAt == lots[j].lot.ExpiresAt {
			return lots[i].lot.LotID < lots[j].lot.LotID
		}
		return lots[i].lot.ExpiresAt < lots[j].lot.ExpiresAt
	})
	remaining := amount
	segments := []lotSlice{}
	consumed := map[string]bool{}
	for _, item := range lots {
		if remaining == 0 {
			break
		}
		take := item.lot.Available
		if take > remaining {
			take = remaining
		}
		item.lot.Available -= take
		remaining -= take
		segments = append(segments, lotSlice{Amount: take, ExpiresAt: item.lot.ExpiresAt, TransferableAt: item.lot.TransferableAt, Source: item.lot.Source, BackingVaultID: item.lot.BackingVaultID})
		if item.lot.Available == 0 && item.lot.Locked == 0 {
			if err := ctx.GetStub().PutPrivateData(financeCollection, item.key, nil); err != nil {
				return nil, fmt.Errorf("delete consumed lot: %w", err)
			}
			consumed[item.lot.LotID] = true
		} else if err := putPrivate(ctx, item.key, &item.lot); err != nil {
			return nil, err
		}
	}
	if remaining != 0 {
		return nil, fmt.Errorf("insufficient non-expired %s lots", bucket)
	}
	if len(consumed) > 0 {
		remainingIDs := make([]string, 0, len(index.LotIDs)-len(consumed))
		for _, lotID := range index.LotIDs {
			if !consumed[lotID] {
				remainingIDs = append(remainingIDs, lotID)
			}
		}
		index.LotIDs = remainingIDs
		if err := putPrivate(ctx, indexKey, index); err != nil {
			return nil, err
		}
	}
	return segments, nil
}

func loadLotIndex(ctx contractapi.TransactionContextInterface, accountID, categoryID, bucket string) (*LotIndex, string, error) {
	key, err := publicKey(ctx, "lotIndex", accountID, categoryID, bucket)
	if err != nil {
		return nil, "", err
	}
	index := &LotIndex{DocType: "lotIndex", AccountID: accountID, CategoryID: categoryID, Bucket: bucket, LotIDs: []string{}}
	found, err := getPrivate(ctx, key, index)
	if err != nil {
		return nil, "", err
	}
	if found && index.LotIDs == nil {
		index.LotIDs = []string{}
	}
	return index, key, nil
}

func loadTransferCounter(ctx contractapi.TransactionContextInterface, accountID string, now time.Time) (*transferCounter, string, error) {
	beijingDay := now.In(time.FixedZone("UTC+8", 8*60*60)).Format("2006-01-02")
	key, err := publicKey(ctx, "transferCounter", accountID, beijingDay)
	if err != nil {
		return nil, "", err
	}
	counter := &transferCounter{DocType: "transferCounter", AccountID: accountID, Day: beijingDay, Recipients: map[string]bool{}}
	found, err := getPrivate(ctx, key, counter)
	if err != nil {
		return nil, "", err
	}
	if found && counter.Recipients == nil {
		counter.Recipients = map[string]bool{}
	}
	return counter, key, nil
}
