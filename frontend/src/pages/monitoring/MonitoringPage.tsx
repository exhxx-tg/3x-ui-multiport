import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Card,
  Col,
  ColorPicker,
  ConfigProvider,
  Drawer,
  Form,
  Input,
  InputNumber,
  Layout,
  message,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Switch,
  Table,
  Tag,
  Tabs,
  Tooltip,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  InfoCircleOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  BellOutlined,
  HistoryOutlined,
  HeartOutlined,
  SettingOutlined,
  FileTextOutlined,
} from '@ant-design/icons';

import { useTheme } from '@/hooks/useTheme';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import AppSidebar from '@/layouts/AppSidebar';
import { HttpUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

// ---- API helpers ----

async function fetchHealthSummary() {
  const msg = await HttpUtil.get('/panel/api/monitor/health', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch health');
  return msg.obj || {};
}

async function fetchAllMetrics() {
  const msg = await HttpUtil.get('/panel/api/monitor/metrics', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch metrics');
  return msg.obj || {};
}

async function fetchAlertRules() {
  const msg = await HttpUtil.get('/panel/api/monitor/alerts/rules', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch alert rules');
  return msg.obj || [];
}

async function fetchAlertHistory(params?: Record<string, string>) {
  const query = params ? '?' + new URLSearchParams(params).toString() : '';
  const msg = await HttpUtil.get(`/panel/api/monitor/alerts/history${query}`, undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch alert history');
  return msg.obj || { items: [], total: 0 };
}

async function createAlertRule(data: any) {
  const msg = await HttpUtil.post('/panel/api/monitor/alerts/rules', data);
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to create rule');
  return msg.obj;
}

async function updateAlertRule(id: string, data: any) {
  const msg = await HttpUtil.put(`/panel/api/monitor/alerts/rules/${id}`, data);
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to update rule');
  return msg.obj;
}

async function deleteAlertRule(id: string) {
  const msg = await HttpUtil.del(`/panel/api/monitor/alerts/rules/${id}`);
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to delete rule');
}

async function testAlertRule(id: string) {
  const msg = await HttpUtil.post(`/panel/api/monitor/alerts/rules/${id}/test`);
  if (!msg?.success) throw new Error(msg?.msg || 'Test alert failed');
  return msg.obj;
}

async function ackAlert(id: number) {
  const msg = await HttpUtil.post(`/panel/api/monitor/alerts/history/${id}/ack`);
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to acknowledge');
}

// ---- Components ----

function ProtocolHealthCard({ data }: { data: any }) {
  const healthy = data.healthy;
  return (
    <Card
      size="small"
      style={{ marginBottom: 8 }}
      bodyStyle={{ padding: '8px 12px' }}
    >
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Space>
          {healthy
            ? <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 16 }} />
            : <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 16 }} />
          }
          <Text strong>{data.protocolId}</Text>
          {data.latency > 0 && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {data.latency}ms
            </Text>
          )}
        </Space>
        <Tag color={healthy ? 'green' : 'red'}>
          {healthy ? 'Healthy' : 'Unhealthy'}
        </Tag>
      </Space>
      {data.error && (
        <Text type="danger" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
          {data.error}
        </Text>
      )}
    </Card>
  );
}

const RULE_FORM_INITIAL: any = {
  name: '',
  description: '',
  protocolId: undefined,
  metric: 'errorCount',
  condition: 'gt',
  threshold: 0,
  duration: 30,
  severity: 'warning',
  enabled: true,
  cooldown: 300,
  channels: [],
  autoRecovery: false,
};

export default function MonitoringPage() {
  const { antdThemeConfig } = useTheme();
  const { isMobile } = useMediaQuery();
  const [messageApi, msgContext] = message.useMessage();
  useEffect(() => { setMessageInstance(messageApi); }, [messageApi]);

  // Data state
  const [healthSummary, setHealthSummary] = useState<any>(null);
  const [metrics, setMetrics] = useState<Record<string, any>>({});
  const [rules, setRules] = useState<any[]>([]);
  const [alertHistory, setAlertHistory] = useState<any[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  // Rule form state
  const [ruleDrawerOpen, setRuleDrawerOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<any>(null);
  const [ruleForm] = Form.useForm();
  const [savingRule, setSavingRule] = useState(false);

  const loadData = useCallback(async () => {
    try {
      const [health, metricsData, rulesData, historyData] = await Promise.all([
        fetchHealthSummary(),
        fetchAllMetrics(),
        fetchAlertRules(),
        fetchAlertHistory({ limit: '50' }),
      ]);
      setHealthSummary(health);
      setMetrics(metricsData);
      setRules(rulesData);
      setAlertHistory(historyData.items || []);
      setHistoryTotal(historyData.total || 0);
    } catch (e: any) {
      messageApi.error(e.message || 'Failed to load monitoring data');
    } finally {
      setLoading(false);
    }
  }, [messageApi]);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 10000);
    return () => clearInterval(interval);
  }, [loadData]);

  // Rule CRUD
  const openCreateRule = useCallback(() => {
    setEditingRule(null);
    ruleForm.resetFields();
    ruleForm.setFieldsValue(RULE_FORM_INITIAL);
    setRuleDrawerOpen(true);
  }, [ruleForm]);

  const openEditRule = useCallback((rule: any) => {
    setEditingRule(rule);
    ruleForm.setFieldsValue({
      name: rule.name,
      description: rule.description || '',
      protocolId: rule.protocolId,
      metric: rule.metric,
      condition: rule.condition,
      threshold: rule.threshold,
      duration: rule.duration,
      severity: rule.severity,
      enabled: rule.enabled,
      cooldown: rule.cooldown,
      channels: rule.channels || [],
      autoRecovery: rule.autoRecovery,
    });
    setRuleDrawerOpen(true);
  }, [ruleForm]);

  const handleSaveRule = useCallback(async () => {
    try {
      const values = await ruleForm.validateFields();
      setSavingRule(true);
      if (editingRule) {
        await updateAlertRule(editingRule.id, values);
        messageApi.success('Rule updated');
      } else {
        await createAlertRule(values);
        messageApi.success('Rule created');
      }
      setRuleDrawerOpen(false);
      loadData();
    } catch (e: any) {
      if (e.errorFields) return; // validation errors
      messageApi.error(e.message || 'Failed to save rule');
    } finally {
      setSavingRule(false);
    }
  }, [ruleForm, editingRule, messageApi, loadData]);

  const handleDeleteRule = useCallback(async (id: string) => {
    try {
      await deleteAlertRule(id);
      messageApi.success('Rule deleted');
      loadData();
    } catch (e: any) {
      messageApi.error(e.message || 'Failed to delete rule');
    }
  }, [messageApi, loadData]);

  const handleTestRule = useCallback(async (id: string) => {
    try {
      const result = await testAlertRule(id);
      messageApi.success(`Test alert sent: ${result?.status || 'ok'}`);
    } catch (e: any) {
      messageApi.error(e.message || 'Test failed');
    }
  }, [messageApi]);

  const handleAck = useCallback(async (id: number) => {
    try {
      await ackAlert(id);
      messageApi.success('Alert acknowledged');
      loadData();
    } catch (e: any) {
      messageApi.error(e.message || 'Failed to acknowledge');
    }
  }, [messageApi, loadData]);

  // Compute stats
  const totalProtocols = healthSummary?.totalProtocols || 0;
  const healthyCount = healthSummary?.healthy || 0;
  const unhealthyCount = healthSummary?.unhealthy || 0;

  const ruleColumns = useMemo(() => [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (_: any, r: any) => (
        <Space>
          {r.enabled ? <BellOutlined style={{ color: '#1677ff' }} /> : <BellOutlined style={{ color: '#d9d9d9' }} />}
          <Text strong>{r.name}</Text>
        </Space>
      ),
    },
    {
      title: 'Protocol',
      dataIndex: 'protocolId',
      key: 'protocolId',
      width: 120,
    },
    {
      title: 'Condition',
      key: 'condition',
      width: 200,
      render: (_: any, r: any) => (
        <Text>
          {r.metric} {r.condition} {r.threshold}
        </Text>
      ),
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (s: string) => {
        const colors: Record<string, string> = { critical: 'red', warning: 'orange', info: 'blue' };
        return <Tag color={colors[s] || 'default'}>{s}</Tag>;
      },
    },
    {
      title: 'Auto',
      dataIndex: 'autoRecovery',
      key: 'autoRecovery',
      width: 80,
      render: (v: boolean) => v ? <Tag color="green">Yes</Tag> : <Tag>No</Tag>,
    },
    {
      title: 'Enabled',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (v: boolean) => v
        ? <Tag icon={<CheckCircleOutlined />} color="success">On</Tag>
        : <Tag color="default">Off</Tag>,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 200,
      render: (_: any, r: any) => (
        <Space>
          <Tooltip title="Edit"><Button size="small" icon={<EditOutlined />} onClick={() => openEditRule(r)} /></Tooltip>
          <Tooltip title="Test"><Button size="small" icon={<PlayCircleOutlined />} onClick={() => handleTestRule(r.id)} /></Tooltip>
          <Popconfirm title="Delete this rule?" onConfirm={() => handleDeleteRule(r.id)}>
            <Tooltip title="Delete"><Button size="small" danger icon={<DeleteOutlined />} /></Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ], [openEditRule, handleTestRule, handleDeleteRule]);

  const historyColumns = useMemo(() => [
    {
      title: 'Time',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (v: number) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: 'Rule',
      dataIndex: 'ruleName',
      key: 'ruleName',
      width: 150,
    },
    {
      title: 'Protocol',
      dataIndex: 'protocolId',
      key: 'protocolId',
      width: 100,
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      width: 90,
      render: (s: string) => {
        const colors: Record<string, string> = { critical: 'red', warning: 'orange', info: 'blue' };
        return <Tag color={colors[s] || 'default'}>{s}</Tag>;
      },
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (s: string) => {
        const meta: Record<string, { color: string; icon: any }> = {
          firing: { color: 'red', icon: <CloseCircleOutlined /> },
          resolved: { color: 'green', icon: <CheckCircleOutlined /> },
          acknowledged: { color: 'blue', icon: <InfoCircleOutlined /> },
        };
        const m = meta[s] || { color: 'default', icon: <WarningOutlined /> };
        return <Tag icon={m.icon} color={m.color}>{s}</Tag>;
      },
    },
    {
      title: 'Message',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 100,
      render: (_: any, r: any) => (
        r.status === 'firing' ? (
          <Button size="small" onClick={() => handleAck(r.id)}>Acknowledge</Button>
        ) : null
      ),
    },
  ], [handleAck]);

  const metricCards = useMemo(() => Object.entries(metrics).map(([id, m]: [string, any]) => (
    <Col xs={24} sm={12} md={8} lg={6} key={id}>
      <Card size="small" title={<Text strong>{id}</Text>} style={{ marginBottom: 8 }}>
        <Space direction="vertical" size={2} style={{ width: '100%' }}>
          <Text type="secondary">↑ {m.upBytes || 0} bytes</Text>
          <Text type="secondary">↓ {m.downBytes || 0} bytes</Text>
          <Text>Connections: {m.connections || 0}</Text>
          <Text>Active users: {m.activeUsers || 0}</Text>
          <Text>Errors: {m.errorCount || 0}</Text>
          <Text>Uptime: {m.uptimeSeconds || 0}s</Text>
        </Space>
      </Card>
    </Col>
  )), [metrics]);

  if (loading) {
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <AppSidebar />
        <Layout.Content style={{ padding: 24, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Spin size="large" tip="Loading monitoring data..." />
        </Layout.Content>
      </Layout>
    );
  }

  return (
    <ConfigProvider theme={antdThemeConfig}>
      <Layout style={{ minHeight: '100vh' }}>
        <AppSidebar />
        <Layout.Content style={{ padding: isMobile ? 12 : 24 }}>
          {msgContext}

          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic title="Protocols" value={totalProtocols} prefix={<HeartOutlined />} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="Healthy"
                  value={healthyCount}
                  valueStyle={{ color: '#3f8600' }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="Unhealthy"
                  value={unhealthyCount}
                  valueStyle={{ color: '#cf1322' }}
                  prefix={<CloseCircleOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic title="Alert Rules" value={rules.length} prefix={<BellOutlined />} />
              </Card>
            </Col>
          </Row>

          <Tabs defaultActiveKey="health" style={{ marginTop: 8 }}>
            <TabPane
              tab={<span><HeartOutlined /> Health Status</span>}
              key="health"
            >
              <Card
                title="Protocol Health"
                extra={<Button size="small" icon={<ReloadOutlined />} onClick={loadData}>Refresh</Button>}
              >
                {healthSummary?.results?.length > 0 ? (
                  <Row gutter={[16, 16]}>
                    {healthSummary.results.map((r: any) => (
                      <Col xs={24} sm={12} md={8} lg={6} key={r.protocolId}>
                        <ProtocolHealthCard data={r} />
                      </Col>
                    ))}
                  </Row>
                ) : (
                  <Text type="secondary">No health data available</Text>
                )}
              </Card>
            </TabPane>

            <TabPane
              tab={<span><FileTextOutlined /> Metrics</span>}
              key="metrics"
            >
              <Card
                title="Protocol Metrics"
                extra={<Button size="small" icon={<ReloadOutlined />} onClick={loadData}>Refresh</Button>}
              >
                <Row gutter={[16, 16]}>
                  {metricCards.length > 0 ? metricCards : (
                    <Col span={24}><Text type="secondary">No metrics data available</Text></Col>
                  )}
                </Row>
              </Card>
            </TabPane>

            <TabPane
              tab={<span><BellOutlined /> Alert Rules</span>}
              key="rules"
            >
              <Card
                title="Alert Rules"
                extra={<Button type="primary" size="small" icon={<PlusOutlined />} onClick={openCreateRule}>Add Rule</Button>}
              >
                <Table
                  dataSource={rules}
                  columns={ruleColumns}
                  rowKey="id"
                  size="small"
                  pagination={false}
                  locale={{ emptyText: 'No alert rules configured' }}
                />
              </Card>
            </TabPane>

            <TabPane
              tab={<span><HistoryOutlined /> Alert History</span>}
              key="history"
            >
              <Card
                title={`Alert History (${historyTotal})`}
                extra={<Button size="small" icon={<ReloadOutlined />} onClick={loadData}>Refresh</Button>}
              >
                <Table
                  dataSource={alertHistory}
                  columns={historyColumns}
                  rowKey="id"
                  size="small"
                  pagination={{ pageSize: 20, total: historyTotal, onChange: () => loadData() }}
                  locale={{ emptyText: 'No alert history' }}
                />
              </Card>
            </TabPane>
          </Tabs>
        </Layout.Content>

        {/* Rule Form Drawer */}
        <Drawer
          title={editingRule ? 'Edit Alert Rule' : 'Create Alert Rule'}
          placement="right"
          width={480}
          open={ruleDrawerOpen}
          onClose={() => setRuleDrawerOpen(false)}
          extra={
            <Space>
              <Button onClick={() => setRuleDrawerOpen(false)}>Cancel</Button>
              <Button type="primary" loading={savingRule} onClick={handleSaveRule}>
                {editingRule ? 'Update' : 'Create'}
              </Button>
            </Space>
          }
        >
          <Form form={ruleForm} layout="vertical" initialValues={RULE_FORM_INITIAL}>
            <Form.Item name="name" label="Rule Name" rules={[{ required: true, message: 'Required' }]}>
              <Input placeholder="e.g. High error rate" />
            </Form.Item>
            <Form.Item name="description" label="Description">
              <Input.TextArea rows={2} placeholder="Optional description" />
            </Form.Item>
            <Form.Item name="protocolId" label="Protocol" rules={[{ required: true, message: 'Required' }]}>
              <Select placeholder="Select protocol">
                {healthSummary?.results?.map((r: any) => (
                  <Select.Option key={r.protocolId} value={r.protocolId}>{r.protocolId}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Space style={{ width: '100%' }}>
              <Form.Item name="metric" label="Metric" rules={[{ required: true }]}>
                <Select style={{ width: 140 }}>
                  <Select.Option value="errorCount">Error Count</Select.Option>
                  <Select.Option value="connections">Connections</Select.Option>
                  <Select.Option value="activeUsers">Active Users</Select.Option>
                  <Select.Option value="uptimeSeconds">Uptime (s)</Select.Option>
                </Select>
              </Form.Item>
              <Form.Item name="condition" label="Condition" rules={[{ required: true }]}>
                <Select style={{ width: 100 }}>
                  <Select.Option value="gt">&gt;</Select.Option>
                  <Select.Option value="gte">&gt;=</Select.Option>
                  <Select.Option value="lt">&lt;</Select.Option>
                  <Select.Option value="lte">&lt;=</Select.Option>
                  <Select.Option value="eq">==</Select.Option>
                  <Select.Option value="neq">!=</Select.Option>
                </Select>
              </Form.Item>
              <Form.Item name="threshold" label="Threshold">
                <InputNumber style={{ width: 100 }} />
              </Form.Item>
            </Space>
            <Space style={{ width: '100%' }}>
              <Form.Item name="severity" label="Severity">
                <Select style={{ width: 120 }}>
                  <Select.Option value="info">Info</Select.Option>
                  <Select.Option value="warning">Warning</Select.Option>
                  <Select.Option value="critical">Critical</Select.Option>
                </Select>
              </Form.Item>
              <Form.Item name="duration" label="Duration (s)">
                <InputNumber style={{ width: 100 }} min={1} />
              </Form.Item>
              <Form.Item name="cooldown" label="Cooldown (s)">
                <InputNumber style={{ width: 100 }} min={0} />
              </Form.Item>
            </Space>
            <Form.Item name="channels" label="Notification Channels">
              <Select mode="multiple" placeholder="Select channels">
                <Select.Option value="log-default">Log</Select.Option>
                <Select.Option value="tg-alert">Telegram</Select.Option>
                <Select.Option value="email-alert">Email</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item name="autoRecovery" label="Auto Recovery" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="enabled" label="Enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Form>
        </Drawer>
      </Layout>
    </ConfigProvider>
  );
}
