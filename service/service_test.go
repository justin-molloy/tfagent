//go:build windows

package service

import (
	"testing"
	"time"

	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/selector"
	"github.com/justin-molloy/tfagent/tracker"
	"golang.org/x/sys/windows/svc"
)

func TestServiceStateToString(t *testing.T) {
	cases := []struct {
		in   svc.State
		want string
	}{
		{svc.Stopped, "Stopped"},
		{svc.StartPending, "Start Pending"},
		{svc.StopPending, "Stop Pending"},
		{svc.Running, "Running"},
		{svc.ContinuePending, "Continue Pending"},
		{svc.PausePending, "Pause Pending"},
		{svc.Paused, "Paused"},
		{svc.State(12345), "Unknown (12345)"},
	}
	for _, tc := range cases {
		got := serviceStateToString(tc.in)
		if got != tc.want {
			t.Fatalf("serviceStateToString(%v) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func minimalConfig() *config.ConfigData {
	// Heartbeat disabled so runHeartbeat won't post to the SCM channel.
	return &config.ConfigData{Heartbeat: false}
}

func TestExecute_StartThenStop_SendsExpectedStatusesAndReturns(t *testing.T) {
	reqCh := make(chan svc.ChangeRequest, 4)
	statusCh := make(chan svc.Status, 8)

	s := &TFAgentService{
		Name:       "tfagent-test",
		Config:     minimalConfig(),
		Tracker:    &tracker.EventTracker{},  // minimal, non-nil
		FileQueue:  make(chan string),        // close so processor/selectors exit if they read
		Processing: &selector.FileSelector{}, // minimal, non-nil
	}
	close(s.FileQueue)

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Execute(nil, reqCh, statusCh)
	}()

	// 1) StartPending should be first
	select {
	case st := <-statusCh:
		if st.State != svc.StartPending {
			t.Fatalf("expected StartPending first, got %v", st.State)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for StartPending")
	}

	// 2) Running should follow with Stop/Shutdown accepted
	select {
	case st := <-statusCh:
		if st.State != svc.Running {
			t.Fatalf("expected Running, got %v", st.State)
		}
		if st.Accepts&(svc.AcceptStop|svc.AcceptShutdown) != (svc.AcceptStop | svc.AcceptShutdown) {
			t.Fatalf("expected to accept Stop+Shutdown, got %+v", st.Accepts)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Running")
	}

	// 3) Send Stop request; expect StopPending, then Execute returns
	reqCh <- svc.ChangeRequest{Cmd: svc.Stop}

	select {
	case st := <-statusCh:
		if st.State != svc.StopPending {
			t.Fatalf("expected StopPending after Stop, got %v", st.State)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for StopPending")
	}

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Execute did not return after Stop")
	}
}

func TestExecute_Interrogate_EchoesCurrentStatus(t *testing.T) {
	reqCh := make(chan svc.ChangeRequest, 2)
	statusCh := make(chan svc.Status, 8)

	s := &TFAgentService{
		Name:       "tfagent-test",
		Config:     minimalConfig(),
		Tracker:    &tracker.EventTracker{},
		FileQueue:  make(chan string),
		Processing: &selector.FileSelector{},
	}
	close(s.FileQueue)

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Execute(nil, reqCh, statusCh)
	}()

	// Drain StartPending and Running
	for i := 0; i < 2; i++ {
		select {
		case <-statusCh:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for initial statuses")
		}
	}

	// Send Interrogate; Execute should echo the provided CurrentStatus
	reqCh <- svc.ChangeRequest{Cmd: svc.Interrogate, CurrentStatus: svc.Status{State: svc.Running}}

	select {
	case st := <-statusCh:
		if st.State != svc.Running {
			t.Fatalf("expected echo of Running, got %v", st.State)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Interrogate echo")
	}

	// Stop cleanly
	reqCh <- svc.ChangeRequest{Cmd: svc.Stop}
	// Expect StopPending then return
	select {
	case <-statusCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for StopPending")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Execute did not return after Stop")
	}
}
