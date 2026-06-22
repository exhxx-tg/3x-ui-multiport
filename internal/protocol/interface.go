package protocol

type Protocol interface {
	ID() ProtocolID
	Info() ProtocolInfo
	Status() Status
	Start() error
	Stop() error
	Restart() error
	Config() (any, error)
	ApplyConfig(cfg any) error
}

type BaseProtocol interface {
	Protocol
	Port() int
	SetPort(port int)
}

type StandaloneService interface {
	Protocol
	ServiceName() string
	IsInstalled() bool
	Install() error
	Uninstall() error
	HealthCheck() error
}

type TransportWrapper interface {
	Protocol
	SupportedProtocols() []ProtocolID
	WrapConfig(baseCfg any, wrapperCfg any) (any, error)
}
