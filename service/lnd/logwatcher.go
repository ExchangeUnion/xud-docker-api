package lnd

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type NeutrinoSyncing struct {
	current int64
	total   int64
	done    bool
}

type LogWatcher struct {
	p               *regexp.Regexp
	p0              *regexp.Regexp
	p1              *regexp.Regexp
	p2              *regexp.Regexp
	neutrinoSyncing NeutrinoSyncing
	logger          *logrus.Entry
	service         *core.SingleContainerService
	stop            func()
}

func initRegex(containerName string) (*regexp.Regexp, *regexp.Regexp, *regexp.Regexp, *regexp.Regexp) {
	p, err := regexp.Compile("^.*NTFN: New block: height=(\\d+), sha=(.+)$")
	if err != nil {
		panic(err)
	}

	p0, err := regexp.Compile("^.*Fully caught up with cfheaders at height (\\d+), waiting at tip for new blocks$")
	if err != nil {
		panic(err)
	}

	var p1 *regexp.Regexp

	if strings.Contains(containerName, "simnet") {
		p1, err = regexp.Compile("^.*Writing cfheaders at height=(\\d+) to next checkpoint$")
		if err != nil {
			panic(err)
		}
	} else {
		p1, err = regexp.Compile("^.*Fetching set of checkpointed cfheaders filters from height=(\\d+).*$")
		if err != nil {
			panic(err)
		}
	}

	p2, err := regexp.Compile("^.*Syncing to block height (\\d+) from peer.*$")
	if err != nil {
		panic(err)
	}

	return p, p0, p1, p2
}

func NewLogWatcher(containerName string, service *core.SingleContainerService) *LogWatcher {
	p, p0, p1, p2 := initRegex(containerName)
	w := &LogWatcher{
		p:               p,
		p0:              p0,
		p1:              p1,
		p2:              p2,
		neutrinoSyncing: NeutrinoSyncing{current: 0, total: 0, done: false},
		logger:          service.GetLogger().WithField("scope", "LogWatcher"),
		service:         service,
	}
	return w
}

func (t *LogWatcher) getLogs() <-chan string {
	for {
		lines, stop, err := t.service.FollowLogs2()
		t.stop = stop
		if err != nil {
			t.logger.Error("Failed to follow logs: %s (will retry in 3 seconds)", err)
			time.Sleep(3 * time.Second)
		}
		return lines
	}
}

func (t *LogWatcher) stopFollowing() {
	if t.stop != nil {
		t.stop()
	}
}

func (t *LogWatcher) getNumber(p *regexp.Regexp, line string) int64 {
	s := p.ReplaceAllString(line, "$1")
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse \"%s\" as Int: %s", s, err))
	}
	return n
}

func (t *LogWatcher) Start() {

	t.logger.Debug("Starting")

	lines := t.getLogs()
	for line := range lines {

		line = strings.TrimSpace(line)

		if t.p0.MatchString(line) {
			t.logger.Debugf("*** %s", line)
			t.neutrinoSyncing.current = t.getNumber(t.p0, line)
			if t.neutrinoSyncing.current < t.neutrinoSyncing.total {
				t.neutrinoSyncing.current = t.neutrinoSyncing.total
			} else if t.neutrinoSyncing.current > t.neutrinoSyncing.total {
				t.neutrinoSyncing.total = t.neutrinoSyncing.current
			}
			t.neutrinoSyncing.done = true
		} else if t.p1.MatchString(line) {
			t.logger.Debugf("*** %s", line)
			t.neutrinoSyncing.current = t.getNumber(t.p1, line)
		} else if t.p2.MatchString(line) {
			t.logger.Debugf("*** %s", line)
			t.neutrinoSyncing.total = t.getNumber(t.p2, line)
		} else if line == "--- EOF ---" {
			t.logger.Debugf("Reset Neutrino syncing state")
			t.neutrinoSyncing.current = 0
			t.neutrinoSyncing.total = 0
			t.neutrinoSyncing.done = false
		}
	}

	t.stopFollowing()

	t.logger.Debug("Stopped")

}

func (t *LogWatcher) GetNeutrinoStatus() string {
	current := t.neutrinoSyncing.current
	total := t.neutrinoSyncing.total
	return syncingText(current, total)
}

func (t *LogWatcher) Stop() {
	t.stopFollowing()
}

func (t *LogWatcher) getCurrentHeight() (uint32, error) {
	logs, err := t.service.GetLogs("10m", "all")
	if err != nil {
		return 0, nil
	}

	var height string

	for _, line := range logs {
		if t.p.MatchString(line) {
			height = t.p.ReplaceAllString(line, "$1")
		}
	}

	if height != "" {
		i64, err := strconv.ParseInt(height, 10, 32)
		if err != nil {
			return 0, nil
		}
		return uint32(i64), nil
	}

	return 0, nil
}
