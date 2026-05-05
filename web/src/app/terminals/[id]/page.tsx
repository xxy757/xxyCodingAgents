'use client';

import { useEffect, useState } from 'react';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { Typography, Tag, Button, Space, Spin, Alert } from 'antd';
import { useTerminal } from '@/lib/hooks/useTerminals';
import { TerminalView } from '@/components/terminal/TerminalView';
import { shortId } from '@/lib/utils';
import Link from 'next/link';

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

export default function TerminalDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState('');
  useEffect(() => { params.then((p) => setId(p.id)); }, [params]);

  const { data: session, error } = useTerminal(id);

  if (!id) {
    return (
      <div style={{ textAlign: 'center', padding: '48px 0' }}>
        <Spin tip="加载中..." />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 24 }}>
        <Alert type="error" message={error.message} showIcon />
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{
        marginBottom: 16, display: 'flex', justifyContent: 'space-between',
        alignItems: 'center', flexShrink: 0,
      }}>
        <Space align="center" size={12}>
          <Link href="/terminals">
            <Button type="text" icon={<ArrowLeftOutlined />} style={{ color: 'rgba(0,0,0,0.45)' }} />
          </Link>
          <div>
            <Title level={4} style={{ margin: 0 }}>终端</Title>
            <Text type="secondary" code>{shortId(id)}</Text>
            <Text type="secondary"> | </Text>
            <Text type="secondary">tmux: </Text>
            <Text code>{session?.tmux_session || '-'}</Text>
          </div>
        </Space>
        {session && statusTag(session.status)}
      </div>
      <div style={{ flex: 1, minHeight: 0 }}>
        <TerminalView sessionId={id} className="h-full" />
      </div>
    </div>
  );
}
