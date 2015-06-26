package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

var Service StatsproxyConfig

type Host struct {
	FQDN string `json:"fqdn"`
	Port int    `json:"port"`
}

type StatsproxyConfig struct {
	Statsd                 Statsd `json:"statsd"`
	MaxUDPPacketSize       int    `json:"max_udp_packet_size"`
	SocketReadBuffer       int    `json:"socket_read_buffer"`
	Port                   string `json:"port"`
	TickerPeriod           int    `json:"ticker_period"`
	Workers                int
	Name                   string
	Logger                 LogConfig `json:"logger"`
	WriterMultiplier       int
	MgmtPort               string `json:"tcp_management_port"`
	HTTPPort               string `json:"http_port"`
	TCPHealthResponse      string `json:"tcp_health_response"`
	MaxCoalescedPacketSize int
	Environment            string `json:"service_environment"`
	MetricsNamespace       string
}

type LogConfig struct {
	Develop bool `json:"develop"`
}

type Statsd struct {
	Hosts []Host `json:"hosts"`
}

func InitializeConfig(f string) error {
	file := path.Clean(f)
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Default the Environment member to an empty string
	Service.Environment = ""

	er := json.Unmarshal(raw, &Service)

	if er != nil {
		return er
	}

	Service.setOtherDefaults()
	Service.setServiceMetricsNamespace()

	return nil
}

func (s *StatsproxyConfig) setOtherDefaults() {
	s.Workers = 2
	s.Name = "Statsproxy"
	s.WriterMultiplier = 2
	s.MaxCoalescedPacketSize = 1400
}

func (s *StatsproxyConfig) setServiceMetricsNamespace() {
	ns := strings.ToLower(s.Name)
	// We are treating the environment string as optional, so if it's actually set,
	// we should append it to the service namespace for metrics
	if len(s.Environment) > 0 {
		ns = ns + "." + s.Environment
	}
	s.MetricsNamespace = ns
}

func newLogger() LogConfig {
	return LogConfig{false}
}

func (h Host) String() string {
	return fmt.Sprintf("%v:%v", h.FQDN, h.Port)
}
