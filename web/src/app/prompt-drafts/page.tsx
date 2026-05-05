'use client';

import { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  BulbOutlined,
  EditOutlined,
  SendOutlined,
  SaveOutlined,
  CloseOutlined,
  ThunderboltOutlined,
  FileTextOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import {
  Card, Input, Select, Button, Tag, Typography, Space, List, Alert, Spin, Tooltip,
} from 'antd';
import { useProjects } from '@/lib/hooks/useProjects';
import {
  usePromptDrafts,
  useGeneratePromptDraft,
  useUpdatePromptDraft,
  useSendPromptDraft,
} from '@/lib/hooks/usePromptDrafts';
import { techStacksApi } from '@/lib/api';
import type { PromptDraft, TechStackOption } from '@/lib/types';
import { formatDate } from '@/lib/utils';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

const TASK_TYPES = [
  { value: '', label: '自动推断' },
  { value: 'bugfix', label: '修复 Bug' },
  { value: 'build', label: '创建功能' },
  { value: 'review', label: '代码审查' },
  { value: 'qa', label: '测试验证' },
  { value: 'docs', label: '写文档' },
  { value: 'architecture', label: '架构设计' },
];

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

export default function PromptDraftsPage() {
  return (
    <Suspense
      fallback={
        <div style={{ textAlign: 'center', padding: '48px 0' }}>
          <Spin tip="加载中..." />
        </div>
      }
    >
      <PromptDraftsContent />
    </Suspense>
  );
}

function PromptDraftsContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { data: projects } = useProjects();
  const [selectedProjectId, setSelectedProjectId] = useState('');
  const { data: drafts } = usePromptDrafts(selectedProjectId);
  const generateMutation = useGeneratePromptDraft();
  const updateMutation = useUpdatePromptDraft();
  const sendMutation = useSendPromptDraft();

  const [techStacks, setTechStacks] = useState<TechStackOption[]>([]);
  const [selectedTechStack, setSelectedTechStack] = useState('custom');
  const [originalInput, setOriginalInput] = useState(searchParams.get('input') || '');
  const [taskType, setTaskType] = useState(searchParams.get('type') || '');
  const [editingDraft, setEditingDraft] = useState<PromptDraft | null>(null);
  const [editContent, setEditContent] = useState('');
  const [success, setSuccess] = useState<string | null>(null);

  // 加载技术方案列表
  useEffect(() => {
    techStacksApi.list().then(setTechStacks).catch(() => {});
  }, []);

  useEffect(() => {
    if (projects && projects.length > 0 && !selectedProjectId) {
      setSelectedProjectId(projects[0].id);
    }
  }, [projects, selectedProjectId]);

  const handleGenerate = async () => {
    if (!selectedProjectId || !originalInput.trim()) return;
    setSuccess(null);
    try {
      const draft = await generateMutation.mutateAsync({
        projectId: selectedProjectId,
        input: originalInput,
        taskType: taskType || undefined,
        techStackId: selectedTechStack || undefined,
      });
      setEditingDraft(draft);
      setEditContent(draft.generated_prompt);
      setOriginalInput('');
      setSuccess('草稿已生成，请编辑后确认发送');
    } catch {
      // 错误由 mutation 管理
    }
  };

  const handleSave = async () => {
    if (!editingDraft) return;
    try {
      const updated = await updateMutation.mutateAsync({
        id: editingDraft.id,
        finalPrompt: editContent,
        taskType: editingDraft.task_type,
      });
      setEditingDraft(updated);
      setSuccess('草稿已保存');
    } catch {
      // 错误由 mutation 管理
    }
  };

  const handleSend = async (draftId: string) => {
    if (editingDraft?.id === draftId && editContent !== (editingDraft.final_prompt || editingDraft.generated_prompt)) {
      try {
        await updateMutation.mutateAsync({ id: editingDraft.id, finalPrompt: editContent, taskType: editingDraft.task_type });
      } catch {
        return;
      }
    }
    setSuccess(null);
    try {
      const result = await sendMutation.mutateAsync(draftId);
      setEditingDraft(null);
      setSuccess(`已发送！Run: ${result.run_id}`);
      setTimeout(() => router.push('/runs'), 2000);
    } catch {
      // 错误由 mutation 管理
    }
  };

  const errorMsg = generateMutation.error?.message || updateMutation.error?.message || sendMutation.error?.message || null;
  const loading = generateMutation.isPending || updateMutation.isPending || sendMutation.isPending;

  return (
    <div style={{ maxWidth: 896 }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>提示词草稿</Title>
        <Text type="secondary">生成、编辑并发送结构化提示词</Text>
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

      {success && (
        <Alert
          type="success"
          message={success}
          showIcon
          closable
          onClose={() => setSuccess(null)}
          style={{ marginBottom: 20 }}
        />
      )}

      {/* 输入区 */}
      <Card
        style={{
          marginBottom: 20,
          background: 'linear-gradient(135deg, #f0f5ff 0%, #e6f4ff 50%, #f0f5ff 100%)',
          border: '1px solid #d6e4ff',
        }}
        styles={{ body: { padding: 24 } }}
      >
        <Space align="center" size={12} style={{ marginBottom: 16 }}>
          <div style={{
            width: 32, height: 32, borderRadius: 8,
            background: '#faad14', display: 'flex', alignItems: 'center', justifyContent: 'center',
            color: '#fff', boxShadow: '0 2px 8px rgba(250,173,20,0.3)',
          }}>
            <ThunderboltOutlined style={{ fontSize: 15 }} />
          </div>
          <div>
            <Text strong style={{ fontSize: 14 }}>生成草稿</Text>
            <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>输入需求，AI 生成结构化提示词</Text>
          </div>
        </Space>

        {/* 技术方案选择 */}
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>
            <AppstoreOutlined style={{ marginRight: 4 }} />
            选择项目技术方案（影响生成提示词中的上下文信息）
          </Text>
          <Select
            value={selectedTechStack || undefined}
            onChange={setSelectedTechStack}
            placeholder="选择技术方案"
            style={{ width: '100%' }}
            size="middle"
            options={techStacks.map((ts) => ({
              value: ts.id,
              label: ts.label,
            }))}
          />
        </div>

        <Space size={12} style={{ marginBottom: 12, width: '100%' }}>
          <Select
            value={selectedProjectId || undefined}
            onChange={setSelectedProjectId}
            placeholder="选择项目"
            style={{ minWidth: 180 }}
            options={(projects || []).map((p) => ({ value: p.id, label: p.name }))}
          />
          <Select
            value={taskType || undefined}
            onChange={setTaskType}
            placeholder="任务类型"
            style={{ minWidth: 140 }}
            options={TASK_TYPES.map((t) => ({ value: t.value, label: t.label }))}
          />
        </Space>

        <TextArea
          value={originalInput}
          onChange={(e) => setOriginalInput(e.target.value)}
          placeholder="描述你想要完成的任务..."
          rows={3}
          style={{ marginBottom: 12 }}
        />

        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <Button
            type="primary"
            icon={<BulbOutlined />}
            loading={loading}
            disabled={!originalInput.trim()}
            onClick={handleGenerate}
          >
            生成草稿
          </Button>
          {['bugfix', 'build', 'review', 'qa', 'docs'].map((t) => (
            <Button
              key={t}
              size="small"
              type={taskType === t ? 'primary' : 'default'}
              ghost={taskType === t}
              onClick={() => setTaskType(t)}
            >
              {TASK_TYPES.find((tt) => tt.value === t)?.label}
            </Button>
          ))}
        </div>
      </Card>

      {/* 编辑区 */}
      {editingDraft && (
        <Card
          style={{ marginBottom: 20, borderLeft: '4px solid #1677ff' }}
          styles={{ body: { padding: 24 } }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <Space>
              <EditOutlined style={{ color: 'rgba(0,0,0,0.45)' }} />
              <Text strong>编辑草稿</Text>
            </Space>
            {statusTag(editingDraft.status)}
          </div>
          <div style={{
            fontSize: 12, color: 'rgba(0,0,0,0.45)', marginBottom: 12,
            background: '#f5f5f5', padding: '4px 10px', borderRadius: 4, display: 'inline-block',
          }}>
            {editingDraft.task_type} | {editingDraft.original_input.slice(0, 60)}...
          </div>
          <TextArea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            rows={10}
            style={{
              fontFamily: 'monospace', marginBottom: 16,
              background: '#fafafa', minHeight: 200,
            }}
          />
          <Space>
            <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={handleSave}>
              保存
            </Button>
            <Button type="primary" icon={<SendOutlined />} loading={loading} onClick={() => handleSend(editingDraft.id)}>
              确认发送
            </Button>
            <Button icon={<CloseOutlined />} onClick={() => setEditingDraft(null)}>
              取消
            </Button>
          </Space>
        </Card>
      )}

      {/* 草稿历史 */}
      <Card
        title={
          <Space>
            <FileTextOutlined style={{ color: 'rgba(0,0,0,0.45)' }} />
            <Text strong>草稿历史</Text>
          </Space>
        }
        styles={{ body: { padding: 0 } }}
      >
        <List<PromptDraft>
          dataSource={drafts || []}
          locale={{ emptyText: '暂无草稿' }}
          renderItem={(draft) => (
            <List.Item
              style={{
                padding: '12px 24px',
                cursor: draft.status === 'draft' ? 'pointer' : 'default',
                opacity: draft.status === 'draft' ? 1 : 0.6,
                transition: 'background 0.2s',
              }}
              actions={[
                statusTag(draft.status),
                draft.status === 'draft' && (
                  <Button
                    key="send"
                    type="primary"
                    size="small"
                    icon={<SendOutlined />}
                    onClick={(e) => { e.stopPropagation(); handleSend(draft.id); }}
                  >
                    发送
                  </Button>
                ),
              ].filter(Boolean)}
              onClick={() => {
                if (draft.status === 'draft') {
                  setEditingDraft(draft);
                  setEditContent(draft.final_prompt || draft.generated_prompt);
                }
              }}
            >
              <List.Item.Meta
                title={
                  <Text strong style={{ fontSize: 14 }}>
                    {draft.original_input.length > 80
                      ? draft.original_input.slice(0, 80) + '...'
                      : draft.original_input}
                  </Text>
                }
                description={
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {draft.task_type} | {formatDate(draft.created_at)}
                  </Text>
                }
              />
            </List.Item>
          )}
        />
      </Card>
    </div>
  );
}
