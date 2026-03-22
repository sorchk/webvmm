package handler

import (
	"io/fs"

	"github.com/gin-gonic/gin"
	"github.com/webvmm/webvmm/internal/auth"
	"github.com/webvmm/webvmm/internal/libvirt"
	"github.com/webvmm/webvmm/internal/middleware"
	"github.com/webvmm/webvmm/internal/static"
)

// SetupRouter 设置路由
func SetupRouter(jwtService *auth.JWTService, libvirtClient *libvirt.Client) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())

	// 创建处理器
	authHandler := NewAuthHandler(jwtService)
	setupHandler := NewSetupHandler(jwtService)
	userHandler := NewUserHandler()
	vmHandler := NewVMHandler(libvirtClient)
	storageHandler := NewStorageHandler()
	networkHandler := NewNetworkHandler()

	// 静态文件服务
	distFS, _ := fs.Sub(static.DistFS, "dist")

	// 提供 assets 目录 - 使用自定义 handler
	router.GET("/assets/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// 去掉开头的斜杠
		if len(filepath) > 0 && filepath[0] == '/' {
			filepath = filepath[1:]
		}
		// 从嵌入的文件系统读取
		data, err := fs.ReadFile(distFS, "assets/"+filepath)
		if err != nil {
			// 尝试直接从 DistFS 读取
			data, err = fs.ReadFile(static.DistFS, "dist/assets/"+filepath)
			if err != nil {
				c.Status(404)
				return
			}
		}
		// 根据文件扩展名设置 Content-Type
		contentType := "application/javascript"
		if len(filepath) > 4 {
			ext := filepath[len(filepath)-4:]
			switch ext {
			case ".css":
				contentType = "text/css"
			case ".svg", ".png", ".jpg":
				contentType = "image/" + ext[1:]
			}
		}
		c.Data(200, contentType, data)
	})

	// SPA fallback - 所有非 API 路由都返回 index.html
	spaHandler := func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.String(500, "读取页面失败")
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	}

	router.GET("/", spaHandler)
	router.GET("/login", spaHandler)
	router.GET("/setup", spaHandler)
	router.GET("/vms", spaHandler)
	router.GET("/vms/*id", spaHandler)
	router.GET("/storage", spaHandler)
	router.GET("/networks", spaHandler)
	router.GET("/users", spaHandler)
	router.GET("/profile", spaHandler)
	router.GET("/logs", spaHandler)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 公开路由
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// 安装向导
		v1.GET("/setup", setupHandler.CheckSetup)
		v1.POST("/setup", setupHandler.Setup)

		// 认证路由
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
		}

		// 需要认证的路由
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(jwtService))
		protected.Use(middleware.ReadOnlyMiddleware())
		{
			// 用户个人操作
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile/password", authHandler.ChangePassword)

			// TOTP 管理
			totpGroup := protected.Group("/totp")
			{
				totpGroup.POST("/setup", authHandler.SetupTOTP)
				totpGroup.POST("/enable", authHandler.EnableTOTP)
				totpGroup.POST("/disable", authHandler.DisableTOTP)
			}

			// 用户管理 (仅管理员)
			users := protected.Group("/users")
			users.Use(middleware.RequireAdmin())
			{
				users.GET("", userHandler.ListUsers)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.POST("/:id/reset-password", userHandler.ResetPassword)
				users.POST("/:id/unlock", userHandler.UnlockUser)
			}

			// 虚拟机管理
			vms := protected.Group("/vms")
			{
				vms.GET("/sync", vmHandler.SyncVMs)
				vms.GET("", vmHandler.ListVMs)
				vms.POST("", vmHandler.CreateVM)
				vms.GET("/:id", vmHandler.GetVM)
				vms.POST("/:id/start", vmHandler.StartVM)
				vms.POST("/:id/stop", vmHandler.StopVM)
				vms.POST("/:id/force-stop", vmHandler.ForceStopVM)
				vms.POST("/:id/reboot", vmHandler.RebootVM)
				vms.POST("/:id/suspend", vmHandler.SuspendVM)
				vms.POST("/:id/resume", vmHandler.ResumeVM)
				vms.DELETE("/:id", vmHandler.DeleteVM)
				vms.GET("/:id/console", vmHandler.GetVNCConsole)
			}

			// 存储池管理
			storage := protected.Group("/storage")
			{
				storage.GET("/pools", storageHandler.ListStoragePools)
				storage.POST("/pools", storageHandler.CreateStoragePool)
				storage.GET("/pools/:id", storageHandler.GetStoragePool)
				storage.DELETE("/pools/:id", storageHandler.DeleteStoragePool)
				storage.GET("/pools/:id/volumes", storageHandler.ListVolumes)
				storage.POST("/pools/:id/volumes", storageHandler.CreateVolume)
				storage.DELETE("/pools/:id/volumes/:volume_id", storageHandler.DeleteVolume)
			}

			// 网络管理
			networks := protected.Group("/networks")
			{
				networks.GET("", networkHandler.ListNetworks)
				networks.POST("", networkHandler.CreateNetwork)
				networks.GET("/:id", networkHandler.GetNetwork)
				networks.PUT("/:id", networkHandler.UpdateNetwork)
				networks.DELETE("/:id", networkHandler.DeleteNetwork)
			}
		}
	}

	return router
}
