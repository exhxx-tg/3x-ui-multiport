import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Badge,
  Button,
  Card,
  Col,
  ConfigProvider,
  Drawer,
  Form,
  Input,
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
  ReloadOutlined,
  SafetyOutlined,
  HistoryOutlined,
  KeyOutlined,
  LockOutlined,
  GlobalOutlined,
  CloudServerOutlined,
  SafetyCertificateOutlined,
  DownloadOutlined,
  UploadOutlined,
  EyeOutlined,
} from '@ant-design/icons';

import { useTheme } from '@/hooks/useTheme';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import AppSidebar from '@/layouts/AppSidebar';
import { HttpUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';

const { Title, Text } = Typography;

async function fetchOverview() {
  const msg = await HttpUtil.get('/panel/api/security/overview', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch security overview');
  return msg.obj || {};
}

async function fetchIPAccessRules() {
  const msg = await HttpUtil.get('/panel/api/security/ip-access', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch IP rules');
  return msg.obj || [];
}

async function fetchSessions() {
  const msg = await HttpUtil.get('/panel/api/security/sessions', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch sessions');
  return msg.obj || [];
}

async function fetchLoginAttempts(offset = 0, limit = 50) {
  const msg = await HttpUtil.get(`/panel/api/security/login-attempts?offset=${offset}&limit=${limit}`, undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch login attempts');
  return msg.obj || { items: [], total: 0 };
}

async function fetchCertificates() {
  const msg = await HttpUtil.get('/panel/api/certificates/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch certificates');
  return msg.obj || [];
}

async function fetchBackups() {
  const msg = await HttpUtil.get('/panel/api/backup/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch backups');
  return msg.obj || [];
}

export default function SecurityPage() {
  const { t } = useTranslation();
  const { antdThemeConfig } = useTheme();
  const { isMobile } = useMediaQuery();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => { setMessageInstance(messageApi); }, [messageApi]);
  const [modal, modalContextHolder] = Modal.useModal();

  const [overview, setOverview] = useState<any>(null);
  const [overviewLoading, setOverviewLoading] = useState(false);

  const [ipRules, setIpRules] = useState<any[]>([]);
  const [ipRulesLoading, setIpRulesLoading] = useState(false);

  const [sessions, setSessions] = useState<any[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);

  const [loginAttempts, setLoginAttempts] = useState<any[]>([]);
  const [loginAttemptsTotal, setLoginAttemptsTotal] = useState(0);
  const [loginAttemptsLoading, setLoginAttemptsLoading] = useState(false);

  const [certificates, setCertificates] = useState<any[]>([]);
  const [certificatesLoading, setCertificatesLoading] = useState(false);

  const [backups, setBackups] = useState<any[]>([]);
  const [backupsLoading, setBackupsLoading] = useState(false);
  const [backupCreateOpen, setBackupCreateOpen] = useState(false);
  const [backupCreating, setBackupCreating] = useState(false);
  const [backupDesc, setBackupDesc] = useState('');
  const [backupEncrypt, setBackupEncrypt] = useState(true);

  const [certUploadOpen, setCertUploadOpen] = useState(false);
  const [certSelfSignedOpen, setCertSelfSignedOpen] = useState(false);
  const [certDomain, setCertDomain] = useState('');
  const [certPem, setCertPem] = useState('');
  const [certKey, setCertKey] = useState('');

  const [ipDrawerOpen, setIpDrawerOpen] = useState(false);
  const [ipDrawerMode, setIpDrawerMode] = useState<'create' | 'edit'>('create');
  const [ipEditId, setIpEditId] = useState<number | null>(null);
  const [ipFormType, setIpFormType] = useState<'allow' | 'block'>('allow');
  const [ipFormCIDR, setIpFormCIDR] = useState('');
  const [ipFormRemark, setIpFormRemark] = useState('');
  const [ipFormEnabled, setIpFormEnabled] = useState(true);
  const [ipFormPriority, setIpFormPriority] = useState(0);

  const loadOverview = useCallback(async () => {
    setOverviewLoading(true);
    try {
      setOverview(await fetchOverview());
    } finally { setOverviewLoading(false); }
  }, []);

  const loadIPRules = useCallback(async () => {
    setIpRulesLoading(true);
    try { setIpRules(await fetchIPAccessRules()); } finally { setIpRulesLoading(false); }
  }, []);

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    try { setSessions(await fetchSessions()); } finally { setSessionsLoading(false); }
  }, []);

  const loadLoginAttempts = useCallback(async () => {
    setLoginAttemptsLoading(true);
    try {
      const res = await fetchLoginAttempts();
      setLoginAttempts(res.items || []);
      setLoginAttemptsTotal(res.total || 0);
    } finally { setLoginAttemptsLoading(false); }
  }, []);

  const loadCertificates = useCallback(async () => {
    setCertificatesLoading(true);
    try { setCertificates(await fetchCertificates()); } finally { setCertificatesLoading(false); }
  }, []);

  const loadBackups = useCallback(async () => {
    setBackupsLoading(true);
    try { setBackups(await fetchBackups()); } finally { setBackupsLoading(false); }
  }, []);

  useEffect(() => { loadOverview(); loadIPRules(); loadSessions(); loadLoginAttempts(); loadCertificates(); loadBackups(); }, []);

  const overviewCards = useMemo(() => {
    if (!overview) return [];
    return [
      { title: 'Active Sessions', value: overview.totalSessions ?? 0, icon: <KeyOutlined />, color: '#1890ff' },
      { title: 'Login Attempts (24h)', value: overview.loginAttempts24h ?? 0, icon: <HistoryOutlined />, color: '#52c41a' },
      { title: 'Failed Logins (24h)', value: overview.failedLogins24h ?? 0, icon: <CloseCircleOutlined />, color: '#ff4d4f' },
      { title: 'IP Access Rules', value: overview.activeRules ?? 0, icon: <GlobalOutlined />, color: '#722ed1' },
      { title: 'Expiring Certs', value: overview.expiringCerts ?? 0, icon: <SafetyCertificateOutlined />, color: '#fa8c16' },
    ];
  }, [overview]);

  async function handleCreateBackup() {
    setBackupCreating(true);
    try {
      const msg = await HttpUtil.post('/panel/api/backup/create', { description: backupDesc, encrypt: backupEncrypt });
      if (msg?.success) {
        messageApi.success('Backup created');
        setBackupCreateOpen(false);
        setBackupDesc('');
        loadBackups();
      }
    } finally { setBackupCreating(false); }
  }

  async function handleRestoreBackup(id: number) {
    const msg = await HttpUtil.post(`/panel/api/backup/restore/${id}`);
    if (msg?.success) {
      messageApi.success('Database restored. Panel will restart.');
    }
  }

  async function handleDeleteBackup(id: number) {
    const msg = await HttpUtil.post(`/panel/api/backup/delete/${id}`);
    if (msg?.success) {
      messageApi.success('Backup deleted');
      loadBackups();
    }
  }

  async function handleUploadCert() {
    const msg = await HttpUtil.post('/panel/api/certificates/create', {
      domain: certDomain,
      certPem: certPem,
      keyPem: certKey,
      autoRenew: false,
    });
    if (msg?.success) {
      messageApi.success('Certificate uploaded');
      setCertUploadOpen(false);
      setCertDomain('');
      setCertPem('');
      setCertKey('');
      loadCertificates();
    }
  }

  async function handleSelfSigned() {
    const msg = await HttpUtil.post('/panel/api/certificates/generate-selfsigned', { domain: certDomain });
    if (msg?.success) {
      messageApi.success('Self-signed certificate generated');
      setCertSelfSignedOpen(false);
      setCertDomain('');
      loadCertificates();
    }
  }

  async function handleSetActiveCert(id: number) {
    const msg = await HttpUtil.post(`/panel/api/certificates/set-active/${id}`);
    if (msg?.success) messageApi.success('Certificate set as active');
  }

  async function handleDeleteCert(id: number) {
    const msg = await HttpUtil.post(`/panel/api/certificates/delete/${id}`);
    if (msg?.success) {
      messageApi.success('Certificate deleted');
      loadCertificates();
    }
  }

  function openIpDrawer(mode: 'create' | 'edit', rule?: any) {
    setIpDrawerMode(mode);
    if (mode === 'edit' && rule) {
      setIpEditId(rule.id);
      setIpFormType(rule.type);
      setIpFormCIDR(rule.cidr);
      setIpFormRemark(rule.remark || '');
      setIpFormEnabled(rule.enabled);
      setIpFormPriority(rule.priority || 0);
    } else {
      setIpEditId(null);
      setIpFormType('allow');
      setIpFormCIDR('');
      setIpFormRemark('');
      setIpFormEnabled(true);
      setIpFormPriority(0);
    }
    setIpDrawerOpen(true);
  }

  async function handleSaveIpRule() {
    if (ipDrawerMode === 'create') {
      const msg = await HttpUtil.post('/panel/api/security/ip-access', {
        type: ipFormType,
        cidr: ipFormCIDR,
        remark: ipFormRemark,
        priority: ipFormPriority,
      });
      if (msg?.success) { messageApi.success('Rule created'); setIpDrawerOpen(false); loadIPRules(); }
    } else {
      const msg = await HttpUtil.put(`/panel/api/security/ip-access/${ipEditId}`, {
        type: ipFormType,
        cidr: ipFormCIDR,
        remark: ipFormRemark,
        enabled: ipFormEnabled,
        priority: ipFormPriority,
      });
      if (msg?.success) { messageApi.success('Rule updated'); setIpDrawerOpen(false); loadIPRules(); }
    }
  }

  async function handleDeleteIpRule(id: number) {
    const msg = await HttpUtil.del(`/panel/api/security/ip-access/${id}`);
    if (msg?.success) { messageApi.success('Rule deleted'); loadIPRules(); }
  }

  async function handleRevokeSession(id: number) {
    const msg = await HttpUtil.post(`/panel/api/security/sessions/revoke/${id}`);
    if (msg?.success) { messageApi.success('Session revoked'); loadSessions(); }
  }

  async function handleRevokeAllSessions() {
    const msg = await HttpUtil.post('/panel/api/security/sessions/revoke-all');
    if (msg?.success) messageApi.success('All sessions revoked');
  }

  const ipRuleColumns = [
    { title: 'Type', dataIndex: 'type', key: 'type', render: (t: string) => <Tag color={t === 'allow' ? 'green' : 'red'}>{t.toUpperCase()}</Tag> },
    { title: 'CIDR', dataIndex: 'cidr', key: 'cidr' },
    { title: 'Remark', dataIndex: 'remark', key: 'remark' },
    { title: 'Priority', dataIndex: 'priority', key: 'priority' },
    { title: 'Enabled', dataIndex: 'enabled', key: 'enabled', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? 'Yes' : 'No'}</Tag> },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, r: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openIpDrawer('edit', r)} />
          <Popconfirm title="Delete this rule?" onConfirm={() => handleDeleteIpRule(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const sessionColumns = [
    { title: 'Username', dataIndex: 'username', key: 'username' },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: 'User Agent', dataIndex: 'userAgent', key: 'userAgent', ellipsis: true },
    { title: 'Last Activity', dataIndex: 'lastActivity', key: 'lastActivity', render: (v: number) => v ? new Date(v).toLocaleString() : '' },
    { title: 'Expires', dataIndex: 'expiresAt', key: 'expiresAt', render: (v: number) => v ? new Date(v).toLocaleString() : '' },
    {
      title: 'Action', key: 'action',
      render: (_: any, r: any) => (
        <Popconfirm title="Revoke this session?" onConfirm={() => handleRevokeSession(r.id)}>
          <Button size="small" danger>Revoke</Button>
        </Popconfirm>
      ),
    },
  ];

  const certColumns = [
    { title: 'Domain', dataIndex: 'domain', key: 'domain' },
    { title: 'Issuer', dataIndex: 'issuer', key: 'issuer' },
    { title: 'Provider', dataIndex: 'provider', key: 'provider', render: (v: string) => <Tag>{v}</Tag> },
    { title: 'Not Before', dataIndex: 'notBefore', key: 'notBefore', render: (v: number) => v ? new Date(v).toLocaleDateString() : '-' },
    { title: 'Not After', dataIndex: 'notAfter', key: 'notAfter', render: (v: number) => v ? new Date(v).toLocaleDateString() : '-', sorter: (a: any, b: any) => a.notAfter - b.notAfter },
    { title: 'Auto Renew', dataIndex: 'autoRenew', key: 'autoRenew', render: (v: boolean) => v ? <Tag color="green">Yes</Tag> : <Tag color="default">No</Tag> },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, r: any) => (
        <Space>
          <Button size="small" onClick={() => handleSetActiveCert(r.id)}>Set Active</Button>
          <Popconfirm title="Delete this certificate?" onConfirm={() => handleDeleteCert(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const backupColumns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Description', dataIndex: 'description', key: 'description' },
    { title: 'Size', dataIndex: 'fileSize', key: 'fileSize', render: (v: number) => v ? `${(v / 1024).toFixed(1)} KB` : '-' },
    { title: 'Encrypted', dataIndex: 'encrypted', key: 'encrypted', render: (v: boolean) => v ? <Tag color="orange">AES-256-GCM</Tag> : <Tag color="default">No</Tag> },
    { title: 'Status', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={v === 'completed' ? 'green' : 'orange'}>{v}</Tag> },
    { title: 'Created', dataIndex: 'createdAt', key: 'createdAt', render: (v: number) => v ? new Date(v).toLocaleString() : '' },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, r: any) => (
        <Space>
          <Popconfirm title="Restore this backup? The panel will restart." onConfirm={() => handleRestoreBackup(r.id)}>
            <Button size="small" icon={<UploadOutlined />}>Restore</Button>
          </Popconfirm>
          <Popconfirm title="Delete this backup?" onConfirm={() => handleDeleteBackup(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      {modalContextHolder}
      <Layout hasSider={!isMobile}>
        {!isMobile && <AppSidebar />}
        <Layout id="content-layout">
          <Layout.Content style={{ padding: 24, minHeight: '100vh' }}>
            <Title level={3}><SafetyOutlined /> Security & Enterprise</Title>

            {/* Overview Cards */}
            <Spin spinning={overviewLoading}>
              <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
                {overviewCards.map((card, idx) => (
                  <Col xs={12} sm={8} md={4} key={idx}>
                    <Card hoverable>
                      <Statistic
                        title={card.title}
                        value={card.value}
                        prefix={card.icon}
                        valueStyle={{ color: card.color }}
                      />
                    </Card>
                  </Col>
                ))}
              </Row>
            </Spin>

            <Tabs defaultActiveKey="ip-access" items={[
              {
                key: 'ip-access',
                label: <span><GlobalOutlined /> IP Access Control</span>,
                children: (
                  <>
                    <Space style={{ marginBottom: 16 }}>
                      <Button type="primary" icon={<PlusOutlined />} onClick={() => openIpDrawer('create')}>Add Rule</Button>
                    </Space>
                    <Table
                      dataSource={ipRules}
                      columns={ipRuleColumns}
                      rowKey="id"
                      loading={ipRulesLoading}
                      size="small"
                      pagination={false}
                    />
                  </>
                ),
              },
              {
                key: 'sessions',
                label: <span><KeyOutlined /> Active Sessions</span>,
                children: (
                  <>
                    <Space style={{ marginBottom: 16 }}>
                      <Popconfirm title="Revoke all sessions?" onConfirm={handleRevokeAllSessions}>
                        <Button danger icon={<DeleteOutlined />}>Revoke All</Button>
                      </Popconfirm>
                    </Space>
                    <Table
                      dataSource={sessions}
                      columns={sessionColumns}
                      rowKey="id"
                      loading={sessionsLoading}
                      size="small"
                    />
                  </>
                ),
              },
              {
                key: 'login-attempts',
                label: <span><HistoryOutlined /> Login Attempts</span>,
                children: (
                  <Table
                    dataSource={loginAttempts}
                    columns={[
                      { title: 'Username', dataIndex: 'username', key: 'username' },
                      { title: 'IP', dataIndex: 'ip', key: 'ip' },
                      { title: 'Success', dataIndex: 'success', key: 'success', render: (v: boolean) => v ? <Tag color="green">Success</Tag> : <Tag color="red">Failed</Tag> },
                      { title: 'Time', dataIndex: 'createdAt', key: 'createdAt', render: (v: number) => v ? new Date(v).toLocaleString() : '' },
                    ]}
                    rowKey="id"
                    loading={loginAttemptsLoading}
                    size="small"
                    pagination={{ total: loginAttemptsTotal, pageSize: 50, showSizeChanger: false }}
                  />
                ),
              },
              {
                key: 'certificates',
                label: <span><SafetyCertificateOutlined /> Certificates</span>,
                children: (
                  <>
                    <Space style={{ marginBottom: 16 }}>
                      <Button type="primary" icon={<PlusOutlined />} onClick={() => setCertUploadOpen(true)}>Upload Certificate</Button>
                      <Button icon={<SafetyCertificateOutlined />} onClick={() => setCertSelfSignedOpen(true)}>Generate Self-Signed</Button>
                    </Space>
                    <Table
                      dataSource={certificates}
                      columns={certColumns}
                      rowKey="id"
                      loading={certificatesLoading}
                      size="small"
                    />
                  </>
                ),
              },
              {
                key: 'backups',
                label: <span><CloudServerOutlined /> Backups</span>,
                children: (
                  <>
                    <Space style={{ marginBottom: 16 }}>
                      <Button type="primary" icon={<PlusOutlined />} onClick={() => setBackupCreateOpen(true)}>Create Backup</Button>
                      <Button icon={<DownloadOutlined />} onClick={() => window.open('/panel/api/backup/export/audit?format=csv')}>Export Audit CSV</Button>
                      <Button icon={<DownloadOutlined />} onClick={() => window.open('/panel/api/backup/export/audit?format=json')}>Export Audit JSON</Button>
                    </Space>
                    <Table
                      dataSource={backups}
                      columns={backupColumns}
                      rowKey="id"
                      loading={backupsLoading}
                      size="small"
                    />
                  </>
                ),
              },
            ]} />
          </Layout.Content>
        </Layout>
      </Layout>

      {/* IP Access Drawer */}
      <Drawer
        title={ipDrawerMode === 'create' ? 'Add IP Access Rule' : 'Edit IP Access Rule'}
        open={ipDrawerOpen}
        onClose={() => setIpDrawerOpen(false)}
        width={400}
        extra={<Button type="primary" onClick={handleSaveIpRule}>Save</Button>}
      >
        <Form layout="vertical">
          <Form.Item label="Type" required>
            <Select value={ipFormType} onChange={setIpFormType}>
              <Select.Option value="allow">Allow</Select.Option>
              <Select.Option value="block">Block</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label="CIDR" required help="e.g. 192.168.1.0/24 or 10.0.0.0/8">
            <Input value={ipFormCIDR} onChange={(e) => setIpFormCIDR(e.target.value)} placeholder="192.168.1.0/24" />
          </Form.Item>
          <Form.Item label="Remark">
            <Input value={ipFormRemark} onChange={(e) => setIpFormRemark(e.target.value)} placeholder="Office network" />
          </Form.Item>
          <Form.Item label="Priority">
            <Input type="number" value={ipFormPriority} onChange={(e) => setIpFormPriority(Number(e.target.value))} />
          </Form.Item>
          <Form.Item label="Enabled">
            <Switch checked={ipFormEnabled} onChange={setIpFormEnabled} />
          </Form.Item>
        </Form>
      </Drawer>

      {/* Create Backup Modal */}
      <Modal
        title="Create Backup"
        open={backupCreateOpen}
        onOk={handleCreateBackup}
        confirmLoading={backupCreating}
        onCancel={() => setBackupCreateOpen(false)}
      >
        <Form layout="vertical">
          <Form.Item label="Description">
            <Input value={backupDesc} onChange={(e) => setBackupDesc(e.target.value)} placeholder="Pre-update backup" />
          </Form.Item>
          <Form.Item label="Encrypt (AES-256-GCM)">
            <Switch checked={backupEncrypt} onChange={setBackupEncrypt} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Upload Certificate Modal */}
      <Modal
        title="Upload Certificate"
        open={certUploadOpen}
        onOk={handleUploadCert}
        onCancel={() => setCertUploadOpen(false)}
        width={600}
      >
        <Form layout="vertical">
          <Form.Item label="Domain" required>
            <Input value={certDomain} onChange={(e) => setCertDomain(e.target.value)} placeholder="example.com" />
          </Form.Item>
          <Form.Item label="Certificate PEM" required>
            <Input.TextArea rows={5} value={certPem} onChange={(e) => setCertPem(e.target.value)} placeholder="-----BEGIN CERTIFICATE-----..." />
          </Form.Item>
          <Form.Item label="Private Key PEM" required>
            <Input.TextArea rows={5} value={certKey} onChange={(e) => setCertKey(e.target.value)} placeholder="-----BEGIN PRIVATE KEY-----..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* Self-Signed Certificate Modal */}
      <Modal
        title="Generate Self-Signed Certificate"
        open={certSelfSignedOpen}
        onOk={handleSelfSigned}
        onCancel={() => setCertSelfSignedOpen(false)}
      >
        <Form layout="vertical">
          <Form.Item label="Domain" required>
            <Input value={certDomain} onChange={(e) => setCertDomain(e.target.value)} placeholder="example.com" />
          </Form.Item>
          <Text type="secondary">A self-signed certificate valid for 365 days will be generated.</Text>
        </Form>
      </Modal>
    </ConfigProvider>
  );
}
