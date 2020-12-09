package xud

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/utils"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func (t *Service) ConfigureRouter(r *gin.RouterGroup) {
	r.GET("/v1/xud/getinfo", func(c *gin.Context) {
		resp, err := t.GetInfo()
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.GET("/v1/xud/getbalance", func(c *gin.Context) {
		resp, err := t.GetBalance("")
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.GET("/v1/xud/getbalance/:currency", func(c *gin.Context) {
		resp, err := t.GetBalance(c.Param("currency"))
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.GET("/v1/xud/tradehistory", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "0")
		limit, err := strconv.ParseUint(limitStr, 10, 32)
		if err != nil {
			msg := fmt.Sprintf("invalid limit: %s", err.Error())
			utils.JsonError(c, msg, http.StatusBadRequest)
			return
		}
		if limit < 0 {
			msg := fmt.Sprintf("invalid limit: %d", limit)
			utils.JsonError(c, msg, http.StatusBadRequest)
			return
		}
		resp, err := t.GetTradeHistory(uint32(limit))
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.GET("/v1/xud/tradinglimits", func(c *gin.Context) {
		resp, err := t.GetTradingLimits("")
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.GET("/v1/xud/tradinglimits/:currency", func(c *gin.Context) {
		resp, err := t.GetTradingLimits(c.Param("currency"))
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.POST("/v1/xud/create", func(c *gin.Context) {
		var params CreateParams
		err := c.BindJSON(&params)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusBadRequest)
		}
		resp, err := t.CreateNode(params.Password)
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.POST("/v1/xud/restore", func(c *gin.Context) {
		var params RestoreParams
		err := c.BindJSON(&params)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusBadRequest)
		}

		lndBTC, err := ioutil.ReadFile(filepath.Join(params.BackupDir, "lnd-BTC"))
		lndLTC, err := ioutil.ReadFile(filepath.Join(params.BackupDir, "lnd-LTC"))
		xud, err := ioutil.ReadFile(filepath.Join(params.BackupDir, "xud"))

		resp, err := t.RestoreNode(
			params.Password,
			strings.Split(params.SeedMnemonic, " "),
			map[string][]byte{
				"BTC": lndBTC,
				"LTC": lndLTC,
			},
			xud,
		)
		utils.HandleProtobufResponse(c, resp, err)
	})

	r.POST("/v1/xud/unlock", func(c *gin.Context) {
		var params UnlockParams
		err := c.BindJSON(&params)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusBadRequest)
		}
		resp, err := t.UnlockNode(params.Password)
		utils.HandleProtobufResponse(c, resp, err)
	})
}

type CreateParams struct {
	Password string `json:"password"`
}

type RestoreParams struct {
	Password     string `json:"password"`
	SeedMnemonic string `json:"seedMnemonic"`
	BackupDir    string `json:"backupDir"`
}

type UnlockParams struct {
	Password string `json:"password"`
}
