package api

import (
	"github.com/gin-gonic/gin"
	"go.dfds.cloud/oops/feats/api/controller"
)

func Configure(router *gin.Engine) {
	controller.AddControllers(router)
}
