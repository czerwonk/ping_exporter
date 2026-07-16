//go:build windows

package main

import (
	"io"

	"github.com/czerwonk/ping_exporter/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type pingExporterService struct {
	cfg *config.Config
}

func (m *pingExporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	go runInteractive(m.cfg)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Info("Service stop requested")
				changes <- svc.Status{State: svc.StopPending}
				changes <- svc.Status{State: svc.Stopped}
				return
			default:
				log.Errorf("unexpected control request %d", c)
			}
		}
	}
}

func runExporter(cfg *config.Config) {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if session is interactive: %v", err)
	}

	if isInteractive {
		runInteractive(cfg)
		return
	}

	runService(cfg)
}

func runService(cfg *config.Config) {
	const serviceName = "ping_exporter"

	hook, err := newEventLogHook(serviceName)
	if err != nil {
		log.Warnf("Failed to create event log hook: %v. Logging to console only.", err)
	} else {
		log.AddHook(hook)
		log.SetOutput(io.Discard)
	}

	log.Infof("Starting service %s", serviceName)
	err = svc.Run(serviceName, &pingExporterService{cfg: cfg})
	if err != nil {
		log.Errorf("Service failed: %v", err)
		return
	}
	log.Infof("Service %s stopped", serviceName)
}

type eventLogHook struct {
	l *eventlog.Log
}

func newEventLogHook(name string) (*eventLogHook, error) {
	l, err := eventlog.Open(name)
	if err != nil {
		err = eventlog.InstallAsEventCreate(name, eventlog.Info|eventlog.Warning|eventlog.Error)
		if err != nil {
			return nil, err
		}
		l, err = eventlog.Open(name)
		if err != nil {
			return nil, err
		}
	}
	return &eventLogHook{l: l}, nil
}

func (h *eventLogHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *eventLogHook) Fire(entry *log.Entry) error {
	msg, err := entry.String()
	if err != nil {
		return err
	}

	switch entry.Level {
	case log.PanicLevel, log.FatalLevel, log.ErrorLevel:
		return h.l.Error(1, msg)
	case log.WarnLevel:
		return h.l.Warning(2, msg)
	default:
		return h.l.Info(3, msg)
	}
}
