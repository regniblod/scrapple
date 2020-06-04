package main

import (
	"strings"

	"github.com/apex/log"
)

type BadgeLogger struct {
	logger *log.Logger
}

func (l *BadgeLogger) Warningf(f string, v ...interface{}) {
	l.logger.WithField("srv", "badge").Warnf(strings.Trim(f, "\n"), v...)
}

func (l *BadgeLogger) Infof(f string, v ...interface{}) {
	l.logger.WithField("srv", "badge").Infof(strings.Trim(f, "\n"), v...)
}

func (l *BadgeLogger) Errorf(f string, v ...interface{}) {
	l.logger.WithField("srv", "badge").Errorf(strings.Trim(f, "\n"), v...)
}

func (l *BadgeLogger) Debugf(f string, v ...interface{}) {
	l.logger.WithField("srv", "badge").Infof(strings.Trim(f, "\n"), v...)
}
