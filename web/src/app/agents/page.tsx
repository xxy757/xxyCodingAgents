'use client';

import { RobotOutlined, PauseCircleOutlined, PlayCircleOutlined, StopOutlined } from '@ant-design/icons';
import { Table, Tag, Button, Space, Typography, Card, Popconfirm, Alert } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useAgents, useAgentAction } from '@/lib/hooks/useAgents';
import type { AgentInstance } from '@/lib/types';
import { formatDate, shortId } from '@/lib/utils';

const { Title, Text } = Typography;

function statusTag(status: string) {
  const map: Record<string, string> = {
    running: 'processing', active: 'success', completed: 'success',
    failed: 'error', stopped: 'error', pending: 'default', queued: 'default',
    cancelled: 'warning', paused: 'orange', draft: 'blue', sent: 'green',
    evicted: 'volcano', starting: 'processing', detached: 'default', closed: 'default',
    recoverable: 'orange', orphaned: 'magenta', admitted: 'processing', blocked: 'default',
  };
  return <Tag color={map[status] || 'default'}>{status}</Tag>;
}

export default function AgentsPage() {
  const { data: agents, isLoading, error } = useAgents();
  const actionMutation = useAgentAction();

  const handleAction = (id: string, action: 'pause' | 'resume' | 'stop') => {
    actionMutation.mutate({ id, action });
  };

  const columns: ColumnsType<AgentInstance> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 160,
      render: (id: string) => (
        <Space>
          <div style={{
            width: 28, height: 28, borderRadius: 6,
            background: '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <RobotOutlined style={{ color: '#1677ff', fontSize: 12 }} />
          </div>
          <Text code style={{ fontSize: 12 }}>{shortId(id)}</Text>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'agent_kind',
      key: 'agent_kind',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => statusTag(status),
    },
    {
      title: 'tmux 会话',
      dataIndex: 'tmux_session',
      key: 'tmux_session',
      render: (val: string) => <Text code style={{ fontSize: 12 }}>{val || '-'}</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (val: string) => <Text type="secondary" style={{ fontSize: 12 }}>{formatDate(val)}</Text>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 240,
      render: (_: unknown, record: AgentInstance) => (
        <Space size={4}>
          {record.status === 'running' && (
            <Button
              size="small"
              icon={<PauseCircleOutlined />}
              onClick={() => handleAction(record.id, 'pause')}
            >
              暂停
            </Button>
          )}
          {record.status === 'paused' && (
            <Button
              size="small"
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={() => handleAction(record.id, 'resume')}
            >
              恢复
            </Button>
          )}
          {record.status !== 'stopped' && record.status !== 'failed' && (
            <Popconfirm
              title="确认停止"
              description="确定要停止此 Agent 实例吗？"
              onConfirm={() => handleAction(record.id, 'stop')}
              okText="确定"
              cancelText="取消"
            >
              <Button size="small" danger icon={<StopOutlined />}>停止</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const errorMsg = error?.message || actionMutation.error?.message;

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>Agent 实例</Title>
        <Text type="secondary">管理和监控 Agent 实例</Text>
      </div>

      {errorMsg && (
        <Alert
          type="error"
          message={errorMsg}
          showIcon
          closable
          style={{ marginBottom: 16 }}
        />
      )}

      <Card styles={{ body: { padding: 0 } }}>
        <Table<AgentInstance>
          columns={columns}
          dataSource={agents || []}
          loading={isLoading}
          rowKey="id"
          locale={{ emptyText: '暂无 Agent 实例，Agent 会在任务调度时自动创建' }}
          pagination={false}
        />
      </Card>
    </div>
  );
}
