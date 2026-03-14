package remote

import (
	"fmt"
	"net"
	"time"
)

// ConnMode represents the remote host connectivity mode.
type ConnMode int

const (
	ConnChecking    ConnMode = iota
	ConnDirect               // gateway LAN reachable — same network
	ConnTunnel               // gateway WAN reachable — use DNAT port
	ConnUnreachable          // neither reachable
)

// String returns a short label for the connectivity mode.
func (m ConnMode) String() string {
	switch m {
	case ConnDirect:
		return "direct"
	case ConnTunnel:
		return "tunnel"
	case ConnUnreachable:
		return "offline"
	default:
		return "checking"
	}
}

// DetectConnMode checks connectivity by attempting TCP dial.
// For LAN gateway, checks port 22. For WAN gateway, checks wanPort (DNAT).
// If both gateway fields are empty, it falls back to checking the host directly.
func DetectConnMode(host, gatewayLAN, gatewayWAN string, wanPort int) ConnMode {
	timeout := 1 * time.Second

	if gatewayLAN == "" && gatewayWAN == "" {
		if tcpReachable(host, "22", timeout) {
			return ConnDirect
		}
		return ConnUnreachable
	}

	if gatewayLAN != "" && tcpReachable(gatewayLAN, "22", timeout) {
		return ConnDirect
	}
	if gatewayWAN != "" {
		port := "22"
		if wanPort > 0 {
			port = fmt.Sprintf("%d", wanPort)
		}
		if tcpReachable(gatewayWAN, port, timeout) {
			return ConnTunnel
		}
	}
	return ConnUnreachable
}

func tcpReachable(host, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
