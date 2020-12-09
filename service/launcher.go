package service

import (
	"fmt"
	"github.com/hpcloud/tail"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type SetupStatus struct {
	Status  string      `json:"status"`
	Details interface{} `json:"details"`
}

type LauncherAgent struct {
	listeners     []chan SetupStatus
	logfile       string
	running       bool
	logger        *logrus.Entry
	statusHistory []SetupStatus
}

func NewLauncherAgent(network string, logger *logrus.Entry) *LauncherAgent {
	a := &LauncherAgent{
		listeners:     []chan SetupStatus{},
		logfile:       fmt.Sprintf("/root/network/logs/%s.log", network),
		running:       true,
		logger:        logger,
		statusHistory: []SetupStatus{},
	}

	go a.followLog()

	return a
}

func (t *LauncherAgent) followLog() {
	for {
		t.logger.Debugf("Trying to follow logfile %s", t.logfile)
		r, err := tail.TailFile(t.logfile, tail.Config{
			Follow: true,
			ReOpen: true,
			//Location: &tail.SeekInfo{
			//	Offset: 0,
			//	Whence: io.SeekStart,
			//},
		})
		if err != nil {
			t.logger.Debugf("Failed to tail file %s: %s", t.logfile, err)
			time.Sleep(1 * time.Second)
			continue
		}
		t.logger.Debugf("Iterating log lines")
		for line := range r.Lines {
			t.logger.Debugf("*** %s", line.Text)
			t.handleLine(line.Text)
		}
		break
	}
}

func (t *LauncherAgent) handleLine(line string) {
	if strings.Contains(line, "Waiting for XUD dependencies to be ready") {
		status := SetupStatus{Status: "Waiting for XUD dependencies to be ready", Details: nil}
		t.emitStatus(status)
	} else if strings.Contains(line, "LightSync") {
		parts := strings.Split(line, " [LightSync] ")
		parts = strings.Split(parts[1], " | ")
		details := map[string]string{}
		status := SetupStatus{Status: "Syncing light clients", Details: details}
		for _, p := range parts {
			kv := strings.Split(p, ": ")
			details[kv[0]] = kv[1]
		}
		t.emitStatus(status)
	} else if strings.Contains(line, "Setup wallets") {
		status := SetupStatus{Status: "Setup wallets", Details: nil}
		t.emitStatus(status)
	} else if strings.Contains(line, "Create wallets") {
		status := SetupStatus{Status: "Create wallets", Details: nil}
		t.emitStatus(status)
	} else if strings.Contains(line, "Restore wallets") {
		status := SetupStatus{Status: "Restore wallets", Details: nil}
		t.emitStatus(status)
	} else if strings.Contains(line, "Setup backup location") {
		status := SetupStatus{Status: "Setup backup location", Details: nil}
		t.emitStatus(status)
	} else if strings.Contains(line, "Unlock wallets") {
		status := SetupStatus{Status: "Unlock wallets", Details: nil}
		t.emitStatus(status)
	} else if strings.Contains(line, "Start shell") {
		status := SetupStatus{Status: "Done", Details: nil}
		t.emitStatus(status)
		//t.statusHistory = []SetupStatus{}
	}
}

func (t *LauncherAgent) emitStatus(status SetupStatus) {
	t.statusHistory = append(t.statusHistory, status)
	for _, listener := range t.listeners {
		listener <- status
	}
}

func (t *LauncherAgent) subscribeSetupStatus(history int) (<-chan SetupStatus, func()) {
	ch := make(chan SetupStatus)
	t.listeners = append(t.listeners, ch)

	// FIXME make sure all history is emited before new status comes in
	go func() {
		if history > 0 {
			if history >= len(t.statusHistory) {
				for _, status := range t.statusHistory {
					ch <- status
				}
			} else {
				for _, status := range t.statusHistory[len(t.statusHistory)-history:] {
					ch <- status
				}
			}
		} else if history == -1 {

			t.logger.Debugf("history count: %d", len(t.statusHistory))
			for _, status := range t.statusHistory {
				ch <- status
			}
		}
	}()

	return ch, func() {
		for i, listener := range t.listeners {
			if listener == ch {
				if i+1 >= len(t.listeners) {
					t.listeners = t.listeners[:i]
				} else {
					t.listeners = append(t.listeners[:i], t.listeners[i+1:]...)
				}
				break
			}
		}
	}
}
