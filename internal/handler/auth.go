package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/webvmm/webvmm/internal/auth"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/middleware"
	"github.com/webvmm/webvmm/internal/models"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	jwtService *auth.JWTService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{jwtService: jwtService}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RequiresTOTP bool   `json:"requires_totp"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 查找用户
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 检查账户状态
	if err := auth.CheckUserLock(&user); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		auth.HandleFailedLogin(&user, 5, 15*time.Minute)
		database.DB.Save(&user)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 检查 TOTP
	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			c.JSON(http.StatusOK, LoginResponse{RequiresTOTP: true})
			return
		}
		if !auth.ValidateTOTP(req.TOTPCode, user.TOTPSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "双因素认证码无效"})
			return
		}
	}

	// 重置失败计数
	auth.ResetFailedLogins(&user)
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = middleware.GetClientIP(c)
	database.DB.Save(&user)

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
	h.logAudit(&user, middleware.GetClientIP(c), "login", "user", "", "success", "用户登录成功")

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
		RequiresTOTP: false,
	})
}

// RefreshToken 刷新令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	accessToken, err := h.jwtService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   86400,
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	user, _ := middleware.GetCurrentUser(c)
	h.logAudit(user, middleware.GetClientIP(c), "logout", "user", "", "success", "用户登出")
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// GetProfile 获取当前用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user, _ := middleware.GetCurrentUser(c)
	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"role":          user.Role,
		"totp_enabled":  user.TOTPEnabled,
		"last_login_at": user.LastLoginAt,
		"created_at":    user.CreatedAt,
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	// 验证旧密码
	if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}

	// 验证新密码强度
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新密码
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码哈希失败"})
		return
	}

	user.PasswordHash = hash
	if err := database.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存密码失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "change_password", "user", "", "success", "修改密码成功")
	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// SetupTOTPResponse TOTP 设置响应
type SetupTOTPResponse struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
}

// SetupTOTP 设置 TOTP
func (h *AuthHandler) SetupTOTP(c *gin.Context) {
	user, _ := middleware.GetCurrentUser(c)

	if user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已启用双因素认证"})
		return
	}

	secret, qrURL, err := auth.GenerateTOTPSecret(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成密钥失败"})
		return
	}

	// 临时保存密钥（未验证）
	user.TOTPSecret = secret
	database.DB.Save(user)

	c.JSON(http.StatusOK, SetupTOTPResponse{
		Secret: secret,
		QRCode: qrURL,
	})
}

// EnableTOTP 启用 TOTP
func (h *AuthHandler) EnableTOTP(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先设置双因素认证"})
		return
	}

	if !auth.ValidateTOTP(req.Code, user.TOTPSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	user.TOTPEnabled = true
	database.DB.Save(user)

	h.logAudit(user, middleware.GetClientIP(c), "enable_totp", "user", "", "success", "启用双因素认证")
	c.JSON(http.StatusOK, gin.H{"message": "双因素认证已启用"})
}

// DisableTOTP 禁用 TOTP
func (h *AuthHandler) DisableTOTP(c *gin.Context) {
	var req struct {
		Code     string `json:"code" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未启用双因素认证"})
		return
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码错误"})
		return
	}

	// 验证 TOTP
	if !auth.ValidateTOTP(req.Code, user.TOTPSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	user.TOTPEnabled = false
	user.TOTPSecret = ""
	database.DB.Save(user)

	h.logAudit(user, middleware.GetClientIP(c), "disable_totp", "user", "", "success", "禁用双因素认证")
	c.JSON(http.StatusOK, gin.H{"message": "双因素认证已禁用"})
}

// logAudit 记录审计日志
func (h *AuthHandler) logAudit(user *models.User, ip, action, resourceType, resourceID, status, message string) {
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
