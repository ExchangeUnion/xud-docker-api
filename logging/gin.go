package logging

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"math"
	"time"
)

func LoggerOverLogrus() gin.HandlerFunc {

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()

		c.Next()

		statusCode := c.Writer.Status()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		clientIp := c.ClientIP()
		method := c.Request.Method

		logger := logrus.NewEntry(logrus.StandardLogger()).WithFields(logrus.Fields{
			"name": "gin",
			"statusCode": statusCode,
			"latency": latency,
			"clientIp": clientIp,
			"method": method,
			"path": path,
		})
		logger.Debugf(fmt.Sprintf("[%s] %s %s | %d | %dms", clientIp, method, path, statusCode, latency))
	}
}