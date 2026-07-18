package main

const (
	financeCollection = "finance"
	amountScale       = int64(1_000_000)
	bpsScale          = int64(10_000)
)

type SystemConfig struct {
	DocType                 string `json:"docType"`
	SchemaVersion           int    `json:"schemaVersion"`
	DemoMode                bool   `json:"demoMode"`
	AmountScale             int64  `json:"amountScale"`
	RedemptionFeeBPS        int64  `json:"redemptionFeeBps"`
	MarketFeeBPS            int64  `json:"marketFeeBps"`
	OrganizerFeeShareBPS    int64  `json:"organizerFeeShareBps"`
	RiskFeeShareBPS         int64  `json:"riskFeeShareBps"`
	BurnFeeShareBPS         int64  `json:"burnFeeShareBps"`
	PaidExpiryDays          int64  `json:"paidExpiryDays"`
	BonusExpiryDays         int64  `json:"bonusExpiryDays"`
	LockGraceDays           int64  `json:"lockGraceDays"`
	TransferCooldownHours   int64  `json:"transferCooldownHours"`
	TransferDailyLimit      int64  `json:"transferDailyLimit"`
	TransferDailyRecipients int    `json:"transferDailyRecipients"`
	CheckInDailyBonus       int64  `json:"checkInDailyBonus"`
	CheckInMonthlyBonus     int64  `json:"checkInMonthlyBonus"`
	ChallengeHours          int64  `json:"challengeHours"`
	ClaimPendingDays        int64  `json:"claimPendingDays"`
	CorrectionWindowDays    int64  `json:"correctionWindowDays"`
	MaxUserExposureBPS      int64  `json:"maxUserExposureBps"`
	MaxRiskGroupExposureBPS int64  `json:"maxRiskGroupExposureBps"`
	DefaultCancellationBPS  int64  `json:"defaultCancellationBps"`
	MaximumCancellationBPS  int64  `json:"maximumCancellationBps"`
	DefaultMarketCap        int64  `json:"defaultMarketCap"`
	MinimumListingBond      int64  `json:"minimumListingBond"`
	MinimumResultBond       int64  `json:"minimumResultBond"`
	UpdatedAt               string `json:"updatedAt"`
}

func defaultSystemConfig(now string) *SystemConfig {
	return &SystemConfig{
		DocType:                 "systemConfig",
		SchemaVersion:           2,
		DemoMode:                true,
		AmountScale:             amountScale,
		RedemptionFeeBPS:        50,
		MarketFeeBPS:            100,
		OrganizerFeeShareBPS:    6000,
		RiskFeeShareBPS:         3000,
		BurnFeeShareBPS:         1000,
		PaidExpiryDays:          365,
		BonusExpiryDays:         90,
		LockGraceDays:           30,
		TransferCooldownHours:   24,
		TransferDailyLimit:      500 * amountScale,
		TransferDailyRecipients: 5,
		CheckInDailyBonus:       5 * amountScale,
		CheckInMonthlyBonus:     100 * amountScale,
		ChallengeHours:          24,
		ClaimPendingDays:        7,
		CorrectionWindowDays:    7,
		MaxUserExposureBPS:      500,
		MaxRiskGroupExposureBPS: 1000,
		DefaultCancellationBPS:  12000,
		MaximumCancellationBPS:  15000,
		DefaultMarketCap:        4000 * amountScale,
		MinimumListingBond:      200 * amountScale,
		MinimumResultBond:       100 * amountScale,
		UpdatedAt:               now,
	}
}

type Category struct {
	DocType   string `json:"docType"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

type Wallet struct {
	DocType     string                     `json:"docType"`
	AccountID   string                     `json:"accountId"`
	AAvailable  int64                      `json:"aAvailable"`
	AReserved   int64                      `json:"aReserved"`
	Categories  map[string]*CategoryWallet `json:"categories"`
	RiskGroupID string                     `json:"riskGroupId"`
	UpdatedAt   string                     `json:"updatedAt"`
}

type CategoryWallet struct {
	Paid  BucketBalance `json:"paid"`
	Bonus BucketBalance `json:"bonus"`
}

type BucketBalance struct {
	Available int64 `json:"available"`
	Locked    int64 `json:"locked"`
}

type AssetLot struct {
	DocType        string `json:"docType"`
	LotID          string `json:"lotId"`
	AccountID      string `json:"accountId"`
	CategoryID     string `json:"categoryId"`
	Bucket         string `json:"bucket"`
	Available      int64  `json:"available"`
	Locked         int64  `json:"locked"`
	IssuedAt       string `json:"issuedAt"`
	ExpiresAt      string `json:"expiresAt"`
	TransferableAt string `json:"transferableAt"`
	Source         string `json:"source"`
	BackingVaultID string `json:"backingVaultId"`
}

type LotIndex struct {
	DocType    string   `json:"docType"`
	AccountID  string   `json:"accountId"`
	CategoryID string   `json:"categoryId"`
	Bucket     string   `json:"bucket"`
	LotIDs     []string `json:"lotIds"`
}

type Vault struct {
	DocType    string `json:"docType"`
	VaultID    string `json:"vaultId"`
	CategoryID string `json:"categoryId"`
	LockedA    int64  `json:"lockedA"`
	UpdatedAt  string `json:"updatedAt"`
}

type Outcome struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Market struct {
	DocType            string           `json:"docType"`
	MarketID           string           `json:"marketId"`
	EventID            string           `json:"eventId"`
	CategoryID         string           `json:"categoryId"`
	OrganizerAccountID string           `json:"organizerAccountId"`
	Outcomes           []Outcome        `json:"outcomes"`
	StakeBucket        string           `json:"stakeBucket"`
	Status             string           `json:"status"`
	MarketCap          int64            `json:"marketCap"`
	DisplayedPool      int64            `json:"displayedPool"`
	OutcomeProbability map[string]int64 `json:"outcomeProbabilityBps"`
	SnapshotAt         string           `json:"snapshotAt"`
	CloseAt            string           `json:"closeAt"`
	CreatedAt          string           `json:"createdAt"`
	LockedAt           string           `json:"lockedAt"`
	ResultProposal     *ResultProposal  `json:"resultProposal"`
	SettlementEpoch    int64            `json:"settlementEpoch"`
	Paused             bool             `json:"paused"`
	PauseReason        string           `json:"pauseReason"`
	ListingBond        int64            `json:"listingBond"`
	Dispute            *Dispute         `json:"dispute"`
}

type MarketLedger struct {
	DocType                 string           `json:"docType"`
	MarketID                string           `json:"marketId"`
	TotalPool               int64            `json:"totalPool"`
	OutcomeTotals           map[string]int64 `json:"outcomeTotals"`
	OutcomeParticipantCount map[string]int64 `json:"outcomeParticipantCount"`
	ParticipantCount        int64            `json:"participantCount"`
}

type ResultProposal struct {
	OutcomeID           string `json:"outcomeId"`
	ProposerAccountID   string `json:"proposerAccountId"`
	EvidenceHash        string `json:"evidenceHash"`
	ProposedAt          string `json:"proposedAt"`
	ConfirmedBy         string `json:"confirmedBy"`
	ConfirmedAt         string `json:"confirmedAt"`
	ChallengeDeadline   string `json:"challengeDeadline"`
	ChallengerAccountID string `json:"challengerAccountId"`
	ChallengeReasonHash string `json:"challengeReasonHash"`
	ProposalBond        int64  `json:"proposalBond"`
	ChallengeBond       int64  `json:"challengeBond"`
}

type Dispute struct {
	Votes             map[string]string `json:"votes"`
	ResolvedOutcomeID string            `json:"resolvedOutcomeId"`
	ResolvedAt        string            `json:"resolvedAt"`
}

type Position struct {
	DocType   string `json:"docType"`
	MarketID  string `json:"marketId"`
	AccountID string `json:"accountId"`
	OutcomeID string `json:"outcomeId"`
	Bucket    string `json:"bucket"`
	Stake     int64  `json:"stake"`
	OpenedAt  string `json:"openedAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Settlement struct {
	DocType            string `json:"docType"`
	MarketID           string `json:"marketId"`
	Epoch              int64  `json:"epoch"`
	Status             string `json:"status"`
	WinningOutcomeID   string `json:"winningOutcomeId"`
	GrossPool          int64  `json:"grossPool"`
	WinningStake       int64  `json:"winningStake"`
	Fee                int64  `json:"fee"`
	NetPool            int64  `json:"netPool"`
	FinalizedAt        string `json:"finalizedAt"`
	CorrectionDeadline string `json:"correctionDeadline"`
	AvailableAt        string `json:"availableAt"`
	PreparedClaims     int64  `json:"preparedClaims"`
	PreparedAmount     int64  `json:"preparedAmount"`
	MaturedClaims      int64  `json:"maturedClaims"`
	MaturedAmount      int64  `json:"maturedAmount"`
	ExpectedClaims     int64  `json:"expectedClaims"`
	FeesActivated      bool   `json:"feesActivated"`
	Reason             string `json:"reason"`
}

type Claim struct {
	DocType     string `json:"docType"`
	ClaimID     string `json:"claimId"`
	MarketID    string `json:"marketId"`
	Epoch       int64  `json:"epoch"`
	AccountID   string `json:"accountId"`
	Bucket      string `json:"bucket"`
	Amount      int64  `json:"amount"`
	Shard       int    `json:"shard"`
	Status      string `json:"status"`
	AvailableAt string `json:"availableAt"`
	PreparedAt  string `json:"preparedAt"`
	MaturedAt   string `json:"maturedAt"`
	TxID        string `json:"txId"`
}

type EscrowShard struct {
	DocType      string `json:"docType"`
	MarketID     string `json:"marketId"`
	Epoch        int64  `json:"epoch"`
	Shard        int    `json:"shard"`
	Allocated    int64  `json:"allocated"`
	Matured      int64  `json:"matured"`
	ClaimCount   int64  `json:"claimCount"`
	MaturedCount int64  `json:"maturedCount"`
}

type Supply struct {
	DocType string `json:"docType"`
	IssuedA int64  `json:"issuedA"`
	BurnedA int64  `json:"burnedA"`
}

type OperationReceipt struct {
	DocType   string `json:"docType"`
	RefID     string `json:"refId"`
	Operation string `json:"operation"`
	AccountID string `json:"accountId"`
	Amount    int64  `json:"amount"`
	Fee       int64  `json:"fee"`
	TxID      string `json:"txId"`
	CreatedAt string `json:"createdAt"`
}

type PositionReceipt struct {
	DocType   string `json:"docType"`
	RefID     string `json:"refId"`
	MarketID  string `json:"marketId"`
	OutcomeID string `json:"outcomeId"`
	AccountID string `json:"accountId"`
	Bucket    string `json:"bucket"`
	Amount    int64  `json:"amount"`
	TxID      string `json:"txId"`
	CreatedAt string `json:"createdAt"`
}

type PlacePositionRequest struct {
	MarketID  string `json:"marketId"`
	OutcomeID string `json:"outcomeId"`
	Amount    string `json:"amount"`
	RefID     string `json:"refId"`
}

type ExpirySweepResult struct {
	AccountID     string `json:"accountId"`
	CategoryID    string `json:"categoryId"`
	ProcessedLots int    `json:"processedLots"`
	RefundedA     int64  `json:"refundedA"`
	BurnedBonus   int64  `json:"burnedBonus"`
}

type EmergencyAction struct {
	DocType     string            `json:"docType"`
	ActionID    string            `json:"actionId"`
	ActionType  string            `json:"actionType"`
	PayloadHash string            `json:"payloadHash"`
	Payload     string            `json:"payload"`
	Nonce       string            `json:"nonce"`
	ExpiresAt   string            `json:"expiresAt"`
	Approvals   map[string]string `json:"approvals"`
	Executed    bool              `json:"executed"`
	CreatedAt   string            `json:"createdAt"`
	ExecutedAt  string            `json:"executedAt"`
}

type EmergencyCorrectionPayload struct {
	MarketID     string `json:"marketId"`
	BaseEpoch    int64  `json:"baseEpoch"`
	NewOutcomeID string `json:"newOutcomeId"`
	ReasonHash   string `json:"reasonHash"`
}

type EmergencyMarketPayload struct {
	MarketID string `json:"marketId"`
	Reason   string `json:"reason"`
}

type ServiceOffer struct {
	DocType             string `json:"docType"`
	OfferID             string `json:"offerId"`
	OrganizerAccountID  string `json:"organizerAccountId"`
	CategoryID          string `json:"categoryId"`
	OfferType           string `json:"offerType"`
	Title               string `json:"title"`
	PricePaidB          int64  `json:"pricePaidB"`
	CancellationBPS     int64  `json:"cancellationBps"`
	GuaranteeBudgetA    int64  `json:"guaranteeBudgetA"`
	GuaranteeAvailableA int64  `json:"guaranteeAvailableA"`
	BonusBudget         int64  `json:"bonusBudget"`
	BonusAvailable      int64  `json:"bonusAvailable"`
	Inventory           int64  `json:"inventory"`
	Status              string `json:"status"`
	CreatedAt           string `json:"createdAt"`
}

type OrderPaymentLot struct {
	Amount    int64  `json:"amount"`
	ExpiresAt string `json:"expiresAt"`
}

type ServiceOrder struct {
	DocType            string            `json:"docType"`
	OrderID            string            `json:"orderId"`
	OfferID            string            `json:"offerId"`
	AccountID          string            `json:"accountId"`
	OrganizerAccountID string            `json:"organizerAccountId"`
	CategoryID         string            `json:"categoryId"`
	PrincipalPaidB     int64             `json:"principalPaidB"`
	GuaranteeReservedA int64             `json:"guaranteeReservedA"`
	PaymentLots        []OrderPaymentLot `json:"paymentLots"`
	Status             string            `json:"status"`
	ScheduledAt        string            `json:"scheduledAt"`
	CreatedAt          string            `json:"createdAt"`
	UpdatedAt          string            `json:"updatedAt"`
	DelayCompensation  int64             `json:"delayCompensation"`
	CancellationReason string            `json:"cancellationReason"`
	RefundedA          int64             `json:"refundedA"`
}

type CreateServiceOrderRequest struct {
	OfferID     string `json:"offerId"`
	ScheduledAt string `json:"scheduledAt"`
	RefID       string `json:"refId"`
}
