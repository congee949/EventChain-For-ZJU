# EventChain Typst 答辩稿

`EventChain_答辩版_10分钟.typ` 是当前答辩主稿。它采用 1 页封面和 9 页正文，版式参考 `hpc101/trsm/slides/2trsm.typ`，讲者备注以源码注释保留。

## 文件说明

- `EventChain_答辩版_10分钟.typ`：10 分钟答辩源码。
- `template.typ`：封面、正文页、双栏、流程、提示框和表格组件。
- `assets/defense/`：答辩稿使用的真实浏览器截图。
- `EventChain_初版复刻.typ`：早期 14 页版本，留作内容对照。
- `EventChain_课程展示.typ`：三页起步示例。

## 编译

在当前目录运行：

```powershell
typst compile EventChain_答辩版_10分钟.typ EventChain_答辩版_10分钟.pdf
```

编辑时可以开启实时预览：

```powershell
typst watch EventChain_答辩版_10分钟.typ EventChain_答辩版_10分钟.pdf
```

## 环境

建议使用 Typst 0.15 或更新版本。模板优先使用 `Segoe UI`、`Noto Sans SC` 和 `Microsoft YaHei`，Windows 通常已安装可用的中文字体。
