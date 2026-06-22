package performance

import (
	"net"
	"net/http"
	"time"
)

const (
	defaultDialTimeout      = 5 * time.Second
	defaultKeepAlive        = 30 * time.Second
	defaultIdleConnTimeout  = 90 * time.Second
	defaultMaxIdleConns     = 100
	defaultMaxIdlePerHost   = 10
	defaultTLSHandshakeTimeout = 10 * time.Second
	defaultResponseHeaderTimeout = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	MaxBodySize             = 10 << 20
)

func OptimizedHTTPTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAlive,
		}).DialContext,
		MaxIdleConns:          defaultMaxIdleConns,
		MaxIdleConnsPerHost:   defaultMaxIdlePerHost,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
		ForceAttemptHTTP2:     true,
	}
}

func OptimizedHTTPClient() *http.Client {
	return &http.Client{
		Transport: OptimizedHTTPTransport(),
		Timeout:   defaultResponseHeaderTimeout + 30*time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

var defaultHTTPClient = OptimizedHTTPClient()

func HTTPClient() *http.Client { return defaultHTTPClient }

type ConnPool struct {
	conns chan net.Conn
	dial  func() (net.Conn, error)
}

func NewConnPool(size int, dial func() (net.Conn, error)) *ConnPool {
	return &ConnPool{
		conns: make(chan net.Conn, size),
		dial:  dial,
	}
}

func (p *ConnPool) Get() (net.Conn, error) {
	select {
	case conn := <-p.conns:
		return conn, nil
	default:
		return p.dial()
	}
}

func (p *ConnPool) Put(conn net.Conn) {
	select {
	case p.conns <- conn:
	default:
		conn.Close()
	}
}

func (p *ConnPool) Close() {
	close(p.conns)
	for conn := range p.conns {
		conn.Close()
	}
}
