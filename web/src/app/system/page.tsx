'use client';

import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Tag,
  Descriptions,
  Space,
  Typography,
  Divider,
} from 'antd';
import {
  DashboardOutlined,
  SettingOutlined,
  CodeOutlined,
  DatabaseOutlined,
  CloudServerOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useSystemMetrics, useDiagnostics } from '@/lib/hooks/useSystem';

const { Title, Text, Paragraph } = Typography;

const PRESSURE_MAP: Record<string, { label: string; color: 'success' | 'warning' | 'error'; icon: React.ReactNode }> = {
  normal: { label: '正常', color: 'success', icon: <CheckCircleOutlined /> },
  warn: { label: '警告', color: 'warning', icon: <ExclamationCircleOutlined /> },
  high: { label: '高压', color: 'warning', icon: <WarningOutlined /> },
  critical: { label: '危险', color: 'error', icon: <CloseCircleOutlined /> },
};

function getResourceColor(val?: number): string {
  if (val === undefined) return '#1677ff';
  if (val > 80) return '#ff4d4f';
  if (val > 60) return '#faad14';
  return '#1677ff';
}

export default function SystemPage() {
  const { data: metrics } = useSystemMetrics(3000);
  const { data: diagnostics } = useDiagnostics();

  const pressureKey = metrics?.pressure_level || 'normal';
  const pressure = PRESSURE_MAP[pressureKey] || PRESSURE_MAP.normal;

  const tmuxLines = diagnostics?.tmux_sessions
    ? diagnostics.tmux_sessions.split('\n').filter((l: string) => l.trim())
    : [];

  return (
    <div style={{ padding: 0 }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>
          系统监控
        </Title>
        <Text type="secondary">资源使用与服务状态</Text>
      </div>

      {/* 资源指标卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="内存使用率"
              value={metrics?.memory_percent ?? 0}
              precision={1}
              suffix="%"
              prefix={<DatabaseOutlined style={{ color: getResourceColor(metrics?.memory_percent) }} />}
              valueStyle={{ color: getResourceColor(metrics?.memory_percent) }}
            />
            <Progress
              percent={metrics?.memory_percent ?? 0}
              showInfo={false}
              strokeColor={getResourceColor(metrics?.memory_percent)}
              size="small"
              style={{ marginTop: 8 }}
            />
            <Text type="secondary" style={{ fontSize: 12 }}>
              系统内存占用
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="CPU 使用率"
              value={metrics?.cpu_percent ?? 0}
              precision={1}
              suffix="%"
              prefix={<DashboardOutlined style={{ color: getResourceColor(metrics?.cpu_percent) }} />}
              valueStyle={{ color: getResourceColor(metrics?.cpu_percent) }}
            />
            <Progress
              percent={metrics?.cpu_percent ?? 0}
              showInfo={false}
              strokeColor={getResourceColor(metrics?.cpu_percent)}
              size="small"
              style={{ marginTop: 8 }}
            />
            <Text type="secondary" style={{ fontSize: 12 }}>
              处理器负载
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card styles={{ body: { padding: '20px 24px' } }}>
            <Statistic
              title="磁盘使用率"
              value={metrics?.disk_percent ?? 0}
              precision={1}
              suffix="%"
              prefix={<CloudServerOutlined style={{ color: getResourceColor(metrics?.disk_percent) }} />}
              valueStyle={{ color: getResourceColor(metrics?.disk_percent) }}
            />
            <Progress
              percent={metrics?.disk_percent ?? 0}
              showInfo={false}
              strokeColor={getResourceColor(metrics?.disk_percent)}
              size="small"
              style={{ marginTop: 8 }}
            />
            <Text type="secondary" style={{ fontSize: 12 }}>
              磁盘空间占用
            </Text>
          </Card>
        </Col>
      </Row>

      {/* 压力 + tmux + 配置 */}
      <Row gutter={[16, 16]}>
        {/* 压力状态 */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <ThunderboltOutlined style={{ color: '#faad14' }} />
                <span>压力状态</span>
              </Space>
            }
            styles={{ body: { padding: '16px 24px' } }}
          >
            <div style={{ textAlign: 'center', marginBottom: 20 }}>
              <Progress
                type="dashboard"
                percent={Math.min(metrics?.memory_percent ?? 0, 100)}
                strokeColor={{
                  '0%': pressure.color === 'success' ? '#52c41a' : pressure.color === 'warning' ? '#faad14' : '#ff4d4f',
                  '100%': pressure.color === 'success' ? '#73d13d' : pressure.color === 'warning' ? '#ffc53d' : '#ff7875',
                }}
                format={() => (
                  <span style={{ fontSize: 18, fontWeight: 700 }}>
                    {pressure.label}
                  </span>
                )}
                size={140}
              />
            </div>
            <Divider style={{ margin: '12px 0' }} />
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 12 }}>
              <Text type="secondary">活跃 Agent</Text>
              <Statistic
                value={metrics?.active_agents ?? 0}
                valueStyle={{ fontWeight: 700, fontSize: 24 }}
              />
            </div>
          </Card>
        </Col>

        {/* tmux 会话 */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <CodeOutlined style={{ color: '#1677ff' }} />
                <span>tmux 会话</span>
              </Space>
            }
            styles={{ body: { padding: '16px 24px' } }}
          >
            <div
              style={{
                background: '#1f1f1f',
                borderRadius: 8,
                border: '1px solid #303030',
                padding: 16,
                minHeight: 180,
                maxHeight: 260,
                overflowY: 'auto',
              }}
            >
              {tmuxLines.length > 0 ? (
                tmuxLines.map((line: string, idx: number) => (
                  <div
                    key={idx}
                    style={{
                      fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', Menlo, monospace",
                      fontSize: 12,
                      lineHeight: 1.8,
                      color: '#52c41a',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-all',
                    }}
                  >
                    {line}
                  </div>
                ))
              ) : (
                <Text
                  style={{
                    color: '#666',
                    fontFamily: "'SF Mono', Menlo, monospace",
                    fontSize: 12,
                  }}
                >
                  无活跃会话
                </Text>
              )}
            </div>
          </Card>
        </Col>

        {/* 调度配置 */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <SettingOutlined style={{ color: '#1677ff' }} />
                <span>调度配置</span>
              </Space>
            }
            styles={{ body: { padding: '16px 24px' } }}
          >
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="最大并发 Agent">
                <Text strong>
                  {diagnostics?.config?.max_concurrent_agents ?? '-'}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="最大重型任务">
                <Text strong>
                  {diagnostics?.config?.max_heavy_agents ?? '-'}
                </Text>
              </Descriptions.Item>
            </Descriptions>

            {diagnostics?.active_agents && diagnostics.active_agents.length > 0 && (
              <>
                <Divider style={{ margin: '16px 0 12px' }} />
                <Text
                  type="secondary"
                  style={{ fontSize: 12, display: 'block', marginBottom: 8 }}
                >
                  已注册 Agent
                </Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                  {diagnostics.active_agents.map((agent: string) => (
                    <Tag key={agent} color="blue" style={{ margin: 0 }}>
                      {agent}
                    </Tag>
                  ))}
                </div>
              </>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}
