package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"net/http"
)

func JsonError(c *gin.Context, message string, code int) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")
	c.JSON(code, gin.H{
		"message": message,
	})
}

func HandleProtobufResponse(c *gin.Context, resp proto.Message, err error) {
	if err != nil {
		JsonError(c, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{EmitDefaults: true}
	err = m.Marshal(c.Writer, resp)
	if err != nil {
		JsonError(c, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
}
