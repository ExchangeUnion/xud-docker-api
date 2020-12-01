package boltz

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"net/http"
	"strconv"
)

func (t *Service) ConfigureRouter(r *gin.RouterGroup) {
	r.GET("/v1/boltz/service-info/:currency", func(c *gin.Context) {
		resp, err := t.GetServiceInfo(c.Param("currency"))
		t.HandleProtobufResponse(c, resp, err)
	})
	r.GET("/v1/boltz/deposit/:currency", func(c *gin.Context) {
		inboundLiquidity, err := strconv.Atoi(c.DefaultQuery("inbound_liquidity", "50"))
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid value %s for inbound_liquidity", c.Query("inbound_liquidity")), http.StatusBadRequest)
			return
		}
		resp, err := t.Deposit(c.Param("currency"), uint32(inboundLiquidity))
		t.HandleProtobufResponse(c, resp, err)
	})
	r.POST("/v1/boltz/withdraw/:currency", func(c *gin.Context) {
		amount, err := strconv.ParseInt(c.PostForm("amount"), 10, 64)
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid amount %s", c.PostForm("amount")), http.StatusBadRequest)
			return
		}
		resp, err := t.Withdraw(c.Param("currency"), amount, c.PostForm("address"))
		t.HandleProtobufResponse(c, resp, err)
	})
}

func (t *Service) HandleProtobufResponse(c *gin.Context, resp proto.Message, err error) {
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
}
