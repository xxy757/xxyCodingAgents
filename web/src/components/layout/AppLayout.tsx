'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Layout, Menu, theme, Avatar, Badge, Breadcrumb } from 'antd';
import {
  DashboardOutlined,
  FolderOpenOutlined,
  PlayCircleOutlined,
  RobotOutlined,
  CodeOutlined,
  SettingOutlined,
  BulbOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SearchOutlined,
  BellOutlined,
  UserOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';

const { Header, Sider, Content } = Layout;

const NAV_ITEMS = [
  { key: '/', icon: DashboardOutlined, label: '仪表盘' },
  { key: '/projects', icon: FolderOpenOutlined, label: '项目管理' },
  { key: '/runs', icon: PlayCircleOutlined, label: '运行中心' },
  { key: '/agents', icon: RobotOutlined, label: 'Agent 实例' },
  { key: '/terminals', icon: CodeOutlined, label: '终端管理' },
  { key: '/system', icon: SettingOutlined, label: '系统监控' },
  { key: '/prompt-drafts', icon: BulbOutlined, label: '提示词工作台' },
];

const ROUTE_LABELS: Record<string, string> = {
  '': '仪表盘',
  projects: '项目管理',
  runs: '运行中心',
  agents: 'Agent 实例',
  terminals: '终端管理',
  system: '系统监控',
  'prompt-drafts': '提示词工作台',
};

export function AppLayout({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);
  const pathname = usePathname();
  const { token } = theme.useToken();

  const selectedKey = NAV_ITEMS.find(
    (item) => (item.key === '/' ? pathname === '/' : pathname.startsWith(item.key)),
  )?.key || '/';

  const menuItems: MenuProps['items'] = NAV_ITEMS.map((item) => ({
    key: item.key,
    icon: <item.icon />,
    label: <Link href={item.key}>{item.label}</Link>,
  }));

  const segments = pathname.split('/').filter(Boolean);
  const breadcrumbItems = [
    { title: <Link href="/">首页</Link> },
    ...segments.map((seg, i) => {
      const href = '/' + segments.slice(0, i + 1).join('/');
      const isLast = i === segments.length - 1;
      return {
        title: isLast
          ? ROUTE_LABELS[seg] || seg
          : <Link href={href}>{ROUTE_LABELS[seg] || seg}</Link>,
      };
    }),
  ];

  return (
    <Layout className="h-screen overflow-hidden">
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={256}
        collapsedWidth={80}
        style={{ overflow: 'auto', height: '100vh', position: 'fixed', left: 0, top: 0, bottom: 0 }}
      >
        {/* Logo */}
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: collapsed ? 'center' : 'flex-start',
            padding: collapsed ? '0' : '0 24px',
            borderBottom: '1px solid rgba(255,255,255,0.06)',
            gap: 12,
            overflow: 'hidden',
          }}
        >
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
              fontWeight: 700,
              fontSize: 14,
              flexShrink: 0,
              boxShadow: '0 4px 12px rgba(22,119,255,0.3)',
            }}
          >
            AI
          </div>
          {!collapsed && (
            <div style={{ display: 'flex', flexDirection: 'column' }}>
              <span style={{ fontSize: 15, fontWeight: 600, color: '#fff', letterSpacing: '-0.3px' }}>
                Dev Platform
              </span>
              <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>v0.1.0</span>
            </div>
          )}
        </div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          style={{ borderRight: 0, marginTop: 8 }}
        />

        {/* 折叠按钮 */}
        <div
          style={{
            position: 'absolute',
            bottom: 0,
            width: '100%',
            borderTop: '1px solid rgba(255,255,255,0.06)',
            padding: 12,
          }}
        >
          <div
            onClick={() => setCollapsed(!collapsed)}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: 36,
              borderRadius: 6,
              color: 'rgba(255,255,255,0.3)',
              cursor: 'pointer',
              transition: 'all 0.15s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'rgba(255,255,255,0.7)';
              e.currentTarget.style.background = 'rgba(255,255,255,0.06)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'rgba(255,255,255,0.3)';
              e.currentTarget.style.background = 'transparent';
            }}
          >
            {collapsed ? <MenuUnfoldOutlined style={{ fontSize: 16 }} /> : <MenuFoldOutlined style={{ fontSize: 16 }} />}
          </div>
        </div>
      </Sider>

      <Layout style={{ marginLeft: collapsed ? 80 : 256, transition: 'margin-left 0.2s' }}>
        <Header
          style={{
            padding: '0 24px',
            background: '#fff',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            height: 56,
            borderBottom: '1px solid #f0f0f0',
            boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
            position: 'sticky',
            top: 0,
            zIndex: 10,
          }}
        >
          <Breadcrumb items={breadcrumbItems} />

          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: 6,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'rgba(0,0,0,0.45)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = '#f5f5f5';
                e.currentTarget.style.color = 'rgba(0,0,0,0.88)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'transparent';
                e.currentTarget.style.color = 'rgba(0,0,0,0.45)';
              }}
            >
              <SearchOutlined style={{ fontSize: 16 }} />
            </div>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: 6,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'rgba(0,0,0,0.45)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = '#f5f5f5';
                e.currentTarget.style.color = 'rgba(0,0,0,0.88)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'transparent';
                e.currentTarget.style.color = 'rgba(0,0,0,0.45)';
              }}
            >
              <QuestionCircleOutlined style={{ fontSize: 16 }} />
            </div>
            <Badge size="small" count={1} offset={[-2, 2]}>
              <div
                style={{
                  width: 32,
                  height: 32,
                  borderRadius: 6,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'rgba(0,0,0,0.45)',
                  cursor: 'pointer',
                  transition: 'all 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = '#f5f5f5';
                  e.currentTarget.style.color = 'rgba(0,0,0,0.88)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                  e.currentTarget.style.color = 'rgba(0,0,0,0.45)';
                }}
              >
                <BellOutlined style={{ fontSize: 16 }} />
              </div>
            </Badge>

            <div style={{ width: 1, height: 20, background: '#f0f0f0', margin: '0 8px' }} />

            <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingLeft: 4 }}>
              <Avatar size={28} style={{ backgroundColor: '#1677ff', fontSize: 12 }}>A</Avatar>
              <span style={{ fontSize: 14, color: 'rgba(0,0,0,0.65)', fontWeight: 500 }}>Admin</span>
            </div>
          </div>
        </Header>

        <Content
          id="main-content"
          style={{
            flex: 1,
            overflow: 'auto',
            background: '#f0f2f5',
            padding: 24,
          }}
        >
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}
