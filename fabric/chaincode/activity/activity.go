package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const activityCollection = "activity_private"

type ActivityContract struct{ contractapi.Contract }

type callerIdentity struct {
	AccountID string
	Role      string
	MSPID     string
}

type Activity struct {
	DocType               string `json:"docType"`
	ID                    string `json:"id"`
	CategoryID            string `json:"categoryId"`
	Title                 string `json:"title"`
	OrganizerAccountID    string `json:"organizerAccountId"`
	Capacity              int    `json:"capacity"`
	ApplicationCloseAt    string `json:"applicationCloseAt"`
	StartsAt              string `json:"startsAt"`
	EndsAt                string `json:"endsAt"`
	Status                string `json:"status"`
	ApplicationCount      int    `json:"applicationCount"`
	IssuedCount           int    `json:"issuedCount"`
	DrawCommitment        string `json:"drawCommitment"`
	DrawVerifierAccountID string `json:"drawVerifierAccountId"`
	DrawSeedHash          string `json:"drawSeedHash"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
}

type Application struct {
	DocType    string `json:"docType"`
	ActivityID string `json:"activityId"`
	AccountID  string `json:"accountId"`
	Status     string `json:"status"`
	AppliedAt  string `json:"appliedAt"`
	DrawScore  string `json:"drawScore"`
}

type Ticket struct {
	DocType     string `json:"docType"`
	TicketID    string `json:"ticketId"`
	ActivityID  string `json:"activityId"`
	AccountID   string `json:"accountId"`
	Status      string `json:"status"`
	ClaimHash   string `json:"claimHash"`
	IssuedAt    string `json:"issuedAt"`
	CheckedInAt string `json:"checkedInAt"`
}

type DrawResult struct {
	ActivityID string   `json:"activityId"`
	Winners    []string `json:"winners"`
	Losers     []string `json:"losers"`
	SeedHash   string   `json:"seedHash"`
}

type ClaimRequest struct {
	Secret string `json:"secret"`
	RefID  string `json:"refId"`
}

type CheckInRequest struct {
	TicketID  string `json:"ticketId"`
	Secret    string `json:"secret"`
	TimeSlice int64  `json:"timeSlice"`
}

type CheckInReceipt struct {
	ActivityID  string `json:"activityId"`
	CategoryID  string `json:"categoryId"`
	AccountID   string `json:"accountId"`
	TicketID    string `json:"ticketId"`
	RefID       string `json:"refId"`
	CheckedInAt string `json:"checkedInAt"`
}

type applicationIndex struct {
	Accounts []string `json:"accounts"`
}

type ticketLookup struct {
	ActivityID string `json:"activityId"`
	AccountID  string `json:"accountId"`
}

func caller(ctx contractapi.TransactionContextInterface) (*callerIdentity, error) {
	cid := ctx.GetClientIdentity()
	account, ok, err := cid.GetAttributeValue("eventchain.accountID")
	if err != nil || !ok || strings.TrimSpace(account) == "" {
		return nil, fmt.Errorf("opaque account identity is required")
	}
	role, ok, err := cid.GetAttributeValue("eventchain.role")
	if err != nil || !ok || strings.TrimSpace(role) == "" {
		return nil, fmt.Errorf("role attribute is required")
	}
	msp, err := cid.GetMSPID()
	if err != nil {
		return nil, err
	}
	return &callerIdentity{AccountID: strings.TrimSpace(account), Role: strings.TrimSpace(role), MSPID: msp}, nil
}

func txTime(ctx contractapi.TransactionContextInterface) (time.Time, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos)).UTC(), nil
}

func timestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid RFC3339 time")
	}
	return parsed.UTC(), nil
}

func validateID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r == ':' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func stateKey(ctx contractapi.TransactionContextInterface, kind string, attrs ...string) (string, error) {
	return ctx.GetStub().CreateCompositeKey(kind, attrs)
}

func putPublic(ctx contractapi.TransactionContextInterface, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, raw)
}

func putPrivate(ctx contractapi.TransactionContextInterface, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutPrivateData(activityCollection, key, raw)
}

func getPublic(ctx contractapi.TransactionContextInterface, key string, value any) (bool, error) {
	raw, err := ctx.GetStub().GetState(key)
	if err != nil || raw == nil {
		return false, err
	}
	return true, json.Unmarshal(raw, value)
}

func getPrivate(ctx contractapi.TransactionContextInterface, key string, value any) (bool, error) {
	raw, err := ctx.GetStub().GetPrivateData(activityCollection, key)
	if err != nil || raw == nil {
		return false, err
	}
	return true, json.Unmarshal(raw, value)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (ac *ActivityContract) CreateActivity(ctx contractapi.TransactionContextInterface, id, categoryID, title, capacityText, applicationCloseAt, startsAt, endsAt string) (*Activity, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "OrganizerMSP" || who.Role != "organizer" {
		return nil, fmt.Errorf("organizer identity is required")
	}
	if !validateID(id) || !validateID(categoryID) {
		return nil, fmt.Errorf("invalid activity or category ID")
	}
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 100 {
		return nil, fmt.Errorf("title must contain 1-100 characters")
	}
	capacity, err := strconv.Atoi(capacityText)
	if err != nil || capacity < 1 || capacity > 100000 {
		return nil, fmt.Errorf("capacity must be 1-100000")
	}
	closeAt, err := parseTimestamp(applicationCloseAt)
	if err != nil {
		return nil, err
	}
	start, err := parseTimestamp(startsAt)
	if err != nil {
		return nil, err
	}
	end, err := parseTimestamp(endsAt)
	if err != nil {
		return nil, err
	}
	now, err := txTime(ctx)
	if err != nil {
		return nil, err
	}
	if !closeAt.After(now) || !start.After(closeAt) || !end.After(start) {
		return nil, fmt.Errorf("times must satisfy now < applicationCloseAt < startsAt < endsAt")
	}
	key, _ := stateKey(ctx, "activity", id)
	var existing Activity
	found, err := getPublic(ctx, key, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("activity already exists")
	}
	activity := &Activity{
		DocType: "activity", ID: id, CategoryID: categoryID, Title: title,
		OrganizerAccountID: who.AccountID, Capacity: capacity,
		ApplicationCloseAt: timestamp(closeAt), StartsAt: timestamp(start), EndsAt: timestamp(end),
		Status: "DRAFT", CreatedAt: timestamp(now), UpdatedAt: timestamp(now),
	}
	if err := putPublic(ctx, key, activity); err != nil {
		return nil, err
	}
	return activity, nil
}

func (ac *ActivityContract) GetActivity(ctx contractapi.TransactionContextInterface, id string) (*Activity, error) {
	return loadActivity(ctx, id)
}

func (ac *ActivityContract) ListActivities(ctx contractapi.TransactionContextInterface) ([]*Activity, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey("activity", []string{})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	result := []*Activity{}
	for iterator.HasNext() {
		row, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var item Activity
		if err := json.Unmarshal(row.Value, &item); err != nil {
			return nil, err
		}
		result = append(result, &item)
	}
	return result, nil
}

func (ac *ActivityContract) SetDrawCommitment(ctx contractapi.TransactionContextInterface, id, seedHash string) (*Activity, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "PlatformMSP" || (who.Role != "verifier" && who.Role != "admin") {
		return nil, fmt.Errorf("platform verifier is required")
	}
	if len(seedHash) != 64 {
		return nil, fmt.Errorf("seed hash must be SHA-256 hex")
	}
	if _, err := hex.DecodeString(seedHash); err != nil {
		return nil, fmt.Errorf("seed hash must be SHA-256 hex")
	}
	activity, err := loadActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	if activity.Status != "DRAFT" || activity.DrawCommitment != "" {
		return nil, fmt.Errorf("draw commitment can only be set once in DRAFT")
	}
	now, _ := txTime(ctx)
	activity.DrawCommitment = strings.ToLower(seedHash)
	activity.DrawVerifierAccountID = who.AccountID
	activity.Status = "APPLICATION_OPEN"
	activity.UpdatedAt = timestamp(now)
	if err := saveActivity(ctx, activity); err != nil {
		return nil, err
	}
	return activity, nil
}

func (ac *ActivityContract) UpdateActivityStatus(ctx contractapi.TransactionContextInterface, id, nextStatus string) (*Activity, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	activity, err := loadActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	isOwner := who.MSPID == "OrganizerMSP" && who.Role == "organizer" && who.AccountID == activity.OrganizerAccountID
	isAdmin := who.MSPID == "PlatformMSP" && who.Role == "admin"
	if !isOwner && !isAdmin {
		return nil, fmt.Errorf("activity owner or platform admin is required")
	}
	nextStatus = strings.ToUpper(strings.TrimSpace(nextStatus))
	allowed := map[string]string{"DRAWN": "ONGOING", "ONGOING": "COMPLETED"}
	if nextStatus == "CANCELLED" {
		if activity.Status == "COMPLETED" || activity.Status == "CANCELLED" {
			return nil, fmt.Errorf("completed or cancelled activity cannot be cancelled")
		}
	} else if allowed[activity.Status] != nextStatus {
		return nil, fmt.Errorf("invalid activity status transition")
	}
	now, _ := txTime(ctx)
	activity.Status = nextStatus
	activity.UpdatedAt = timestamp(now)
	if err := saveActivity(ctx, activity); err != nil {
		return nil, err
	}
	return activity, nil
}

func (ac *ActivityContract) Apply(ctx contractapi.TransactionContextInterface, id string) (*Application, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "StudentMSP" || who.Role != "student" {
		return nil, fmt.Errorf("student identity is required")
	}
	activity, err := loadActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	now, _ := txTime(ctx)
	closeAt, _ := parseTimestamp(activity.ApplicationCloseAt)
	if activity.Status != "APPLICATION_OPEN" || !now.Before(closeAt) {
		return nil, fmt.Errorf("applications are closed")
	}
	appKey, _ := stateKey(ctx, "application", id, who.AccountID)
	var existing Application
	found, err := getPrivate(ctx, appKey, &existing)
	if err != nil {
		return nil, err
	}
	if found {
		return &existing, nil
	}
	application := &Application{DocType: "application", ActivityID: id, AccountID: who.AccountID, Status: "PENDING", AppliedAt: timestamp(now)}
	indexKey, _ := stateKey(ctx, "applicationIndex", id)
	index := applicationIndex{Accounts: []string{}}
	_, _ = getPrivate(ctx, indexKey, &index)
	index.Accounts = append(index.Accounts, who.AccountID)
	sort.Strings(index.Accounts)
	activity.ApplicationCount++
	activity.UpdatedAt = timestamp(now)
	if err := putPrivate(ctx, appKey, application); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, indexKey, &index); err != nil {
		return nil, err
	}
	if err := saveActivity(ctx, activity); err != nil {
		return nil, err
	}
	return application, nil
}

func (ac *ActivityContract) RunDraw(ctx contractapi.TransactionContextInterface, id, seed string) (*DrawResult, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	activity, err := loadActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	if who.AccountID != activity.OrganizerAccountID || who.Role != "organizer" {
		return nil, fmt.Errorf("activity organizer is required")
	}
	now, _ := txTime(ctx)
	closeAt, _ := parseTimestamp(activity.ApplicationCloseAt)
	if now.Before(closeAt) {
		return nil, fmt.Errorf("application window is still open")
	}
	if activity.Status != "APPLICATION_OPEN" {
		return nil, fmt.Errorf("activity is not drawable")
	}
	if len(seed) < 32 || len(seed) > 256 || digest(seed) != activity.DrawCommitment {
		return nil, fmt.Errorf("draw seed does not match verifier commitment")
	}
	indexKey, _ := stateKey(ctx, "applicationIndex", id)
	index := applicationIndex{Accounts: []string{}}
	_, err = getPrivate(ctx, indexKey, &index)
	if err != nil {
		return nil, err
	}
	type candidate struct{ account, score string }
	candidates := make([]candidate, 0, len(index.Accounts))
	for _, account := range index.Accounts {
		candidates = append(candidates, candidate{account: account, score: digest(seed + ":" + id + ":" + account)})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].account < candidates[j].account
		}
		return candidates[i].score < candidates[j].score
	})
	winnerCount := activity.Capacity
	if winnerCount > len(candidates) {
		winnerCount = len(candidates)
	}
	winners, losers := []string{}, []string{}
	for i, candidate := range candidates {
		appKey, _ := stateKey(ctx, "application", id, candidate.account)
		var application Application
		found, err := getPrivate(ctx, appKey, &application)
		if err != nil || !found {
			return nil, fmt.Errorf("application index is inconsistent")
		}
		application.DrawScore = candidate.score
		if i < winnerCount {
			application.Status = "WON"
			winners = append(winners, candidate.account)
		} else {
			application.Status = "LOST"
			losers = append(losers, candidate.account)
		}
		if err := putPrivate(ctx, appKey, &application); err != nil {
			return nil, err
		}
	}
	activity.Status = "DRAWN"
	activity.IssuedCount = winnerCount
	activity.DrawSeedHash = digest(seed)
	activity.UpdatedAt = timestamp(now)
	if err := saveActivity(ctx, activity); err != nil {
		return nil, err
	}
	return &DrawResult{ActivityID: id, Winners: winners, Losers: losers, SeedHash: digest(seed)}, nil
}

func (ac *ActivityContract) ClaimTicket(ctx contractapi.TransactionContextInterface, id string) (*Ticket, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "StudentMSP" || who.Role != "student" {
		return nil, fmt.Errorf("student identity is required")
	}
	transient, err := ctx.GetStub().GetTransient()
	if err != nil {
		return nil, err
	}
	raw := transient["claim"]
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, fmt.Errorf("transient claim is required")
	}
	var request ClaimRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return nil, fmt.Errorf("invalid transient claim")
	}
	if len(request.Secret) < 32 || len(request.Secret) > 256 || !validateID(request.RefID) {
		return nil, fmt.Errorf("invalid claim secret or reference")
	}
	appKey, _ := stateKey(ctx, "application", id, who.AccountID)
	var application Application
	found, err := getPrivate(ctx, appKey, &application)
	if err != nil || !found {
		return nil, fmt.Errorf("winning application not found")
	}
	ticketKey, _ := stateKey(ctx, "ticket", id, who.AccountID)
	if application.Status == "CLAIMED" {
		var existing Ticket
		found, err := getPrivate(ctx, ticketKey, &existing)
		if err != nil || !found {
			return nil, fmt.Errorf("ticket state is inconsistent")
		}
		if existing.ClaimHash != digest(request.Secret) {
			return nil, fmt.Errorf("idempotency reference was reused with another secret")
		}
		return &existing, nil
	}
	if application.Status != "WON" {
		return nil, fmt.Errorf("application did not win")
	}
	now, _ := txTime(ctx)
	ticket := &Ticket{
		DocType: "ticket", TicketID: digest(id + ":" + who.AccountID)[:32], ActivityID: id,
		AccountID: who.AccountID, Status: "ISSUED", ClaimHash: digest(request.Secret), IssuedAt: timestamp(now),
	}
	lookupKey, _ := stateKey(ctx, "ticketLookup", ticket.TicketID)
	application.Status = "CLAIMED"
	if err := putPrivate(ctx, appKey, &application); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, ticketKey, ticket); err != nil {
		return nil, err
	}
	if err := putPrivate(ctx, lookupKey, &ticketLookup{ActivityID: id, AccountID: who.AccountID}); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (ac *ActivityContract) GetMyApplication(ctx contractapi.TransactionContextInterface, id string) (*Application, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	key, _ := stateKey(ctx, "application", id, who.AccountID)
	var application Application
	found, err := getPrivate(ctx, key, &application)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("application not found")
	}
	return &application, nil
}

func (ac *ActivityContract) GetMyTicket(ctx contractapi.TransactionContextInterface, id string) (*Ticket, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	key, _ := stateKey(ctx, "ticket", id, who.AccountID)
	var ticket Ticket
	found, err := getPrivate(ctx, key, &ticket)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("ticket not found")
	}
	return &ticket, nil
}

func (ac *ActivityContract) CheckIn(ctx contractapi.TransactionContextInterface) (*CheckInReceipt, error) {
	who, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if who.MSPID != "PlatformMSP" || (who.Role != "operator" && who.Role != "admin") {
		return nil, fmt.Errorf("platform check-in operator is required")
	}
	transient, err := ctx.GetStub().GetTransient()
	if err != nil {
		return nil, err
	}
	raw := transient["checkIn"]
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, fmt.Errorf("transient checkIn is required")
	}
	var request CheckInRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return nil, fmt.Errorf("invalid transient checkIn")
	}
	lookupKey, _ := stateKey(ctx, "ticketLookup", request.TicketID)
	var lookup ticketLookup
	found, err := getPrivate(ctx, lookupKey, &lookup)
	if err != nil || !found {
		return nil, fmt.Errorf("ticket lookup not found")
	}
	ticketKey, _ := stateKey(ctx, "ticket", lookup.ActivityID, lookup.AccountID)
	var ticket Ticket
	found, err = getPrivate(ctx, ticketKey, &ticket)
	if err != nil || !found {
		return nil, fmt.Errorf("ticket not found")
	}
	if digest(request.Secret) != ticket.ClaimHash {
		return nil, fmt.Errorf("invalid ticket proof")
	}
	receiptKey, _ := stateKey(ctx, "checkInReceipt", request.TicketID)
	if ticket.Status == "USED" {
		var existing CheckInReceipt
		found, err := getPrivate(ctx, receiptKey, &existing)
		if err != nil || !found {
			return nil, fmt.Errorf("used ticket receipt is missing")
		}
		return &existing, nil
	}
	if ticket.Status != "ISSUED" {
		return nil, fmt.Errorf("ticket is not usable")
	}
	now, _ := txTime(ctx)
	currentSlice := now.Unix() / 30
	if request.TimeSlice < currentSlice-1 || request.TimeSlice > currentSlice+1 {
		return nil, fmt.Errorf("QR proof expired")
	}
	activity, err := loadActivity(ctx, lookup.ActivityID)
	if err != nil {
		return nil, err
	}
	startsAt, _ := parseTimestamp(activity.StartsAt)
	endsAt, _ := parseTimestamp(activity.EndsAt)
	if now.Before(startsAt.Add(-2*time.Hour)) || now.After(endsAt.Add(2*time.Hour)) {
		return nil, fmt.Errorf("check-in is outside the event window")
	}
	ticket.Status = "USED"
	ticket.CheckedInAt = timestamp(now)
	if err := putPrivate(ctx, ticketKey, &ticket); err != nil {
		return nil, err
	}
	refID := "checkin:" + activity.ID + ":" + lookup.AccountID
	receipt := &CheckInReceipt{
		ActivityID: activity.ID, CategoryID: activity.CategoryID, AccountID: lookup.AccountID,
		TicketID: ticket.TicketID, RefID: refID, CheckedInAt: timestamp(now),
	}
	if err := putPrivate(ctx, receiptKey, receipt); err != nil {
		return nil, err
	}
	return receipt, nil
}

func loadActivity(ctx contractapi.TransactionContextInterface, id string) (*Activity, error) {
	key, _ := stateKey(ctx, "activity", id)
	var activity Activity
	found, err := getPublic(ctx, key, &activity)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("activity not found")
	}
	return &activity, nil
}

func saveActivity(ctx contractapi.TransactionContextInterface, activity *Activity) error {
	key, _ := stateKey(ctx, "activity", activity.ID)
	return putPublic(ctx, key, activity)
}
