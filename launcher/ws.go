package launcher

import (
"encoding/json"
"fmt"
"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
)

var (
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
	}

	launchers []*Launcher
	requests = make(chan LauncherRequest)

	logger *logrus.Entry

	errNoLauncher = fmt.Errorf("no launcher")
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logger = logrus.NewEntry(logrus.StandardLogger()).WithField("name", "launcher")
}

type RequestHandler interface {
	HandleResponse(resp *Response)
}

type Launcher struct {
	conn *websocket.Conn

	counter uint16
	requests map[uint16]chan Response
}

type Request struct {
	Id uint16 `json:"id"`
	Method string `json:"method"`
	Params []string `json:"params"`
}

type Response struct {
	Id uint16 `json:"id"`
	Result string `json:"result"`
	Error interface{} `json:"error"`
}

type LauncherRequest struct {
	Launcher *Launcher
	Request *Request
}

type Info struct {
	Test string `json:"test"`
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("Failed to set websocket upgrade: %+v", err)
		return
	}

	logger.Debugf("New launcher attached. Launchers = %d", len(launchers))

	launcher := Launcher{
		conn: conn,
		counter: 0,
		requests: make(map[uint16]chan Response),
	}

	launchers = append(launchers, &launcher)

	go launcher.Listen()
}

func StartLauncherRegistry() {
	for {
		req := <-requests
		logger.Debugf("launcher %v request %v", req.Launcher, req.Request)
	}
}

func GetInfo() (interface{}, error) {
	if len(launchers) == 0 {
		return nil, errNoLauncher
	}
	return launchers[0].GetInfo()
}

func (t *Launcher) GetInfo() (*json.RawMessage, error){
	t.counter += 1
	req := Request{
		Id: t.counter,
		Method: "getinfo",
		Params: []string{},
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	t.conn.WriteMessage(websocket.TextMessage, payload)

	respChan := make(chan Response)

	t.requests[req.Id] = respChan

	resp := <-respChan

	close(respChan)
	delete(t.requests, req.Id)

	var info json.RawMessage

	err = json.Unmarshal([]byte(resp.Result), &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

type BackupSettings struct {
	Location string
}

func UpdateBackup(settings BackupSettings) (bool, error) {
	if len(launchers) == 0 {
		return false, errNoLauncher
	}
	return launchers[0].UpdateBackup(settings)
}

func (t *Launcher) UpdateBackup(settings BackupSettings) (bool, error) {
	t.counter += 1
	req := Request{
		Id: t.counter,
		Method: "backupto",
		Params: []string{settings.Location},
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return false, err
	}
	t.conn.WriteMessage(websocket.TextMessage, payload)

	respChan := make(chan Response)

	t.requests[req.Id] = respChan

	resp := <-respChan

	close(respChan)
	delete(t.requests, req.Id)

	if resp.Error == nil {
		return true, nil
	}

	return false, fmt.Errorf("%s", resp.Error)
}


func (t *Launcher) Listen() {
	for {
		msgType, msg, err := t.conn.ReadMessage()
		if err != nil {
			logger.Debugf("Failed to listen for messages from launcher: %s", err)
			// remove launcher from launchers
			for i, launcher := range launchers {
				if launcher == t {
					launchers = append(launchers[:i], launchers[i+1:]...)
				}
			}
			break
		}
		if msgType == websocket.TextMessage {
			logger.Debugf("[Attach] Got text message: %s", string(msg))
			var resp Response
			err = json.Unmarshal(msg, &resp)
			if err != nil {

				var req Request
				err = json.Unmarshal(msg, &req)
				if err != nil {
					logrus.Debugf("Failed to parse message: %s", err)
				}

				logger.Debugf("[Attach] The message is a request")
				requests <- LauncherRequest{Launcher: t, Request: &req}

			} else {

				logger.Debugf("[Attach] The message is a response")
				t.requests[resp.Id] <- resp

			}
		} else {
			logger.Debugf("[Attach] Got non-text message: %v", msg)
		}
	}
}
