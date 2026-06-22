package standalone

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"
)

func checkPortAvailable(port int) error {
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil
	}
	conn.Close()
	return fmt.Errorf("port %d is already in use", port)
}

func tcpPortCheck(host string, port int, timeout time.Duration) bool {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func binaryExists(name string) bool {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func executableExists(path string) bool {
	if _, err := exec.LookPath(path); err == nil {
		return true
	}
	_, err := exec.LookPath(path)
	return err == nil
}
