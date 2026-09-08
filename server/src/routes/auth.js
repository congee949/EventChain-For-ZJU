import { Router } from 'express';
import jwt from 'jsonwebtoken';
import crypto from 'node:crypto';
import config from '../config/index.js';
import { createUser, findUser, hashPassword, verifyPassword } from '../services/wallet.js';
import { registerAndEnrollUser } from '../services/caService.js';
import { rateLimit } from '../middleware/rateLimit.js';

const router = Router();

// POST /api/v2/auth/register
router.post('/register', rateLimit({ windowMs: 15 * 60_000, max: 5 }), async (req, res, next) => {
  try {
    if (!config.openStudentRegistration) {
      const err = new Error('公开注册已关闭；课程 Demo 账号由名单预置，以防女巫账号');
      err.code = 'FORBIDDEN';
      throw err;
    }
    const { studentID, password, name, role: requestedRole } = req.body;

    if (!studentID || !password || !name) {
      const err = new Error('studentID, password, name 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    if (requestedRole && requestedRole !== 'student') {
      const err = new Error('公开注册仅允许 student 角色');
      err.code = 'FORBIDDEN';
      throw err;
    }
    validateRegistrationInput(studentID, password, name);
    const role = 'student';
    const orgMSP = 'StudentMSP';

    // Check if user already exists in SQLite
    if (findUser(studentID)) {
      const err = new Error('该学号已注册');
      err.code = 'USER_EXISTS';
      throw err;
    }

    const accountID = createAccountID();
    const passwordHash = await hashPassword(password);
    await registerAndEnrollUser(accountID, orgMSP, role);
    createUser(accountID, studentID, name, passwordHash, orgMSP, role);

    // 4. Issue JWT
    const token = jwt.sign(
      { sub: accountID, userId: accountID, orgMSP, role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.status(201).json({
      error: false,
      data: { token, userId: accountID, name, role, balance: 0 },
    });
  } catch (err) {
    next(err);
  }
});

// Demo-only privileged account bootstrap. The shared key is deliberately
// separate from JWT auth and the route is disabled when DEMO_MODE=false.
router.post('/bootstrap', rateLimit({ windowMs: 15 * 60_000, max: 20 }), async (req, res, next) => {
  try {
    if (!config.demoMode || req.get('X-Demo-Bootstrap-Key') !== config.identity.demoBootstrapKey) {
      const err = new Error('bootstrap access denied');
      err.code = 'FORBIDDEN';
      throw err;
    }
    const { studentID, password, name, role } = req.body;
    validateRegistrationInput(studentID, password, name);
    const roleToMSP = {
      admin: 'PlatformMSP', operator: 'PlatformMSP', verifier: 'PlatformMSP',
      arbitrator: 'PlatformMSP', organizer: 'OrganizerMSP',
      student: 'StudentMSP',
    };
    const orgMSP = roleToMSP[role];
    if (!orgMSP) {
      const err = new Error('bootstrap role is invalid');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }
    if (findUser(studentID)) {
      const err = new Error('该登录账号已注册');
      err.code = 'USER_EXISTS';
      throw err;
    }
    const accountID = createAccountID();
    const passwordHash = await hashPassword(password);
    await registerAndEnrollUser(accountID, orgMSP, role);
    createUser(accountID, studentID, name, passwordHash, orgMSP, role);
    const token = jwt.sign(
      { sub: accountID, userId: accountID, orgMSP, role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );
    res.status(201).json({ error: false, data: { token, userId: accountID, name, role } });
  } catch (err) {
    next(err);
  }
});

// POST /api/v2/auth/login
// Body: { studentID, password }
router.post('/login', rateLimit({ windowMs: 15 * 60_000, max: 10, key: (req) => `${req.ip}:${String(req.body?.studentID || '').toLowerCase()}` }), async (req, res, next) => {
  try {
    const { studentID, password } = req.body;

    if (!studentID || !password) {
      const err = new Error('studentID 和 password 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const user = findUser(studentID);
    if (!user) {
      const err = new Error('学号或密码错误');
      err.code = 'INVALID_CREDENTIALS';
      throw err;
    }

    const valid = await verifyPassword(password, user.password);
    if (!valid) {
      const err = new Error('学号或密码错误');
      err.code = 'INVALID_CREDENTIALS';
      throw err;
    }

    const token = jwt.sign(
      { sub: user.account_id, userId: user.account_id, orgMSP: user.org_msp, role: user.role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.json({
      error: false,
      data: { token, userId: user.account_id, name: user.name, role: user.role },
    });
  } catch (err) {
    next(err);
  }
});

function createAccountID() {
  return `acct_${crypto.randomBytes(24).toString('hex')}`;
}

function validateRegistrationInput(loginID, password, name) {
  if (!loginID || !password || !name) {
    const err = new Error('studentID, password, name 均为必填');
    err.code = 'VALIDATION_ERROR';
    throw err;
  }
  if (String(loginID).length > 128 || String(password).length < 8 || String(password).length > 128 || String(name).length > 64) {
    const err = new Error('登录账号、密码或姓名长度不符合要求');
    err.code = 'VALIDATION_ERROR';
    throw err;
  }
}

export default router;
