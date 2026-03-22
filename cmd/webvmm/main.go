package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/webvmm/webvmm/internal/auth"
	"github.com/webvmm/webvmm/internal/config"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/handler"
	"github.com/webvmm/webvmm/internal/libvirt"
	"github.com/webvmm/webvmm/internal/utils"
)

var (
	version   = "1.0.0"
	buildDate = "unknown"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	devMode := flag.Bool("dev", false, "开发模式 (使用 test/config.yaml, HTTP)")
	showVersion := flag.Bool("version", false, "显示版本信息")
	genCert := flag.Bool("gen-cert", false, "生成自签名证书")
	installService := flag.Bool("install-service", false, "安装为 systemd 服务")
	uninstallService := flag.Bool("uninstall-service", false, "卸载 systemd 服务")
	resetAdmin := flag.Bool("reset-admin", false, "重置管理员密码")
	backupNow := flag.Bool("backup-now", false, "手动触发备份")

	flag.Parse()

	// 开发模式使用默认配置路径
	if *devMode && *configPath == "" {
		*configPath = "test/config.yaml"
	}
	if *configPath == "" {
		*configPath = "/etc/webvmm/config.yaml"
	}

	// 显示版本
	if *showVersion {
		fmt.Printf("WebVMM v%s (built: %s)\n", version, buildDate)
		os.Exit(0)
	}

	// CLI 命令处理
	if *genCert {
		if err := generateCert(*configPath); err != nil {
			log.Fatalf("生成证书失败: %v", err)
		}
		fmt.Println("证书生成成功")
		os.Exit(0)
	}

	if *installService {
		if err := installSystemService(); err != nil {
			log.Fatalf("安装服务失败: %v", err)
		}
		fmt.Println("服务安装成功")
		os.Exit(0)
	}

	if *uninstallService {
		if err := uninstallSystemService(); err != nil {
			log.Fatalf("卸载服务失败: %v", err)
		}
		fmt.Println("服务卸载成功")
		os.Exit(0)
	}

	if *resetAdmin {
		if err := resetAdminPassword(*configPath); err != nil {
			log.Fatalf("重置管理员密码失败: %v", err)
		}
		fmt.Println("管理员密码已重置")
		os.Exit(0)
	}

	if *backupNow {
		if err := triggerBackup(*configPath); err != nil {
			log.Fatalf("备份失败: %v", err)
		}
		fmt.Println("备份完成")
		os.Exit(0)
	}

	// 启动服务器
	if err := startServer(*configPath); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}

func generateCert(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	return utils.GenerateSelfSignedCert(cfg.Server.CertFile, cfg.Server.KeyFile)
}

func installSystemService() error {
	// 创建 systemd 服务文件
	serviceContent := `[Unit]
Description=WebVMM - KVM Virtual Machine Manager
After=network.target libvirtd.service

[Service]
Type=simple
ExecStart=/usr/local/bin/webvmm
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`

	// 写入服务文件
	if err := os.WriteFile("/etc/systemd/system/webvmm.service", []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("写入服务文件失败: %w", err)
	}

	// 创建必要目录
	os.MkdirAll("/etc/webvmm", 0755)
	os.MkdirAll("/var/lib/webvmm", 0755)
	os.MkdirAll("/var/log/webvmm", 0755)

	// 生成默认配置
	cfg := config.DefaultConfig()
	cfg.Save("/etc/webvmm/config.yaml")

	// 生成证书
	utils.GenerateSelfSignedCert(cfg.Server.CertFile, cfg.Server.KeyFile)

	return nil
}

func uninstallSystemService() error {
	if err := os.Remove("/etc/systemd/system/webvmm.service"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除服务文件失败: %w", err)
	}
	return nil
}

func resetAdminPassword(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化数据库
	if err := database.Init(cfg.Database.Path, cfg.Log.Level); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer database.Close()

	// 检查是否有用户
	if !database.IsInitialized() {
		return fmt.Errorf("系统未初始化，请先启动服务完成安装向导")
	}

	// 获取管理员用户名
	adminUsername, _ := database.GetSetting("admin_username")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	// 生成新密码
	newPassword, err := auth.GenerateRandomString(12)
	if err != nil {
		return fmt.Errorf("生成密码失败: %w", err)
	}

	// 哈希密码
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("哈希密码失败: %w", err)
	}

	// 更新密码
	result := database.DB.Model(&struct {
		Username string
	}{}).Table("users").Where("username = ?", adminUsername).Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("更新密码失败: %w", result.Error)
	}

	fmt.Printf("管理员 '%s' 的新密码: %s\n", adminUsername, newPassword)
	fmt.Println("请登录后立即修改密码")

	return nil
}

func triggerBackup(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化数据库
	if err := database.Init(cfg.Database.Path, cfg.Log.Level); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer database.Close()

	// TODO: 实现备份逻辑
	fmt.Println("备份功能将在后续版本实现")

	return nil
}

func startServer(configPath string) error {
	log.Printf("正在从 %s 加载配置...", configPath)

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	log.Printf("配置加载成功: Port=%d, DevMode=%v", cfg.Server.Port, cfg.Server.DevMode)

	// 确保数据目录存在
	if err := cfg.EnsureDataDirs(); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 初始化数据库
	if err := database.Init(cfg.Database.Path, cfg.Log.Level); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer database.Close()

	// 生成 JWT 密钥（如果未配置）
	if cfg.Auth.JWTSecret == "" {
		secret, err := auth.GenerateRandomString(32)
		if err != nil {
			return fmt.Errorf("生成 JWT 密钥失败: %w", err)
		}
		cfg.Auth.JWTSecret = secret
	}

	// 创建 JWT 服务
	jwtService := auth.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.TokenExpiry, cfg.Auth.RefreshExpiry)

	// 初始化 Libvirt 客户端
	libvirtClient, err := libvirt.GetClient()
	if err != nil {
		log.Printf("警告: 连接 libvirt 失败: %v (虚拟机功能将不可用)", err)
	}

	// 设置路由
	router := handler.SetupRouter(jwtService, libvirtClient)

	// 创建服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	// 开发模式使用 HTTP
	if cfg.Server.DevMode {
		log.Printf("WebVMM 服务器启动（开发模式），监听 http://%s", addr)

		// 优雅关闭
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan
			log.Println("正在关闭服务器...")
			database.Close()
			os.Exit(0)
		}()

		return router.Run(addr)
	}

	// 生产模式使用 HTTPS
	// 检查并生成证书
	if cfg.Server.AutoGenCert {
		if _, err := os.Stat(cfg.Server.CertFile); os.IsNotExist(err) {
			log.Println("证书文件不存在，正在生成自签名证书...")
			if err := utils.GenerateSelfSignedCert(cfg.Server.CertFile, cfg.Server.KeyFile); err != nil {
				return fmt.Errorf("生成证书失败: %w", err)
			}
		}
	}

	// 加载证书
	cert, err := tls.LoadX509KeyPair(cfg.Server.CertFile, cfg.Server.KeyFile)
	if err != nil {
		return fmt.Errorf("加载证书失败: %w", err)
	}

	// 配置 TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("创建监听器失败: %w", err)
	}

	log.Printf("WebVMM 服务器启动，监听 https://%s", addr)

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("正在关闭服务器...")
		database.Close()
		os.Exit(0)
	}()

	// 启动服务器
	return router.RunListener(listener)
}
