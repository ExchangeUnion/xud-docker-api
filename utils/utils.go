package utils

import (
	"github.com/gin-gonic/gin"
)

func JsonError(c *gin.Context, message string, code int) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")
	c.JSON(code, gin.H{
		"message": message,
	})
}
