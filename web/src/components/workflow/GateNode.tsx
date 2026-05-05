'use client';

import { Handle, Position } from '@xyflow/react';
import { Tag, Button } from 'antd';

interface GateNodeData {
  label: string;
  status: string;
  gate_type?: string;
  gate_id?: string;
  onApprove?: (gateId: string) => void;
}

const STATUS_COLORS: Record<string, string> = {
  pending: 'warning', approved: 'success', rejected: 'error',
};

export function GateNode({ data }: { data: GateNodeData }) {
  return (
    <div style={{
      background: '#fffbe6',
      borderRadius: 8,
      border: '1px solid #ffe58f',
      boxShadow: '0 1px 2px rgba(0,0,0,0.06)',
      minWidth: 160,
    }}>
      <Handle type="target" position={Position.Top} style={{ width: 8, height: 8, background: '#bfbfbf' }} />
      <div style={{ padding: '8px 12px' }}>
        <div style={{ fontSize: 12, color: '#faad14', marginBottom: 4 }}>门禁</div>
        <div style={{ fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.88)', marginBottom: 8 }}>{data.label}</div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Tag color={STATUS_COLORS[data.status] || 'default'}>{data.status}</Tag>
          {data.status === 'pending' && data.gate_id && data.onApprove && (
            <Button
              type="primary"
              size="small"
              onClick={() => data.onApprove!(data.gate_id!)}
            >
              审批
            </Button>
          )}
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ width: 8, height: 8, background: '#bfbfbf' }} />
    </div>
  );
}
