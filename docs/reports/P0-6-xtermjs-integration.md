# P0-6 开发报告：前端集成 xterm.js 终端展示

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P0-6 |
| 任务名称 | 前端集成 xterm.js 终端展示 |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 1h |
| 关联问题 | N1：缺少 xterm.js |

---

## 1. 任务概述

### 1.1 任务目标
在前端集成 xterm.js 终端模拟器，通过 WebSocket 连接后端实现实时终端输出展示和键盘输入。

### 1.2 完成标准

- [x] 安装 xterm.js 和 xterm-addon-fit 依赖
- [x] 新增终端详情页 `/terminals/[id]`
- [x] WebSocket 连接 `/api/terminals/:id/ws` 实时显示终端输出
- [x] 支持键盘输入发送到 tmux session
- [x] 终端自适应容器大小（FitAddon）
- [x] 终端列表页增加"打开终端"链接

---

## 2. 实现方法

### 2.1 总体方案
1. 安装 `@xterm/xterm` 和 `@xterm/addon-fit` 包
2. 创建 `/terminals/[id]/page.tsx` 终端详情页
3. 使用动态 import 加载 xterm（避免 SSR 问题）
4. 通过 WebSocket 连接后端，接收输出并发送输入
5. 在终端列表页添加"打开"链接

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| xterm 包 | @xterm/xterm（v5+） | 官方推荐的新包名 |
| 加载方式 | 动态 import | 避免 Next.js SSR 问题 |
| WebSocket 协议 | JSON 格式消息 | 便于扩展消息类型 |
| 终端主题 | VS Code 暗色主题 | 与开发环境一致 |

---

## 3. 技术难点及解决方案

### 难点 1：Next.js SSR 与 xterm.js 冲突
**问题描述：** xterm.js 依赖 DOM API，Next.js 的 SSR 环境中没有 DOM

**解决方案：** 使用动态 import（`await import()`）在 useEffect 中加载 xterm 模块，确保只在客户端运行

### 难点 2：WebSocket 消息格式
**问题描述：** 需要区分输出消息和错误消息

**解决方案：** 使用 JSON 格式消息，包含 `type` 字段区分消息类型：
- `{ type: "output", data: "..." }` — 终端输出
- `{ type: "input", data: "..." }` — 用户输入
- `{ type: "error", message: "..." }` — 错误消息

---

## 4. 代码变更记录

### 4.1 新增文件

| 文件路径 | 说明 |
|----------|------|
| `web/src/app/terminals/[id]/page.tsx` | 终端详情页（xterm.js 集成） |

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `web/src/app/terminals/page.tsx` | 添加 Link 导入，表格新增"操作"列和"打开"链接 |
| `web/package.json` | 新增 @xterm/xterm 和 @xterm/addon-fit 依赖 |

---

## 5. 测试结果

### 5.2 手动验证

| 验证项 | 结果 | 备注 |
|--------|------|------|
| xterm.js 安装 | ✅ | npm install 成功 |
| 终端详情页创建 | ✅ | 文件已创建 |
| 列表页链接添加 | ✅ | "打开"链接已添加 |

---

## 6. 后续优化建议

1. 添加终端重连机制（WebSocket 断开后自动重连）
2. 添加终端会话录制回放功能
3. 支持多标签页终端
4. 添加终端字体大小调整
5. 添加搜索功能（xterm-addon-search）

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更 |
| API | 无变更（使用已有 WebSocket 接口） |
| 前端 | 新增终端详情页，修改终端列表页 |
| 配置 | 无变更 |
| 兼容性 | 完全兼容 |
