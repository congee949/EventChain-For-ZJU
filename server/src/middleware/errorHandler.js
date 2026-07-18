// Maps chaincode / application error codes to HTTP status + Chinese message.

const ERROR_MAP = {
  INSUFFICIENT_BALANCE: { status: 400, message: '余额不足' },
  EVENT_NOT_FOUND: { status: 404, message: '赛事不存在' },
  INVALID_STATUS: { status: 400, message: '赛事状态不允许此操作' },
  BET_CLOSED: { status: 400, message: '预测已关闭' },
  ALREADY_APPLIED: { status: 409, message: '已提交过申请' },
  TICKET_NOT_FOUND: { status: 404, message: '票据不存在' },
  UNAUTHORIZED: { status: 401, message: '未登录或登录已过期' },
  FORBIDDEN: { status: 403, message: '无权执行此操作' },
  USER_EXISTS: { status: 409, message: '该学号已注册' },
  INVALID_CREDENTIALS: { status: 401, message: '学号或密码错误' },
  VALIDATION_ERROR: { status: 400, message: '请求参数不合法' },
  RATE_LIMITED: { status: 429, message: '请求过于频繁，请稍后再试' },
};

export function errorHandler(err, _req, res, _next) {
  console.error('[ErrorHandler]', err);

  // Application-level coded errors (thrown as { code, message })
  if (err.code && ERROR_MAP[err.code]) {
    const mapped = ERROR_MAP[err.code];
    return res.status(mapped.status).json({
      error: true,
      code: err.code,
      message: err.message || mapped.message,
    });
  }

  // Fabric Gateway errors often include a "details" array
  if (err.details && Array.isArray(err.details)) {
    const detail = err.details[0]?.message || err.message;
    // Try to extract a known code from the chaincode error string
    for (const [code, meta] of Object.entries(ERROR_MAP)) {
      if (detail.includes(code)) {
        return res.status(meta.status).json({
          error: true,
          code,
          message: meta.message,
        });
      }
    }
  }

  // Fallback
  const status = err.status || err.statusCode || 500;
  res.status(status).json({
    error: true,
    code: 'INTERNAL_ERROR',
    message: process.env.NODE_ENV === 'production' ? '服务器内部错误' : err.message,
  });
}

// Helper to throw coded errors from route handlers.
export function throwCoded(code, message) {
  const e = new Error(message || ERROR_MAP[code]?.message || code);
  e.code = code;
  throw e;
}
