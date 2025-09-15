package misc

import (
	"github.com/gin-gonic/gin"
)

func MiscController(router *gin.Engine) {
	routes := router.Group("/misc")
	
	routes.GET("/", func(c *gin.Context) {
		c.String(200, "wee")
	})

}
