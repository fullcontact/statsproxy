package workers

import (
	"github.com/frightenedmonkey/statsproxy/config"
	"log"
	"net"
)

func InitializeTCPHealthCheck(l *net.TCPListener) {
	go tcpHealthCheckListener(l)
}

func tcpHealthCheckListener(listener *net.TCPListener) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Fatal("not able to accept connection: %s", err)
		}

		m := make([]byte, 6)
		conn.Read(m)
		conn.Write([]byte(config.Service.TCPHealthResponse))
	}
}
