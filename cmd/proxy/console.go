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
	mutex      = sync.Mutex{}
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
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

type StartRequest struct {
	Id   string       `json:"id"`
	Size TerminalSize `json:"size"`
}

type ResizeRequest struct {
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

var help = `\
Xucli shortcut commands
  addcurrency <currency>                    add a currency
  <swap_client> [decimal_places]
  [token_address]
  addpair <pair_id|base_currency>           add a trading pair
  [quote_currency]
  ban <node_identifier>                     ban a remote node
  buy <quantity> <pair_id> <price>          place a buy order
  [order_id]
  closechannel <currency>                   close any payment channels with a
  [node_identifier ] [--force]              peer
  connect <node_uri>                        connect to a remote node
  create                                    create a new xud instance and set a
                                            password
  discovernodes <node_identifier>           discover nodes from a specific peer
  getbalance [currency]                     get total balance for a given
                                            currency
  getinfo                                   get general info from the local xud
                                            node
  getnodeinfo <node_identifier>             get general information about a
                                            known node
  listcurrencies                            list available currencies
  listorders [pair_id] [owner]              list orders from the order book
  [limit]
  listpairs                                 get order book's available pairs
  listpeers                                 list connected peers
  openchannel <currency> <amount>           open a payment channel with a peer
  [node_identifier] [push_amount]
  orderbook [pair_id] [precision]           display the order book, with orders
                                            aggregated per price point
  removecurrency <currency>                 remove a currency
  removeorder <order_id> [quantity]         remove an order
  removepair <pair_id>                      remove a trading pair
  restore [backup_directory]                restore an xud instance from seed
  sell <quantity> <pair_id> <price>         place a sell order
  [order_id]
  shutdown                                  gracefully shutdown local xud node
  streamorders [existing]                   stream order added, removed, and
                                            swapped events (DEMO)
  tradehistory [limit]                      list completed trades
  tradinglimits [currency]                  trading limits for a given currency
  unban <node_identifier>                   unban a previously banned remote
  [--reconnect]                             node
  unlock                                    unlock local xud node
  walletdeposit <currency>                  gets an address to deposit funds to
                                            xud
  walletwithdraw [amount] [currency]        withdraws on-chain funds from xud
  [destination] [fee]
  
General commands
  status                                    show service status
  report                                    report issue
  logs                                      show service log
  start                                     start service
  stop                                      stop service
  restart                                   restart service
  down                                      shutdown the environment
  up                                        bring up the environment
  help                                      show this help
  exit                                      exit xud-ctl shell

CLI commands
  bitcoin-cli                               bitcoind cli
  litecoin-cli                              litecoind cli
  lndbtc-lncli                              lnd cli
  lndltc-lncli                              lnd cli
  geth                                      geth cli
  xucli                                     xud cli
  boltzcli                                  boltz cli

Boltzcli shortcut commands  
  deposit <chain> deposit 
  --inbound [inbound_balance]               deposit from boltz (btc/ltc)
  boltzcli <chain> withdraw 
  <amount> <address>                        withdraw from boltz channel
`

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
function help() {
	echo "` + help + `"
}
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

func startShell(console *Console, size TerminalSize) error {
	writeInitScript(console.Network)
	c := exec.Command("/bin/bash", "--init-file", "init.bash")

	ptmx, err := pty.StartWithSize(c, &pty.Winsize{Cols: size.Cols, Rows: size.Rows, X: 0, Y: 0})
	if err != nil {
		return err
	}

	console.Pty = ptmx

	return nil
}

func initSioConsole() {
	sioServer.OnEvent("/", "start", func(s socketio.Conn, data string) {
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
		err = startShell(console, req.Size)
		if err != nil {
			s.Emit("start", fmt.Sprintf("failed to start: %s", err))
			return
		}

		inputEvent := fmt.Sprintf("console.%s.input", console.Id)
		outputEvent := fmt.Sprintf("console.%s.output", console.Id)

		sioServer.OnEvent("/", inputEvent, func(s socketio.Conn, data string) {
			logger.Debugf("[console/%s] ---> %v", console.Id, data)

			pty_ := console.Pty

			_, err := pty_.WriteString(data)
			if err != nil {
				logger.Errorf("Failed to write to console %s: %s", console.Id, err)
			}
		})

		go func() {
			var buf = make([]byte, 65536)
			for {
				pty_ := console.Pty

				n, err := pty_.Read(buf)
				if err != nil {
					logger.Errorf("Failed to read from console %s: %s", req.Id, err)
					break
				}
				data := buf[:n]
				logger.Debugf("[console/%s] <--- %v", console.Id, data)

				sioServer.BroadcastToRoom("/", s.ID(), outputEvent, string(data))
			}
		}()
	})

	sioServer.OnEvent("/", "resize", func(s socketio.Conn, data string) {
		req := ResizeRequest{}
		err := json.Unmarshal([]byte(data), &req)
		if err != nil {
			s.Emit("resize", fmt.Sprintf("invalid request: %s", err))
			return
		}

		console := findById(req.Id)
		if console == nil {
			s.Emit("resize", "console not found")
			return
		}

		err = pty.Setsize(console.Pty, &pty.Winsize{Rows: req.Size.Rows, Cols: req.Size.Cols, X: 0, Y: 0})
		if err != nil {
			s.Emit("resize", fmt.Sprintf("failed to resize: %s", err))
		}
	})
}
