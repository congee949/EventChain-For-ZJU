# Typst PPT 套件

这套文件可以直接用于 EventChain 的课程展示。核心模板是 `template.typ`，正文文件只需在第一行导入它。

## 初版复刻

`EventChain_初版复刻.typ` 按原始讲解版的 14 页结构重建，画布、颜色、流程卡片、证据路径和钱包截图均与初版对应。它是独立版式，不依赖 `template.typ`，只需保留 `assets/wallet-before-convert.png`。

```powershell
typst compile EventChain_初版复刻.typ EventChain_初版复刻.pdf
```

## 文件说明

- `EventChain_课程展示.typ` 是 EventChain 的三页起步稿，直接从这里开始改。
- `template.typ` 提供封面、正文页、双栏、提示框、表格和公式框等版式。
- `TRSM_完整示例.typ` 和对应 PDF 是完整的 10 页技术答辩示例，可参考组件写法和信息密度。
- `TRSM_旧版备份.typ` 保留旧版页面，通常不需要修改。

## 编译

先安装 Typst，然后在本目录运行：

```powershell
typst compile EventChain_课程展示.typ EventChain_课程展示.pdf
```

编辑时可使用实时预览：

```powershell
typst watch EventChain_课程展示.typ EventChain_课程展示.pdf
```

所有 `.typ` 文件都只依赖同目录的 `template.typ`，迁移整个 `PPT套件` 文件夹即可继续使用。模板的底栏文案由 `presentation.with(... footer: [...])` 控制，不再固定写 HPC 课程。

## 环境

模板优先使用 `Segoe UI`、`Noto Sans SC` 和 `Microsoft YaHei`。Windows 上通常无需额外安装字体。另一台电脑只要安装 Typst，并保留上述字体中的一种中文字体，就能正常编译。
