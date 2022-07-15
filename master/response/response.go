package response

import (
	"cronTab/master/response/code"
	"github.com/gin-gonic/gin"
	"net/http"
)

//对gin 框架返回值的封装

//Success 返回成功
func Success(c *gin.Context, statusCode code.Code, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": statusCode.GetCode(),
		"msg":  statusCode.GetCodeMsg(),
		"data": data,
	})
}

//Fail 返回失败
func Fail(c *gin.Context, httpCode int, statusCode code.Code, err error) {
	c.JSON(httpCode, gin.H{
		"code":      statusCode.GetCode(),
		"errMsg":    statusCode.GetCodeMsg(),
		"errDetail": err.Error(),
	})
}
