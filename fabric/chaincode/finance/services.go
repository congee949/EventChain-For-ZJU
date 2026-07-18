package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (fc *FinanceContract) CreateServiceOffer(ctx contractapi.TransactionContextInterface, offerID, categoryID, offerType, title, priceText, cancellationBPSText, guaranteeBudgetText, bonusBudgetText, inventoryText string) (*ServiceOffer, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "OrganizerMSP" || identity.Role != "organizer" {
		return nil, fmt.Errorf("organizer identity is required")
	}
	if err := validateID("offer ID", offerID); err != nil {
		return nil, err
	}
	if _, err := fc.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	offerType = strings.ToUpper(strings.TrimSpace(offerType))
	if offerType != "EQUIPMENT" && offerType != "VENUE" && offerType != "EVENT_PASS" {
		return nil, fmt.Errorf("offer type must be EQUIPMENT, VENUE, or EVENT_PASS")
	}
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 100 {
		return nil, fmt.Errorf("offer title must contain 1-100 characters")
	}
	price, err := parseAmount(priceText)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	cancellationBPS := config.DefaultCancellationBPS
	if strings.TrimSpace(cancellationBPSText) != "" {
		cancellationBPS, err = strconv.ParseInt(cancellationBPSText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cancellation multiplier")
		}
	}
	if cancellationBPS < bpsScale || cancellationBPS > config.MaximumCancellationBPS {
		return nil, fmt.Errorf("cancellation multiplier must be 10000-%d bps", config.MaximumCancellationBPS)
	}
	guaranteeBudget, err := parseOptionalAmount(guaranteeBudgetText)
	if err != nil {
		return nil, err
	}
	bonusBudget, err := parseOptionalAmount(bonusBudgetText)
	if err != nil {
		return nil, err
	}
	inventory, err := strconv.ParseInt(inventoryText, 10, 64)
	if err != nil || inventory < 1 || inventory > 1_000_000 {
		return nil, fmt.Errorf("inventory must be 1-1000000")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	key, err := publicKey(ctx, "serviceOffer", offerID)
	if err != nil {
		return nil, err
	}
	var existing ServiceOffer
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("service offer already exists")
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if wallet.AAvailable < guaranteeBudget {
		return nil, fmt.Errorf("insufficient A guarantee budget")
	}
	category := categoryWallet(wallet, categoryID)
	if category.Bonus.Available < bonusBudget {
		return nil, fmt.Errorf("insufficient B_bonus compensation budget")
	}
	if bonusBudget > 0 {
		if _, err := consumeLots(ctx, identity.AccountID, categoryID, "BONUS", bonusBudget, now, false); err != nil {
			return nil, err
		}
	}
	wallet.AAvailable -= guaranteeBudget
	wallet.AReserved += guaranteeBudget
	category.Bonus.Available -= bonusBudget
	category.Bonus.Locked += bonusBudget
	wallet.UpdatedAt = timestamp(now)
	offer := &ServiceOffer{
		DocType: "serviceOffer", OfferID: offerID, OrganizerAccountID: identity.AccountID,
		CategoryID: categoryID, OfferType: offerType, Title: title, PricePaidB: price,
		CancellationBPS: cancellationBPS, GuaranteeBudgetA: guaranteeBudget,
		GuaranteeAvailableA: guaranteeBudget, BonusBudget: bonusBudget, BonusAvailable: bonusBudget,
		Inventory: inventory, Status: "ACTIVE", CreatedAt: timestamp(now),
	}
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := putPublic(ctx, key, offer); err != nil {
		return nil, err
	}
	return offer, nil
}

func (fc *FinanceContract) GetServiceOffer(ctx contractapi.TransactionContextInterface, offerID string) (*ServiceOffer, error) {
	return loadServiceOffer(ctx, offerID)
}

func (fc *FinanceContract) ListServiceOffers(ctx contractapi.TransactionContextInterface) ([]*ServiceOffer, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("serviceOffer", []string{})
	if err != nil {
		return nil, fmt.Errorf("list service offers: %w", err)
	}
	defer iterator.Close()
	offers := make([]*ServiceOffer, 0)
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("iterate service offers: %w", err)
		}
		var offer ServiceOffer
		if err := decodeJSONStrict(string(item.Value), &offer); err != nil {
			return nil, fmt.Errorf("decode service offer: %w", err)
		}
		offers = append(offers, &offer)
	}
	return offers, nil
}

func (fc *FinanceContract) GetServiceOrder(ctx contractapi.TransactionContextInterface, orderID string) (*ServiceOrder, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	order, found, err := loadServiceOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("service order not found")
	}
	allowed := identity.AccountID == order.AccountID || identity.AccountID == order.OrganizerAccountID ||
		(identity.MSPID == "PlatformMSP" && (identity.Role == "operator" || identity.Role == "admin"))
	if !allowed {
		return nil, fmt.Errorf("service order access denied")
	}
	return order, nil
}

func (fc *FinanceContract) CreateServiceOrder(ctx contractapi.TransactionContextInterface) (*ServiceOrder, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if identity.MSPID != "StudentMSP" || identity.Role != "student" {
		return nil, fmt.Errorf("student identity is required")
	}
	transient, err := ctx.GetStub().GetTransient()
	if err != nil {
		return nil, err
	}
	raw := transient["order"]
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, fmt.Errorf("transient order is required")
	}
	var request CreateServiceOrderRequest
	if err := decodeJSONStrict(string(raw), &request); err != nil {
		return nil, err
	}
	if err := validateID("order reference", request.RefID); err != nil {
		return nil, err
	}
	if existing, found, err := loadServiceOrder(ctx, request.RefID); err != nil {
		return nil, err
	} else if found {
		if existing.AccountID != identity.AccountID || existing.OfferID != request.OfferID {
			return nil, fmt.Errorf("idempotency key was used for another service order")
		}
		return existing, nil
	}
	offer, err := loadServiceOffer(ctx, request.OfferID)
	if err != nil {
		return nil, err
	}
	if offer.Status != "ACTIVE" || offer.Inventory < 1 {
		return nil, fmt.Errorf("service offer is unavailable")
	}
	scheduledAt, err := parseTimestamp(request.ScheduledAt)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if !scheduledAt.After(now) {
		return nil, fmt.Errorf("scheduled time must be in the future")
	}
	compensationBPS := offer.CancellationBPS - bpsScale
	guaranteeRequired, err := mulBPS(offer.PricePaidB, compensationBPS, true)
	if err != nil {
		return nil, err
	}
	if offer.GuaranteeAvailableA < guaranteeRequired {
		return nil, fmt.Errorf("offer guarantee reserve is insufficient")
	}
	wallet, err := loadWallet(ctx, identity.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	category := categoryWallet(wallet, offer.CategoryID)
	if category.Paid.Available < offer.PricePaidB {
		return nil, fmt.Errorf("insufficient B_paid balance")
	}
	segments, err := consumeLots(ctx, identity.AccountID, offer.CategoryID, "PAID", offer.PricePaidB, now, false)
	if err != nil {
		return nil, err
	}
	paymentLots := make([]OrderPaymentLot, 0, len(segments))
	for _, segment := range segments {
		paymentLots = append(paymentLots, OrderPaymentLot{Amount: segment.Amount, ExpiresAt: segment.ExpiresAt})
	}
	category.Paid.Available -= offer.PricePaidB
	wallet.UpdatedAt = timestamp(now)
	offer.Inventory--
	offer.GuaranteeAvailableA -= guaranteeRequired
	order := &ServiceOrder{
		DocType: "serviceOrder", OrderID: request.RefID, OfferID: offer.OfferID,
		AccountID: identity.AccountID, OrganizerAccountID: offer.OrganizerAccountID,
		CategoryID: offer.CategoryID, PrincipalPaidB: offer.PricePaidB,
		GuaranteeReservedA: guaranteeRequired, PaymentLots: paymentLots, Status: "PAID",
		ScheduledAt: timestamp(scheduledAt), CreatedAt: timestamp(now), UpdatedAt: timestamp(now),
	}
	if err := saveWallet(ctx, wallet); err != nil {
		return nil, err
	}
	if err := saveServiceOffer(ctx, offer); err != nil {
		return nil, err
	}
	if err := saveServiceOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (fc *FinanceContract) FulfillServiceOrder(ctx contractapi.TransactionContextInterface, orderID string) (*ServiceOrder, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	order, found, err := loadServiceOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("service order not found")
	}
	if identity.MSPID != "OrganizerMSP" || identity.Role != "organizer" || identity.AccountID != order.OrganizerAccountID {
		return nil, fmt.Errorf("order organizer is required")
	}
	if order.Status == "FULFILLED" {
		return order, nil
	}
	if order.Status != "PAID" && order.Status != "DELAYED" && order.Status != "RESCHEDULED" {
		return nil, fmt.Errorf("order cannot be fulfilled from status %s", order.Status)
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := creditB(ctx, order.OrganizerAccountID, order.CategoryID, "PAID", order.PrincipalPaidB, "SERVICE_REVENUE", now, config); err != nil {
		return nil, err
	}
	offer, err := loadServiceOffer(ctx, order.OfferID)
	if err != nil {
		return nil, err
	}
	offer.GuaranteeAvailableA += order.GuaranteeReservedA
	order.GuaranteeReservedA = 0
	order.Status = "FULFILLED"
	order.UpdatedAt = timestamp(now)
	if err := saveServiceOffer(ctx, offer); err != nil {
		return nil, err
	}
	if err := saveServiceOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (fc *FinanceContract) CancelServiceOrder(ctx contractapi.TransactionContextInterface, orderID, reason string) (*ServiceOrder, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	order, found, err := loadServiceOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("service order not found")
	}
	if order.Status == "CANCELLED" || order.Status == "REFUNDED" {
		return order, nil
	}
	if order.Status == "FULFILLED" {
		return nil, fmt.Errorf("fulfilled order cannot be cancelled")
	}
	reason = strings.ToUpper(strings.TrimSpace(reason))
	if reason != "USER_REQUEST" && reason != "ORGANIZER_FAULT" && reason != "FORCE_MAJEURE" {
		return nil, fmt.Errorf("unsupported cancellation reason")
	}
	if identity.AccountID == order.AccountID {
		if reason != "USER_REQUEST" {
			return nil, fmt.Errorf("student can only request USER_REQUEST cancellation")
		}
	} else if identity.AccountID == order.OrganizerAccountID && identity.MSPID == "OrganizerMSP" {
		if reason == "USER_REQUEST" {
			return nil, fmt.Errorf("organizer cannot assert user cancellation")
		}
	} else if identity.MSPID != "PlatformMSP" || (identity.Role != "operator" && identity.Role != "admin") {
		return nil, fmt.Errorf("cancellation access denied")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	offer, err := loadServiceOffer(ctx, order.OfferID)
	if err != nil {
		return nil, err
	}
	offer.Inventory++
	user, err := loadWallet(ctx, order.AccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	organizer, err := loadWallet(ctx, order.OrganizerAccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	if reason == "USER_REQUEST" {
		category := categoryWallet(user, order.CategoryID)
		category.Paid.Available += order.PrincipalPaidB
		for index, payment := range order.PaymentLots {
			originalExpiry, _ := parseTimestamp(payment.ExpiresAt)
			pausedDuration := now.Sub(mustParse(order.CreatedAt))
			expiry := originalExpiry.Add(pausedDuration)
			minimum := now.Add(time.Duration(config.LockGraceDays) * 24 * time.Hour)
			if expiry.Before(minimum) {
				expiry = minimum
			}
			lot := &AssetLot{
				DocType: "assetLot", LotID: fmt.Sprintf("%s-refund-%02d", ctx.GetStub().GetTxID(), index),
				AccountID: order.AccountID, CategoryID: order.CategoryID, Bucket: "PAID",
				Available: payment.Amount, IssuedAt: timestamp(now), ExpiresAt: timestamp(expiry),
				TransferableAt: timestamp(now), Source: "SERVICE_REFUND", BackingVaultID: order.CategoryID,
			}
			if err := putLot(ctx, lot); err != nil {
				return nil, err
			}
		}
		offer.GuaranteeAvailableA += order.GuaranteeReservedA
		order.GuaranteeReservedA = 0
	} else {
		vault, err := loadVault(ctx, order.CategoryID, timestamp(now))
		if err != nil {
			return nil, err
		}
		if vault.LockedA < order.PrincipalPaidB {
			return nil, fmt.Errorf("backing invariant violated")
		}
		vault.LockedA -= order.PrincipalPaidB
		vault.UpdatedAt = timestamp(now)
		refund := order.PrincipalPaidB
		if reason == "ORGANIZER_FAULT" {
			if organizer.AReserved < order.GuaranteeReservedA {
				return nil, fmt.Errorf("organizer guarantee invariant violated")
			}
			organizer.AReserved -= order.GuaranteeReservedA
			refund += order.GuaranteeReservedA
			order.GuaranteeReservedA = 0
		} else {
			offer.GuaranteeAvailableA += order.GuaranteeReservedA
			order.GuaranteeReservedA = 0
		}
		user.AAvailable, err = safeAdd(user.AAvailable, refund)
		if err != nil {
			return nil, err
		}
		order.RefundedA = refund
		if err := saveVault(ctx, vault); err != nil {
			return nil, err
		}
	}
	user.UpdatedAt = timestamp(now)
	organizer.UpdatedAt = timestamp(now)
	order.Status = "REFUNDED"
	order.CancellationReason = reason
	order.UpdatedAt = timestamp(now)
	if err := saveWallet(ctx, user); err != nil {
		return nil, err
	}
	if err := saveWallet(ctx, organizer); err != nil {
		return nil, err
	}
	if err := saveServiceOffer(ctx, offer); err != nil {
		return nil, err
	}
	if err := saveServiceOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (fc *FinanceContract) CompensateServiceDelay(ctx contractapi.TransactionContextInterface, orderID string) (*ServiceOrder, error) {
	identity, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	order, found, err := loadServiceOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("service order not found")
	}
	if identity.AccountID != order.OrganizerAccountID && !(identity.MSPID == "PlatformMSP" && (identity.Role == "operator" || identity.Role == "admin")) {
		return nil, fmt.Errorf("delay compensation access denied")
	}
	if order.Status != "PAID" && order.Status != "DELAYED" {
		return nil, fmt.Errorf("order is not eligible for delay compensation")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	scheduledAt, _ := parseTimestamp(order.ScheduledAt)
	if !now.After(scheduledAt) {
		return nil, fmt.Errorf("order is not delayed")
	}
	days := int64(now.Sub(scheduledAt) / (24 * time.Hour))
	rate := int64(0)
	switch {
	case days >= 15:
		rate = 1500
	case days >= 8:
		rate = 1000
	case days >= 3:
		rate = 500
	default:
		return nil, fmt.Errorf("delay is below the three-day compensation threshold")
	}
	target, err := mulBPS(order.PrincipalPaidB, rate, false)
	if err != nil {
		return nil, err
	}
	additional := target - order.DelayCompensation
	if additional <= 0 {
		return order, nil
	}
	offer, err := loadServiceOffer(ctx, order.OfferID)
	if err != nil {
		return nil, err
	}
	if offer.BonusAvailable < additional {
		return nil, fmt.Errorf("offer bonus compensation budget is insufficient")
	}
	organizer, err := loadWallet(ctx, order.OrganizerAccountID, timestamp(now))
	if err != nil {
		return nil, err
	}
	category := categoryWallet(organizer, order.CategoryID)
	if category.Bonus.Locked < additional {
		return nil, fmt.Errorf("organizer bonus reserve invariant violated")
	}
	category.Bonus.Locked -= additional
	organizer.UpdatedAt = timestamp(now)
	config, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := creditB(ctx, order.AccountID, order.CategoryID, "BONUS", additional, "SERVICE_DELAY", now, config); err != nil {
		return nil, err
	}
	offer.BonusAvailable -= additional
	order.DelayCompensation += additional
	order.Status = "DELAYED"
	order.UpdatedAt = timestamp(now)
	if err := saveWallet(ctx, organizer); err != nil {
		return nil, err
	}
	if err := saveServiceOffer(ctx, offer); err != nil {
		return nil, err
	}
	if err := saveServiceOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func loadServiceOffer(ctx contractapi.TransactionContextInterface, offerID string) (*ServiceOffer, error) {
	if err := validateID("offer ID", offerID); err != nil {
		return nil, err
	}
	key, err := publicKey(ctx, "serviceOffer", offerID)
	if err != nil {
		return nil, err
	}
	var offer ServiceOffer
	found, err := getPublic(ctx, key, &offer)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("service offer not found")
	}
	return &offer, nil
}

func saveServiceOffer(ctx contractapi.TransactionContextInterface, offer *ServiceOffer) error {
	key, err := publicKey(ctx, "serviceOffer", offer.OfferID)
	if err != nil {
		return err
	}
	return putPublic(ctx, key, offer)
}

func loadServiceOrder(ctx contractapi.TransactionContextInterface, orderID string) (*ServiceOrder, bool, error) {
	if err := validateID("order ID", orderID); err != nil {
		return nil, false, err
	}
	key, err := publicKey(ctx, "serviceOrder", orderID)
	if err != nil {
		return nil, false, err
	}
	var order ServiceOrder
	found, err := getPrivate(ctx, key, &order)
	return &order, found, err
}

func saveServiceOrder(ctx contractapi.TransactionContextInterface, order *ServiceOrder) error {
	key, err := publicKey(ctx, "serviceOrder", order.OrderID)
	if err != nil {
		return err
	}
	return putPrivate(ctx, key, order)
}

func parseOptionalAmount(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || value == "0.0" {
		return 0, nil
	}
	return parseAmount(value)
}

func mustParse(value string) time.Time {
	parsed, _ := parseTimestamp(value)
	return parsed
}
