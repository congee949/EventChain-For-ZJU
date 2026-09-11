// ==============================================================================
// TRSM 高性能计算答辩专用 Typst PPT 模板
// 风格：简洁大方 · 现代学术工程向 (Clean, Tech-oriented)
// 适配 Typst 0.13+ / 0.15+
// ==============================================================================

// --- 色彩系统 (Clean Tech & Engineering Palette) ---
#let tech-dark      = rgb("#0a192f")      // 科技深蓝黑 (封面、关键深色底)
#let tech-navy      = rgb("#0f2744")      // 军舰深蓝 (大标题、主结构强调)
#let tech-primary   = rgb("#0284c7")      // 科技青蓝 (主要聚焦色、高亮、状态条)
#let tech-accent    = rgb("#2563eb")      // 皇家科技蓝 (次级链接与按钮)
#let tech-cyan      = rgb("#0891b2")      // 电光深青色 (流程节点、徽章)
#let tech-text-main = rgb("#1e293b")      // 正文字体颜色 (深石板灰)
#let tech-dark-gray = tech-text-main      // 别名兼容
#let tech-text-sub  = rgb("#64748b")      // 次级文字灰 (辅助说明)
#let tech-muted-gray= tech-text-sub       // 别名兼容
#let tech-bg        = rgb("#f8fafc")      // 幻灯片主背景色 (柔和浅灰白)
#let tech-card-bg   = rgb("#ffffff")      // 卡片白色背景
#let tech-border    = rgb("#e2e8f0")      // 柔和卡片边框
#let tech-border-mid= rgb("#cbd5e1")      // 中度分割线边框

// 状态强调色
#let tech-success   = rgb("#059669")
#let tech-success-bg= rgb("#ecfdf5")
#let tech-warning   = rgb("#d97706")
#let tech-warning-bg= rgb("#fffbeb")
#let tech-danger    = rgb("#dc2626")
#let tech-danger-bg = rgb("#fef2f2")
#let tech-purple    = rgb("#7c3aed")
#let tech-purple-bg = rgb("#f5f3ff")

// 全局元数据状态
#let _meta = state("meta", (
  title: [TRSM 算法优化与工程实现],
  short-title: [TRSM V4 答辩],
  author: "答辩人",
  date: [2026-09],
  version: "V4",
  total-slides: 10,
  footer: [课程展示],
))

#let _slide-counter = counter("slide-page-counter")

// ==============================================================================
// 1. 全局 Presentation 文档环境
// ==============================================================================
#let presentation(
  title: [TRSM 算法优化与工程实现],
  short-title: [TRSM V4 答辩],
  author: "答辩人",
  date: [2026-09],
  version: "V4",
  total-slides: 10,
  footer: [课程展示],
  body
) = {
  // 文档元数据 (类型安全)
  let doc-title = if type(title) == str { title } else { "TRSM 算法优化与工程实现" }
  let doc-author = if type(author) == str { author } else { "答辩人" }
  set document(title: doc-title, author: doc-author)

  // 16:9 标准比例 (297mm x 167.06mm)
  set page(
    paper: "presentation-16-9",
    margin: (top: 2.3cm, bottom: 1.4cm, left: 1.8cm, right: 1.8cm),
    header-ascent: 0.65cm,
    footer-descent: 0.5cm,
    fill: tech-bg,
  )

  // 字体配置：优雅清晰的无衬线字体为主
  set text(
    font: ("Segoe UI", "Noto Sans SC", "Microsoft YaHei", "Arial"),
    size: 13.5pt,
    fill: tech-text-main,
    lang: "zh",
  )

  // 段落排版
  set par(leading: 0.62em, justify: false)

  // 列表符号样式
  set list(marker: text(fill: tech-primary, font: "Segoe UI", [▸ ]), spacing: 0.75em)

  // 代码块美化
  show raw: set text(font: ("Cascadia Code", "Consolas", "Courier New"), size: 0.88em)
  show raw.where(block: true): it => block(
    fill: rgb("#0f172a"),
    inset: 10pt,
    radius: 6pt,
    width: 100%,
    stroke: 1pt + rgb("#334155"),
    text(fill: rgb("#f1f5f9"), it)
  )

  // 数学公式默认样式
  show math.equation: set text(weight: "regular")

  // 初始化全局状态
  _meta.update((
    title: title,
    short-title: short-title,
    author: author,
    date: date,
    version: version,
    total-slides: total-slides,
    footer: footer,
  ))

  body
}

// ==============================================================================
// 2. 基础组件库 (UI Components)
// ==============================================================================

// 徽章 / 状态标签
#let badge(content, fill: rgb("#f1f5f9"), stroke: tech-border, color: tech-text-main) = {
  box(
    fill: fill,
    stroke: 0.6pt + stroke,
    radius: 4pt,
    inset: (x: 6pt, y: 2.5pt),
    baseline: 0%,
    text(size: 0.75em, weight: "medium", fill: color, content)
  )
}

#let badge-tech(content)    = badge(content, fill: rgb("#e0f2fe"), stroke: rgb("#bae6fd"), color: tech-primary)
#let badge-accent(content)  = badge(content, fill: rgb("#eff6ff"), stroke: rgb("#bfdbfe"), color: tech-accent)
#let badge-success(content) = badge(content, fill: tech-success-bg, stroke: rgb("#a7f3d0"), color: tech-success)
#let badge-warning(content) = badge(content, fill: tech-warning-bg, stroke: rgb("#fde68a"), color: tech-warning)
#let badge-danger(content)  = badge(content, fill: tech-danger-bg, stroke: rgb("#fecaca"), color: tech-danger)
#let badge-purple(content)  = badge(content, fill: tech-purple-bg, stroke: rgb("#ddd6fe"), color: tech-purple)

// 现代技术卡片 (白底微框，无表情包)
#let card(
  title: none,
  tag: none,
  fill: tech-card-bg,
  stroke: 1pt + tech-border,
  radius: 6pt,
  inset: 10pt,
  width: 100%,
  body
) = {
  block(
    fill: fill,
    stroke: stroke,
    radius: radius,
    inset: inset,
    width: width,
    [
      #if title != none or tag != none [
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          if title != none {
            text(weight: "bold", size: 1.05em, fill: tech-navy, title)
          },
          if tag != none { tag }
        )
        #v(3pt)
        #line(length: 100%, stroke: 0.6pt + tech-border)
        #v(4pt)
      ]
      #body
    ]
  )
}

// 强调提示框 (Callout - 纯净工程风格，不带 Emoji)
#let callout(
  type: "info", // "info", "success", "warning", "danger", "tech"
  title: none,
  body
) = {
  let (bar-color, bg-color, text-color) = if type == "success" {
    (tech-success, tech-success-bg, tech-success)
  } else if type == "warning" {
    (tech-warning, tech-warning-bg, tech-warning)
  } else if type == "danger" {
    (tech-danger, tech-danger-bg, tech-danger)
  } else if type == "tech" {
    (tech-purple, tech-purple-bg, tech-purple)
  } else {
    (tech-primary, rgb("#f0f9ff"), tech-primary)
  }

  block(
    fill: bg-color,
    stroke: (left: 3.5pt + bar-color, rest: 0.6pt + bar-color.lighten(65%)),
    radius: (right: 6pt),
    inset: (x: 10pt, y: 7pt),
    width: 100%,
    [
      #if title != none [
        #text(weight: "bold", fill: text-color, title)
        #v(2pt)
      ]
      #text(fill: tech-text-main, size: 0.94em, body)
    ]
  )
}

// 核心指标数据卡片 (Stat Card)
#let stat-card(
  value: "",
  unit: "",
  label: "",
  desc: none,
  color: tech-primary,
  fill: tech-card-bg,
  width: 100%
) = {
  block(
    fill: fill,
    stroke: 1pt + tech-border,
    radius: 6pt,
    inset: (x: 12pt, y: 9pt),
    width: width,
    [
      #text(weight: "black", size: 1.55em, fill: color, value)
      #if unit != "" [
        #h(3pt)
        #text(weight: "bold", size: 0.85em, fill: color.lighten(20%), unit)
      ]
      #v(2pt)
      #text(weight: "bold", size: 0.92em, fill: tech-navy, label)
      #if desc != none [
        #v(1pt)
        #text(size: 0.74em, fill: tech-text-sub, desc)
      ]
    ]
  )
}

// 步骤卡片 (Step Card)
#let step-card(num: "1", title: "", body: "", active: false) = {
  let main-color = if active { tech-primary } else { tech-navy }
  let bg = if active { rgb("#f0f9ff") } else { tech-card-bg }
  block(
    fill: bg,
    stroke: if active { 1.5pt + tech-primary } else { 1pt + tech-border },
    radius: 6pt,
    inset: 8pt,
    width: 100%,
    [
      #grid(
        columns: (auto, 1fr),
        gutter: 6pt,
        align: horizon,
        circle(radius: 8.5pt, fill: main-color, text(fill: white, size: 0.75em, weight: "bold", num)),
        text(weight: "bold", size: 0.9em, fill: main-color, title)
      )
      #v(3pt)
      #text(size: 0.8em, fill: tech-text-main, body)
    ]
  )
}

// 横向四步/多步流程组件
#let steps-row(steps, active-index: -1) = {
  let count = steps.len()
  let cols = ()
  for i in range(count) {
    cols.push(1fr)
  }
  grid(
    columns: cols,
    gutter: 8pt,
    ..steps.enumerate().map(((idx, it)) => {
      let is-act = (idx == active-index)
      step-card(num: str(idx + 1), title: it.at(0), body: it.at(1), active: is-act)
    })
  )
}

// 多列排版助手
#let two-col(left, right, ratio: (1fr, 1fr), gutter: 14pt) = {
  grid(
    columns: ratio,
    gutter: gutter,
    left,
    right
  )
}

// 双栏居中分割线布局 (纯净学术向：中间一条分割线，两边文字，避免过多 Card 边框)
#let two-col-split(col1, col2, ratio: (1fr, 1fr), gap: 24pt, stroke: 0.8pt + tech-border-mid) = {
  grid(
    columns: ratio,
    stroke: (x, y) => if x == 0 { (right: stroke) },
    inset: (col, row) => if col == 0 { (right: gap) } else { (left: gap) },
    align: (top + left, top + left),
    col1,
    col2
  )
}

// 纯净小节标题组件 (取代沉重的外层 Card 边框)
#let subhead(title, tag: none) = {
  block(width: 100%, [
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      text(weight: "bold", size: 1.08em, fill: tech-navy, title),
      if tag != none { tag }
    )
    #v(3pt)
    #line(length: 100%, stroke: 0.8pt + tech-border)
    #v(5pt)
  ])
}

#let three-col(c1, c2, c3, ratio: (1fr, 1fr, 1fr), gutter: 12pt) = {
  grid(
    columns: ratio,
    gutter: gutter,
    c1,
    c2,
    c3
  )
}

// 公式展示框
#let formula-card(title: none, sub: none, body) = {
  block(
    fill: rgb("#f8fafc"),
    stroke: 1pt + tech-border-mid,
    radius: 6pt,
    inset: (x: 12pt, y: 8pt),
    width: 100%,
    [
      #if title != none [
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          text(size: 0.82em, weight: "bold", fill: tech-primary, title),
          if sub != none { text(size: 0.72em, fill: tech-text-sub, sub) }
        )
        #v(2pt)
      ]
      #align(center, body)
    ]
  )
}

// 美化数据表格
#let tech-table(
  columns: auto,
  headers: (),
  rows: (),
  align: auto,
) = {
  table(
    columns: columns,
    stroke: (x, y) => if y == 0 {
      (bottom: 1.5pt + tech-primary)
    } else {
      (bottom: 0.5pt + tech-border)
    },
    fill: (col, row) => if row == 0 {
      rgb("#f1f5f9")
    } else if calc.even(row) {
      rgb("#fafbfc")
    } else {
      white
    },
    inset: (x: 10pt, y: 7pt),
    align: if align != auto { align } else {
      (col, row) => if row == 0 { center + horizon } else { left + horizon }
    },
    ..headers.map(h => text(weight: "bold", size: 0.88em, fill: tech-navy, h)),
    ..rows.flatten()
  )
}

// ==============================================================================
// 3. 幻灯片页面模板 (Title, Slide, Section, Backup)
// ==============================================================================

// 封面页 (纯净白底学术工程向)
#let title-slide(
  title: [TRSM 算法优化与工程实现],
  subtitle: [双精度下三角求解 $L X = B$],
  author: [答辩人],
  date: [2026-09],
  platform: none,
  extra-badges: (),
  version-tag: none,
  sha-hash: none,
) = {
  page(
    header: none,
    footer: none,
    fill: white,
    margin: (top: 2.8cm, bottom: 2.2cm, left: 2.4cm, right: 2.4cm)
  )[
    #v(1.2fr)

    // 大标题 (深海蓝黑)
    #text(
      weight: "black",
      size: 2.4em,
      fill: tech-navy,
      title
    )

    #v(0.4cm)

    // 副标题 (科技青蓝)
    #text(
      weight: "medium",
      size: 1.2em,
      fill: tech-primary,
      subtitle
    )

    #v(1.2cm)
    #line(length: 100%, stroke: 1.5pt + tech-primary)
    #v(0.5cm)

    // 底部作者与平台信息
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #text(weight: "bold", size: 1.05em, fill: tech-text-main, author)
        #h(16pt)
        #text(size: 0.9em, fill: tech-text-sub, [时间：#date])
        #if platform != none [
          #v(6pt)
          #text(size: 0.85em, fill: tech-text-sub, [运行环境：#platform])
        ]
      ],
      [
        #for b in extra-badges [
          #badge(b, fill: rgb("#f1f5f9"), stroke: tech-border, color: tech-text-main)
          #h(4pt)
        ]
      ]
    )

    #v(1.5fr)
  ]
}

// 章节过渡页
#let section-slide(
  number: "01",
  title: [章节标题],
  desc: none
) = {
  page(
    header: none,
    footer: none,
    fill: tech-navy,
    margin: (top: 3cm, bottom: 2.5cm, left: 2.5cm, right: 2.5cm)
  )[
    #v(1.5fr)
    #grid(
      columns: (auto, 1fr),
      gutter: 22pt,
      align: horizon,
      [
        #text(
          weight: "black",
          size: 3.6em,
          fill: tech-primary,
          number
        )
      ],
      [
        #line(length: 50pt, stroke: 3pt + tech-primary)
        #v(6pt)
        #text(
          weight: "bold",
          size: 2.1em,
          fill: rgb("#f8fafc"),
          title
        )
        #if desc != none [
          #v(6pt)
          #text(size: 1.05em, fill: rgb("#94a3b8"), desc)
        ]
      ]
    )
    #v(2fr)
  ]
}

// 正文幻灯片页 (右上角留空，无多余元素)
#let slide(
  title: [页面标题],
  subtitle: none,
  section: none,
  body
) = {
  _slide-counter.step()

  // 顶栏构造 (纯净单行，右上角清空)
  let slide-header = context {
    block(
      width: 100%,
      [
        #grid(
          columns: (1fr,),
          align: (left + horizon,),
          [
            #if section != none [
              #badge-tech([#section])
              #h(6pt)
            ]
            #text(size: 1.25em, weight: "bold", fill: tech-navy, title)
            #if subtitle != none [
              #h(8pt)
              #text(size: 0.76em, weight: "regular", fill: tech-text-sub, [| #subtitle])
            ]
          ]
        )
        #v(4pt)
        #line(length: 100%, stroke: 0.6pt + tech-border)
      ]
    )
  }

  // 底栏构造
  let slide-footer = context {
    let meta = _meta.get()
    let cur-num = _slide-counter.get().first()

    block(
      width: 100%,
      [
        #line(length: 100%, stroke: 0.5pt + tech-border)
        #v(3pt)
        #grid(
          columns: (auto, 1fr, auto),
          align: (left + horizon, center + horizon, right + horizon),
          text(size: 0.72em, fill: tech-text-sub, [#meta.short-title]),
          text(size: 0.72em, fill: tech-text-sub.lighten(30%), meta.footer),
          [
            #text(size: 0.78em, weight: "bold", fill: tech-primary, [#str(cur-num)])
            #text(size: 0.72em, fill: tech-text-sub, [ \/ #str(meta.total-slides)])
          ]
        )
      ]
    )
  }

  page(
    header: slide-header,
    footer: slide-footer,
  )[
    #body
  ]
}

// 备份/附录页 (Backup Slide - 右上角留空)
#let backup-slide(
  id: "A",
  title: [备份页标题],
  ref: none,
  body
) = {
  let backup-header = context {
    block(
      width: 100%,
      [
        #grid(
          columns: (1fr,),
          align: (left + horizon,),
          [
            #badge-purple([备份页 #id])
            #h(6pt)
            #text(size: 1.25em, weight: "bold", fill: tech-navy, title)
            #if ref != none [
              #h(8pt)
              #text(size: 0.76em, weight: "regular", fill: tech-text-sub, [| 讲稿参考: #ref])
            ]
          ]
        )
        #v(4pt)
        #line(length: 100%, stroke: 0.6pt + tech-border)
      ]
    )
  }

  let backup-footer = context {
    block(
      width: 100%,
      [
        #line(length: 100%, stroke: 0.5pt + tech-border)
        #v(3pt)
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          text(size: 0.72em, fill: tech-purple, [备答附录 · 仅在针对性追问时展示]),
          text(size: 0.75em, weight: "bold", fill: tech-purple, [备份页 #id])
        )
      ]
    )
  }

  page(
    header: backup-header,
    footer: backup-footer,
    fill: rgb("#fafafa")
  )[
    #body
  ]
}
