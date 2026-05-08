package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type adminUsernameRequest struct {
	Username string `json:"username"`
}

func ProvisionUserToken(c *gin.Context) {
	var req adminUsernameRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	result, err := service.ProvisionUserToken(req.Username)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, result)
}

func QueryUserKeys(c *gin.Context) {
	var req adminUsernameRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	result, err := service.QueryUserTokensByUsername(req.Username)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, result)
}
