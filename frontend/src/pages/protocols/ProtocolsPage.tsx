import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  ConfigProvider,
  Descriptions,
  Layout,
  Modal,
  Row,
  Spin,
  Statistic,
  Tag,
  Table,
  Tooltip,
  Typography,
  Space,
  message,
} from 'antd';
import {
  PlayCircleOutlined,
  StopOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  QuestionCircleOutlined,
  ApiOutlined,
  CloudServerOutlined,
  BranchesOutlined,
  DashboardOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';

import { useTheme } from '@/hooks/useTheme';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import AppSidebar from '@/layouts/AppSidebar';
import { useProtocolsQuery, useProtocolMutations } from '@/api/queries/useProtocolsQuery';
import { setMessageInstance } from '@/utils/messageBus';

const { Title, Text } = Typography;

const STATUS_META: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
  running: { label: 'Running', color: 'green', icon: <CheckCircleOutlined /> },
  stopped: { label: 'Stopped', color: 'default', icon: <CloseCircleOutlined /> },
  error: { label: 'Error', color: 'red', icon: <CloseCircleOutlined /> },
  installing: { label: 'Installing', color: 'processing', icon: <ReloadOutlined spin /> },
  unknown: { label: 'Unknown', color: 'warning', icon: <QuestionCircleOutlined /> },
};

export default function ProtocolsPage() {
  const { antdThemeConfig } = useTheme();
  const { isMobile } = useMediaQuery();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => { setMessageInstance(messageApi); }, [messageApi]);

  const { protocols, fetched, fetchError, refresh } = useProtocolsQuery();
  const { startProtocol, stopProtocol, restartProtocol, isStarting, isStopping, isRestarting } = useProtocolMutations();

  const [detailProtocol, setDetailProtocol] = useState<any>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const baseProtocols = useMemo(() => protocols.filter(p => p.category === 'base'), [protocols]);
  const standaloneProtocols = useMemo(() => protocols.filter(p => p.category === 'standalone'), [protocols]);
  const wrappers = useMemo(() => protocols.filter(p => p.category === 'wrapper'), [protocols]);

  const showDetail = useCallback((record: any) => {
    setDetailProtocol(record);
    setDetailOpen(true);
  }, []);

  const handleStart = useCallback(async (id: string) => {
    try {
      await startProtocol(id);
      messageApi.success(`${id} started`);
    } catch (e: any) {
      messageApi.error(e.message || `Failed to start ${id}`);
    }
  }, [startProtocol, messageApi]);

  const handleStop = useCallback(async (id: string) => {
    try {
      await stopProtocol(id);
      messageApi.success(`${id} stopped`);
    } catch (e: any) {
      messageApi.error(e.message || `Failed to stop ${id}`);
    }
  }, [stopProtocol, messageApi]);

  const handleRestart = useCallback(async (id: string) => {
    try {
      await restartProtocol(id);
      messageApi.success(`${id} restarted`);
    } catch (e: any) {
      messageApi.error(e.message || `Failed to restart ${id}`);
    }
  }, [restartProtocol, messageApi]);

  const columns = useMemo(() => [
    {
      title: 'Protocol',
      dataIndex: 'name',
      key: 'name',
      render: (_: any, record: any) => (
        <Space>
          <Text strong>{record.name}</Text>
          {record.xrayNative && <Tag color="geekblue">Xray</Tag>}
          <Text type="secondary" style={{ fontSize: 12 }}>{record.id}</Text>
        </Space>
      ),
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      responsive: ['md' as const],
    },
    {
      title: 'Port',
      dataIndex: 'port',
      key: 'port',
      width: 80,
      render: (port: number | undefined) => port ? <Text code>{port}</Text> : <Text type="secondary">-</Text>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const meta = STATUS_META[status] || STATUS_META.unknown;
        return <Tag icon={meta.icon} color={meta.color}>{meta.label}</Tag>;
      },
    },
    {
      title: 'Health',
      key: 'health',
      width: 100,
      render: (_: any, record: any) => (
        record.healthy
          ? <Tag icon={<CheckCircleOutlined />} color="green">OK</Tag>
          : <Tag icon={<CloseCircleOutlined />} color="red">Down</Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 280,
      render: (_: any, record: any) => (
        <Space>
          <Tooltip title="View details">
            <Button size="small" icon={<InfoCircleOutlined />} onClick={() => showDetail(record)} />
          </Tooltip>
          <Tooltip title={`Start ${record.name}`}>
            <Button
              type="primary"
              size="small"
              icon={<PlayCircleOutlined />}
              disabled={record.status === 'running'}
              loading={isStarting}
              onClick={() => handleStart(record.id)}
            />
          </Tooltip>
          <Tooltip title={`Stop ${record.name}`}>
            <Button
              danger
              size="small"
              icon={<StopOutlined />}
              disabled={record.status === 'stopped' || record.status === 'unknown'}
              loading={isStopping}
              onClick={() => handleStop(record.id)}
            />
          </Tooltip>
          <Tooltip title={`Restart ${record.name}`}>
            <Button
              size="small"
              icon={<ReloadOutlined />}
              disabled={record.status !== 'running'}
              loading={isRestarting}
              onClick={() => handleRestart(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ], [isStarting, isStopping, isRestarting, handleStart, handleStop, handleRestart, showDetail]);

  const renderCategorySection = (title: string, icon: React.ReactNode, color: string, data: any[]) => (
    <Card
      title={<Space><span style={{ color }}>{icon}</span><span>{title}</span></Space>}
      extra={
        <Space>
          <Tag>{data.length} protocols</Tag>
          <Button size="small" icon={<ReloadOutlined />} onClick={refresh}>Refresh</Button>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Table
        dataSource={data}
        columns={columns}
        rowKey="id"
        pagination={false}
        size="small"
        locale={{ emptyText: 'No protocols in this category' }}
      />
    </Card>
  );

  if (!fetched) {
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <AppSidebar />
        <Layout.Content style={{ padding: 24, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Spin size="large" tip="Loading protocols..." />
        </Layout.Content>
      </Layout>
    );
  }

  if (fetchError) {
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <AppSidebar />
        <Layout.Content style={{ padding: 24 }}>
          <Card>
            <Space direction="vertical" align="center" style={{ width: '100%' }}>
              <CloseCircleOutlined style={{ fontSize: 48, color: '#ff4d4f' }} />
              <Title level={4}>Failed to load protocols</Title>
              <Text type="secondary">{fetchError}</Text>
              <Button type="primary" onClick={refresh}>Retry</Button>
            </Space>
          </Card>
        </Layout.Content>
      </Layout>
    );
  }

  const allRunning = protocols.filter(p => p.status === 'running').length;
  const allHealthy = protocols.filter(p => p.healthy).length;

  return (
    <ConfigProvider theme={antdThemeConfig}>
      <Layout style={{ minHeight: '100vh' }}>
        <AppSidebar />
        <Layout.Content style={{ padding: isMobile ? 12 : 24 }}>
          {messageContextHolder}

          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="Total Protocols"
                  value={protocols.length}
                  prefix={<DashboardOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="Running"
                  value={allRunning}
                  suffix={`/ ${protocols.length}`}
                  valueStyle={{ color: allRunning === protocols.length ? '#3f8600' : '#faad14' }}
                  prefix={<PlayCircleOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title="Healthy"
                  value={allHealthy}
                  suffix={`/ ${protocols.length}`}
                  valueStyle={{ color: allHealthy === protocols.length ? '#3f8600' : '#ff4d4f' }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
          </Row>

          {renderCategorySection('Base Protocols (Xray-native)', <ApiOutlined />, '#1677ff', baseProtocols)}
          {renderCategorySection('Standalone Services', <CloudServerOutlined />, '#52c41a', standaloneProtocols)}
          {renderCategorySection('Transport Wrappers', <BranchesOutlined />, '#722ed1', wrappers)}

          <Modal
            title={detailProtocol ? `${detailProtocol.name} Details` : 'Protocol Details'}
            open={detailOpen}
            onCancel={() => setDetailOpen(false)}
            footer={<Button onClick={() => setDetailOpen(false)}>Close</Button>}
            width={600}
          >
            {detailProtocol && (
              <Descriptions column={1} bordered size="small">
                <Descriptions.Item label="ID">{detailProtocol.id}</Descriptions.Item>
                <Descriptions.Item label="Name">{detailProtocol.name}</Descriptions.Item>
                <Descriptions.Item label="Category">
                  <Tag color={
                    detailProtocol.category === 'base' ? 'blue' :
                    detailProtocol.category === 'standalone' ? 'green' : 'purple'
                  }>
                    {detailProtocol.category}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Description">{detailProtocol.description || '-'}</Descriptions.Item>
                <Descriptions.Item label="Source">{detailProtocol.source || '-'}</Descriptions.Item>
                <Descriptions.Item label="Xray Native">
                  {detailProtocol.xrayNative ? <Tag color="geekblue">Yes</Tag> : <Tag>No</Tag>}
                </Descriptions.Item>
                <Descriptions.Item label="Status">
                  <Tag icon={STATUS_META[detailProtocol.status]?.icon} color={STATUS_META[detailProtocol.status]?.color}>
                    {STATUS_META[detailProtocol.status]?.label || 'Unknown'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Healthy">
                  {detailProtocol.healthy
                    ? <Tag icon={<CheckCircleOutlined />} color="green">OK</Tag>
                    : <Tag icon={<CloseCircleOutlined />} color="red">Down</Tag>}
                </Descriptions.Item>
                {detailProtocol.port && (
                  <Descriptions.Item label="Port"><Text code>{detailProtocol.port}</Text></Descriptions.Item>
                )}
                {detailProtocol.installed !== undefined && (
                  <Descriptions.Item label="Installed">
                    {detailProtocol.installed ? <Tag color="green">Yes</Tag> : <Tag color="red">No</Tag>}
                  </Descriptions.Item>
                )}
                {detailProtocol.serviceName && (
                  <Descriptions.Item label="Service Name">{detailProtocol.serviceName}</Descriptions.Item>
                )}
                {detailProtocol.supportedProtocols && detailProtocol.supportedProtocols.length > 0 && (
                  <Descriptions.Item label="Supported Protocols">
                    <Space wrap>
                      {detailProtocol.supportedProtocols.map((sp: string) => (
                        <Tag key={sp}>{sp}</Tag>
                      ))}
                    </Space>
                  </Descriptions.Item>
                )}
              </Descriptions>
            )}
          </Modal>
        </Layout.Content>
      </Layout>
    </ConfigProvider>
  );
}
