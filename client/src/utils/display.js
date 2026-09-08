const labels = {
  role: {
    student: '学生',
    organizer: '活动组织者',
    verifier: '独立验证员',
    operator: '结算运营员',
    arbitrator: '仲裁员',
    admin: '系统管理员',
  },
  marketStatus: {
    CREATED: '待开放',
    OPEN: '预测开放',
    LOCKED: '已锁盘',
    RESULT_PROPOSED: '结果待确认',
    CHALLENGED: '争议处理中',
    PROVISIONAL_FINALIZED: '临时结算',
    FINALIZED: '已结算',
    PAUSED: '已暂停',
    VOID: '已作废',
  },
  activityStatus: {
    DRAFT: '草稿',
    APPLICATION_OPEN: '报名中',
    APPLICATION_CLOSED: '报名结束',
    DRAWN: '已抽签',
    ACTIVE: '进行中',
    COMPLETED: '已结束',
    CANCELLED: '已取消',
  },
  applicationStatus: {
    PENDING: '等待抽签',
    WON: '已中签',
    LOST: '未中签',
    CLAIMED: '已领票',
  },
  ticketStatus: {
    ISSUED: '有效',
    ACTIVE: '有效',
    USED: '已核销',
    CANCELLED: '已取消',
  },
  bucket: {
    PAID: '可兑回积分',
    BONUS: '奖励积分',
  },
  offerType: {
    VENUE: '场馆',
    EQUIPMENT: '器材',
    EVENT_PASS: '活动通行',
  },
};

function label(group, value) {
  return labels[group]?.[value] || value || '未知';
}

export const roleLabel = (value) => label('role', value);
export const marketStatusLabel = (value) => label('marketStatus', value);
export const activityStatusLabel = (value) => label('activityStatus', value);
export const applicationStatusLabel = (value) => label('applicationStatus', value);
export const ticketStatusLabel = (value) => label('ticketStatus', value);
export const bucketLabel = (value) => label('bucket', value);
export const offerTypeLabel = (value) => label('offerType', value);

export function readStoredJSON(key) {
  try {
    const value = localStorage.getItem(key);
    return value ? JSON.parse(value) : null;
  } catch {
    localStorage.removeItem(key);
    return null;
  }
}
