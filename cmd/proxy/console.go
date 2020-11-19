package main

import (
	"encoding/json"
	"fmt"
	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

var (
	consoleMap = make(map[string]Console)
	mutex = sync.Mutex{}
)

type Console struct {
	Id      string   `json:"id"`
	Network string   `json:"network"`
	Pty     *os.File `json:"-"`
}

func init() {
	router.GET("/api/v1/consoles", listConsoles)
	router.GET("/api/v1/consoles/:id", getConsole)
	router.POST("/api/v1/consoles", createConsole)
}

func findById(id string) *Console {
	console, ok := consoleMap[id]
	if !ok {
		return nil
	}
	return &console
}

func getConsole(c *gin.Context) {
	id := c.Param("id")
	console := findById(id)
	if console == nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	c.JSON(http.StatusOK, console)
}

func listConsoles(c *gin.Context) {
	c.JSON(http.StatusOK, consoleMap)
}

type TerminalSize struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

type StartRequest struct {
	Id   string       `json:"id"`
	Size TerminalSize `json:"size"`
}

type StopRequest struct {
	Id string `json:"id"`
}

func createConsole(c *gin.Context) {
	network := os.Getenv("NETWORK")
	id := fmt.Sprint(uuid.New())
	console := Console{
		Id:      id,
		Network: network,
	}
	consoleMap[id] = console
	e := fmt.Sprintf("console-%s", id)
	logger.Debugf("[console] Register event %s", e)
	c.JSON(http.StatusOK, console)
}

func writeInitScript(network string) {
	f, err := os.Create("init.bash")
	if err != nil {
		logger.Errorf("Failed to write init.bash: %s", err)
		return
	}
	defer f.Close()

	f.WriteString(`\
cat <<EOF

                           .___           __  .__   
          ___  _____ __  __| _/     _____/  |_|  |  
          \  \/  /  |  \/ __ |    _/ ___\   __\  |  
           >    <|  |  / /_/ |    \  \___|  | |  |__
          /__/\_ \____/\____ |     \___  >__| |____/
                \/          \/         \/           
--------------------------------------------------------------

EOF

export NETWORK=` + network + `
export PS1="$NETWORK > "
function start() {
	docker start ${NETWORK}_${1}_1 
}
function stop() {
	docker stop ${NETWORK}_${1}_1
}
function restart() {
	docker restart ${NETWORK}_${1}_1
}
function down() {
	echo "Not implemented yet!"
}
function logs() {
	docker logs --tail=100 ${NETWORK}_${1}_1
}
function report() {
	cat <<EOF
Please click on https://github.com/ExchangeUnion/xud/issues/\
new?assignees=kilrau&labels=bug&template=bug-report.md&title=Short%2C+concise+\
description+of+the+bug, describe your issue, drag and drop the file "${NETWORK}\
.log" which is located in "{logs_dir}" into your browser window and submit \
your issue.
EOF
}
function xucli() {
	docker exec -it ${NETWORK}_xud_1 xucli $@
}
function lndbtc-lncli() {
	docker exec -it ${NETWORK}_lndbtc_1 lncli -n ${NETWORK} -c bitcoin $@
}
function lndltc-lncli() {
	docker exec -it ${NETWORK}_lndltc_1 lncli -n ${NETWORK} -c litecoin $@
}
function geth() {
	docker exec -it ${NETWORK}_geth_1 geth $@
}
function bitcoin-ctl() {	
	if [[ $NETWORK == "testnet" ]]; then
		docker exec -it ${NETWORK}_bitcoind_1 -testnet -user xu -password xu bitcoind $@
	else
		docker exec -it ${NETWORK}_bitcoind_1 -user xu -password xu bitcoind $@
	fi
}
function litecoin-ctl() {
	if [[ $NETWORK == "testnet" ]]; then
		docker exec -it ${NETWORK}_litecoind_1 -testnet -user xu -password xu litecoind $@
	else
		docker exec -it ${NETWORK}_litecoind_1 -user xu -password xu litecoind $@
	fi
}
function boltzcli() {
	docker exec -it ${NETWORK}_boltz_1 boltzcli $@
}

alias getinfo='xucli getinfo'
alias addcurrency='xucli addcurrency'
alias addpair='xucli addpair'
alias ban='xucli ban'
alias buy='xucli buy'
alias closechannel='xucli closechannel'
alias connect='xucli connect'
alias create='xucli create'
alias discovernodes='xucli discovernodes'
alias getbalance='xucli getbalance'
alias getnodeinfo='xucli getnodeinfo'
alias listcurrencies='xucli listcurrencies'
alias listorders='xucli listorders'
alias listpairs='xucli listpairs'
alias listpeers='xucli listpeers'
alias openchannel='xucli openchannel'
alias orderbook='xucli orderbook'
alias removeallorders='xucli removeallorders'
alias removecurrency='xucli removecurrency'
alias removeorder='xucli removeorder'
alias removepair='xucli removepair'
alias restore='xucli restore'
alias sell='xucli sell'
alias shutdown='xucli shutdown'
alias streamorders='xucli streamorders'
alias tradehistory='xucli tradehistory'
alias tradelimits='xucli tradelimits'
alias unban='xucli unban'
alias unlock='xucli unlock'
alias walletdeposit='xucli walletdeposit'
alias walletwithdraw='xucli walletwithdraw'
`)
}

func startShell(console *Console) error {
	writeInitScript(console.Network)
	c := exec.Command("bash", "--init-file", "init.bash")

	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}

	console.Pty = ptmx

	return nil
}

func initSioConsole() {
	sioServer.OnEvent("/", "start", func(s socketio.Conn, data string) {
		// data = id + size
		req := StartRequest{}
		err := json.Unmarshal([]byte(data), &req)
		if err != nil {
			s.Emit("start", fmt.Sprintf("invalid request: %s", err))
			return
		}

		logger.Debugf("[console] Start %s", req.Id)

		console := findById(req.Id)
		if console == nil {
			s.Emit("start", "console not found")
			return
		}
		err = startShell(console)
		if err != nil {
			s.Emit("start", fmt.Sprintf("failed to start: %s", err))
			return
		}

		e := fmt.Sprintf("console-%s", console.Id)

		sioServer.OnEvent("/", e, func(s socketio.Conn, data string) {
			logger.Debugf("[console] %s: <input> %v", console.Id, data)

			pty_ := console.Pty

			_, err := pty_.WriteString(data)
			if err != nil {
				logger.Errorf("Failed to write to console %s: %s", console.Id, err)
			}
		})



		s.Join("output")
		// TODO resize

		go func() {
			//io.Copy(SioWriter{Conn: s}, ptmx)
			var buf = make([]byte, 1024)
			for {
				pty_ := console.Pty

				n, err := pty_.Read(buf)
				if err != nil {
					logger.Errorf("Failed to read from console %s: %s", req.Id, err)
					break
				}
				buf = buf[:n]
				logger.Debugf("[console] %s: <output> %v", console.Id, buf)

				sioServer.BroadcastToRoom("/", "output", e, string(buf))
			}
		}()
	})
}
