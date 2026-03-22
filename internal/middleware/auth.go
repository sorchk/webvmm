package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/webvmm/webvmm/internal/auth"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/models"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}

		tokenString := auth.ParseBearerToken(authHeader)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证格式"})
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// 验证用户是否仍然有效
		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		if !user.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "账户已被禁用"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("user", &user)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求认证）
func OptionalAuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := auth.ParseBearerToken(authHeader)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err == nil && user.IsActive {
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("email", claims.Email)
			c.Set("role", claims.Role)
			c.Set("user", &user)
		}

		c.Next()
	}
}

// RequireRoles 要求特定角色的中间件
func RequireRoles(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}

		role := userRole.(models.Role)
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		c.Abort()
	}
}

// RequireAdmin 要求管理员角色
func RequireAdmin() gin.HandlerFunc {
	return RequireRoles(models.RoleAdmin)
}

// RequireAdminOrAuditor 要求管理员或审计员角色
func RequireAdminOrAuditor() gin.HandlerFunc {
	return RequireRoles(models.RoleAdmin, models.RoleAuditor)
}

// RequireOwnerOrAdmin 要求资源所有者或管理员
func RequireOwnerOrAdmin(resourceOwnerID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, _ := c.Get("role")
		userID, _ := c.Get("userID")

		if userRole == models.RoleAdmin {
			c.Next()
			return
		}

		if userID.(uint) == resourceOwnerID {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此资源"})
		c.Abort()
	}
}

// ReadOnlyMiddleware 只读模式中间件（用于审计员）
func ReadOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			userRole, _ := c.Get("role")
			if userRole == models.RoleAuditor {
				c.JSON(http.StatusForbidden, gin.H{"error": "审计员只有只读权限"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// CORSMiddleware CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware 安全头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// GetCurrentUser 从上下文获取当前用户
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	return user.(*models.User), true
}

// GetCurrentUserID 从上下文获取当前用户ID
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetClientIP 获取客户端 IP
func GetClientIP(c *gin.Context) string {
	// 检查 X-Forwarded-For 头
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 检查 X-Real-IP 头
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	return c.ClientIP()
}
