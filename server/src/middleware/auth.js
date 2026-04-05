import jwt from 'jsonwebtoken';
import config from '../config/index.js';

// JWT verification middleware.
// Sets req.user = { userId, orgMSP } on success.
export function authenticate(req, _res, next) {
  const header = req.headers.authorization;
  if (!header || !header.startsWith('Bearer ')) {
    const err = new Error('未提供认证令牌');
    err.code = 'UNAUTHORIZED';
    return next(err);
  }

  const token = header.slice(7);
  try {
    const payload = jwt.verify(token, config.jwt.secret);
    req.user = { userId: payload.userId, orgMSP: payload.orgMSP, role: payload.role };
    next();
  } catch {
    const err = new Error('认证令牌无效或已过期');
    err.code = 'UNAUTHORIZED';
    next(err);
  }
}

// Require a specific role (e.g. 'organizer', 'admin').
export function requireRole(...roles) {
  return (req, _res, next) => {
    if (!roles.includes(req.user.role)) {
      const err = new Error('无权执行此操作');
      err.code = 'FORBIDDEN';
      return next(err);
    }
    next();
  };
}
