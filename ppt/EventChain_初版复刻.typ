// EventChain 课程展示初版复刻
// 视觉基准：EventChain_课程展示_讲解版.pdf

#set document(
  title: "EventChain 课程展示初版复刻",
  author: "EventChain 课程项目组",
)

#set page(
  width: 960pt,
  height: 540pt,
  margin: 0pt,
  fill: white,
)

#set text(
  font: ("Microsoft YaHei", "Noto Sans SC", "Arial"),
  fill: rgb("#18354C"),
  lang: "zh",
)

#let navy = rgb("#18354C")
#let blue = rgb("#2475A8")
#let muted = rgb("#486477")
#let card-fill = rgb("#EFF5FA")
#let card-border = rgb("#BBD5E7")
#let shadow-fill = rgb("#D3DBE0")

#let flow-card(number, label, size: 20pt) = box(
  width: 200pt,
  height: 124pt,
)[
  #place(top + left, dx: 2pt, dy: 5pt)[
    #rect(
      width: 200pt,
      height: 118pt,
      radius: 18pt,
      fill: shadow-fill,
    )
  ]
  #place(top + left)[
    #rect(
      width: 200pt,
      height: 118pt,
      radius: 18pt,
      fill: card-fill,
      stroke: 1pt + card-border,
    )
  ]
  #place(top + left, dx: 16pt, dy: 17pt)[
    #text(size: 14pt, weight: "regular", fill: blue, number)
  ]
  #place(top + left, dx: 16pt, dy: 58pt)[
    #box(width: 170pt)[
      #set par(leading: 0.24em)
      #text(size: size, weight: "bold", fill: navy, label)
    ]
  ]
]

#let arrow-cell = box(width: 30pt, height: 124pt)[
  #place(top + center, dy: 48pt)[
    #text(size: 18pt, fill: blue)[→]
  ]
]

#let flow-row(cards, arrows: true) = {
  if arrows {
    grid(
      columns: (200pt, 30pt, 200pt, 30pt, 200pt, 30pt, 200pt),
      ..cards.enumerate().map(((index, card)) => {
        if index == 0 { (card,) } else { (arrow-cell, card) }
      }).flatten(),
    )
  } else {
    grid(
      columns: (200pt, 200pt, 200pt, 200pt),
      column-gutter: 30pt,
      ..cards,
    )
  }
}

#let course-slide(
  number,
  section,
  title,
  evidence,
  body,
  note: none,
) = page[
  #place(top + left, dx: 50pt, dy: 29pt)[
    #text(size: 14pt, weight: "bold", fill: blue)[#number / #section]
  ]

  #place(top + left, dx: 50pt, dy: 72pt)[
    #text(size: 30pt, weight: "bold", fill: navy, title)
  ]

  #body

  #if note != none {
    place(top + left, dx: 54pt, dy: 365pt)[
      #box(width: 852pt)[
        #text(size: 17.5pt, fill: navy, note)
      ]
    ]
  }

  #place(top + left, dx: 54pt, dy: 477pt)[
    #text(size: 13pt, fill: muted)[证据： #evidence]
  ]
]

#let standard-body(cards, arrows: true) = place(
  top + left,
  dx: 46pt,
  dy: 190pt,
  flow-row(cards, arrows: arrows),
)

#course-slide(
  [01],
  [项目定位],
  [参与过程可复核，不只显示成功],
  [evidence-2026-09-10/prediction-replay.json],
  standard-body((
    flow-card([01], [封闭积分]),
    flow-card([02], [预测仓位]),
    flow-card([03], [票务核销]),
    flow-card([04], [赛季徽章]),
  )),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [02],
  [问题],
  [资格、抽签和奖励需要共享证据],
  [evidence-2026-09-11/draw-current-student-readback.json],
  standard-body((
    flow-card([01], [输入不透明]),
    flow-card([02], [结果难复算]),
    flow-card([03], [重复核销风险]),
    flow-card([04], [共同回执]),
  ), arrows: false),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [03],
  [角色地图],
  [角色分工避免单方完成全部步骤],
  [evidence-2026-09-10/ticket-application-receipt.json],
  [
    #standard-body((
      flow-card([01], [student 申请]),
      flow-card([02], [verifier 承诺]),
      flow-card([03], [organizer 抽签]),
      flow-card([04], [operator 核销]),
    ), arrows: false)
    #place(top + left, dx: 54pt, dy: 337pt)[
      #text(size: 18pt, weight: "bold")[admin 管理员：配置课程基线、权限与徽章系列]
    ]
  ],
  note: [五类角色共同参与；前端提示、后端鉴权与链码证书校验相互配合。],
)

#course-slide(
  [04],
  [积分系统],
  [A 可兑换 B_paid，奖励积分独立发放],
  [evidence-2026-09-10/wallet-after-convert.txt],
  [
    #place(top + left, dx: 46pt, dy: 118pt)[
      #image("assets/wallet-before-convert.png", width: 420pt, height: 315pt, fit: "cover")
    ]
    #place(top + left, dx: 505pt, dy: 175pt)[
      #box(width: 390pt)[
        #set par(leading: 0.28em)
        #text(size: 21pt, fill: navy)[
          真实兑换 1 A → 1 B_paid \
          A：975.999999 → 974.999999 \
          羽毛球 B_paid：20 → 21 \
          截图为操作前余额 \
          B_bonus 不参与兑换
        ]
      ]
    ]
  ],
)

#course-slide(
  [05],
  [预测市场],
  [公开快照与本人仓位分开读取],
  [evidence-2026-09-10/prediction-readback.json],
  standard-body((
    flow-card([01], [公开延迟快照]),
    flow-card([02], [B_bonus 投入]),
    flow-card([03], [私有仓位 P]),
    flow-card([04], [本人证书读回]),
  )),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [06],
  [承诺-揭示抽签],
  [先承诺输入，再揭示并读回结果],
  [evidence-2026-09-11/draw-current-student-readback.json],
  standard-body((
    flow-card([01], [verifier: H(seed)], size: 19pt),
    flow-card([02], [学生申请]),
    flow-card([03], [截止后 reveal]),
    flow-card([04], [本人 WON/LOST], size: 18pt),
  )),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [07],
  [票务大厅],
  [票据秘密不应出现在动态二维码里],
  [evidence-2026-09-11/draw-ticket-readback.webm],
  standard-body((
    flow-card([01], [申请]),
    flow-card([02], [中签]),
    flow-card([03], [领取票据]),
    flow-card([04], [HMAC 时间片]),
  )),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [08],
  [签到奖励],
  [核销成功不代表本次一定发奖],
  [evidence-2026-09-10/checkin-browser-receipt.json],
  standard-body((
    flow-card([01], [200 核销成功], size: 19pt),
    flow-card([02], [GRANTED 已发放], size: 18pt),
    flow-card([03], [LIMIT 未发放], size: 19pt),
    flow-card([04], [409 重放拒绝], size: 19pt),
  ), arrows: false),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [09],
  [赛季徽章],
  [徽章是纪念凭证，不改变积分与仓位],
  [evidence-2026-09-10/badge-financial-invariants.json],
  standard-body((
    flow-card([01], [免费领取]),
    flow-card([02], [固定 maxSupply], size: 18pt),
    flow-card([03], [No. 0001 / 2], size: 18pt),
    flow-card([04], [不可交易兑换], size: 19pt),
  ), arrows: false),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [10],
  [技术架构],
  [隐私边界是组织级，不是绝对匿名],
  [evidence-2026-09-10/activity-committed-latest.json],
  standard-body((
    flow-card([01], [Vue 浏览器]),
    flow-card([02], [Express 身份], size: 19pt),
    flow-card([03], [Fabric 双链码], size: 19pt),
    flow-card([04], [MSP + PDC], size: 19pt),
  )),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [11],
  [证据梯],
  [写入、读回、重放是三类独立证据],
  [evidence-2026-09-10/prediction-replay.json],
  standard-body((
    flow-card([01], [提交]),
    flow-card([02], [交易回执]),
    flow-card([03], [对象 ID 读回], size: 19pt),
    flow-card([04], [原键重放]),
  )),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [12],
  [边界与结论],
  [账本约束明确，但线下治理仍然必要],
  [evidence-2026-09-10/plan-audit.md],
  standard-body((
    flow-card([01], [封闭积分]),
    flow-card([02], [组织级隐私]),
    flow-card([03], [非绝对公平]),
    flow-card([04], [未覆盖项明示]),
  ), arrows: false),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)

#course-slide(
  [13],
  [附录：操作顺序],
  [现场按角色顺序操作，不重置账本],
  [evidence-2026-09-10/ticket-network-status.txt],
  standard-body((
    flow-card([01], [student \ student-demo], size: 18pt),
    flow-card([02], [verifier \ verifier01], size: 18pt),
    flow-card([03], [organizer \ organizer01], size: 18pt),
    flow-card([04], [operator \ operator01], size: 18pt),
  )),
  note: [student-demo 为脱敏展示别名，不是登录账号。真实账号由课程管理渠道提供；勿使用本页别名登录。],
)

#course-slide(
  [14],
  [附录：常见问题],
  [回答质疑时，说明证据和限制],
  [evidence-2026-09-10/badge-financial-invariants.json],
  standard-body((
    flow-card([01], [能否兑现金？不能], size: 17pt),
    flow-card([02], [绝对公平？ \ 不能保证], size: 17pt),
    flow-card([03], [重复领取？ \ 同一实例], size: 17pt),
    flow-card([04], [隐私？组织级边界], size: 17pt),
  ), arrows: false),
  note: [验证方式：操作角色、对象编号与状态共同核对；不以页面成功提示替代链上证据。],
)
