package launcher

import (
	"github.com/ExchangeUnion/xud-docker-api/utils"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ConfigureRouter(r *gin.Engine) {
	r.Use(static.Serve("/", static.LocalFile("/ui", false)))

	api := r.Group("/api")
	{
		api.GET("/v1/info", func(c *gin.Context) {
			info, err := GetInfo()
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusInternalServerError)
				return
			}
			c.JSON(http.StatusOK, info)
		})
		api.PUT("/v1/backup", func(c *gin.Context) {
			var settings BackupSettings
			err := c.BindJSON(&settings)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusBadRequest)
			}
			ok, err := UpdateBackup(settings)
			if ok {
				c.Status(http.StatusNoContent)
			}
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusBadRequest)
			} else {
				utils.JsonError(c, "", http.StatusBadRequest)
			}
		})
		api.GET("/v1/launcher/setup-status", func(c *gin.Context) {

		})
	}
}

