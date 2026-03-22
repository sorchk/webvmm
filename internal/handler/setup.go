package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/webvmm/webvmm/internal/auth"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/middleware"
	"github.com/webvmm/webvmm/internal/models"
)

// SetupHandler 安装向导处理器
type SetupHandler struct {
	jwtService *auth.JWTService
}

// NewSetupHandler 创建安装向导处理器
func NewSetupHandler(jwtService *auth.JWTService) *SetupHandler {
	return &SetupHandler{jwtService: jwtService}
}

// SetupRequest 安装请求
type SetupRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// SetupResponse 安装响应
type SetupResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Message      string `json:"message"`
}

// CheckSetup 检查系统是否已初始化
func (h *SetupHandler) CheckSetup(c *gin.Context) {
	initialized := database.IsInitialized()
	c.JSON(http.StatusOK, gin.H{
		"initialized": initialized,
	})
}

// Setup 执行首次安装
func (h *SetupHandler) Setup(c *gin.Context) {
	// 检查是否已初始化
	if database.IsInitialized() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "系统已初始化"})
		return
	}

	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	// 验证密码强度
	if err := auth.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 哈希密码
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
		return
	}

	// 创建管理员用户
	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         models.RoleAdmin,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	// 设置系统已初始化标记
	database.SetSetting("system_initialized", "true")
	database.SetSetting("admin_username", req.Username)

	// 生成令牌
	accessToken, err := h.jwtService.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成刷新令牌失败"})
		return
	}

	// 记录审计日志
	h.logAudit(&user, middleware.GetClientIP(c), "setup", "system", "", "success", "系统初始化完成")

	c.JSON(http.StatusOK, SetupResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
		Message:      "系统初始化成功",
	})
}

// logAudit 记录审计日志
func (h *SetupHandler) logAudit(user *models.User, ip, action, resourceType, resourceID, status, message string) {
	log := models.AuditLog{
		UserID:       user.ID,
		Username:     user.Username,
		IP:           ip,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       status,
		Message:      message,
	}
	database.DB.Create(&log)
}
