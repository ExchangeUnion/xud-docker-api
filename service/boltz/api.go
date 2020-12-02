package boltz

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (t *Service) ConfigureRouter(r *gin.RouterGroup) {
	r.GET("/v1/boltz/service-info/:currency", func(c *gin.Context) {
		resp, err := t.GetServiceInfo(c.Param("currency"))
		utils.HandleProtobufResponse(c, resp, err)
	})
	r.GET("/v1/boltz/deposit/:currency", func(c *gin.Context) {
		inboundLiquidity, err := strconv.Atoi(c.DefaultQuery("inbound_liquidity", "50"))
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid value %s for inbound_liquidity", c.Query("inbound_liquidity")), http.StatusBadRequest)
			return
		}
		resp, err := t.Deposit(c.Param("currency"), uint32(inboundLiquidity))
		utils.HandleProtobufResponse(c, resp, err)
	})
	r.POST("/v1/boltz/withdraw/:currency", func(c *gin.Context) {
		amount, err := strconv.ParseInt(c.PostForm("amount"), 10, 64)
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid amount %s", c.PostForm("amount")), http.StatusBadRequest)
			return
		}
		resp, err := t.Withdraw(c.Param("currency"), amount, c.PostForm("address"))
		utils.HandleProtobufResponse(c, resp, err)
	})
}
