const buckets = new Map();

// Small in-process limiter for the single-instance course demo. A distributed
// deployment must replace this with a shared store (for example Redis).
export function rateLimit({ windowMs, max, key = (req) => req.ip }) {
  return (req, _res, next) => {
    const now = Date.now();
    const bucketKey = `${req.path}:${key(req)}`;
    let bucket = buckets.get(bucketKey);
    if (!bucket || now >= bucket.resetAt) {
      bucket = { count: 0, resetAt: now + windowMs };
      buckets.set(bucketKey, bucket);
    }
    bucket.count++;
    if (bucket.count > max) {
      const error = new Error('请求过于频繁，请稍后再试');
      error.code = 'RATE_LIMITED';
      error.status = 429;
      return next(error);
    }
    if (buckets.size > 10_000) {
      for (const [entryKey, entry] of buckets) {
        if (now >= entry.resetAt) buckets.delete(entryKey);
      }
    }
    next();
  };
}
