# P1-4: 前端集成 React Flow 工作流可视化

## 任务概述

| 项目 | 内容 |
|------|------|
| 任务编号 | P1-4 |
| 优先级 | P1 (高) |
| 状态 | ✅ 已完成 |
| 预计工时 | 4h |
| 关联任务 | P1-1 (DAG执行), P1-3 (前端完善) |

## 实现方法

### 后端 API

在 `internal/api/handlers.go` 中新增 `handleRunWorkflow` 处理器，提供 `GET /api/runs/{id}/workflow` 端点：

1. 获取 Run 对象及其关联的 Tasks 列表
2. 若 Run 关联了 WorkflowTemplate，解析其 NodesJSON/EdgesJSON
3. 将 Task 状态映射到工作流节点上（通过 TaskSpecID 匹配）
4. 自动计算节点布局位置（网格排列，4列）
5. 若无模板，退化为以 Task 列表为节点的简单图

### 前端可视化

在 `web/src/app/runs/[id]/page.tsx` 中集成 React Flow：

1. 定义 `WorkflowGraph` 接口匹配后端响应
2. 创建自定义 `TaskNode` 组件，按状态着色边框和背景
3. 添加 `statusBorderColor` / `statusBgColor` 辅助函数映射6种状态颜色
4. 使用 `ReactFlow` 组件渲染图，配置 Background、Controls、MiniMap
5. 边线启用动画效果，MiniMap 按状态颜色显示节点

## 技术难点及解决方案

### 难点1: React Flow 与 Next.js SSR 兼容

React Flow 依赖 DOM API，在 Next.js SSR 环境下会报错。

**解决方案**: 页面已使用 `"use client"` 指令，React Flow 组件仅在客户端渲染，避免了 SSR 兼容性问题。

### 难点2: 节点状态可视化

需要在图中直观展示每个任务节点的当前执行状态。

**解决方案**: 创建自定义 `TaskNode` 组件，根据 status 字段动态计算边框色和背景色，同时显示状态标签和任务类型。Handle 组件的颜色也跟随状态变化，使整个节点视觉一致。

### 难点3: 自动布局

后端 API 返回的节点位置使用简单的网格排列，对于复杂 DAG 可能不够美观。

**解决方案**: 当前采用 4 列网格布局（`x = col * 250, y = row * 120`），配合 React Flow 的 `fitView` 自动缩放。后续可集成 dagre/elkjs 等自动布局算法优化。

## 测试结果

| 测试项 | 结果 |
|--------|------|
| Go 编译 (`go build ./...`) | ✅ 通过 |
| Go 静态检查 (`go vet ./...`) | ✅ 通过 |
| Next.js 构建 (`next build`) | ✅ 通过 |
| TypeScript 类型检查 | ✅ 通过 |
| 页面路由注册 | ✅ `/runs/[id]` 动态路由正常 |

## 代码变更记录

### 新增文件

无

### 修改文件

| 文件 | 变更内容 |
|------|----------|
| `internal/api/server.go` | 新增 `GET /api/runs/{id}/workflow` 路由 |
| `internal/api/handlers.go` | 新增 `handleRunWorkflow` 处理器（约 130 行） |
| `web/src/app/runs/[id]/page.tsx` | 新增 React Flow 导入、WorkflowGraph 接口、TaskNode 组件、工作流标签页 |
| `web/package.json` | 新增 `@xyflow/react` 依赖 |

### 关键代码片段

**自定义 TaskNode 组件**:
```tsx
function TaskNode({ data }: { data: { label: string; status: string; task_type: string } }) {
  const borderColor = statusBorderColor(data.status);
  const bgColor = statusBgColor(data.status);
  return (
    <div style={{ padding: "10px 16px", borderRadius: 8, border: `2px solid ${borderColor}`, background: bgColor }}>
      <Handle type="target" position={Position.Top} />
      <div style={{ fontWeight: 600 }}>{data.label}</div>
      <span style={{ background: borderColor, color: "#fff" }}>{data.status}</span>
      <Handle type="source" position={Position.Bottom} />
    </div>
  );
}
```

**React Flow 渲染**:
```tsx
<ReactFlow
  nodes={graph.nodes.map(n => ({ id: n.id, type: "task", data: n.data, position: n.position }))}
  edges={graph.edges.map(e => ({ id: e.id, source: e.source, target: e.target, animated: true }))}
  nodeTypes={nodeTypes}
  fitView
>
  <Background /><Controls /><MiniMap />
</ReactFlow>
```

## 后续优化建议

1. **自动布局算法**: 集成 dagre 或 elkjs 实现 DAG 层次化自动布局，替代当前的网格排列
2. **节点交互**: 支持点击节点跳转到对应 Task 详情或终端页面
3. **实时更新**: 通过 WebSocket 推送任务状态变更，实时更新节点颜色
4. **边线标签**: 在边上显示依赖关系类型（如 `on_success`、`on_failure`）
5. **缩略图优化**: MiniMap 中使用更精细的节点形状区分不同任务类型
