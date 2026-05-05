'use client';

import { useEffect, useState } from 'react';
import {
  Card, Table, Tabs, Tag, Button, Space, Typography, Spin, Badge, Empty, message,
} from 'antd';
import {
  ArrowLeftOutlined, UnorderedListOutlined, ClockCircleOutlined, BranchesOutlined,
  RedoOutlined, StopOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useRun, useRunTasks, useRunTimeline, useRunWorkflow } from '@/lib/hooks/useRuns';
import { useApproveGate } from '@/lib/hooks/useGates';
import { useRetryTask, useCancelTask } from '@/lib/hooks/useTasks';
import { WorkflowGraph } from '@/components/workflow/WorkflowGraph';
import type { Task, Event } from '@/lib/types';
import { formatDate, shortId } from '@/lib/utils';

const { Title, Text, Paragraph } = Typography;

/** 状态 Tag 颜色映射 */
function statusTag(status: string) {
  const map: Record<string, string> = {
    running: 'processing',
    completed: 'success',
    active: 'success',
    failed: 'error',
    stopped: 'error',
    pending: 'default',
    queued: 'default',
    cancelled: 'warning',
    paused: 'orange',
    evicted: 'volcano',
    draft: 'blue',
  };
  return <Tag color={map[status] || 'default'}>{status}</Tag>;
}

/** 事件类型对应的 Badge 颜色 */
function eventTypeColor(type: string): string {
  const map: Record<string, string> = {
    task_created: 'blue',
    task_started: 'processing',
    task_completed: 'success',
    task_failed: 'error',
    task_cancelled: 'warning',
    gate_pending: 'orange',
    gate_approved: 'green',
    gate_rejected: 'red',
    checkpoint: 'purple',
  };
  return map[type] || 'default';
}

/** 任务输出展示组件 */
function TaskOutput({ data }: { data?: string }) {
  if (!data) {
    return <Text type="secondary" style={{ fontSize: 13 }}>暂无输出</Text>;
  }

  // 尝试解析为 JSON 格式化展示
  let formatted: string;
  try {
    const parsed = JSON.parse(data);
    formatted = JSON.stringify(parsed, null, 2);
  } catch {
    formatted = data;
  }

  return (
    <div
      style={{
        background: '#1f1f1f',
        borderRadius: 8,
        border: '1px solid #303030',
        padding: 16,
        maxHeight: 400,
        overflowY: 'auto',
      }}
    >
      <pre
        style={{
          margin: 0,
          fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', Menlo, monospace",
          fontSize: 12,
          lineHeight: 1.7,
          color: '#d4d4d4',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all',
        }}
      >
        {formatted}
      </pre>
    </div>
  );
}

export default function RunDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState('');
  useEffect(() => { params.then((p) => setId(p.id)); }, [params]);

  const { data: run, error: runError, isLoading: runLoading } = useRun(id);
  const { data: tasks } = useRunTasks(id);
  const { data: events } = useRunTimeline(id);
  const { data: workflow } = useRunWorkflow(id);
  const approveMutation = useApproveGate(id);
  const retryMutation = useRetryTask(id);
  const cancelMutation = useCancelTask(id);

  if (!id) {
    return (
      <div style={{ textAlign: 'center', padding: '48px 0' }}>
        <Spin tip="加载中..." />
      </div>
    );
  }

  const errorMessage = runError?.message || approveMutation.error?.message || retryMutation.error?.message || cancelMutation.error?.message;

  const taskColumns: ColumnsType<Task> = [
    {
      title: '任务',
      dataIndex: 'title',
      key: 'title',
      render: (title: string, record: Task) => (
        <div>
          <div style={{ fontWeight: 500 }}>{title || record.task_type}</div>
          <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)', marginTop: 2 }}>
            {record.task_type} | {record.priority} | {record.resource_class} | 尝试 #{record.attempt_no}
          </div>
        </div>
      ),
    },
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 13, color: 'rgba(0,0,0,0.45)' }}>{shortId(id)}</span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => statusTag(status),
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (_: any, record: Task) => (
        <Space size="small">
          {(record.status === 'failed' || record.status === 'evicted') && (
            <Button
              type="primary"
              size="small"
              icon={<RedoOutlined />}
              loading={retryMutation.isPending}
              onClick={() => retryMutation.mutate(record.id)}
            >
              重试
            </Button>
          )}
          {(record.status === 'queued' || record.status === 'running') && (
            <Button
              danger
              size="small"
              icon={<StopOutlined />}
              loading={cancelMutation.isPending}
              onClick={() => cancelMutation.mutate(record.id)}
            >
              取消
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const tabItems = [
    {
      key: 'tasks',
      label: (
        <Space size={4}>
          <UnorderedListOutlined />
          任务
          <Badge count={tasks?.length ?? 0} showZero size="small" style={{ marginLeft: 4 }} />
        </Space>
      ),
      children: (
        <Table<Task>
          columns={taskColumns}
          dataSource={tasks || []}
          rowKey="id"
          locale={{ emptyText: '暂无任务' }}
          pagination={false}
          size="middle"
          expandable={{
            rowExpandable: (record) => !!(record.output_data || record.input_data),
            expandedRowRender: (record) => (
              <div style={{ padding: '8px 0' }}>
                {record.input_data && (
                  <div style={{ marginBottom: 12 }}>
                    <Text type="secondary" style={{ fontSize: 12, marginBottom: 6, display: 'block' }}>
                      输入数据
                    </Text>
                    <TaskOutput data={record.input_data} />
                  </div>
                )}
                {record.output_data && (
                  <div>
                    <Text type="secondary" style={{ fontSize: 12, marginBottom: 6, display: 'block' }}>
                      执行结果
                    </Text>
                    <TaskOutput data={record.output_data} />
                  </div>
                )}
              </div>
            ),
          }}
        />
      ),
    },
    {
      key: 'timeline',
      label: (
        <Space size={4}>
          <ClockCircleOutlined />
          时间线
          <Badge count={events?.length ?? 0} showZero size="small" style={{ marginLeft: 4 }} />
        </Space>
      ),
      children: (events || []).length === 0 ? (
        <Empty description="暂无事件" style={{ padding: '48px 0' }} />
      ) : (
        <div style={{ padding: '16px 0' }}>
          {(events || []).map((event: Event) => (
            <div
              key={event.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 16,
                padding: '12px 24px',
                borderBottom: '1px solid #f0f0f0',
                transition: 'background 0.2s',
              }}
              onMouseEnter={(e) => { e.currentTarget.style.background = '#fafafa'; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
            >
              <div style={{ minWidth: 160, fontSize: 12, color: 'rgba(0,0,0,0.45)', fontFamily: 'monospace', paddingTop: 2 }}>
                {formatDate(event.created_at)}
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Tag color={eventTypeColor(event.event_type)} style={{ margin: 0 }}>
                    {event.event_type}
                  </Tag>
                  <span style={{ fontSize: 14 }}>{event.message}</span>
                </div>
                {(event.task_id || event.agent_id) && (
                  <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)', marginTop: 4, fontFamily: 'monospace' }}>
                    {event.task_id && `Task: ${shortId(event.task_id)}`}
                    {event.task_id && event.agent_id && ' | '}
                    {event.agent_id && `Agent: ${shortId(event.agent_id)}`}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      ),
    },
    {
      key: 'workflow',
      label: (
        <Space size={4}>
          <BranchesOutlined />
          工作流
        </Space>
      ),
      children: (
        <div style={{ padding: 16 }}>
          <WorkflowGraph
            graph={workflow || null}
            onApproveGate={(gateId: string) => approveMutation.mutate({ gateId })}
          />
        </div>
      ),
    },
  ];

  return (
    <div>
      {/* PageHeader */}
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => window.history.back()}
          style={{ marginRight: 12 }}
        />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Title level={4} style={{ margin: 0 }}>
              {runLoading ? '加载中...' : (run?.title || `Run ${shortId(id)}`)}
            </Title>
            {run && statusTag(run.status)}
          </div>
          <div style={{ fontSize: 13, color: 'rgba(0,0,0,0.45)', fontFamily: 'monospace', marginTop: 4 }}>
            {shortId(id)}
          </div>
        </div>
      </div>

      {/* Error Banner */}
      {errorMessage && (
        <Card size="small" style={{ marginBottom: 16, borderColor: '#ff4d4f' }}>
          <span style={{ color: '#ff4d4f' }}>{errorMessage}</span>
        </Card>
      )}

      {/* Tabs Card */}
      <Card styles={{ body: { padding: 0 } }}>
        <Tabs
          defaultActiveKey="tasks"
          items={tabItems}
          style={{ padding: '0 16px' }}
        />
      </Card>
    </div>
  );
}
