package xud

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"net/http"
	"strconv"
)

func (t *Service) ConfigureRouter(r *gin.RouterGroup) {
	r.GET("/v1/xud/getinfo", func(c *gin.Context) {
		resp, err := t.GetInfo()
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
	})

	r.GET("/v1/xud/getbalance", func(c *gin.Context) {
		resp, err := t.GetBalance("")
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
	})

	r.GET("/v1/xud/getbalance/:currency", func(c *gin.Context) {
		resp, err := t.GetBalance(c.Param("currency"))
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
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
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
	})

	r.GET("/v1/xud/tradinglimits", func(c *gin.Context) {
		resp, err := t.GetTradingLimits("")
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
	})

	r.GET("/v1/xud/tradinglimits/:currency", func(c *gin.Context) {
		resp, err := t.GetTradingLimits(c.Param("currency"))
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
	})

	r.POST("/v1/xud/create", func(c *gin.Context) {

	})

	r.POST("/v1/xud/restore", func(c *gin.Context) {

	})

	r.POST("/v1/xud/unlock", func(c *gin.Context) {

	})
}
