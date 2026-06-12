import { useState, useEffect, useCallback, useMemo } from 'react';
import type { CSSProperties } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import {
  ApiOutlined,
  ArrowLeftOutlined,
  CalendarOutlined,
  CheckCircleOutlined,
  CloudServerOutlined,
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  GlobalOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  StopOutlined,
  ThunderboltOutlined,
  UserOutlined,
  WifiOutlined,
} from '@ant-design/icons';
import { ClipboardManager, HttpUtil } from '@/utils';
import axios from 'axios';
import './ExtraProtocolsPage.css';

const { Text, Title, Paragraph } = Typography;

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } } as const;

const PROTOCOL_OPTIONS = [
  { value: 'SSH', label: 'SSH' },
  { value: 'SSWS', label: 'SSH-WS' },
  { value: 'SLOW-DNS', label: 'SlowDNS (DNSTT)' },
  { value: 'Psiphon', label: 'Psiphon' },
  { value: 'UDP Custom (BadVPN)', label: 'UDP Custom' },
  { value: 'Dropbear', label: 'Dropbear' },
  { value: 'SSL (Stunnel)', label: 'SSL (Stunnel)' },
  { value: 'OpenVPN', label: 'OpenVPN' },
  { value: 'Squid', label: 'Squid' },
  { value: 'OHP', label: 'OHP' },
] as const;

const PROTOCOL_META: Record<string, { icon: string; color: string; accent: string; name: string }> = {
  SSH: { icon: '⌁', color: 'green', accent: '#22c55e', name: 'SSH' },
  SSWS: { icon: 'WS', color: 'cyan', accent: '#06b6d4', name: 'SSH-WS' },
  'SLOW-DNS': { icon: 'DNS', color: 'purple', accent: '#a855f7', name: 'SlowDNS' },
  Psiphon: { icon: 'Ψ', color: 'magenta', accent: '#ec4899', name: 'Psiphon' },
  'UDP Custom (BadVPN)': { icon: 'UDP', color: 'orange', accent: '#f97316', name: 'UDP Custom' },
  Dropbear: { icon: 'DB', color: 'blue', accent: '#3b82f6', name: 'Dropbear' },
  'SSL (Stunnel)': { icon: 'TLS', color: 'gold', accent: '#eab308', name: 'SSL Tunnel' },
  OpenVPN: { icon: 'OV', color: 'lime', accent: '#84cc16', name: 'OpenVPN' },
  Squid: { icon: 'SQ', color: 'geekblue', accent: '#6366f1', name: 'Squid' },
  OHP: { icon: 'OHP', color: 'volcano', accent: '#ef4444', name: 'OHP' },
};

type ExtraUserPayload = {
  username: string;
  password: string;
  protocolType: string;
  expiryDate: number;
  status: string;
  configPayload: string;
};

interface ExtraUser {
  id: number;
  username: string;
  password: string;
  protocolType: string;
  expiryDate: number;
  status: string;
  configPayload: string;
  configString?: string;
  formattedDetails?: Record<string, string>;
}

interface ExtraSetting {
  protocolName: string;
  listeningPort: number;
  isEnabled: boolean;
  bannerText?: string;
}

function protocolMeta(protocol: string) {
  return PROTOCOL_META[protocol] || { icon: 'VPN', color: 'default', accent: '#64748b', name: protocol || 'Protocol' };
}

function normalizeExpiry(expiryDate: number) {
  if (!expiryDate || expiryDate <= 0) return 'Never';
  const ms = expiryDate < 1_000_000_000_000 ? expiryDate * 1000 : expiryDate;
  return new Date(ms).toLocaleString();
}

function configText(user: ExtraUser | null) {
  if (!user) return '';
  return user.configString || `Config will be generated after saving ${user.username}.`;
}

export default function ExtraProtocolsPage() {
  const navigate = useNavigate();
  const [users, setUsers] = useState<ExtraUser[]>([]);
  const [settings, setSettings] = useState<ExtraSetting[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [configUser, setConfigUser] = useState<ExtraUser | null>(null);
  const [editingUser, setEditingUser] = useState<ExtraUser | null>(null);
  const [form] = Form.useForm();

  const settingsByProtocol = useMemo(() => new Map(settings.map((s) => [s.protocolName, s])), [settings]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const usersMsg = await HttpUtil.get<ExtraUser[]>('/panel/api/extra/users', undefined, { silent: true });
      const settingsMsg = await HttpUtil.get<ExtraSetting[]>('/panel/api/extra/settings', undefined, { silent: true });
      if (usersMsg?.success) setUsers(Array.isArray(usersMsg.obj) ? usersMsg.obj : []);
      if (settingsMsg?.success) setSettings(Array.isArray(settingsMsg.obj) ? settingsMsg.obj : []);
      if (!usersMsg?.success || !settingsMsg?.success) message.error(usersMsg?.msg || settingsMsg?.msg || 'Failed to load extra protocols');
    } catch (err) {
      message.error('Failed to load extra protocols');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const buildUserPayload = (values: Record<string, unknown>): ExtraUserPayload => {
    const rawExpiry = values.expiryDate;
    const expiryDate = rawExpiry === undefined || rawExpiry === null || rawExpiry === ''
      ? 0
      : Number.parseInt(String(rawExpiry), 10);

    return {
      username: String(values.username ?? '').trim(),
      password: String(values.password ?? ''),
      protocolType: String(values.protocolType ?? ''),
      expiryDate: Number.isFinite(expiryDate) ? expiryDate : 0,
      status: String(values.status ?? 'active'),
      configPayload: String(values.configPayload ?? ''),
    };
  };

  const openUserModal = (record?: ExtraUser) => {
    setEditingUser(record || null);
    form.resetFields();
    form.setFieldsValue(record || { status: 'active', expiryDate: 0, protocolType: 'SSH' });
    setModalVisible(true);
  };

  const handleSaveUser = async () => {
    try {
      const values = await form.validateFields();
      const payload = buildUserPayload(values);
      const msg = editingUser
        ? (await axios.put(`/panel/api/extra/users/${editingUser.id}`, JSON.stringify(payload), JSON_HEADERS)).data
        : await HttpUtil.post('/panel/api/extra/users', JSON.stringify(payload), { ...JSON_HEADERS, silent: true });
      if (msg?.success) {
        message.success('Saved successfully');
        setModalVisible(false);
        setEditingUser(null);
        form.resetFields();
        fetchAll();
      } else {
        message.error(msg?.msg || 'Save failed');
      }
    } catch (err) {
      // Form validation errors are displayed inline by Ant Design.
    }
  };

  const handleDeleteUser = async (id: number) => {
    try {
      const response = await axios.delete(`/panel/api/extra/users/${id}`);
      const msg = response.data;
      if (msg?.success) {
        message.success('Deleted successfully');
        fetchAll();
      } else {
        message.error(msg?.msg || 'Delete failed');
      }
    } catch (err) {
      message.error('Delete failed');
    }
  };

  const handleUpdateSetting = async (protocolName: string, port: number, enabled: boolean, bannerText?: string) => {
    try {
      const safePort = Number.parseInt(String(port), 10);
      const payload = {
        protocolName,
        listeningPort: Number.isFinite(safePort) ? safePort : 0,
        isEnabled: Boolean(enabled),
        bannerText: bannerText ?? '',
      };
      const response = await axios.put('/panel/api/extra/settings', JSON.stringify(payload), JSON_HEADERS);
      const msg = response.data;
      if (msg?.success) {
        message.success('Settings updated');
        fetchAll();
      } else {
        message.error(msg?.msg || 'Update failed');
      }
    } catch (err) {
      message.error('Update failed');
    }
  };

  const copySelectedConfig = async () => {
    const ok = await ClipboardManager.copyText(configText(configUser));
    if (ok) message.success('Copied to clipboard');
  };

  const userCards = users.map((user) => {
    const meta = protocolMeta(user.protocolType);
    const setting = settingsByProtocol.get(user.protocolType);
    const active = String(user.status).toLowerCase() === 'active';
    return (
      <Col xs={24} md={12} xl={8} key={user.id}>
        <Card
          hoverable
          className="extra-user-card"
          style={{ '--protocol-accent': meta.accent } as CSSProperties}
          title={
            <div className="extra-card-title">
              <span className="protocol-badge">{meta.icon}</span>
              <div>
                <Text strong>{meta.name}</Text>
                <div className="extra-subtitle">#{user.id} • {user.username}</div>
              </div>
            </div>
          }
          extra={<Tag color={active ? 'success' : 'default'} icon={active ? <CheckCircleOutlined /> : <StopOutlined />}>{active ? 'Active' : 'Inactive'}</Tag>}
          actions={[
            <Tooltip title="View Config" key="payload"><Button type="text" icon={<EyeOutlined />} onClick={() => setConfigUser(user)}>Payload</Button></Tooltip>,
            <Tooltip title="Edit" key="edit"><Button type="text" icon={<EditOutlined />} onClick={() => openUserModal(user)} /></Tooltip>,
            <Tooltip title="Delete" key="delete"><Button danger type="text" icon={<DeleteOutlined />} onClick={() => handleDeleteUser(user.id)} /></Tooltip>,
          ]}
        >
          <div className="extra-card-grid">
            <div className="extra-stat"><UserOutlined /><span>Username</span><strong>{user.username}</strong></div>
            <div className="extra-stat"><CloudServerOutlined /><span>Port</span><strong>{setting?.listeningPort || user.formattedDetails?.Port || '—'}</strong></div>
            <div className="extra-stat"><CalendarOutlined /><span>Expiry</span><strong>{normalizeExpiry(user.expiryDate)}</strong></div>
            <div className="extra-stat"><SafetyCertificateOutlined /><span>Service</span><strong>{setting?.isEnabled ? 'Enabled' : 'Disabled'}</strong></div>
          </div>
          <Paragraph copyable={{ text: user.configString || '' }} ellipsis={{ rows: 2 }} className="config-preview">
            <pre>{user.configString || 'Config will be generated after saving.'}</pre>
          </Paragraph>
        </Card>
      </Col>
    );
  });

  return (
    <div className="extra-page">
      <Card className="extra-hero-card">
        <div className="extra-hero">
          <div>
            <Space align="center" wrap>
              <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/')}>
                Back to Panel
              </Button>
              <Tag color="cyan" icon={<WifiOutlined />}>VPN Manager Ecosystem</Tag>
            </Space>
            <Title level={2}>Extra Protocols</Title>
            <Paragraph>
              Manage SSH, Dropbear, Stunnel, SSH-WS, SlowDNS, Psiphon and UDP Custom users with generated payloads ready for clients.
            </Paragraph>
          </div>
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={fetchAll} loading={loading}>Refresh</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openUserModal()}>Add User</Button>
          </Space>
        </div>
      </Card>

      <Tabs
        defaultActiveKey="users"
        items={[
          {
            key: 'users',
            label: <span><UserOutlined /> Users</span>,
            children: users.length ? <Row gutter={[16, 16]}>{userCards}</Row> : <Empty description="No extra protocol users yet" />,
          },
          {
            key: 'settings',
            label: <span><SettingOutlined /> Ports & Services</span>,
            children: (
              <div className="settings-grid">
                {settings.map((s) => {
                  const meta = protocolMeta(s.protocolName);
                  return (
                    <Card key={s.protocolName} className="protocol-setting-card" size="small">
                      <div className="setting-card-head">
                        <span className="protocol-badge small" style={{ background: meta.accent }}>{meta.icon}</span>
                        <div>
                          <Text strong>{meta.name}</Text>
                          <div className="extra-subtitle">systemd managed endpoint</div>
                        </div>
                        <Switch checked={s.isEnabled} onChange={(checked) => handleUpdateSetting(s.protocolName, s.listeningPort, checked, s.bannerText)} />
                      </div>
                      <Input
                        type="number"
                        prefix={<ApiOutlined />}
                        defaultValue={s.listeningPort}
                        onPressEnter={(e) => {
                          const val = Number.parseInt((e.target as HTMLInputElement).value, 10);
                          handleUpdateSetting(s.protocolName, val, s.isEnabled, s.bannerText);
                        }}
                      />
                    </Card>
                  );
                })}

                <Card title={<span><ThunderboltOutlined /> Server Customization</span>} className="banner-card">
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Text strong>Connection Banner (SSH/Dropbear)</Text>
                    <Input.TextArea
                      rows={6}
                      placeholder="Enter ASCII art or welcome message here..."
                      defaultValue={settings.find((s) => s.protocolName === 'SSH')?.bannerText || ''}
                      onBlur={(e) => {
                        const text = e.target.value;
                        const sshSetting = settings.find((s) => s.protocolName === 'SSH');
                        handleUpdateSetting('SSH', sshSetting?.listeningPort || 22, sshSetting?.isEnabled || false, text);
                      }}
                    />
                    <Text type="secondary">Displayed when users connect through SSH or Dropbear.</Text>
                  </Space>
                </Card>
              </div>
            ),
          },
        ]}
      />

      <Modal
        title={editingUser ? 'Edit Extra Protocol User' : 'Add Extra Protocol User'}
        open={modalVisible}
        onOk={handleSaveUser}
        okText="Save"
        onCancel={() => {
          setModalVisible(false);
          setEditingUser(null);
          form.resetFields();
        }}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="Username" rules={[{ required: true, message: 'Username is required' }]}>
            <Input prefix={<UserOutlined />} />
          </Form.Item>
          <Form.Item name="password" label="Password" rules={[{ required: true, message: 'Password is required' }]}>
            <Input.Password prefix={<LockOutlined />} />
          </Form.Item>
          <Form.Item name="protocolType" label="Protocol" rules={[{ required: true, message: 'Protocol is required' }]}>
            <Select options={[...PROTOCOL_OPTIONS]} />
          </Form.Item>
          <Form.Item name="expiryDate" label="Expiry (Unix timestamp, 0 = never)">
            <Input type="number" prefix={<CalendarOutlined />} />
          </Form.Item>
          <Form.Item
            name="configPayload"
            label="Protocol Config Payload"
            extra="Optional JSON/key-value data. Examples: {&quot;path&quot;:&quot;/ssh&quot;,&quot;host&quot;:&quot;example.com&quot;} or {&quot;domain&quot;:&quot;dns.example.com&quot;,&quot;publicKey&quot;:&quot;...&quot;}"
          >
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="status" label="Status">
            <Select options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Inactive' }]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={configUser ? `${protocolMeta(configUser.protocolType).name} Payload — ${configUser.username}` : 'Payload'}
        open={!!configUser}
        onCancel={() => setConfigUser(null)}
        footer={[
          <Button key="copy" type="primary" icon={<CopyOutlined />} onClick={copySelectedConfig}>Copy to Clipboard</Button>,
          <Button key="close" onClick={() => setConfigUser(null)}>Close</Button>,
        ]}
        width={760}
        className="config-modal"
        destroyOnHidden
      >
        {configUser && (
          <div className="config-modal-body">
            <Descriptions bordered size="small" column={{ xs: 1, sm: 1, md: 2 }}>
              {Object.entries(configUser.formattedDetails || {}).map(([key, value]) => (
                <Descriptions.Item key={key} label={key}>{value}</Descriptions.Item>
              ))}
            </Descriptions>
            <div className="payload-box">
              <div className="payload-box-head">
                <GlobalOutlined /> Fully formatted connection details
              </div>
              <pre>{configText(configUser)}</pre>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}