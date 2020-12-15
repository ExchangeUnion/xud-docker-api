package boltz

import (
	"context"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (t *Service) ConfigureRouter(r *gin.RouterGroup) {
	r.GET("/v1/boltz/service-info/:currency", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), config.DefaultApiTimeout)
		defer cancel()
		resp, err := t.GetServiceInfo(ctx, c.Param("currency"))
		utils.HandleProtobufResponse(c, resp, err)
	})
	r.GET("/v1/boltz/deposit/:currency", func(c *gin.Context) {
		inboundLiquidity, err := strconv.Atoi(c.DefaultQuery("inbound_liquidity", "50"))
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid value %s for inbound_liquidity", c.Query("inbound_liquidity")), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), config.DefaultApiTimeout)
		defer cancel()
		resp, err := t.Deposit(ctx, c.Param("currency"), uint32(inboundLiquidity))
		utils.HandleProtobufResponse(c, resp, err)
	})
	r.POST("/v1/boltz/withdraw/:currency", func(c *gin.Context) {
		amount, err := strconv.ParseInt(c.PostForm("amount"), 10, 64)
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid amount %s", c.PostForm("amount")), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), config.DefaultApiTimeout)
		defer cancel()
		resp, err := t.Withdraw(ctx, c.Param("currency"), amount, c.PostForm("address"))
		utils.HandleProtobufResponse(c, resp, err)
	})
}
