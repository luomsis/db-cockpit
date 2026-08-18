package envelope

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应包裹：{code, message, data}。
// 成功 code=0；错误时 HTTP 状态码保留语义，body.code 镜像错误码。
const (
	CodeOK      = 0
	CodeNotImpl = 1001
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: code, Message: msg, Data: nil})
}

func BadRequest(c *gin.Context, msg string) {
	Fail(c, http.StatusBadRequest, http.StatusBadRequest, msg)
}

func NotFound(c *gin.Context, msg string) {
	Fail(c, http.StatusNotFound, http.StatusNotFound, msg)
}

func Internal(c *gin.Context, err error) {
	Fail(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
}

func NotImplemented(c *gin.Context) {
	Fail(c, http.StatusNotImplemented, CodeNotImpl, "not implemented in MVP")
}
