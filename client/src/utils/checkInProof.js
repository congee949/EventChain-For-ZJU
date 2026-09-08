export function parseCheckInProof(rawValue) {
  const raw = typeof rawValue === 'string' ? rawValue.trim() : '';
  if (!raw) throw new Error('请扫描或粘贴签到二维码内容');

  let decoded;
  try {
    decoded = JSON.parse(raw);
  } catch {
    throw new Error('无法识别此二维码，请使用 EventChain 动态票据');
  }

  if (!decoded || typeof decoded !== 'object' || Array.isArray(decoded)) {
    throw new Error('二维码内容格式不正确');
  }

  const ticketId = typeof decoded.ticketId === 'string' ? decoded.ticketId.trim() : '';
  const secret = typeof decoded.secret === 'string' ? decoded.secret.trim() : '';
  const timeSlice = decoded.timeSlice;

  if (!ticketId || ticketId.length > 256) throw new Error('二维码缺少有效票据编号');
  if (secret.length < 32 || secret.length > 256) throw new Error('二维码缺少有效票据密钥');
  if (!Number.isSafeInteger(timeSlice)) throw new Error('二维码缺少有效时间片');

  return { ticketId, secret, timeSlice };
}
