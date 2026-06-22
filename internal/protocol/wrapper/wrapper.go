package wrapper

import (
	"fmt"

	"github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

func init() {
	protocol.RegisterWrapperFactory(func(id protocol.ProtocolID) protocol.TransportWrapper {
		switch id {
		case protocol.WrapperWebSocket:
			return NewWebSocketWrapper()
		case protocol.WrapperTLS:
			return NewTLSWrapper()
		case protocol.WrapperHTTP2:
			return NewHTTP2Wrapper()
		case protocol.WrapperGRPC:
			return NewGRPCWrapper()
		case protocol.WrapperNaive:
			return NewNaiveWrapper()
		default:
			return nil
		}
	})
}

type BaseWrapper struct {
	id       protocol.ProtocolID
	status   protocol.Status
	port     int
	config   map[string]any
	supported []protocol.ProtocolID
}

func (w *BaseWrapper) ID() protocol.ProtocolID { return w.id }

func (w *BaseWrapper) Info() protocol.ProtocolInfo {
	info := protocol.GetProtocolInfo(w.id)
	if info != nil {
		return *info
	}
	return protocol.ProtocolInfo{}
}

func (w *BaseWrapper) Status() protocol.Status { return w.status }

func (w *BaseWrapper) Start() error {
	w.status = protocol.StatusRunning
	return nil
}

func (w *BaseWrapper) Stop() error {
	w.status = protocol.StatusStopped
	return nil
}

func (w *BaseWrapper) Restart() error {
	_ = w.Stop()
	return w.Start()
}

func (w *BaseWrapper) Config() (any, error) {
	return w.config, nil
}

func (w *BaseWrapper) ApplyConfig(cfg any) error {
	c, ok := cfg.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	w.config = c
	if p, ok := c["port"].(int); ok {
		w.port = p
	}
	return nil
}

func (w *BaseWrapper) SupportedProtocols() []protocol.ProtocolID {
	return w.supported
}

func (w *BaseWrapper) WrapConfig(baseCfg any, wrapperCfg any) (any, error) {
	return map[string]any{
		"base":    baseCfg,
		"wrapper": wrapperCfg,
		"type":    string(w.id),
	}, nil
}

func NewWebSocketWrapper() protocol.TransportWrapper {
	return &BaseWrapper{
		id:     protocol.WrapperWebSocket,
		status: protocol.StatusUnknown,
		port:   80,
		config: map[string]any{
			"path": "/ws",
			"host": "",
		},
		supported: []protocol.ProtocolID{
			protocol.ProtocolVMess,
			protocol.ProtocolVLESS,
			protocol.ProtocolTrojan,
			protocol.ProtocolShadowsocks,
		},
	}
}

func NewTLSWrapper() protocol.TransportWrapper {
	return &BaseWrapper{
		id:     protocol.WrapperTLS,
		status: protocol.StatusUnknown,
		port:   443,
		config: map[string]any{
			"tls":     true,
			"certFile": "",
			"keyFile":  "",
		},
		supported: []protocol.ProtocolID{
			protocol.ProtocolVMess,
			protocol.ProtocolVLESS,
			protocol.ProtocolTrojan,
			protocol.ProtocolShadowsocks,
		},
	}
}

func NewHTTP2Wrapper() protocol.TransportWrapper {
	return &BaseWrapper{
		id:     protocol.WrapperHTTP2,
		status: protocol.StatusUnknown,
		port:   443,
		config: map[string]any{
			"path": "/h2",
			"host": "",
		},
		supported: []protocol.ProtocolID{
			protocol.ProtocolVLESS,
			protocol.ProtocolTrojan,
		},
	}
}

func NewGRPCWrapper() protocol.TransportWrapper {
	return &BaseWrapper{
		id:     protocol.WrapperGRPC,
		status: protocol.StatusUnknown,
		port:   443,
		config: map[string]any{
			"serviceName": "grpc",
			"multiMode":   false,
		},
		supported: []protocol.ProtocolID{
			protocol.ProtocolVLESS,
			protocol.ProtocolTrojan,
		},
	}
}

func NewNaiveWrapper() protocol.TransportWrapper {
	return &BaseWrapper{
		id:     protocol.WrapperNaive,
		status: protocol.StatusUnknown,
		port:   8080,
		config: map[string]any{
			"proxyType": "http",
			"auth":      "",
		},
		supported: []protocol.ProtocolID{
			protocol.ProtocolVMess,
			protocol.ProtocolVLESS,
			protocol.ProtocolTrojan,
			protocol.ProtocolShadowsocks,
			protocol.ProtocolHysteria,
		},
	}
}
