package common

import (
	"github.com/fullcontact/statsproxy/config"
	"time"
)

var StatsTicker *time.Ticker

func InitializeTickers() {
	Logger.Info("Initializing the tickers to send stats about service")
	StatsTicker = time.NewTicker(time.Duration(
		config.Service.TickerPeriod) * time.Second)
}
