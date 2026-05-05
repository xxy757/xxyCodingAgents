'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Card, Table, Button, Modal, Form, Input, Select, Tag, Space, Typography, message } from 'antd';
import { RocketOutlined, PlusOutlined, ArrowRightOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useRuns, useCreateRun } from '@/lib/hooks/useRuns';
import { useProjects } from '@/lib/hooks/useProjects';
import { useWorkflowTemplates } from '@/lib/hooks/useWorkflowTemplates';
import type { Run } from '@/lib/types';
import { formatDate, shortId } from '@/lib/utils';

const { Title } = Typography;

/** 状态 Tag 颜色映射 */
function statusTag(status: string) {
  const map: Record<string, string> = {
    running: 'processing',
    completed: 'success',
    failed: 'error',
    pending: 'default',
    queued: 'default',
    cancelled: 'warning',
    paused: 'orange',
    evicted: 'volcano',
  };
  return <Tag color={map[status] || 'default'}>{status}</Tag>;
}

export default function RunsPage() {
  const router = useRouter();
  const { data: runs, isLoading, error } = useRuns();
  const { data: projects } = useProjects();
  const { data: templates } = useWorkflowTemplates();
  const createMutation = useCreateRun();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const handleCreate = async (values: { title: string; project_id?: string; workflow_template_id?: string }) => {
    try {
      const body: { title: string; project_id?: string; workflow_template_id?: string } = { title: values.title };
      if (values.project_id) body.project_id = values.project_id;
      if (values.workflow_template_id) body.workflow_template_id = values.workflow_template_id;
      await createMutation.mutateAsync(body);
      message.success('Run 创建成功');
      form.resetFields();
      setModalOpen(false);
    } catch (err: any) {
      message.error(err?.message || '创建失败');
    }
  };

  const columns: ColumnsType<Run> = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (title: string, record: Run) => (
        <Space>
          <RocketOutlined style={{ color: '#1677ff' }} />
          <Link href={`/runs/${record.id}`} style={{ fontWeight: 500 }}>
            {title}
          </Link>
        </Space>
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
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 200,
      render: (v: string) => (
        <span style={{ fontSize: 13, color: 'rgba(0,0,0,0.45)' }}>{formatDate(v)}</span>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      render: (_: any, record: Run) => (
        <Link href={`/runs/${record.id}`} style={{ color: 'rgba(0,0,0,0.25)' }}>
          <ArrowRightOutlined />
        </Link>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>运行中心</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          新建 Run
        </Button>
      </div>

      {(error || createMutation.error) && (
        <Card size="small" style={{ marginBottom: 16, borderColor: '#ff4d4f' }}>
          <span style={{ color: '#ff4d4f' }}>{error?.message || createMutation.error?.message}</span>
        </Card>
      )}

      <Card>
        <Table<Run>
          columns={columns}
          dataSource={runs || []}
          rowKey="id"
          loading={isLoading}
          locale={{ emptyText: '暂无运行记录' }}
          pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (total) => `共 ${total} 条运行` }}
          onRow={(record) => ({
            onClick: () => { router.push(`/runs/${record.id}`); },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      <Modal
        title="新建 Run"
        open={modalOpen}
        onCancel={() => { setModalOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
        okText="创建"
        cancelText="取消"
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 16 }}>
          <Form.Item label="项目" name="project_id">
            <Select placeholder="选择项目" allowClear>
              {(projects || []).map((p) => (
                <Select.Option key={p.id} value={p.id}>{p.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label="工作流模板" name="workflow_template_id">
            <Select placeholder="无模板" allowClear>
              {(templates || []).map((t) => (
                <Select.Option key={t.id} value={t.id}>{t.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label="标题" name="title" rules={[{ required: true, message: '请输入 Run 标题' }]}>
            <Input placeholder="请输入标题" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
