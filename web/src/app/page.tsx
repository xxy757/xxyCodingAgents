'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Tag,
  Button,
  Input,
  Space,
  Typography,
} from 'antd';
import {
  RobotOutlined,
  DashboardOutlined,
  ThunderboltOutlined,
  ArrowRightOutlined,
  ApiOutlined,
  DatabaseOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useSystemMetrics, useHealth, useReady } from '@/lib/hooks/useSystem';

const { Title, Text, Paragraph } = Typography;

const QUICK_TASKS = [
  { type: 'bugfix', label: '修复 Bug', icon: '🐛' },
  { type: 'build', label: '创建 API', icon: '⚡' },
  { type: 'qa', label: '添加测试', icon: '✅' },
  { type: 'review', label: '代码审查', icon: '🔍' },
  { type: 'docs', label: '写文档', icon: '📝' },
];

const PRESSURE_MAP: Record<string, { label: string; color: 'success' | 'warning' | 'error'; percent: number; desc: string }> = {
  normal: { label: '正常', color: 'success', percent: 30, desc: '系统资源充足，可正常调度任务' },
  warn: { label: '警告', color: 'warning', percent: 60, desc: '资源使用偏高，建议关注' },
  high: { label: '高压', color: 'warning', percent: 85, desc: '资源紧张，低优先级任务可能被暂停' },
  critical: { label: '危险', color: 'error', percent: 100, desc: '资源严重不足，可抢占任务将被驱逐' },
};

function getMemColor(val?: number): 'blue' | 'gold' | 'red' {
  if (val === undefined) return 'blue';
  if (val > 80) return 'red';
  if (val > 60) return 'gold';
  return 'blue';
}

export default function DashboardPage() {
  const router = useRouter();
  const { data: metrics } = useSystemMetrics(5000);
  const { data: health } = useHealth();
  const { data: ready } = useReady();
  const [quickInput, setQuickInput] = useState('');

  const handleSubmit = (type?: string) => {
    const p = new URLSearchParams();
    if (quickInput.trim()) p.set('input', quickInput.trim());
    if (type) p.set('type', type);
    router.push(`/prompt-drafts?${p.toString()}`);
  };

  const pressureKey = metrics?.pressure_level || 'normal';
  const pressure = PRESSURE_MAP[pressureKey] || PRESSURE_MAP.normal;

  const apiHealthy = health?.status === 'ok';
  const dbReady = ready?.status === 'ready';

  return (
    <div style={{ padding: 0 }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>
          仪表盘
        </Title>
        <Text type="secondary">系统资源与服务状态概览</Text>
      </div>

      {/* 快速任务输入区 */}
      <Card
        style={{
          marginBottom: 24,
          background: 'linear-gradient(135deg, #f0f5ff 0%, #e6f4ff 50%, #f0f5ff 100%)',
          borderColor: '#d6e4ff',
        }}
        styles={{ body: { padding: '20px 24px' } }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 16 }}>
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              background: '#1677ff',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#fff',
              boxShadow: '0 2px 8px rgba(22,119,255,0.35)',
            }}
          >
            <ThunderboltOutlined style={{ fontSize: 15 }} />
          </div>
          <div>
            <Text strong style={{ fontSize: 14 }}>
              快速任务
            </Text>
            <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
              输入一句话，AI 帮你拆解执行
            </Text>
          </div>
        </div>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            size="large"
            value={quickInput}
            onChange={(e) => setQuickInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
            placeholder="描述你想完成的任务，例如：给 user 表添加 email 唯一索引..."
            style={{ flex: 1 }}
          />
          <Button
            type="primary"
            size="large"
            icon={<ArrowRightOutlined />}
            onClick={() => handleSubmit()}
          >
            优化提示词
          </Button>
        </Space.Compact>
        <div style={{ marginTop: 12, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {QUICK_TASKS.map((t) => (
            <Button
              key={t.type}
              size="small"
              onClick={() => handleSubmit(t.type)}
            >
              {t.icon} {t.label}
            </Button>
          ))}
        </div>
      </Card>

      {/* 4 个指标卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="活跃 Agent"
              value={metrics?.active_agents ?? 0}
              prefix={<RobotOutlined style={{ color: '#1677ff' }} />}
              valueStyle={{ color: '#1677ff' }}
            />
            <Text type="secondary" style={{ fontSize: 12 }}>
              正在运行的 Agent 实例
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="内存使用"
              value={metrics?.memory_percent ?? 0}
              precision={1}
              suffix="%"
              prefix={<DashboardOutlined />}
              valueStyle={{ color: getMemColor(metrics?.memory_percent) === 'red' ? '#ff4d4f' : getMemColor(metrics?.memory_percent) === 'gold' ? '#d48806' : '#1677ff' }}
            />
            <Progress
              percent={metrics?.memory_percent ?? 0}
              showInfo={false}
              strokeColor={getMemColor(metrics?.memory_percent) === 'red' ? '#ff4d4f' : getMemColor(metrics?.memory_percent) === 'gold' ? '#faad14' : '#1677ff'}
              size="small"
              style={{ marginTop: 8 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="CPU 使用"
              value={metrics?.cpu_percent ?? 0}
              precision={1}
              suffix="%"
              prefix={<DashboardOutlined />}
              valueStyle={{ color: getMemColor(metrics?.cpu_percent) === 'red' ? '#ff4d4f' : getMemColor(metrics?.cpu_percent) === 'gold' ? '#d48806' : '#1677ff' }}
            />
            <Progress
              percent={metrics?.cpu_percent ?? 0}
              showInfo={false}
              strokeColor={getMemColor(metrics?.cpu_percent) === 'red' ? '#ff4d4f' : getMemColor(metrics?.cpu_percent) === 'gold' ? '#faad14' : '#1677ff'}
              size="small"
              style={{ marginTop: 8 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="磁盘使用"
              value={metrics?.disk_percent ?? 0}
              precision={1}
              suffix="%"
              prefix={<DashboardOutlined />}
              valueStyle={{ color: getMemColor(metrics?.disk_percent) === 'red' ? '#ff4d4f' : getMemColor(metrics?.disk_percent) === 'gold' ? '#d48806' : '#1677ff' }}
            />
            <Progress
              percent={metrics?.disk_percent ?? 0}
              showInfo={false}
              strokeColor={getMemColor(metrics?.disk_percent) === 'red' ? '#ff4d4f' : getMemColor(metrics?.disk_percent) === 'gold' ? '#faad14' : '#1677ff'}
              size="small"
              style={{ marginTop: 8 }}
            />
          </Card>
        </Col>
      </Row>

      {/* 底部：压力等级 + 服务状态 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <DashboardOutlined />
                <span>系统压力</span>
              </Space>
            }
            styles={{ body: { padding: '16px 24px' } }}
          >
            <div style={{ textAlign: 'center', marginBottom: 16 }}>
              <Progress
                type="dashboard"
                percent={Math.min(metrics?.memory_percent ?? 0, 100)}
                strokeColor={{
                  '0%': pressure.color === 'success' ? '#52c41a' : pressure.color === 'warning' ? '#faad14' : '#ff4d4f',
                  '100%': pressure.color === 'success' ? '#73d13d' : pressure.color === 'warning' ? '#ffc53d' : '#ff7875',
                }}
                format={() => (
                  <span
                    style={{
                      fontSize: 20,
                      fontWeight: 700,
                      color: pressure.color === 'success' ? '#52c41a' : pressure.color === 'warning' ? '#d48806' : '#ff4d4f',
                    }}
                  >
                    {pressure.label}
                  </span>
                )}
                size={120}
              />
            </div>
            <Paragraph
              type="secondary"
              style={{ textAlign: 'center', fontSize: 12, marginBottom: 0 }}
            >
              {pressure.desc}
            </Paragraph>
          </Card>
        </Col>
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <ApiOutlined />
                <span>服务状态</span>
              </Space>
            }
            styles={{ body: { padding: '16px 24px' } }}
          >
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12}>
                <Card
                  size="small"
                  style={{ background: '#fafafa', borderColor: apiHealthy ? '#b7eb8f' : '#ffa39e' }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Space>
                      <div
                        style={{
                          width: 36,
                          height: 36,
                          borderRadius: 8,
                          background: '#e6f4ff',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          color: '#1677ff',
                        }}
                      >
                        <ApiOutlined style={{ fontSize: 16 }} />
                      </div>
                      <div>
                        <Text strong>后端 API</Text>
                        <br />
                        <Text type="secondary" style={{ fontSize: 11 }}>
                          localhost:8080
                        </Text>
                      </div>
                    </Space>
                    <Tag
                      icon={apiHealthy ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
                      color={apiHealthy ? 'success' : 'error'}
                    >
                      {apiHealthy ? '正常' : '异常'}
                    </Tag>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={12}>
                <Card
                  size="small"
                  style={{ background: '#fafafa', borderColor: dbReady ? '#b7eb8f' : '#ffa39e' }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Space>
                      <div
                        style={{
                          width: 36,
                          height: 36,
                          borderRadius: 8,
                          background: '#f6ffed',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          color: '#52c41a',
                        }}
                      >
                        <DatabaseOutlined style={{ fontSize: 16 }} />
                      </div>
                      <div>
                        <Text strong>数据库</Text>
                        <br />
                        <Text type="secondary" style={{ fontSize: 11 }}>
                          SQLite WAL
                        </Text>
                      </div>
                    </Space>
                    <Tag
                      icon={dbReady ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
                      color={dbReady ? 'success' : 'error'}
                    >
                      {dbReady ? '就绪' : '异常'}
                    </Tag>
                  </div>
                </Card>
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
