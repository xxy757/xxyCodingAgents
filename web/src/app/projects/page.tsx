'use client';

import { useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, Space, Typography, message } from 'antd';
import { FolderOpenOutlined, PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useProjects, useCreateProject } from '@/lib/hooks/useProjects';
import type { Project } from '@/lib/types';
import { formatDate, repoDisplayPath } from '@/lib/utils';

const { Title } = Typography;

export default function ProjectsPage() {
  const { data: projects, isLoading, error } = useProjects();
  const createMutation = useCreateProject();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const handleCreate = async (values: { name: string; repo_url?: string; description?: string }) => {
    try {
      await createMutation.mutateAsync(values);
      message.success('项目创建成功');
      form.resetFields();
      setModalOpen(false);
    } catch (err: any) {
      message.error(err?.message || '创建失败');
    }
  };

  const columns: ColumnsType<Project> = [
    {
      title: '项目名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space>
          <FolderOpenOutlined style={{ color: '#1677ff' }} />
          <span style={{ fontWeight: 500 }}>{name}</span>
        </Space>
      ),
    },
    {
      title: '仓库地址',
      dataIndex: 'repo_url',
      key: 'repo_url',
      render: (url: string) => (
        <span style={{ color: 'rgba(0,0,0,0.45)', fontFamily: 'monospace', fontSize: 13 }}>
          {url ? repoDisplayPath(url) : '-'}
        </span>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (desc: string) => (
        <span style={{ color: 'rgba(0,0,0,0.45)' }}>{desc || '-'}</span>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: string) => (
        <span style={{ color: 'rgba(0,0,0,0.45)', fontSize: 13 }}>{formatDate(v)}</span>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>项目</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          新建项目
        </Button>
      </div>

      {error && (
        <Card size="small" style={{ marginBottom: 16, borderColor: '#ff4d4f' }}>
          <span style={{ color: '#ff4d4f' }}>{error.message}</span>
        </Card>
      )}

      <Card>
        <Table<Project>
          columns={columns}
          dataSource={projects || []}
          rowKey="id"
          loading={isLoading}
          locale={{ emptyText: '暂无项目，点击右上角创建第一个项目' }}
          pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (total) => `共 ${total} 个项目` }}
        />
      </Card>

      <Modal
        title="新建项目"
        open={modalOpen}
        onCancel={() => { setModalOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
        okText="创建"
        cancelText="取消"
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 16 }}>
          <Form.Item label="项目名称" name="name" rules={[{ required: true, message: '请输入项目名称' }]}>
            <Input placeholder="请输入项目名称" />
          </Form.Item>
          <Form.Item label="仓库地址" name="repo_url">
            <Input placeholder="https://github.com/..." />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input placeholder="请输入项目描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
