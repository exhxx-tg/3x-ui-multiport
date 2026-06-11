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
  type InputRef,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, ReloadOutlined } from '@ant-design/icons';
import { HttpUtil } from '@/utils';

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
      const usersMsg = await HttpUtil.get('/panel/api/extra/users');
      const settingsMsg = await HttpUtil.get('/panel/api/extra/settings');
      if (usersMsg?.success) setUsers(usersMsg.obj);
      if (settingsMsg?.success) setSettings(settingsMsg.obj);
    } catch (err) {
      message.error(t('somethingWentWrong'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const handleAddUser = async () => {
    try {
      const values = await form.validateFields();
      const msg = await HttpUtil.post('/panel/api/extra/users', values);
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
      const msg = await HttpUtil.put(`/panel/api/extra/users/${editingUser.id}`, values);
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
      const msg = await HttpUtil.delete(`/panel/api/extra/users/${id}`);
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
      const msg = await HttpUtil.put('/panel/api/extra/settings', { protocolName, listeningPort: port, isEnabled: enabled, bannerText });
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
              {settings.map(s => (
                <Card key={s.protocolName} size="small" title={s.protocolName}>
                  <Space align="baseline">
                    <span style={{ width: 100 }}>Port:</span>
                    <Input 
                      type="number" 
                      value={s.listeningPort} 
                      style={{ width: 120 }}
                      onPressEnter={(e) => {
                        const val = parseInt((e.target as HTMLInputElement).value);
                        handleUpdateSetting(s.protocolName, val, s.isEnabled);
                      }}
                    />
                    <Switch 
                      checked={s.isEnabled} 
                      onChange={(checked) => handleUpdateSetting(s.protocolName, s.listeningPort, checked)} 
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
                  value={settings.find(s => s.protocolName === 'SSH')?.bannerText || ''}
                  onChange={(e) => {
                    const text = e.target.value;
                    // We update the banner under the 'SSH' setting for convenience
                    handleUpdateSetting('SSH', 2222, true, text);
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
