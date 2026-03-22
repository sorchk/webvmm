package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/webvmm/webvmm/internal/auth"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/middleware"
	"github.com/webvmm/webvmm/internal/models"
	"gorm.io/gorm"
)

// UserHandler 用户管理处理器
type UserHandler struct{}

// NewUserHandler 创建用户管理处理器
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// ListUsers 列出所有用户
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")

	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{})

	if search != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	// 清除敏感信息
	for i := range users {
		users[i].PasswordHash = ""
		users[i].TOTPSecret = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUser 获取单个用户
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	user.PasswordHash = ""
	user.TOTPSecret = ""

	c.JSON(http.StatusOK, user)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string      `json:"username" binding:"required,min=3,max=50"`
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password" binding:"required"`
	Role     models.Role `json:"role" binding:"required"`
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 验证角色
	if req.Role != models.RoleAdmin && req.Role != models.RoleAuditor && req.Role != models.RoleUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	// 验证密码强度
	if err := auth.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}

	// 检查邮箱是否已存在
	database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已存在"})
		return
	}

	// 哈希密码
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
		return
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	currentUser, _ := middleware.GetCurrentUser(c)
	h.logAudit(currentUser, middleware.GetClientIP(c), "create_user", "user", strconv.FormatUint(uint64(user.ID), 10), "success", "创建用户: "+user.Username)

	user.PasswordHash = ""
	user.TOTPSecret = ""
	c.JSON(http.StatusCreated, user)
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    string      `json:"email"`
	Role     models.Role `json:"role"`
	IsActive *bool       `json:"is_active"`
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	currentUser, _ := middleware.GetCurrentUser(c)

	// 不允许管理员禁用自己
	if currentUser.ID == user.ID && req.IsActive != nil && !*req.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能禁用自己的账户"})
		return
	}

	// 不允许修改自己的角色
	if currentUser.ID == user.ID && req.Role != "" && req.Role != user.Role {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能修改自己的角色"})
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户失败"})
		return
	}

	h.logAudit(currentUser, middleware.GetClientIP(c), "update_user", "user", strconv.FormatUint(uint64(user.ID), 10), "success", "更新用户: "+user.Username)

	user.PasswordHash = ""
	user.TOTPSecret = ""
	c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	currentUser, _ := middleware.GetCurrentUser(c)

	// 不允许删除自己
	if strconv.FormatUint(uint64(currentUser.ID), 10) == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己的账户"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	h.logAudit(currentUser, middleware.GetClientIP(c), "delete_user", "user", strconv.FormatUint(uint64(user.ID), 10), "success", "删除用户: "+user.Username)

	c.JSON(http.StatusOK, gin.H{"message": "用户已删除"})
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// ResetPassword 重置用户密码
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 验证密码强度
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 哈希密码
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
		return
	}

	user.PasswordHash = passwordHash
	user.FailedLogins = 0
	user.LockedUntil = nil

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存密码失败"})
		return
	}

	currentUser, _ := middleware.GetCurrentUser(c)
	h.logAudit(currentUser, middleware.GetClientIP(c), "reset_password", "user", strconv.FormatUint(uint64(user.ID), 10), "success", "重置用户密码: "+user.Username)

	c.JSON(http.StatusOK, gin.H{"message": "密码已重置"})
}

// UnlockUser 解锁用户
func (h *UserHandler) UnlockUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	user.FailedLogins = 0
	user.LockedUntil = nil
	user.IsActive = true

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解锁用户失败"})
		return
	}

	currentUser, _ := middleware.GetCurrentUser(c)
	h.logAudit(currentUser, middleware.GetClientIP(c), "unlock_user", "user", strconv.FormatUint(uint64(user.ID), 10), "success", "解锁用户: "+user.Username)

	c.JSON(http.StatusOK, gin.H{"message": "用户已解锁"})
}

// logAudit 记录审计日志
func (h *UserHandler) logAudit(user *models.User, ip, action, resourceType, resourceID, status, message string) {
	log := models.AuditLog{
		UserID:       user.ID,
		Username:     user.Username,
		IP:           ip,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       status,
		Message:      message,
		CreatedAt:    time.Now(),
	}
	database.DB.Create(&log)
}
