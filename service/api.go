package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/utils"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"time"
)

func (t *Manager) ConfigureRouter(r *gin.Engine) {
	r.Use(static.Serve("/", static.LocalFile("/ui", false)))

	api := r.Group("/api")
	{
		api.GET("/v1/services", func(c *gin.Context) {
			var result []ServiceEntry

			result = append(result, ServiceEntry{"xud", "XUD"})
			result = append(result, ServiceEntry{"lndbtc", "LND (Bitcoin)"})
			result = append(result, ServiceEntry{"lndltc", "LND (Litecoin)"})
			result = append(result, ServiceEntry{"connext", "Connext"})
			result = append(result, ServiceEntry{"bitcoind", "Bitcoind"})
			result = append(result, ServiceEntry{"litecoind", "Litecoind"})
			result = append(result, ServiceEntry{"geth", "Geth"})
			result = append(result, ServiceEntry{"arby", "Arby"})
			result = append(result, ServiceEntry{"boltz", "Boltz"})
			result = append(result, ServiceEntry{"webui", "Web UI"})

			c.JSON(http.StatusOK, result)
		})

		api.GET("/v1/status", func(c *gin.Context) {
			status := t.GetStatus()

			var result []ServiceStatus

			for _, svc := range t.services {
				result = append(result, ServiceStatus{Service: svc.GetName(), Status: status[svc.GetName()]})
			}

			c.JSON(http.StatusOK, result)
		})

		api.GET("/v1/status/:service", func(c *gin.Context) {
			service := c.Param("service")
			s, err := t.GetService(service)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusNotFound)
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			status := s.GetStatus(ctx)
			c.JSON(http.StatusOK, ServiceStatus{Service: service, Status: status})
		})

		api.GET("/v1/logs/:service", func(c *gin.Context) {
			service := c.Param("service")
			s, err := t.GetService(service)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusNotFound)
				return
			}
			since := c.DefaultQuery("since", "1h")
			tail := c.DefaultQuery("tail", "all")
			logs, err := s.GetLogs(since, tail)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusInternalServerError)
				return
			}
			c.Header("Content-Type", "text/plain")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.log\"", service))
			for _, line := range logs {
				_, err = c.Writer.WriteString(line + "\n")
				if err != nil {
					utils.JsonError(c, err.Error(), http.StatusInternalServerError)
				}
			}
		})

		api.GET("/v1/setup-status", func(c *gin.Context) {

			statusChan, cancel := t.subscribeSetupStatus(-1)

			c.Stream(func(w io.Writer) bool {
				for status := range statusChan {
					j, _ := json.Marshal(status)
					c.Writer.Write(j)
					c.Writer.Write([]byte("\n"))
					c.Writer.Flush()

					if status.Status == "Done" {
						cancel()
						break
					}
				}
				return false
			})

		})
	}

	for _, svc := range t.services {
		svc.ConfigureRouter(api)
	}
}
