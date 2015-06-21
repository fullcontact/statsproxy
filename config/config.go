package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
)

var Service StatsproxyConfig

type Host struct {
	FQDN string `json:"fqdn"`
	Port int    `json:"port"`
}

type StatsproxyConfig struct {
	Statsd           Statsd `json:"statsd"`
	MaxUDPPacketSize int    `json:"max_udp_packet_size"`
	SocketReadBuffer int    `json:"socket_read_buffer"`
	Port             string `json:"port"`
	TickerPeriod     int    `json:"ticker_period"`
	Workers          int
	Logger           LogConfig `json:"logger"`
	Name             string
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

	er := json.Unmarshal(raw, &Service)

	if er != nil {
		return er
	}

	Service.setOtherDefaults()

	return nil
}

func (s StatsproxyConfig) setOtherDefaults() {
	s.Workers = 2
	s.Name = "Statsproxy"
}

func newLogger() LogConfig {
	return LogConfig{false}
}

func (h Host) String() string {
	return fmt.Sprintf("%v:%v", h.FQDN, h.Port)
}
