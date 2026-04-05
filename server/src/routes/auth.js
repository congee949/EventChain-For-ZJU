import { Router } from 'express';
import jwt from 'jsonwebtoken';
import config from '../config/index.js';
import { createUser, findUser, hashPassword, verifyPassword } from '../services/wallet.js';
import { registerAndEnrollUser } from '../services/caService.js';
import { submitTransaction } from '../services/fabricGateway.js';

const router = Router();

// POST /api/v1/auth/register
// Body: { studentID, password, name }
router.post('/register', async (req, res, next) => {
  try {
    const { studentID, password, name } = req.body;

    if (!studentID || !password || !name) {
      const err = new Error('studentID, password, name 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Check if user already exists in SQLite
    if (findUser(studentID)) {
      const err = new Error('该学号已注册');
      err.code = 'USER_EXISTS';
      throw err;
    }

    // 1. Hash password and store in SQLite
    const passwordHash = await hashPassword(password);
    createUser(studentID, name, passwordHash, 'StudentMSP', 'student');

    // 2. Register + enroll with Fabric CA → cert stored in wallet
    await registerAndEnrollUser(studentID, 'StudentMSP');

    // 3. Mint 1000 tokens as registration bonus
    await submitTransaction(studentID, config.fabric.chaincode.token, 'Mint', studentID, '1000');

    // 4. Issue JWT
    const token = jwt.sign(
      { userId: studentID, orgMSP: 'StudentMSP', role: 'student' },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.status(201).json({
      error: false,
      data: { token, userId: studentID, name, balance: 1000 },
    });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/auth/login
// Body: { studentID, password }
router.post('/login', async (req, res, next) => {
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
      { userId: user.user_id, orgMSP: user.org_msp, role: user.role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.json({
      error: false,
      data: { token, userId: user.user_id, name: user.name, role: user.role },
    });
  } catch (err) {
    next(err);
  }
});

export default router;
