'use client';

import { Handle, Position } from '@xyflow/react';
import { Tag } from 'antd';

interface TaskNodeData {
  label: string;
  status: string;
  task_type: string;
  task_id: string;
}

const STATUS_COLORS: Record<string, string> = {
  running: 'processing', completed: 'success', failed: 'error',
  pending: 'default', queued: 'default', cancelled: 'warning',
  paused: 'orange', evicted: 'volcano', admitted: 'processing', blocked: 'default',
};

export function TaskNode({ data }: { data: TaskNodeData }) {
  return (
    <div style={{
      background: '#fff',
      borderRadius: 8,
      border: '1px solid #d9d9d9',
      boxShadow: '0 1px 2px rgba(0,0,0,0.06)',
      minWidth: 160,
    }}>
      <Handle type="target" position={Position.Top} style={{ width: 8, height: 8, background: '#bfbfbf' }} />
      <div style={{ padding: '8px 12px' }}>
        <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)', marginBottom: 4 }}>{data.task_type}</div>
        <div style={{ fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.88)', marginBottom: 8 }}>{data.label}</div>
        <Tag color={STATUS_COLORS[data.status] || 'default'}>{data.status}</Tag>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ width: 8, height: 8, background: '#bfbfbf' }} />
    </div>
  );
}
