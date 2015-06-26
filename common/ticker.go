package common

import (
	"github.com/frightenedmonkey/statsproxy/config"
	"time"
)

var StatsTicker *time.Ticker

func InitializeTickers() {
	StatsTicker = time.NewTicker(time.Duration(
		config.Service.TickerPeriod) * time.Second)
}
