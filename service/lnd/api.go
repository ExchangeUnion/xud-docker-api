package lnd

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"net/http"
)

func (t *Service) ConfigureRouter(r *gin.RouterGroup) {
	r.GET(fmt.Sprintf("/v1/%s/getinfo", t.GetName()), func(c *gin.Context) {
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
}
