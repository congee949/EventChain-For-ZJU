const HIDDEN_TEST_SEASONS = new Set(['2026-e2e', '2026-fabric-test']);

export function isVisibleBadgeSeries(series) {
  return Boolean(series) && series.status !== 'DRAFT' && !HIDDEN_TEST_SEASONS.has(series.seasonId);
}

const HIDDEN_ACTIVITY_PREFIXES = ['badge-fab-', 'bulk-', 'badge-e2e-'];

export function isVisibleActivity(activity) {
  return Boolean(activity) && !HIDDEN_ACTIVITY_PREFIXES.some((prefix) => activity.id?.startsWith(prefix));
}
