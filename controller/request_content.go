package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetRequestContent(c *gin.Context) {
	// The admin route is separately protected by AdminAuth. Self routes always enforce ownership.
	admin := c.FullPath() == "/api/log/content/:request_id"
	entry, err := model.GetRequestContent(c.Request.Context(), c.Param("request_id"), c.GetInt("id"), admin)
	c.Header("Cache-Control", "no-store")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiSuccess(c, nil)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Unable to load request content"})
		return
	}
	common.ApiSuccess(c, entry)
}

func RequestContentAccess(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !canManageTargetRole(c.GetInt("role"), user.Role) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if c.Request.Method == http.MethodPut {
		var request struct {
			Enabled *bool `json:"enabled"`
		}
		if common.DecodeJson(c.Request.Body, &request) != nil || request.Enabled == nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if err := model.SetRequestContentAccess(c.Request.Context(), userId, *request.Enabled); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	enabled, err := model.CanViewRequestContent(c.Request.Context(), userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"enabled": enabled})
}
