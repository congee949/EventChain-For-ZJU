import { Router } from 'express';
import jwt from 'jsonwebtoken';
import config from '../config/index.js';
import { createUser, findUser, hashPassword, verifyPassword } from '../services/wallet.js';
import { registerAndEnrollUser } from '../services/caService.js';
import { submitTransaction } from '../services/fabricGateway.js';

const router = Router();

// POST /api/v1/auth/register
// Body: { studentID, password, name, role? }  (role defaults to 'student')
//
// SECURITY: NOT FOR PRODUCTION. This endpoint is publicly accessible AND
// honors a `role` field from the request body, so anyone can self-register
// as `organizer` or `admin` and obtain a privileged JWT. The damage isn't
// just in-app: registerAndEnrollUser also enrolls a real Fabric CA identity
// under the corresponding MSP (PlatformMSP for admin, OrganizerMSP for
// organizer), so the attacker walks away with an actual on-chain admin cert
// in the server wallet. This is acceptable for the school project demo only.
// Before any real deployment, either:
//   (a) drop the role parameter and only create students here, then expose
//       a separate /admin/users endpoint behind admin auth, OR
//   (b) require an existing admin token to register privileged roles.
router.post('/register', async (req, res, next) => {
  try {
    const { studentID, password, name, role: requestedRole } = req.body;

    if (!studentID || !password || !name) {
      const err = new Error('studentID, password, name 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Validate role and map to corresponding MSP
    const role = requestedRole || 'student';
    const ROLE_TO_MSP = {
      student: 'StudentMSP',
      organizer: 'OrganizerMSP',
      admin: 'PlatformMSP',
    };
    const orgMSP = ROLE_TO_MSP[role];
    if (!orgMSP) {
      const err = new Error(`无效的角色: ${role}`);
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
    createUser(studentID, name, passwordHash, orgMSP, role);

    // 2. Register + enroll with Fabric CA → cert stored in wallet
    await registerAndEnrollUser(studentID, orgMSP);

    // 3. Mint 1000 tokens as registration bonus (students only)
    if (role === 'student') {
      await submitTransaction(studentID, config.fabric.chaincode.token, 'Mint', studentID, '1000');
    }

    // 4. Issue JWT
    const token = jwt.sign(
      { userId: studentID, orgMSP, role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.status(201).json({
      error: false,
      data: { token, userId: studentID, name, role, balance: role === 'student' ? 1000 : 0 },
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
