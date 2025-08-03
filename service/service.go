//go:build windows

package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/tracker"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type TFAgentService struct {
	Name    string
	Config  *config.ConfigData
	Tracker *tracker.EventTracker
}

func (m *TFAgentService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {

	s <- svc.Status{State: svc.StartPending}

	// entry point to the file system tracker

	go tracker.StartTracker(m.Config, m.Tracker)
	go runHeartbeat(s, m.Name, m.Config.Heartbeat)

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	slog.Info("Service started", "servicename", m.Name)

	for {
		req := <-r

		switch req.Cmd {
		case svc.Interrogate:
			s <- req.CurrentStatus

		case svc.Stop, svc.Shutdown:
			slog.Info("Service stop requested", "command", int(req.Cmd), "servicename", m.Name)
			s <- svc.Status{State: svc.StopPending}
			return false, 0

		default:
			slog.Warn("Unhandled service command", "command", int(req.Cmd), "servicename", m.Name)
		}
	}
}

func runHeartbeat(statusChan chan<- svc.Status, serviceName string, heartbeat bool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		state, err := queryServiceStatus(serviceName)
		if err != nil {
			slog.Error("Failed to query service status", "servicename", serviceName, "error", err)
		}

		if heartbeat {
			slog.Info("Service heartbeat", "servicename", serviceName, "state", state)
			// heartbeat update to SCM
			statusChan <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
		}
	}
}

// human readable labels

func queryServiceStatus(serviceName string) (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return "", err
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", err
	}

	return serviceStateToString(status.State), nil
}

func serviceStateToString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Start Pending"
	case svc.StopPending:
		return "Stop Pending"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "Continue Pending"
	case svc.PausePending:
		return "Pause Pending"
	case svc.Paused:
		return "Paused"
	default:
		return fmt.Sprintf("Unknown (%d)", state)
	}
}
