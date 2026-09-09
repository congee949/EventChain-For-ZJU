import { expect, test } from '@playwright/test';

const canonicalRef = 'b_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abc';
function makeSeries() {
  const now = Date.now();
  return { seriesId: 'series-2026', seasonId: '2026', categoryId: 'basketball', title: '院际篮球纪念章', description: '完成活动签到后可领取的赛季纪念。', status: 'ACTIVE', maxSupply: 100, issuedCount: 0, assetUri: '/badges/basketball-2026-demo/v1/badge.png', assetSha256: 'a'.repeat(64), metadataSha256: 'b'.repeat(64), eligibilityPolicyHash: 'c'.repeat(64), claimOpenAt: new Date(now - 60_000).toISOString(), claimCloseAt: new Date(now + 3_600_000).toISOString() };
}

async function installBadgeApi(page, { pending = false, loseFirstClaim = false } = {}) {
  const series = makeSeries();
  const state = { series, awarded: false, pending, claimRequests: [], statusRefs: [], firstClaimLost: false };
  await page.route('**/api/v2/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace('/api/v2', '');
    const json = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(status >= 400 ? data : { error: false, data }) });
    if (path === '/auth/login' && request.method() === 'POST') return json({ token: 'student-token', userId: 'acct_student', name: '学生一', role: 'student' });
    if (!request.headers().authorization) return json({ error: true, code: 'UNAUTHORIZED', message: '请先登录' }, 401);
    if (path === '/finance/wallet') return json({ aAvailable: 0, categories: {} });
    if (path === '/finance/categories') return json([]);
    if (path === '/finance/markets') return json([]);
    if (path === '/finance/offers') return json([]);
    if (path === '/activities') return json([]);
    if (path === '/badges/series') return json([series]);
    if (path === `/badges/series/${series.seriesId}`) return json(series);
    if (path === `/badges/series/${series.seriesId}/activities`) return json([]);
    if (path === `/badges/series/${series.seriesId}/my-award`) return json(state.awarded ? { series, award: { instanceId: 'instance-1', serialNumber: 1 }, instance: { instanceId: 'instance-1', serialNumber: 1 }, eligibility: { status: 'CLAIMABLE' } } : { series, award: null, instance: null, eligibility: { status: 'CLAIMABLE' } });
    if (path === '/badges/my-awards') return json(state.awarded ? [{ series, award: { instanceId: 'instance-1', serialNumber: 1 }, instance: { instanceId: 'instance-1', serialNumber: 1 } }] : []);
    if (path === `/badges/series/${series.seriesId}/claim` && request.method() === 'POST') {
      const key = request.headers()['idempotency-key'] || '';
      state.claimRequests.push(key);
      if (loseFirstClaim && !state.firstClaimLost) { state.firstClaimLost = true; return route.abort('failed'); }
      if (state.pending) return json({ seriesId: series.seriesId, refId: canonicalRef, status: 'UNKNOWN' }, 202);
      state.awarded = true;
      return json({ series, award: { instanceId: 'instance-1', serialNumber: 1 }, instance: { instanceId: 'instance-1', serialNumber: 1 }, refId: canonicalRef }, 201);
    }
    if (path.startsWith(`/badges/series/${series.seriesId}/claims/`)) { const ref = path.split('/').at(-1); state.statusRefs.push(ref); state.pending = false; state.awarded = true; return json({ seriesId: series.seriesId, refId: ref, status: 'CONFIRMED' }); }
    return json({ error: true, code: 'NOT_FOUND', message: 'not found' }, 404);
  });
  return state;
}

async function login(page) {
  // Badge tests focus on the badge contract; seed the exact session shape
  // consumed by the router and auth store, while API calls still carry JWT.
  await page.addInitScript(() => {
    localStorage.setItem('ec_token', 'student-token');
    localStorage.setItem('ec_user', JSON.stringify({ userId: 'acct_student', name: '学生一', role: 'student' }));
  });
}

test('gallery and detail show the limited non-financial badge', async ({ page }) => {
  const state = await installBadgeApi(page); await login(page); await page.goto('/badges');
  await expect(page.getByRole('heading', { name: '赛季徽章', exact: true })).toBeVisible(); await expect(page.getByText(state.series.title)).toBeVisible();
  await page.locator(`a[href="/badges/${state.series.seriesId}"]`).click(); await expect(page.getByRole('heading', { name: '领取状态', exact: true })).toBeVisible(); await expect(page.getByText('不可交易 · 不可兑换 · 无现金价值')).toBeVisible(); await expect(page.locator('body')).not.toContainText(/转售|升值|投资回报/);
});

test('successful claim sends a request key and returns canonical b_ ref', async ({ page }) => {
  const state = await installBadgeApi(page); await login(page); await page.goto(`/badges/${state.series.seriesId}`); await page.getByRole('button', { name: '免费领取' }).click(); await page.getByRole('dialog').getByRole('button', { name: '确认领取' }).click(); await expect(page.getByRole('heading', { name: '已领取', exact: true })).toBeVisible();
  expect(state.claimRequests).toHaveLength(1); expect(state.claimRequests[0]).toMatch(/^[A-Za-z0-9._:-]{8,128}$/); expect(canonicalRef).toMatch(/^b_[a-z0-9_-]{7,63}$/);
});

test('202 pending claim polls the canonical b_ ref', async ({ page }) => {
  const state = await installBadgeApi(page, { pending: true }); await login(page); await page.goto(`/badges/${state.series.seriesId}`); await page.getByRole('button', { name: '免费领取' }).click(); await page.getByRole('dialog').getByRole('button', { name: '确认领取' }).click(); await expect(page.getByRole('heading', { name: '已领取', exact: true })).toBeVisible(); expect(state.statusRefs).toEqual([canonicalRef]);
});

test('request loss retries with the same client key', async ({ page }) => {
  const state = await installBadgeApi(page, { loseFirstClaim: true }); await login(page); await page.goto(`/badges/${state.series.seriesId}`); await page.getByRole('button', { name: '免费领取' }).click(); await page.getByRole('dialog').getByRole('button', { name: '确认领取' }).click(); await expect(page.getByRole('alert')).toBeVisible();
  await page.getByRole('button', { name: '免费领取' }).click(); await page.getByRole('dialog').getByRole('button', { name: '确认领取' }).click(); await expect(page.getByRole('heading', { name: '已领取', exact: true })).toBeVisible(); expect(state.claimRequests).toHaveLength(2); expect(state.claimRequests[0]).toBe(state.claimRequests[1]);
});

test('badge detail exposes no transfer, redeem, sell, or burn controls', async ({ page }) => {
  const state = await installBadgeApi(page); await login(page); await page.goto(`/badges/${state.series.seriesId}`); for (const word of ['转让', '兑换', '出售', '销毁']) await expect(page.getByText(word, { exact: true })).toHaveCount(0);
});
