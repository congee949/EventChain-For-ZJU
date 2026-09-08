import { expect, test } from '@playwright/test';

const MARKET_ID = 'basketball-final';
const ACTIVITY_TITLE = '院际篮球决赛';
const TICKET_ID = 'ticket-basketball-final-student-01';

function apiData(data) {
  return { error: false, data };
}

async function installApiScenario(page, { expired = false } = {}) {
  const future = (minutes) => new Date(Date.now() + minutes * 60_000).toISOString();
  const market = {
    marketId: MARKET_ID,
    eventId: MARKET_ID,
    categoryId: 'basketball',
    outcomes: [
      { id: 'home', label: '主队胜' },
      { id: 'draw', label: '平局' },
      { id: 'away', label: '客队胜' },
    ],
    outcomeProbabilityBps: { home: 5000, draw: 2000, away: 3000 },
    closeAt: future(expired ? -60 : 60),
    stakeBucket: 'BONUS',
    marketCap: 4_000_000_000,
    displayedPool: 120_000_000,
    participantCount: 5,
    status: 'OPEN',
  };
  const activity = {
    id: MARKET_ID,
    categoryId: 'basketball',
    title: ACTIVITY_TITLE,
    capacity: 3,
    applicationCount: 1,
    applicationCloseAt: future(expired ? -30 : 120),
    startsAt: future(180),
    endsAt: future(300),
    status: 'APPLICATION_OPEN',
  };
  const offer = {
    offerId: 'basketball-court',
    categoryId: 'basketball',
    offerType: 'VENUE',
    title: '篮球馆 1 小时',
    pricePaidB: 50_000_000,
    cancellationBps: 15_000,
    guaranteeAvailableA: 500_000_000,
    inventory: 20,
  };
  const wallet = {
    aAvailable: 1_000_000_000,
    categories: {
      basketball: {
        paid: { available: 50_000_000 },
        bonus: { available: 300_000_000 },
      },
    },
  };
  const state = {
    applicationStatus: null,
    ticket: null,
    claimedSecret: '',
    position: null,
    checkInProof: null,
  };

  await page.route('**/api/v2/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace('/api/v2', '');
    const method = request.method();
    const json = (data, status = 200) => route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(status >= 400 ? data : apiData(data)),
    });
    const body = () => request.postDataJSON();

    if (method === 'POST' && path === '/auth/login') {
      const credentials = body();
      if (credentials.studentID === '3220100001') {
        return json({ token: 'student-token', userId: 'student-01', name: '学生一', role: 'student' });
      }
      if (credentials.studentID === 'operator01') {
        return json({ token: 'operator-token', userId: 'operator-01', name: '结算运营员', role: 'operator' });
      }
      return json({ error: true, code: 'UNAUTHORIZED', message: '账号或密码错误' }, 401);
    }

    const authorization = request.headers().authorization;
    const expectedToken = path === '/activities/check-in/verify'
      ? 'Bearer operator-token'
      : method === 'POST'
        ? 'Bearer student-token'
        : null;
    const authorized = expectedToken
      ? authorization === expectedToken
      : ['Bearer student-token', 'Bearer operator-token'].includes(authorization);
    if (!authorized) {
      return json({ error: true, code: 'UNAUTHORIZED', message: '测试场景收到错误的登录身份' }, 401);
    }

    if (method === 'GET' && path === '/finance/wallet') return json(wallet);
    if (method === 'GET' && path === '/finance/categories') return json([{ id: 'basketball', name: '篮球' }]);
    if (method === 'GET' && path === '/finance/markets') return json([market]);
    if (method === 'GET' && path === '/finance/offers') return json([offer]);
    if (method === 'GET' && path === `/finance/markets/${MARKET_ID}`) return json(market);
    if (method === 'GET' && path === '/activities') return json([activity]);

    if (method === 'POST' && path === `/finance/markets/${MARKET_ID}/positions`) {
      state.position = body();
      return json({ positionId: 'position-01', ...state.position });
    }

    if (method === 'GET' && path === `/activities/${MARKET_ID}/application`) {
      if (!state.applicationStatus) {
        return json({ error: true, code: 'NOT_FOUND', message: '尚未报名' }, 404);
      }
      return json({ activityId: MARKET_ID, accountId: 'student-01', status: state.applicationStatus });
    }

    if (method === 'POST' && path === `/activities/${MARKET_ID}/applications`) {
      state.applicationStatus = 'PENDING';
      activity.applicationCount += 1;
      return json({ activityId: MARKET_ID, accountId: 'student-01', status: 'PENDING' });
    }

    if (method === 'GET' && path === `/activities/${MARKET_ID}/ticket`) {
      if (!state.ticket) return json({ error: true, code: 'NOT_FOUND', message: '尚未领票' }, 404);
      return json(state.ticket);
    }

    if (method === 'POST' && path === `/activities/${MARKET_ID}/ticket`) {
      state.claimedSecret = body().secret;
      state.applicationStatus = 'CLAIMED';
      state.ticket = { ticketId: TICKET_ID, activityId: MARKET_ID, status: 'ISSUED' };
      return json({ ticket: state.ticket });
    }

    if (method === 'POST' && path === '/activities/check-in/verify') {
      state.checkInProof = body();
      const valid = state.checkInProof.ticketId === TICKET_ID
        && state.checkInProof.secret === state.claimedSecret
        && Number.isSafeInteger(state.checkInProof.timeSlice);
      if (!valid) return json({ error: true, code: 'INVALID_PROOF', message: '签到凭证无效' }, 400);
      state.ticket.status = 'USED';
      return json({
        receipt: {
          ticketId: TICKET_ID,
          activityId: MARKET_ID,
          categoryId: 'basketball',
          checkedInAt: new Date().toISOString(),
        },
        rewardPending: false,
        reward: { amount: 5_000_000 },
      });
    }

    return json({ error: true, code: 'UNEXPECTED_REQUEST', message: `${method} ${path}` }, 500);
  });

  return state;
}

async function login(page, studentID, password) {
  await page.goto('/login');
  await page.locator('input[autocomplete="username"]').fill(studentID);
  await page.locator('input[autocomplete="current-password"]').fill(password);
  await page.getByRole('button', { name: '登录', exact: true }).click();
}

test('登录 → 建立仓位 → 报名 → 领票 → 签到', async ({ page }) => {
  const state = await installApiScenario(page);

  await test.step('学生登录并建立仓位', async () => {
    await login(page, '3220100001', 'eventchain-student-2026');
    await expect(page.getByRole('heading', { name: '预测市场进度' })).toBeVisible();

    await page.locator('.market-row').filter({ hasText: ACTIVITY_TITLE }).click();
    await expect(page.getByRole('heading', { name: '建立预测仓位' })).toBeVisible();
    await page.getByRole('button', { name: /主队胜/ }).click();
    await page.locator('.position-panel .el-input-number input').fill('25');
    await page.getByRole('button', { name: '确认建立仓位' }).click();

    await expect(page.getByText('仓位已写入私有账本')).toBeVisible();
    expect(state.position).toEqual({ outcomeId: 'home', amount: 25 });
  });

  await test.step('学生报名，抽签推进后领取票据', async () => {
    await page.getByRole('link', { name: '票务大厅', exact: true }).click();
    await expect(page.getByRole('heading', { name: '活动票务大厅' })).toBeVisible();
    await page.getByRole('button', { name: '提交抽签申请' }).click();
    await expect(page.locator('.application-list .app-status')).toHaveText('等待抽签');

    // 模拟组织方在报名截止后完成链上抽签；刷新后学生看到中签结果。
    state.applicationStatus = 'WON';
    await page.reload();
    await expect(page.locator('.application-list .app-status')).toHaveText('已中签');
    await page.getByRole('button', { name: '领取票据' }).click();

    await expect(page.getByText(TICKET_ID)).toBeVisible();
    await expect(page.locator('.qrcode-canvas')).toBeVisible();
    expect(state.claimedSecret).toMatch(/^[0-9a-f]{64}$/);
  });

  await test.step('运营员登录并核验动态票据', async () => {
    await page.getByRole('button', { name: '退出' }).click();
    await login(page, 'operator01', 'eventchain-operator-2026');
    await page.getByRole('link', { name: '签到核验', exact: true }).click();
    await expect(page.getByRole('heading', { name: '签到扫码' })).toBeVisible();

    const proof = JSON.stringify({
      ticketId: TICKET_ID,
      secret: state.claimedSecret,
      timeSlice: Math.floor(Date.now() / 30_000),
    });
    await page.getByPlaceholder(/粘贴 \{"ticketId"/).fill(proof);
    await page.getByRole('button', { name: '提交核验' }).click();

    await expect(page.getByText('签到成功，奖励积分已发放。')).toBeVisible();
    await expect(page.getByText(TICKET_ID)).toBeVisible();
    await expect(page.getByText('已发放 5 B_bonus')).toBeVisible();
    expect(state.checkInProof).toEqual(JSON.parse(proof));
  });
});

test('截止状态、余额提示和运营员界面保持一致', async ({ page }) => {
  await installApiScenario(page, { expired: true });

  await login(page, '3220100001', 'eventchain-student-2026');
  await page.setViewportSize({ width: 390, height: 844 });
  for (const name of ['赛事市场', '票务大厅', '服务兑换', 'A / B 钱包', '我的']) {
    const box = await page.getByRole('link', { name, exact: true }).boundingBox();
    expect(box).not.toBeNull();
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(390);
  }
  await expect(page.locator('.market-row').filter({ hasText: ACTIVITY_TITLE })).toContainText('待运营方锁盘');

  await page.getByRole('link', { name: '票务大厅', exact: true }).click();
  await expect(page.locator('.status-pill--waiting')).toHaveText('报名已截止，等待抽签');

  await page.getByRole('link', { name: '服务兑换', exact: true }).click();
  await expect(page.getByText('当前可用 50 B_paid')).toBeVisible();
  await expect(page.getByText('先选择预约时间')).toBeVisible();
  await expect(page.getByRole('button', { name: '兑换', exact: true })).toBeDisabled();

  await page.setViewportSize({ width: 700, height: 900 });
  await page.getByRole('button', { name: '退出' }).click();
  await login(page, 'operator01', 'eventchain-operator-2026');
  await page.getByRole('link', { name: '运营台', exact: true }).click();
  await expect(page.getByRole('heading', { name: '市场动作' })).toBeVisible();
  await expect(page.getByText('结果编号', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: '抽签与活动状态' })).toHaveCount(0);
  await expect(page.getByRole('link', { name: '运营台', exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
