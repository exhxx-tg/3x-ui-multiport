export type ProtocolCategory = 'base' | 'standalone' | 'wrapper';

export interface ProtocolInfo {
  id: string;
  name: string;
  category: ProtocolCategory;
  description: string;
  source: string;
  xrayNative: boolean;
  status: ProtocolStatus;
  healthy: boolean;
  port?: number;
  installed?: boolean;
  serviceName?: string;
  supportedProtocols?: string[];
}

export type ProtocolStatus = 'running' | 'stopped' | 'error' | 'installing' | 'unknown';

export interface ProtocolConfig {
  id: number;
  protocol: string;
  config: string;
  version: string;
  enabled: boolean;
  createdAt: number;
  updatedAt: number;
}

export interface ServiceModel {
  id: number;
  serviceType: string;
  name: string;
  remark: string;
  port: number;
  protocol: string;
  status: ServiceStatus;
  enable: boolean;
  up: number;
  down: number;
  total: number;
  expiryTime: number;
  lastStartedAt: number;
  lastStoppedAt: number;
  errorMsg: string;
  installPath: string;
  configPath: string;
  createdAt: number;
  updatedAt: number;
}

export type ServiceStatus = 'running' | 'stopped' | 'error' | 'installing' | 'unknown';

export interface TransportWrapperModel {
  id: number;
  wrapperType: string;
  name: string;
  remark: string;
  protocols: string;
  config: string;
  enable: boolean;
  port: number;
  tlsEnabled: boolean;
  certFile: string;
  keyFile: string;
  createdAt: number;
  updatedAt: number;
}
