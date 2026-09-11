#import "template.typ": *

// 先修改封面信息，再按需要复制 #slide(...) 页面。
#show: presentation.with(
  title: "EventChain 课程展示",
  short-title: "EventChain",
  author: "小组成员",
  date: [2026-09],
  version: "G 交付包",
  total-slides: 3,
  footer: [区块链课程展示],
)

#title-slide(
  title: [EventChain],
  subtitle: [区块链课程项目展示],
  author: [小组成员],
  date: [2026年9月],
)

#slide(
  title: [项目概览],
  subtitle: [在这里写项目要解决的问题],
  section: [01 项目概览],
)[
  #subhead([核心内容])
  - 在这里概括项目背景和目标
  - 在这里说明系统的主要功能
  - 在这里放一张系统结构图或流程图
]

#slide(
  title: [实现与演示],
  subtitle: [在这里写技术方案或演示结果],
  section: [02 实现与演示],
)[
  #two-col(
    [
      #subhead([实现])
      - 在这里说明链上和链下部分
      - 在这里说明关键的数据流
    ],
    [
      #subhead([结果])
      - 在这里放演示截图或测试结果
      - 保持每页只有一个要讲清楚的重点
    ]
  )
]
