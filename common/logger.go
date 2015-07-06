package common

import (
	"github.com/fullcontact/statsproxy/config"
	"log"
	"log/syslog"
)

var Logger LogAdapter

type LogAdapter interface {
	Dev(m string)
	Debug(m string)
	Info(m string)
	Warning(m string)
	Err(m string)
	Crit(m string)
}

type DevelopLogger struct{}
type SyslogLogger struct {
	w *syslog.Writer
}

func (d *DevelopLogger) Dev(m string) {
	log.Printf("[DEV] %v", m)
}

func (d *DevelopLogger) Debug(m string) {
	log.Printf("[DEBUG] %v", m)
}
func (d *DevelopLogger) Info(m string) {
	log.Printf("[INFO] %v", m)
}
func (d *DevelopLogger) Warning(m string) {
	log.Printf("[WARNING] %v", m)
}
func (d *DevelopLogger) Err(m string) {
	log.Printf("[ERR] %v", m)
}
func (d *DevelopLogger) Crit(m string) {
	log.Printf("[CRIT] %v", m)
}

func InitializeLogger() error {
	if config.Service.Logger.Develop {
		Logger = &DevelopLogger{}
		return nil
	} else {
		w, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_LOCAL0, config.Service.Name)
		if err != nil {
			return err
		}
		Logger = newSyslogLogger(w)
		return nil
	}
	return nil
}

func newSyslogLogger(w *syslog.Writer) *SyslogLogger {
	return &SyslogLogger{w: w}
}

func (s *SyslogLogger) Dev(m string) {
}

func (s *SyslogLogger) Debug(m string) {
	if err := s.w.Debug(m); err != nil {
		log.Printf("Error writing '%v' to syslog: %v", m, err)
	}
}
func (s *SyslogLogger) Info(m string) {
	if err := s.w.Info(m); err != nil {
		log.Printf("Error writing '%v' to syslog: %v", m, err)
	}
}
func (s *SyslogLogger) Warning(m string) {
	if err := s.w.Warning(m); err != nil {
		log.Printf("Error writing '%v' to syslog: %v", m, err)
	}
}
func (s *SyslogLogger) Err(m string) {
	if err := s.w.Err(m); err != nil {
		log.Printf("Error writing '%v' to syslog: %v", m, err)
	}
}
func (s *SyslogLogger) Crit(m string) {
	if err := s.w.Crit(m); err != nil {
		log.Printf("Error writing '%v' to syslog: %v", m, err)
	}
}
