package service

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
	"strings"
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
	c := exec.Command("tail", "-F", t.logfile)
	r, _ := c.StdoutPipe()
	c.Stderr = c.Stdout

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			t.logger.Debugf("*** %s", line)
			t.handleLine(line)
		}
	}()

	err := c.Run()
	if err != nil {
		t.logger.Errorf("Failed to tail %s: %s", t.logfile, err)
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
	t.logger.Debugf("Emit %s", status)
	t.statusHistory = append(t.statusHistory, status)
	for _, listener := range t.listeners {
		listener <- status
	}
}

func (t *LauncherAgent) subscribeSetupStatus(history int) (<-chan SetupStatus, func(), []SetupStatus) {
	ch := make(chan SetupStatus, 100)
	t.listeners = append(t.listeners, ch)

	var h []SetupStatus

	if history > 0 {
		if history >= len(t.statusHistory) {
			h = append(h, t.statusHistory...)
		} else {
			h = append(h, t.statusHistory[len(t.statusHistory)-history:]...)
		}
	} else if history == -1 {
		h = append(h, t.statusHistory...)
	}

	var cancel = func() {
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

	return ch, cancel, h
}
