package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const badgeCollection = "badge_private"

type BadgeSeries struct {
	DocType                  string `json:"docType"`
	SchemaVersion            int    `json:"schemaVersion"`
	SeriesID                 string `json:"seriesId"`
	SeasonID                 string `json:"seasonId"`
	CategoryID               string `json:"categoryId"`
	Title                    string `json:"title"`
	Description              string `json:"description"`
	AssetURI                 string `json:"assetUri"`
	AssetSHA256              string `json:"assetSha256"`
	MetadataSHA256           string `json:"metadataSha256"`
	EligibilityPolicyHash    string `json:"eligibilityPolicyHash"`
	IssuanceMode             string `json:"issuanceMode"`
	MaxSupply                int    `json:"maxSupply"`
	IssuedCount              int    `json:"issuedCount"`
	Status                   string `json:"status"`
	EligibilityOpenAt        string `json:"eligibilityOpenAt"`
	EligibilityCloseAt       string `json:"eligibilityCloseAt"`
	ClaimOpenAt              string `json:"claimOpenAt"`
	ClaimCloseAt             string `json:"claimCloseAt"`
	IssuerOrganization       string `json:"issuerOrganization"`
	SupplyRationale          string `json:"supplyRationale"`
	CreatedAt                string `json:"createdAt"`
	ActivatedAt              string `json:"activatedAt"`
	ClosedAt                 string `json:"closedAt"`
	FinalizedAt              string `json:"finalizedAt"`
	LastPauseReasonHash      string `json:"lastPauseReasonHash"`
	LastAdministrativeUpdate string `json:"lastAdministrativeUpdate"`
}

type createBadgeSeriesRequest struct {
	SeriesID              string `json:"seriesId"`
	SeasonID              string `json:"seasonId"`
	CategoryID            string `json:"categoryId"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	AssetURI              string `json:"assetUri"`
	AssetSHA256           string `json:"assetSha256"`
	MetadataSHA256        string `json:"metadataSha256"`
	EligibilityPolicyHash string `json:"eligibilityPolicyHash"`
	IssuanceMode          string `json:"issuanceMode"`
	MaxSupply             int    `json:"maxSupply"`
	EligibilityOpenAt     string `json:"eligibilityOpenAt"`
	EligibilityCloseAt    string `json:"eligibilityCloseAt"`
	ClaimOpenAt           string `json:"claimOpenAt"`
	ClaimCloseAt          string `json:"claimCloseAt"`
	SupplyRationale       string `json:"supplyRationale"`
}

type BadgeActivityLink struct {
	DocType          string `json:"docType"`
	SeriesID         string `json:"seriesId"`
	ActivityID       string `json:"activityId"`
	EligibilityType  string `json:"eligibilityType"`
	EligibilityQuota int    `json:"eligibilityQuota"`
	CreatedAt        string `json:"createdAt"`
}

type BadgeEligibility struct {
	DocType            string `json:"docType"`
	SeriesID           string `json:"seriesId"`
	AccountID          string `json:"accountId"`
	SourceType         string `json:"sourceType"`
	SourceActivityID   string `json:"sourceActivityId"`
	SourceTicketID     string `json:"sourceTicketId"`
	EvidenceCommitment string `json:"evidenceCommitment"`
	EligibleAt         string `json:"eligibleAt"`
	Status             string `json:"status"`
}

type BadgeEligibilityResult struct {
	SeriesID   string `json:"seriesId"`
	Status     string `json:"status"`
	EligibleAt string `json:"eligibleAt"`
}

type BadgePublicInstance struct {
	DocType           string `json:"docType"`
	InstanceID        string `json:"instanceId"`
	SeriesID          string `json:"seriesId"`
	SerialNumber      int    `json:"serialNumber"`
	Status            string `json:"status"`
	IssuanceBatchDate string `json:"issuanceBatchDate"`
	MetadataSHA256    string `json:"metadataSha256"`
	StateVersion      int    `json:"stateVersion"`
}

type BadgeAward struct {
	DocType          string `json:"docType"`
	InstanceID       string `json:"instanceId"`
	SeriesID         string `json:"seriesId"`
	AccountID        string `json:"accountId"`
	SerialNumber     int    `json:"serialNumber"`
	SourceActivityID string `json:"sourceActivityId"`
	Status           string `json:"status"`
	IssuedAt         string `json:"issuedAt"`
}

type BadgeAwardIndex struct {
	DocType   string   `json:"docType"`
	AccountID string   `json:"accountId"`
	SeriesIDs []string `json:"seriesIds"`
}

type BadgeClaimReceipt struct {
	DocType      string `json:"docType"`
	SeriesID     string `json:"seriesId"`
	AccountID    string `json:"accountId"`
	RefID        string `json:"refId"`
	RequestHash  string `json:"requestHash"`
	InstanceID   string `json:"instanceId"`
	SerialNumber int    `json:"serialNumber"`
	TxID         string `json:"txId"`
	CreatedAt    string `json:"createdAt"`
}

type BadgeAwardView struct {
	Series      *BadgeSeries         `json:"series"`
	Instance    *BadgePublicInstance `json:"instance"`
	Award       *BadgeAward          `json:"award"`
	Eligibility *BadgeEligibility    `json:"eligibility"`
}

type BadgeAuditRecord struct {
	DocType    string `json:"docType"`
	SeriesID   string `json:"seriesId"`
	Action     string `json:"action"`
	ActorMSP   string `json:"actorMsp"`
	ReasonHash string `json:"reasonHash"`
	TxID       string `json:"txId"`
	CreatedAt  string `json:"createdAt"`
}

// BadgeReadiness is a capability probe response. The probe deliberately
// touches badge_private so callers can distinguish an installed collection
// from an empty badge index.
type BadgeReadiness struct {
	DocType    string `json:"docType"`
	Collection string `json:"collection"`
	Status     string `json:"status"`
}

func (ac *ActivityContract) GetBadgeReadiness(ctx contractapi.TransactionContextInterface) (*BadgeReadiness, error) {
	if _, err := requireBadgeAdmin(ctx); err != nil {
		return nil, err
	}
	probeKey, err := stateKey(ctx, "badgeReadinessProbe")
	if err != nil {
		return nil, err
	}
	if _, err := ctx.GetStub().GetPrivateData(badgeCollection, probeKey); err != nil {
		return nil, err
	}
	return &BadgeReadiness{DocType: "badgeReadiness", Collection: badgeCollection, Status: "READY"}, nil
}

func (ac *ActivityContract) CreateBadgeSeries(ctx contractapi.TransactionContextInterface, seriesJSON string) (*BadgeSeries, error) {
	who, err := requireBadgeAdmin(ctx)
	if err != nil {
		return nil, err
	}
	var request createBadgeSeriesRequest
	decoder := json.NewDecoder(strings.NewReader(seriesJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return nil, fmt.Errorf("invalid badge series JSON: %w", err)
	}
	if err := validateBadgeSeriesRequest(&request); err != nil {
		return nil, err
	}
	key, err := stateKey(ctx, "badgeSeries", request.SeriesID)
	if err != nil {
		return nil, err
	}
	var existing BadgeSeries
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("badge series already exists")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	series := &BadgeSeries{
		DocType: "badgeSeries", SchemaVersion: 1, SeriesID: request.SeriesID,
		SeasonID: request.SeasonID, CategoryID: request.CategoryID, Title: strings.TrimSpace(request.Title),
		Description: strings.TrimSpace(request.Description), AssetURI: strings.TrimSpace(request.AssetURI),
		AssetSHA256: request.AssetSHA256, MetadataSHA256: request.MetadataSHA256,
		EligibilityPolicyHash: request.EligibilityPolicyHash, IssuanceMode: "CHECK_IN_GUARANTEED",
		MaxSupply: request.MaxSupply, Status: "DRAFT", EligibilityOpenAt: request.EligibilityOpenAt,
		EligibilityCloseAt: request.EligibilityCloseAt, ClaimOpenAt: request.ClaimOpenAt,
		ClaimCloseAt: request.ClaimCloseAt, IssuerOrganization: who.MSPID,
		SupplyRationale: strings.TrimSpace(request.SupplyRationale), CreatedAt: timestamp(now),
		LastAdministrativeUpdate: timestamp(now),
	}
	if err := putPublic(ctx, key, series); err != nil {
		return nil, err
	}
	if err := writeBadgeAudit(ctx, who, series.SeriesID, "CREATE", "", now); err != nil {
		return nil, err
	}
	return series, nil
}

func (ac *ActivityContract) LinkBadgeActivity(ctx contractapi.TransactionContextInterface, seriesID, activityID, eligibilityQuotaText string) (*BadgeActivityLink, error) {
	who, err := requireBadgeAdmin(ctx)
	if err != nil {
		return nil, err
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Status != "DRAFT" {
		return nil, fmt.Errorf("badge activity can only be linked while series is DRAFT")
	}
	activity, err := loadActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if activity.Status != "DRAFT" {
		return nil, fmt.Errorf("badge activity can only be linked while activity is DRAFT")
	}
	quota, err := strconv.Atoi(eligibilityQuotaText)
	if err != nil || quota != activity.Capacity {
		return nil, fmt.Errorf("eligibility quota must equal activity capacity")
	}
	key, err := stateKey(ctx, "badgeActivityLink", activityID, seriesID)
	if err != nil {
		return nil, err
	}
	var existing BadgeActivityLink
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("badge activity link already exists")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	link := &BadgeActivityLink{DocType: "badgeActivityLink", SeriesID: seriesID, ActivityID: activityID, EligibilityType: "VALID_CHECK_IN", EligibilityQuota: quota, CreatedAt: timestamp(now)}
	if err := putPublic(ctx, key, link); err != nil {
		return nil, err
	}
	return link, writeBadgeAudit(ctx, who, seriesID, "LINK_ACTIVITY", "", now)
}

func (ac *ActivityContract) ActivateBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeSeries, error) {
	who, err := requireBadgeAdmin(ctx)
	if err != nil {
		return nil, err
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Status != "DRAFT" {
		return nil, fmt.Errorf("only a DRAFT badge series can be activated")
	}
	links, err := listBadgeLinksForSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("badge series requires at least one linked activity")
	}
	totalQuota := 0
	for _, link := range links {
		activity, err := loadActivity(ctx, link.ActivityID)
		if err != nil {
			return nil, fmt.Errorf("linked activity %s is unavailable: %w", link.ActivityID, err)
		}
		if link.EligibilityQuota != activity.Capacity {
			return nil, fmt.Errorf("linked activity eligibility quota no longer equals capacity")
		}
		if totalQuota > series.MaxSupply-link.EligibilityQuota {
			return nil, fmt.Errorf("badge max supply is below linked activity capacity total")
		}
		totalQuota += link.EligibilityQuota
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	eligibilityOpen, _ := parseTimestamp(series.EligibilityOpenAt)
	eligibilityClose, _ := parseTimestamp(series.EligibilityCloseAt)
	claimOpen, _ := parseTimestamp(series.ClaimOpenAt)
	claimClose, _ := parseTimestamp(series.ClaimCloseAt)
	if !eligibilityOpen.After(now) || !eligibilityClose.After(eligibilityOpen) || eligibilityClose.After(claimClose) || !claimOpen.After(now) || !claimClose.After(claimOpen) {
		return nil, fmt.Errorf("badge series windows must satisfy activation constraints")
	}
	series.Status = "ACTIVE"
	series.ActivatedAt = timestamp(now)
	series.LastAdministrativeUpdate = timestamp(now)
	if err := saveBadgeSeries(ctx, series); err != nil {
		return nil, err
	}
	return series, writeBadgeAudit(ctx, who, seriesID, "ACTIVATE", "", now)
}

func (ac *ActivityContract) PauseBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID, reasonHash string) (*BadgeSeries, error) {
	return ac.transitionBadgeSeries(ctx, seriesID, "ACTIVE", "PAUSED", "PAUSE", reasonHash)
}

func (ac *ActivityContract) ResumeBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeSeries, error) {
	return ac.transitionBadgeSeries(ctx, seriesID, "PAUSED", "ACTIVE", "RESUME", "")
}

func (ac *ActivityContract) CloseBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeSeries, error) {
	who, err := requireBadgeAdmin(ctx)
	if err != nil {
		return nil, err
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Status != "ACTIVE" && series.Status != "PAUSED" {
		return nil, fmt.Errorf("only ACTIVE or PAUSED badge series can be closed")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	series.Status = "CLOSED"
	series.ClosedAt = timestamp(now)
	series.LastAdministrativeUpdate = timestamp(now)
	if err := saveBadgeSeries(ctx, series); err != nil {
		return nil, err
	}
	return series, writeBadgeAudit(ctx, who, seriesID, "CLOSE", "", now)
}

func (ac *ActivityContract) FinalizeBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeSeries, error) {
	who, err := requireBadgeAdmin(ctx)
	if err != nil {
		return nil, err
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Status != "CLOSED" {
		return nil, fmt.Errorf("only a CLOSED badge series can be finalized")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	series.Status = "FINALIZED"
	series.FinalizedAt = timestamp(now)
	series.LastAdministrativeUpdate = timestamp(now)
	if err := saveBadgeSeries(ctx, series); err != nil {
		return nil, err
	}
	return series, writeBadgeAudit(ctx, who, seriesID, "FINALIZE", "", now)
}

func (ac *ActivityContract) GetBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeSeries, error) {
	return loadBadgeSeries(ctx, seriesID)
}

func (ac *ActivityContract) ListBadgeSeries(ctx contractapi.TransactionContextInterface) ([]*BadgeSeries, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("badgeSeries", []string{})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	result := []*BadgeSeries{}
	for iterator.HasNext() {
		row, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var series BadgeSeries
		if err := json.Unmarshal(row.Value, &series); err != nil {
			return nil, err
		}
		result = append(result, &series)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SeriesID < result[j].SeriesID })
	return result, nil
}

func (ac *ActivityContract) ListBadgeActivityLinks(ctx contractapi.TransactionContextInterface, seriesID string) ([]*BadgeActivityLink, error) {
	if _, err := loadBadgeSeries(ctx, seriesID); err != nil {
		return nil, err
	}
	return listBadgeLinksForSeries(ctx, seriesID)
}

func (ac *ActivityContract) GetBadgePublicInstance(ctx contractapi.TransactionContextInterface, seriesID, serialNumberText string) (*BadgePublicInstance, error) {
	serial, err := strconv.Atoi(serialNumberText)
	if err != nil || serial < 1 || serial > 100000 {
		return nil, fmt.Errorf("invalid badge serial number")
	}
	return loadBadgeInstance(ctx, seriesID, serial)
}

func (ac *ActivityContract) GetBadgePublicInstanceByID(ctx contractapi.TransactionContextInterface, instanceID string) (*BadgePublicInstance, error) {
	if !isLowerHash(instanceID) {
		return nil, fmt.Errorf("invalid badge instance ID")
	}
	lookupKey, _ := stateKey(ctx, "badgeInstanceLookup", instanceID)
	var lookup struct {
		SeriesID     string `json:"seriesId"`
		SerialNumber int    `json:"serialNumber"`
	}
	found, err := getPublic(ctx, lookupKey, &lookup)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("badge instance not found")
	}
	return loadBadgeInstance(ctx, lookup.SeriesID, lookup.SerialNumber)
}

func (ac *ActivityContract) ClaimMyBadge(ctx contractapi.TransactionContextInterface, seriesID, refID string) (*BadgeAwardView, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "StudentMSP" || who.Role != "student" {
		return nil, fmt.Errorf("student identity is required")
	}
	if !validBadgeRefID(refID) {
		return nil, fmt.Errorf("invalid badge claim reference")
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	requestHash := digest("ClaimMyBadge:" + seriesID)
	receipt, found, err := loadBadgeClaimReceipt(ctx, seriesID, who.AccountID, refID)
	if err != nil {
		return nil, err
	}
	if found {
		if receipt.RequestHash != requestHash {
			return nil, fmt.Errorf("IDEMPOTENCY_CONFLICT: badge claim reference was reused")
		}
		return loadBadgeAwardView(ctx, series, who.AccountID)
	}
	award, awardFound, err := loadBadgeAward(ctx, seriesID, who.AccountID)
	if err != nil {
		return nil, err
	}
	if awardFound {
		return nil, fmt.Errorf("BADGE_ALREADY_CLAIMED: current account already owns this badge series")
	}
	eligibility, eligible, err := loadBadgeEligibility(ctx, seriesID, who.AccountID)
	if err != nil {
		return nil, err
	}
	if !eligible || eligibility.Status != "CLAIMABLE" {
		return nil, fmt.Errorf("BADGE_INELIGIBLE: current account has no claimable badge eligibility")
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	claimOpen, _ := parseTimestamp(series.ClaimOpenAt)
	claimClose, _ := parseTimestamp(series.ClaimCloseAt)
	if now.After(claimClose) {
		return nil, fmt.Errorf("BADGE_CLAIM_WINDOW_CLOSED: badge claim window has ended")
	}
	if series.Status != "ACTIVE" || now.Before(claimOpen) {
		return nil, fmt.Errorf("BADGE_SERIES_NOT_ACTIVE: badge series is not claimable")
	}
	if series.IssuedCount >= series.MaxSupply {
		return nil, fmt.Errorf("BADGE_SUPPLY_EXHAUSTED: badge fixed supply is exhausted")
	}
	serial := series.IssuedCount + 1
	serialKey := badgeSerialKey(serial)
	instance := &BadgePublicInstance{
		DocType: "badgePublicInstance", InstanceID: digest(seriesID + ":" + serialKey), SeriesID: seriesID,
		SerialNumber: serial, Status: "ACTIVE", IssuanceBatchDate: now.UTC().Format("2006-01-02"),
		MetadataSHA256: series.MetadataSHA256, StateVersion: 1,
	}
	instanceKey, _ := stateKey(ctx, "badgeInstance", seriesID, serialKey)
	if err := putPublic(ctx, instanceKey, instance); err != nil {
		return nil, err
	}
	lookupKey, _ := stateKey(ctx, "badgeInstanceLookup", instance.InstanceID)
	if err := putPublic(ctx, lookupKey, map[string]any{"seriesId": seriesID, "serialNumber": serial}); err != nil {
		return nil, err
	}
	award = &BadgeAward{
		DocType: "badgeAward", InstanceID: instance.InstanceID, SeriesID: seriesID, AccountID: who.AccountID,
		SerialNumber: serial, SourceActivityID: eligibility.SourceActivityID, Status: "ACTIVE", IssuedAt: timestamp(now),
	}
	awardKey, _ := stateKey(ctx, "badgeAward", seriesID, who.AccountID)
	if err := putBadgePrivate(ctx, awardKey, award); err != nil {
		return nil, err
	}
	awardIndex, awardIndexKey, err := loadBadgeAwardIndex(ctx, who.AccountID)
	if err != nil {
		return nil, err
	}
	if !containsString(awardIndex.SeriesIDs, seriesID) {
		awardIndex.SeriesIDs = append(awardIndex.SeriesIDs, seriesID)
		sort.Strings(awardIndex.SeriesIDs)
	}
	if err := putBadgePrivate(ctx, awardIndexKey, awardIndex); err != nil {
		return nil, err
	}
	receipt = &BadgeClaimReceipt{
		DocType: "badgeClaimReceipt", SeriesID: seriesID, AccountID: who.AccountID, RefID: refID,
		RequestHash: requestHash, InstanceID: instance.InstanceID, SerialNumber: serial,
		TxID: ctx.GetStub().GetTxID(), CreatedAt: timestamp(now),
	}
	receiptKey, _ := stateKey(ctx, "badgeClaimReceipt", seriesID, who.AccountID, refID)
	if err := putBadgePrivate(ctx, receiptKey, receipt); err != nil {
		return nil, err
	}
	eligibility.Status = "CLAIMED"
	eligibilityKey, _ := stateKey(ctx, "badgeEligibility", seriesID, who.AccountID)
	if err := putBadgePrivate(ctx, eligibilityKey, eligibility); err != nil {
		return nil, err
	}
	series.IssuedCount = serial
	if err := saveBadgeSeries(ctx, series); err != nil {
		return nil, err
	}
	return &BadgeAwardView{Series: series, Instance: instance, Award: award, Eligibility: eligibility}, nil
}

func (ac *ActivityContract) GetMyBadge(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeAwardView, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	return loadBadgeAwardView(ctx, series, who.AccountID)
}

func (ac *ActivityContract) ListMyBadges(ctx contractapi.TransactionContextInterface) ([]*BadgeAwardView, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	index, _, err := loadBadgeAwardIndex(ctx, who.AccountID)
	if err != nil {
		return nil, err
	}
	result := []*BadgeAwardView{}
	for _, seriesID := range index.SeriesIDs {
		award, found, err := loadBadgeAward(ctx, seriesID, who.AccountID)
		if err != nil {
			return nil, err
		}
		if !found || award.AccountID != who.AccountID {
			return nil, fmt.Errorf("badge award account index is inconsistent")
		}
		series, err := loadBadgeSeries(ctx, award.SeriesID)
		if err != nil {
			return nil, err
		}
		view, err := badgeAwardViewFromAward(ctx, series, award, nil)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Series.SeriesID < result[j].Series.SeriesID })
	return result, nil
}

func (ac *ActivityContract) GetMyBadgeClaimReceipt(ctx contractapi.TransactionContextInterface, seriesID, refID string) (*BadgeClaimReceipt, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	receipt, found, err := loadBadgeClaimReceipt(ctx, seriesID, who.AccountID, refID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("badge claim receipt not found")
	}
	return receipt, nil
}

func (ac *ActivityContract) transitionBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID, from, to, action, reasonHash string) (*BadgeSeries, error) {
	who, err := requireBadgeAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if action == "PAUSE" && !isLowerHash(reasonHash) {
		return nil, fmt.Errorf("pause reason hash must be lowercase SHA-256 hex")
	}
	series, err := loadBadgeSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Status != from {
		return nil, fmt.Errorf("badge series must be %s before %s", from, strings.ToLower(action))
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	series.Status = to
	series.LastPauseReasonHash = reasonHash
	series.LastAdministrativeUpdate = timestamp(now)
	if err := saveBadgeSeries(ctx, series); err != nil {
		return nil, err
	}
	return series, writeBadgeAudit(ctx, who, seriesID, action, reasonHash, now)
}

func requireBadgeAdmin(ctx contractapi.TransactionContextInterface) (*callerIdentity, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "PlatformMSP" || who.Role != "admin" {
		return nil, fmt.Errorf("platform admin identity is required")
	}
	return who, nil
}

func validateBadgeSeriesRequest(request *createBadgeSeriesRequest) error {
	if !validateID(request.SeriesID) || !validateID(request.SeasonID) || !validateID(request.CategoryID) {
		return fmt.Errorf("invalid badge series, season, or category ID")
	}
	if request.IssuanceMode != "" && request.IssuanceMode != "CHECK_IN_GUARANTEED" {
		return fmt.Errorf("only CHECK_IN_GUARANTEED issuance is supported")
	}
	if request.MaxSupply < 1 || request.MaxSupply > 100000 {
		return fmt.Errorf("badge max supply must be 1-100000")
	}
	if textLengthOutside(request.Title, 1, 100) || textLengthOutside(request.Description, 1, 500) || textLengthOutside(request.SupplyRationale, 1, 500) {
		return fmt.Errorf("invalid badge title, description, or supply rationale length")
	}
	if !validBadgeAssetURI(request.AssetURI) {
		return fmt.Errorf("invalid badge asset URI")
	}
	if !isLowerHash(request.AssetSHA256) || !isLowerHash(request.MetadataSHA256) || !isLowerHash(request.EligibilityPolicyHash) {
		return fmt.Errorf("badge hashes must be lowercase SHA-256 hex")
	}
	eligibilityOpen, err := parseTimestamp(request.EligibilityOpenAt)
	if err != nil {
		return err
	}
	eligibilityClose, err := parseTimestamp(request.EligibilityCloseAt)
	if err != nil {
		return err
	}
	claimOpen, err := parseTimestamp(request.ClaimOpenAt)
	if err != nil {
		return err
	}
	claimClose, err := parseTimestamp(request.ClaimCloseAt)
	if err != nil {
		return err
	}
	if !eligibilityClose.After(eligibilityOpen) || eligibilityClose.After(claimClose) || !claimClose.After(claimOpen) {
		return fmt.Errorf("invalid badge eligibility or claim windows")
	}
	return nil
}

func textLengthOutside(value string, minimum, maximum int) bool {
	length := len([]rune(strings.TrimSpace(value)))
	return length < minimum || length > maximum
}

func validBadgeAssetURI(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 1 || len(value) > 512 || strings.ContainsAny(value, "\\?#\r\n\t ") || strings.Contains(value, "..") {
		return false
	}
	return strings.HasPrefix(value, "/badges/") || strings.HasPrefix(value, "ipfs://")
}

func isLowerHash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validBadgeRefID(value string) bool {
	if len(value) < 8 || len(value) > 64 || value != strings.ToLower(value) {
		return false
	}
	for index, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || index > 0 && (r == '_' || r == '-') {
			continue
		}
		return false
	}
	return true
}

func putBadgePrivate(ctx contractapi.TransactionContextInterface, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutPrivateData(badgeCollection, key, raw)
}

func getBadgePrivate(ctx contractapi.TransactionContextInterface, key string, value any) (bool, error) {
	raw, err := ctx.GetStub().GetPrivateData(badgeCollection, key)
	if err != nil || raw == nil {
		return false, err
	}
	return true, json.Unmarshal(raw, value)
}

func loadBadgeSeries(ctx contractapi.TransactionContextInterface, seriesID string) (*BadgeSeries, error) {
	if !validateID(seriesID) {
		return nil, fmt.Errorf("invalid badge series ID")
	}
	key, _ := stateKey(ctx, "badgeSeries", seriesID)
	var series BadgeSeries
	found, err := getPublic(ctx, key, &series)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("badge series not found")
	}
	return &series, nil
}

func saveBadgeSeries(ctx contractapi.TransactionContextInterface, series *BadgeSeries) error {
	key, _ := stateKey(ctx, "badgeSeries", series.SeriesID)
	return putPublic(ctx, key, series)
}

func listBadgeLinksForSeries(ctx contractapi.TransactionContextInterface, seriesID string) ([]*BadgeActivityLink, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("badgeActivityLink", []string{})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	result := []*BadgeActivityLink{}
	for iterator.HasNext() {
		row, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var link BadgeActivityLink
		if err := json.Unmarshal(row.Value, &link); err != nil {
			return nil, err
		}
		if link.SeriesID == seriesID {
			result = append(result, &link)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ActivityID < result[j].ActivityID })
	return result, nil
}

func listBadgeLinksForActivity(ctx contractapi.TransactionContextInterface, activityID string) ([]*BadgeActivityLink, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("badgeActivityLink", []string{activityID})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	result := []*BadgeActivityLink{}
	for iterator.HasNext() {
		row, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var link BadgeActivityLink
		if err := json.Unmarshal(row.Value, &link); err != nil {
			return nil, err
		}
		result = append(result, &link)
	}
	return result, nil
}

func recordBadgeEligibility(ctx contractapi.TransactionContextInterface, activity *Activity, ticket *Ticket, accountID string, now time.Time) ([]BadgeEligibilityResult, error) {
	if activity.Status == "CANCELLED" {
		return []BadgeEligibilityResult{}, nil
	}
	links, err := listBadgeLinksForActivity(ctx, activity.ID)
	if err != nil {
		return nil, err
	}
	results := []BadgeEligibilityResult{}
	todo := []struct {
		key         string
		eligibility BadgeEligibility
	}{}
	for _, link := range links {
		series, err := loadBadgeSeries(ctx, link.SeriesID)
		if err != nil {
			return nil, err
		}
		if series.Status != "ACTIVE" && series.Status != "PAUSED" {
			continue
		}
		openAt, _ := parseTimestamp(series.EligibilityOpenAt)
		closeAt, _ := parseTimestamp(series.EligibilityCloseAt)
		if now.Before(openAt) || now.After(closeAt) {
			continue
		}
		key, _ := stateKey(ctx, "badgeEligibility", series.SeriesID, accountID)
		var eligibility BadgeEligibility
		found, err := getBadgePrivate(ctx, key, &eligibility)
		if err != nil {
			return nil, err
		}
		if !found {
			eligibility = BadgeEligibility{
				DocType: "badgeEligibility", SeriesID: series.SeriesID, AccountID: accountID,
				SourceType: "CHECK_IN_RECEIPT", SourceActivityID: activity.ID, SourceTicketID: ticket.TicketID,
				EvidenceCommitment: digest("badge-eligibility:" + series.SeriesID + ":" + accountID + ":" + ticket.TicketID + ":" + ctx.GetStub().GetTxID()),
				EligibleAt:         timestamp(now), Status: "CLAIMABLE",
			}
			todo = append(todo, struct {
				key         string
				eligibility BadgeEligibility
			}{key: key, eligibility: eligibility})
		}
		results = append(results, BadgeEligibilityResult{SeriesID: series.SeriesID, Status: eligibility.Status, EligibleAt: eligibility.EligibleAt})
	}
	for _, pending := range todo {
		if err := putBadgePrivate(ctx, pending.key, &pending.eligibility); err != nil {
			return nil, err
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].SeriesID < results[j].SeriesID })
	return results, nil
}

func loadBadgeEligibility(ctx contractapi.TransactionContextInterface, seriesID, accountID string) (*BadgeEligibility, bool, error) {
	key, _ := stateKey(ctx, "badgeEligibility", seriesID, accountID)
	var eligibility BadgeEligibility
	found, err := getBadgePrivate(ctx, key, &eligibility)
	return &eligibility, found, err
}

func loadBadgeAward(ctx contractapi.TransactionContextInterface, seriesID, accountID string) (*BadgeAward, bool, error) {
	key, _ := stateKey(ctx, "badgeAward", seriesID, accountID)
	var award BadgeAward
	found, err := getBadgePrivate(ctx, key, &award)
	return &award, found, err
}

func loadBadgeAwardIndex(ctx contractapi.TransactionContextInterface, accountID string) (*BadgeAwardIndex, string, error) {
	key, _ := stateKey(ctx, "badgeAwardIndex", accountID)
	index := &BadgeAwardIndex{DocType: "badgeAwardIndex", AccountID: accountID, SeriesIDs: []string{}}
	found, err := getBadgePrivate(ctx, key, index)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return index, key, nil
	}
	return index, key, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func loadBadgeClaimReceipt(ctx contractapi.TransactionContextInterface, seriesID, accountID, refID string) (*BadgeClaimReceipt, bool, error) {
	key, _ := stateKey(ctx, "badgeClaimReceipt", seriesID, accountID, refID)
	var receipt BadgeClaimReceipt
	found, err := getBadgePrivate(ctx, key, &receipt)
	return &receipt, found, err
}

func badgeSerialKey(serial int) string { return fmt.Sprintf("%06d", serial) }

func loadBadgeInstance(ctx contractapi.TransactionContextInterface, seriesID string, serial int) (*BadgePublicInstance, error) {
	key, _ := stateKey(ctx, "badgeInstance", seriesID, badgeSerialKey(serial))
	var instance BadgePublicInstance
	found, err := getPublic(ctx, key, &instance)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("badge instance not found")
	}
	return &instance, nil
}

func loadBadgeAwardView(ctx contractapi.TransactionContextInterface, series *BadgeSeries, accountID string) (*BadgeAwardView, error) {
	award, found, err := loadBadgeAward(ctx, series.SeriesID, accountID)
	if err != nil {
		return nil, err
	}
	eligibility, eligibilityFound, err := loadBadgeEligibility(ctx, series.SeriesID, accountID)
	if err != nil {
		return nil, err
	}
	if !eligibilityFound {
		eligibility = nil
	}
	if !found {
		return &BadgeAwardView{Series: series, Eligibility: eligibility}, nil
	}
	return badgeAwardViewFromAward(ctx, series, award, eligibility)
}

func badgeAwardViewFromAward(ctx contractapi.TransactionContextInterface, series *BadgeSeries, award *BadgeAward, eligibility *BadgeEligibility) (*BadgeAwardView, error) {
	instance, err := loadBadgeInstance(ctx, series.SeriesID, award.SerialNumber)
	if err != nil {
		return nil, err
	}
	if eligibility == nil {
		eligibility, _, err = loadBadgeEligibility(ctx, series.SeriesID, award.AccountID)
		if err != nil {
			return nil, err
		}
	}
	return &BadgeAwardView{Series: series, Instance: instance, Award: award, Eligibility: eligibility}, nil
}

func writeBadgeAudit(ctx contractapi.TransactionContextInterface, who *callerIdentity, seriesID, action, reasonHash string, now time.Time) error {
	txID := ctx.GetStub().GetTxID()
	key, _ := stateKey(ctx, "badgeAudit", seriesID, txID)
	record := &BadgeAuditRecord{
		DocType: "badgeAudit", SeriesID: seriesID, Action: action,
		ActorMSP: who.MSPID, ReasonHash: reasonHash, TxID: txID, CreatedAt: timestamp(now),
	}
	return putPublic(ctx, key, record)
}
