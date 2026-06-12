import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  message,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, ReloadOutlined } from '@ant-design/icons';
import { HttpUtil } from '@/utils';
import axios from 'axios';

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } } as const;

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
}

interface ExtraSetting {
  protocolName: string;
  listeningPort: number;
  isEnabled: boolean;
  bannerText?: string;
}

export default function ExtraProtocolsPage() {
  const { t } = useTranslation();
  const [users, setUsers] = useState<ExtraUser[]>([]);
  const [settings, setSettings] = useState<ExtraSetting[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<ExtraUser | null>(null);
  const [form] = Form.useForm();

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const usersMsg = await HttpUtil.get<ExtraUser[]>('/panel/api/extra/users');
      const settingsMsg = await HttpUtil.get<ExtraSetting[]>('/panel/api/extra/settings');
      if (usersMsg?.success) setUsers(Array.isArray(usersMsg.obj) ? usersMsg.obj : []);
      if (settingsMsg?.success) setSettings(Array.isArray(settingsMsg.obj) ? settingsMsg.obj : []);
    } catch (err) {
      message.error(t('somethingWentWrong'));
    } finally {
      setLoading(false);
    }
  }, [t]);

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

  const handleAddUser = async () => {
    try {
      const values = await form.validateFields();
      const msg = await HttpUtil.post('/panel/api/extra/users', JSON.stringify(buildUserPayload(values)), JSON_HEADERS);
      if (msg?.success) {
        message.success(t('success'));
        setModalVisible(false);
        form.resetFields();
        fetchAll();
      } else {
        message.error(msg?.msg || t('fail'));
      }
    } catch (err) {
      // Validation failed
    }
  };

  const handleUpdateUser = async () => {
    try {
      const values = await form.validateFields();
      if (!editingUser) return;
      const response = await axios.put(`/panel/api/extra/users/${editingUser.id}`, JSON.stringify(buildUserPayload(values)), JSON_HEADERS);
      const msg = response.data;
      if (msg?.success) {
        message.success(t('success'));
        setModalVisible(false);
        setEditingUser(null);
        form.resetFields();
        fetchAll();
      } else {
        message.error(msg?.msg || t('fail'));
      }
    } catch (err) {
      // Validation failed
    }
  };

  const handleDeleteUser = async (id: number) => {
    try {
      const response = await axios.delete(`/panel/api/extra/users/${id}`);
      const msg = response.data;
      if (msg?.success) {
        message.success(t('success'));
        fetchAll();
      } else {
        message.error(msg?.msg || t('fail'));
      }
    } catch (err) {
      message.error(t('fail'));
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
        message.success(t('success'));
        fetchAll();
      } else {
        message.error(msg?.msg || t('fail'));
      }
    } catch (err) {
      message.error(t('fail'));
    }
  };

  const userColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: 'Username', dataIndex: 'username', key: 'username' },
    { title: 'Protocol', dataIndex: 'protocolType', key: 'protocolType' },
    { 
      title: 'Expiry', 
      dataIndex: 'expiryDate', 
      key: 'expiryDate',
      render: (date: number) => date === 0 ? 'Never' : new Date(date).toLocaleDateString()
    },
    { title: 'Status', dataIndex: 'status', key: 'status' },
    { 
      title: 'Actions', 
      key: 'actions',
      render: (_: any, record: ExtraUser) => (
        <Space>
          <Button icon={<EditOutlined />} onClick={() => {
            setEditingUser(record);
            form.setFieldsValue(record);
            setModalVisible(true);
          }} />
          <Button danger icon={<DeleteOutlined />} onClick={() => handleDeleteUser(record.id)} />
        </Space>
      )
    },
  ];

  return (
    <Card title="Extra Protocols" extra={<Button icon={<ReloadOutlined />} onClick={fetchAll} loading={loading} />}>
      <Tabs defaultActiveKey="users">
        <Tabs.TabPane tab="User Management" key="users">
          <Space direction="vertical" style={{ width: '100%' }}>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
              setEditingUser(null);
              form.resetFields();
              setModalVisible(true);
            }}>
              Add User
            </Button>
            <Table 
              dataSource={users} 
              columns={userColumns} 
              rowKey="id" 
              pagination={{ pageSize: 10 }}
              loading={loading}
            />
          </Space>
        </Tabs.TabPane>
        <Tabs.TabPane tab="Port Settings" key="settings">
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {settings.map((s: ExtraSetting) => (
                <Card key={s.protocolName} size="small" title={s.protocolName}>
                  <Space align="baseline">
                    <span style={{ width: 100 }}>Port:</span>
                    <Input
                      type="number"
                      defaultValue={s.listeningPort}
                      style={{ width: 120 }}
                      onPressEnter={(e: any) => {
                        const val = Number.parseInt((e.target as HTMLInputElement).value, 10);
                        handleUpdateSetting(s.protocolName, val, s.isEnabled);
                      }}
                    />
                    <Switch
                      checked={s.isEnabled}
                      onChange={(checked: boolean) => handleUpdateSetting(s.protocolName, s.listeningPort, checked)}
                    />
                  </Space>
                </Card>
              ))}
            </div>

            <Card title="Server Customization" size="small">
              <Space direction="vertical" style={{ width: '100%' }}>
                <span style={{ fontWeight: 'bold' }}>Connection Banner (SSH/Dropbear)</span>
                <Input.TextArea
                  rows={6}
                  placeholder="Enter ASCII art or welcome message here..."
                  value={settings.find((s: ExtraSetting) => s.protocolName === 'SSH')?.bannerText || ''}
                  onChange={(e: any) => {
                    const text = e.target.value;
                    // We update the banner under the 'SSH' setting for convenience
                    const sshSetting = settings.find((s: ExtraSetting) => s.protocolName === 'SSH');
                    handleUpdateSetting('SSH', sshSetting?.listeningPort || 22, sshSetting?.isEnabled || false, text);
                  }}
                />
                <p style={{ fontSize: '12px', color: 'gray' }}>
                  This banner will be displayed when users connect via SSH or Dropbear.
                </p>
              </Space>
            </Card>
          </div>
        </Tabs.TabPane>
      </Tabs>

      <Modal
        title={editingUser ? "Edit User" : "Add User"}
        open={modalVisible}
        onOk={editingUser ? handleUpdateUser : handleAddUser}
        onCancel={() => {
          setModalVisible(false);
          setEditingUser(null);
          form.resetFields();
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="Username" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="Password" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="protocolType" label="Protocol" rules={[{ required: true }]}>
            <Select options={[
              { value: 'SSH', label: 'SSH' },
              { value: 'SSWS', label: 'SSWS' },
              { value: 'SLOW-DNS', label: 'SLOW-DNS' },
              { value: 'Psiphon', label: 'Psiphon' },
              { value: 'UDP Custom (BadVPN)', label: 'UDP Custom (BadVPN)' },
              { value: 'Dropbear', label: 'Dropbear' },
              { value: 'SSL (Stunnel)', label: 'SSL (Stunnel)' },
              { value: 'OpenVPN', label: 'OpenVPN' },
              { value: 'Squid', label: 'Squid' },
              { value: 'OHP', label: 'OHP' },
            ]} />
          </Form.Item>
          <Form.Item name="expiryDate" label="Expiry (Unix Timestamp)">
            <Input type="number" />
          </Form.Item>
          <Form.Item name="configPayload" label="Protocol Config Payload">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="status" label="Status">
            <Select options={[
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
            ]} defaultValue="active" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
