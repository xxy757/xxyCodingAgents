'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { CodeOutlined, PlusOutlined, ArrowRightOutlined } from '@ant-design/icons';
import { Table, Tag, Button, Typography, Card, Modal, Input, Form, Space, Alert } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTerminals, useCreateTerminal } from '@/lib/hooks/useTerminals';
import type { TerminalSession } from '@/lib/types';
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

export default function TerminalsPage() {
  const router = useRouter();
  const { data: terminals, isLoading, error } = useTerminals();
  const createMutation = useCreateTerminal();
  const [showForm, setShowForm] = useState(false);
  const [form] = Form.useForm();

  const handleCreate = async (values: { task_id: string }) => {
    try {
      await createMutation.mutateAsync({ task_id: values.task_id });
      form.resetFields();
      setShowForm(false);
    } catch {
      // 错误由 mutation 管理
    }
  };

  const columns: ColumnsType<TerminalSession> = [
    {
      title: 'tmux 会话',
      dataIndex: 'tmux_session',
      key: 'tmux_session',
      width: '40%',
      ellipsis: true,
      render: (val: string) => (
        <Space size={8}>
          <div style={{
            width: 28, height: 28, borderRadius: 6, flexShrink: 0,
            background: '#f6ffed', display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <CodeOutlined style={{ color: '#52c41a', fontSize: 12 }} />
          </div>
          <Text code style={{ fontSize: 13 }}>{val}</Text>
        </Space>
      ),
    },
    {
      title: '关联任务',
      dataIndex: 'task_id',
      key: 'task_id',
      width: '25%',
      ellipsis: true,
      render: (val: string) => <Text code style={{ fontSize: 13 }}>{shortId(val)}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => statusTag(status),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (val: string) => <Text type="secondary" style={{ fontSize: 13 }}>{formatDate(val)}</Text>,
    },
    {
      title: '',
      key: 'actions',
      width: 60,
      align: 'center',
      render: (_: unknown, record: TerminalSession) => (
        <Link href={`/terminals/${record.id}`}>
          <Button type="text" icon={<ArrowRightOutlined />} style={{ color: 'rgba(0,0,0,0.25)' }} />
        </Link>
      ),
    },
  ];

  const errorMsg = error?.message || createMutation.error?.message;

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>终端管理</Title>
          <Text type="secondary">tmux 会话管理</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setShowForm(true)}>
          新建终端
        </Button>
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

      <Modal
        title="新建终端"
        open={showForm}
        onCancel={() => { setShowForm(false); form.resetFields(); }}
        footer={null}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 16 }}>
          <Form.Item
            label="Task ID"
            name="task_id"
            rules={[{ required: true, message: '请输入关联的 Task ID' }]}
          >
            <Input placeholder="输入关联的 Task ID" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setShowForm(false); form.resetFields(); }}>取消</Button>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending}>创建</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Card styles={{ body: { padding: 0 } }}>
        <Table<TerminalSession>
          columns={columns}
          dataSource={terminals || []}
          loading={isLoading}
          rowKey="id"
          locale={{ emptyText: '暂无终端会话' }}
          pagination={false}
          tableLayout="fixed"
          onRow={(record) => ({
            onClick: () => { router.push(`/terminals/${record.id}`); },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>
    </div>
  );
}
