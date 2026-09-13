package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func badgeSeriesJSON(base time.Time, seriesID string, maxSupply int) string {
	request := createBadgeSeriesRequest{
		SeriesID: seriesID, SeasonID: "2026-autumn", CategoryID: "basketball",
		Title: "Autumn badge", Description: "A commemorative badge for valid check-in",
		AssetURI:    "/badges/" + seriesID + "/v1/badge.webp",
		AssetSHA256: digest("asset:" + seriesID), MetadataSHA256: digest("metadata:" + seriesID),
		EligibilityPolicyHash: digest("policy:" + seriesID), IssuanceMode: "CHECK_IN_GUARANTEED",
		MaxSupply: maxSupply, EligibilityOpenAt: timestamp(base.Add(30 * time.Minute)),
		EligibilityCloseAt: timestamp(base.Add(5 * time.Hour)), ClaimOpenAt: timestamp(base.Add(30 * time.Minute)),
		ClaimCloseAt: timestamp(base.Add(6 * time.Hour)), SupplyRationale: "capacity bounded supply",
	}
	raw, _ := json.Marshal(request)
	return string(raw)
}

func TestBadgeCheckInEligibilityAndAtomicClaim(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	admin := activityIdentity("PlatformMSP", "admin01", "admin")
	organizer := activityIdentity("OrganizerMSP", "organizer01", "organizer")
	verifier := activityIdentity("PlatformMSP", "verifier01", "verifier")
	operator := activityIdentity("PlatformMSP", "operator01", "operator")
	student := activityIdentity("StudentMSP", "student01", "student")
	seed := "independent-verifier-seed-badge-0001"

	h.tx("create-activity", organizer, base, func() {
		_, err := h.contract.CreateActivity(h.ctx, "badge-event", "basketball", "Final", "1",
			timestamp(base.Add(time.Hour)), timestamp(base.Add(2*time.Hour)), timestamp(base.Add(4*time.Hour)))
		require.NoError(t, err)
	})
	h.tx("create-series", admin, base.Add(time.Minute), func() {
		series, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, "basketball-2026", 1))
		require.NoError(t, err)
		require.Equal(t, "DRAFT", series.Status)
		require.Zero(t, series.IssuedCount)
	})
	h.tx("link", admin, base.Add(2*time.Minute), func() {
		link, err := h.contract.LinkBadgeActivity(h.ctx, "basketball-2026", "badge-event", "1")
		require.NoError(t, err)
		require.Equal(t, 1, link.EligibilityQuota)
	})
	h.tx("activate", admin, base.Add(3*time.Minute), func() {
		series, err := h.contract.ActivateBadgeSeries(h.ctx, "basketball-2026")
		require.NoError(t, err)
		require.Equal(t, "ACTIVE", series.Status)
		links, err := h.contract.ListBadgeActivityLinks(h.ctx, "basketball-2026")
		require.NoError(t, err)
		require.Len(t, links, 1)
		require.Equal(t, "badge-event", links[0].ActivityID)
	})
	h.tx("commit", verifier, base.Add(4*time.Minute), func() {
		_, err := h.contract.SetDrawCommitment(h.ctx, "badge-event", digest(seed))
		require.NoError(t, err)
	})
	h.tx("apply", student, base.Add(5*time.Minute), func() {
		_, err := h.contract.Apply(h.ctx, "badge-event")
		require.NoError(t, err)
	})
	h.tx("draw", organizer, base.Add(61*time.Minute), func() {
		_, err := h.contract.RunDraw(h.ctx, "badge-event", seed)
		require.NoError(t, err)
	})

	secret := "student-held-badge-ticket-secret-001"
	var ticket *Ticket
	h.tx("ticket", student, base.Add(70*time.Minute), func() {
		raw, _ := json.Marshal(ClaimRequest{Secret: secret, RefID: "claim-badge-event"})
		h.stub.TransientMap = map[string][]byte{"claim": raw}
		var err error
		_, err = h.contract.ClaimTicket(h.ctx, "badge-event")
		require.NoError(t, err)
		ticket, err = h.contract.GetMyTicket(h.ctx, "badge-event")
		require.NoError(t, err)
	})
	// A malformed linked series must fail before CheckIn writes the ticket or
	// receipt. Fabric rolls back the complete transaction; the mock harness
	// still lets us assert that no writes occur before eligibility succeeds.
	h.tx("inject-broken-link", admin, base.Add(71*time.Minute), func() {
		key, err := stateKey(h.ctx, "badgeActivityLink", "badge-event", "missing-series")
		require.NoError(t, err)
		require.NoError(t, putPublic(h.ctx, key, &BadgeActivityLink{DocType: "badgeActivityLink", SeriesID: "missing-series", ActivityID: "badge-event", EligibilityType: "VALID_CHECK_IN", EligibilityQuota: 1}))
	})
	h.tx("check-in-broken-link", operator, base.Add(2*time.Hour), func() {
		raw, _ := json.Marshal(CheckInRequest{TicketID: ticket.TicketID, Secret: secret, TimeSlice: base.Add(2*time.Hour).Unix() / 30})
		h.stub.TransientMap = map[string][]byte{"checkIn": raw}
		_, err := h.contract.CheckIn(h.ctx)
		require.ErrorContains(t, err, "badge series not found")
		ticketKey, keyErr := stateKey(h.ctx, "ticket", "badge-event", "student01")
		require.NoError(t, keyErr)
		var current Ticket
		found, getErr := getPrivate(h.ctx, ticketKey, &current)
		require.NoError(t, getErr)
		require.True(t, found)
		require.Equal(t, "ISSUED", current.Status)
		_, found, getErr = loadBadgeEligibility(h.ctx, "basketball-2026", "student01")
		require.NoError(t, getErr)
		require.False(t, found)
	})
	h.tx("remove-broken-link", admin, base.Add(72*time.Minute), func() {
		key, err := stateKey(h.ctx, "badgeActivityLink", "badge-event", "missing-series")
		require.NoError(t, err)
		require.NoError(t, h.stub.DelState(key))
	})
	h.tx("check-in", operator, base.Add(2*time.Hour), func() {
		raw, _ := json.Marshal(CheckInRequest{TicketID: ticket.TicketID, Secret: secret, TimeSlice: base.Add(2*time.Hour).Unix() / 30})
		h.stub.TransientMap = map[string][]byte{"checkIn": raw}
		receipt, err := h.contract.CheckIn(h.ctx)
		require.NoError(t, err)
		require.False(t, receipt.Replayed)
		require.Equal(t, []BadgeEligibilityResult{{SeriesID: "basketball-2026", Status: "CLAIMABLE", EligibleAt: timestamp(base.Add(2 * time.Hour))}}, receipt.BadgeEligibility)
	})
	h.tx("check-in-replay", operator, base.Add(2*time.Hour+time.Second), func() {
		raw, _ := json.Marshal(CheckInRequest{TicketID: ticket.TicketID, Secret: secret, TimeSlice: base.Add(2*time.Hour).Unix() / 30})
		h.stub.TransientMap = map[string][]byte{"checkIn": raw}
		receipt, err := h.contract.CheckIn(h.ctx)
		require.NoError(t, err)
		require.True(t, receipt.Replayed)
		require.Equal(t, []BadgeEligibilityResult{{SeriesID: "basketball-2026", Status: "CLAIMABLE", EligibleAt: timestamp(base.Add(2 * time.Hour))}}, receipt.BadgeEligibility)
		eligibility, found, err := loadBadgeEligibility(h.ctx, "basketball-2026", "student01")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, ticket.TicketID, eligibility.SourceTicketID)
		require.Equal(t, timestamp(base.Add(2*time.Hour)), eligibility.EligibleAt)
	})

	h.tx("view-before", student, base.Add(2*time.Hour+time.Minute), func() {
		view, err := h.contract.GetMyBadge(h.ctx, "basketball-2026")
		require.NoError(t, err)
		require.Nil(t, view.Award)
		require.Nil(t, view.Instance)
		require.NotNil(t, view.Eligibility)
		require.Equal(t, "CLAIMABLE", view.Eligibility.Status)
	})

	var first *BadgeAwardView
	h.tx("badge-claim", student, base.Add(2*time.Hour+2*time.Minute), func() {
		var err error
		first, err = h.contract.ClaimMyBadge(h.ctx, "basketball-2026", "badge-claim-0001")
		require.NoError(t, err)
		require.Equal(t, 1, first.Instance.SerialNumber)
		require.Equal(t, digest("basketball-2026:000001"), first.Instance.InstanceID)
		require.Equal(t, "CLAIMED", first.Eligibility.Status)
		require.Equal(t, 1, first.Series.IssuedCount)
	})
	h.tx("badge-replay", student, base.Add(2*time.Hour+3*time.Minute), func() {
		replayed, err := h.contract.ClaimMyBadge(h.ctx, "basketball-2026", "badge-claim-0001")
		require.NoError(t, err)
		require.Equal(t, first.Instance.InstanceID, replayed.Instance.InstanceID)
		require.Equal(t, 1, replayed.Series.IssuedCount)
		receipt, err := h.contract.GetMyBadgeClaimReceipt(h.ctx, "basketball-2026", "badge-claim-0001")
		require.NoError(t, err)
		require.Equal(t, first.Instance.InstanceID, receipt.InstanceID)
		mine, err := h.contract.ListMyBadges(h.ctx)
		require.NoError(t, err)
		require.Len(t, mine, 1)
		require.Equal(t, "student01", mine[0].Award.AccountID)
	})
	h.tx("other-student-list", activityIdentity("StudentMSP", "student02", "student"), base.Add(2*time.Hour+3*time.Minute), func() {
		mine, err := h.contract.ListMyBadges(h.ctx)
		require.NoError(t, err)
		require.Empty(t, mine)
	})
	h.tx("public-instance", admin, base.Add(2*time.Hour+4*time.Minute), func() {
		instance, err := h.contract.GetBadgePublicInstanceByID(h.ctx, first.Instance.InstanceID)
		require.NoError(t, err)
		require.Equal(t, 1, instance.SerialNumber)
	})
}

func TestBadgeAdminStateAndCapacityGuards(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	admin := activityIdentity("PlatformMSP", "admin01", "admin")
	organizer := activityIdentity("OrganizerMSP", "organizer01", "organizer")
	student := activityIdentity("StudentMSP", "student01", "student")

	h.tx("activity", organizer, base, func() {
		_, err := h.contract.CreateActivity(h.ctx, "capacity-two", "basketball", "Final", "2",
			timestamp(base.Add(time.Hour)), timestamp(base.Add(2*time.Hour)), timestamp(base.Add(4*time.Hour)))
		require.NoError(t, err)
	})
	h.tx("forbidden-create", student, base.Add(time.Minute), func() {
		_, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, "forbidden", 2))
		require.ErrorContains(t, err, "platform admin")
	})
	h.tx("create", admin, base.Add(2*time.Minute), func() {
		_, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, "underfunded", 1))
		require.NoError(t, err)
	})
	h.tx("bad-quota", admin, base.Add(3*time.Minute), func() {
		_, err := h.contract.LinkBadgeActivity(h.ctx, "underfunded", "capacity-two", "1")
		require.ErrorContains(t, err, "must equal activity capacity")
	})
	h.tx("link", admin, base.Add(4*time.Minute), func() {
		_, err := h.contract.LinkBadgeActivity(h.ctx, "underfunded", "capacity-two", "2")
		require.NoError(t, err)
	})
	h.tx("underfunded", admin, base.Add(5*time.Minute), func() {
		_, err := h.contract.ActivateBadgeSeries(h.ctx, "underfunded")
		require.ErrorContains(t, err, "max supply")
	})
}

func TestBadgeSeriesRejectsInvalidSupplyAndMetadata(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	admin := activityIdentity("PlatformMSP", "admin01", "admin")

	for index, maxSupply := range []int{0, -1, 100001} {
		h.tx(fmt.Sprintf("invalid-supply-%d", index), admin, base.Add(time.Duration(index)*time.Minute), func() {
			_, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, fmt.Sprintf("invalid-supply-%d", index), maxSupply))
			require.ErrorContains(t, err, "max supply")
		})
	}

	var request createBadgeSeriesRequest
	require.NoError(t, json.Unmarshal([]byte(badgeSeriesJSON(base, "invalid-asset", 1)), &request))
	request.AssetURI = "/badges/../unsafe.svg?script=1"
	raw, err := json.Marshal(request)
	require.NoError(t, err)
	h.tx("invalid-asset-uri", admin, base.Add(4*time.Minute), func() {
		_, createErr := h.contract.CreateBadgeSeries(h.ctx, string(raw))
		require.ErrorContains(t, createErr, "asset URI")
	})

	request.AssetURI = "/badges/invalid-asset/v1/badge.webp"
	request.AssetSHA256 = "ABCDEF"
	raw, err = json.Marshal(request)
	require.NoError(t, err)
	h.tx("invalid-asset-hash", admin, base.Add(5*time.Minute), func() {
		_, createErr := h.contract.CreateBadgeSeries(h.ctx, string(raw))
		require.ErrorContains(t, createErr, "lowercase SHA-256")
	})
}

func TestBadgeManagementRequiresPlatformAdminForEveryRole(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	identities := []struct {
		name     string
		identity *fakeIdentity
	}{
		{"student", activityIdentity("StudentMSP", "student01", "student")},
		{"organizer", activityIdentity("OrganizerMSP", "organizer01", "organizer")},
		{"operator", activityIdentity("PlatformMSP", "operator01", "operator")},
		{"verifier", activityIdentity("PlatformMSP", "verifier01", "verifier")},
	}
	for index, candidate := range identities {
		h.tx("forbidden-"+candidate.name, candidate.identity, base.Add(time.Duration(index)*time.Minute), func() {
			_, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, "forbidden-"+candidate.name, 1))
			require.ErrorContains(t, err, "platform admin")
		})
	}
}

func TestBadgeReadinessTouchesPrivateCollection(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	admin := activityIdentity("PlatformMSP", "admin01", "admin")
	student := activityIdentity("StudentMSP", "student01", "student")
	h.tx("readiness", admin, base, func() {
		readiness, err := h.contract.GetBadgeReadiness(h.ctx)
		require.NoError(t, err)
		require.Equal(t, &BadgeReadiness{DocType: "badgeReadiness", Collection: "badge_private", Status: "READY"}, readiness)
	})
	h.tx("readiness-forbidden", student, base.Add(time.Minute), func() {
		_, err := h.contract.GetBadgeReadiness(h.ctx)
		require.ErrorContains(t, err, "platform admin")
	})
}

func TestBadgePauseCloseFinalizeAndNoEligibilityOutsideWindow(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	admin := activityIdentity("PlatformMSP", "admin01", "admin")
	organizer := activityIdentity("OrganizerMSP", "organizer01", "organizer")

	h.tx("activity", organizer, base, func() {
		_, err := h.contract.CreateActivity(h.ctx, "lifecycle", "basketball", "Final", "1",
			timestamp(base.Add(time.Hour)), timestamp(base.Add(2*time.Hour)), timestamp(base.Add(4*time.Hour)))
		require.NoError(t, err)
	})
	h.tx("create", admin, base.Add(time.Minute), func() {
		_, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, "lifecycle-series", 1))
		require.NoError(t, err)
	})
	h.tx("link", admin, base.Add(2*time.Minute), func() {
		_, err := h.contract.LinkBadgeActivity(h.ctx, "lifecycle-series", "lifecycle", "1")
		require.NoError(t, err)
	})
	h.tx("activate", admin, base.Add(3*time.Minute), func() {
		_, err := h.contract.ActivateBadgeSeries(h.ctx, "lifecycle-series")
		require.NoError(t, err)
	})
	h.tx("pause", admin, base.Add(4*time.Minute), func() {
		series, err := h.contract.PauseBadgeSeries(h.ctx, "lifecycle-series", digest("incident"))
		require.NoError(t, err)
		require.Equal(t, "PAUSED", series.Status)
	})
	h.tx("resume", admin, base.Add(5*time.Minute), func() {
		series, err := h.contract.ResumeBadgeSeries(h.ctx, "lifecycle-series")
		require.NoError(t, err)
		require.Equal(t, "ACTIVE", series.Status)
	})
	h.tx("close", admin, base.Add(6*time.Minute), func() {
		series, err := h.contract.CloseBadgeSeries(h.ctx, "lifecycle-series")
		require.NoError(t, err)
		require.Equal(t, "CLOSED", series.Status)
	})
	h.tx("cannot-resume", admin, base.Add(7*time.Minute), func() {
		_, err := h.contract.ResumeBadgeSeries(h.ctx, "lifecycle-series")
		require.Error(t, err)
	})
	h.tx("finalize", admin, base.Add(8*time.Minute), func() {
		series, err := h.contract.FinalizeBadgeSeries(h.ctx, "lifecycle-series")
		require.NoError(t, err)
		require.Equal(t, "FINALIZED", series.Status)
	})
	h.tx("missing-view", admin, base.Add(9*time.Minute), func() {
		_, err := h.contract.GetMyBadge(h.ctx, "does-not-exist")
		require.ErrorContains(t, err, "series not found")
	})

	_ = fmt.Sprintf("%s", base) // Keep the test fixture deliberately deterministic.
}

func TestCancelledActivityCannotCreateBadgeEligibility(t *testing.T) {
	h := newActivityHarness()
	base := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	admin := activityIdentity("PlatformMSP", "admin01", "admin")
	organizer := activityIdentity("OrganizerMSP", "organizer01", "organizer")
	operator := activityIdentity("PlatformMSP", "operator01", "operator")

	h.tx("cancelled-activity", organizer, base, func() {
		_, err := h.contract.CreateActivity(h.ctx, "cancelled-badge-event", "basketball", "Cancelled", "1",
			timestamp(base.Add(time.Hour)), timestamp(base.Add(2*time.Hour)), timestamp(base.Add(4*time.Hour)))
		require.NoError(t, err)
	})
	h.tx("cancelled-series", admin, base.Add(time.Minute), func() {
		_, err := h.contract.CreateBadgeSeries(h.ctx, badgeSeriesJSON(base, "cancelled-series", 1))
		require.NoError(t, err)
		_, err = h.contract.LinkBadgeActivity(h.ctx, "cancelled-series", "cancelled-badge-event", "1")
		require.NoError(t, err)
		_, err = h.contract.ActivateBadgeSeries(h.ctx, "cancelled-series")
		require.NoError(t, err)
	})
	h.tx("cancel-after-activation", organizer, base.Add(2*time.Minute), func() {
		activity, err := loadActivity(h.ctx, "cancelled-badge-event")
		require.NoError(t, err)
		activity.Status = "CANCELLED"
		require.NoError(t, saveActivity(h.ctx, activity))
	})
	h.tx("cancelled-no-eligibility", operator, base.Add(2*time.Hour), func() {
		activity, err := loadActivity(h.ctx, "cancelled-badge-event")
		require.NoError(t, err)
		results, err := recordBadgeEligibility(h.ctx, activity, &Ticket{TicketID: "cancelled-ticket"}, "student01", base.Add(2*time.Hour))
		require.NoError(t, err)
		require.Empty(t, results)
		_, found, err := loadBadgeEligibility(h.ctx, "cancelled-series", "student01")
		require.NoError(t, err)
		require.False(t, found)
	})
}
