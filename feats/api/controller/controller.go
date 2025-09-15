package controller

import (
	"github.com/gin-gonic/gin"
	"go.dfds.cloud/oops/feats/api/controller/misc"
)

func AddControllers(router *gin.Engine) {
	misc.MiscController(router)
}
