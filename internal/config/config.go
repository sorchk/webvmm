package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Backup   BackupConfig   `yaml:"backup"`
	Log      LogConfig      `yaml:"log"`
	Auth     AuthConfig     `yaml:"auth"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        int    `yaml:"port"`
	CertFile    string `yaml:"cert_file"`
	KeyFile     string `yaml:"key_file"`
	AutoGenCert bool   `yaml:"auto_gen_cert"`
	DevMode     bool   `yaml:"dev_mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// BackupConfig 备份配置
type BackupConfig struct {
	Enabled       bool         `yaml:"enabled"`
	Schedule      string       `yaml:"schedule"`
	RetentionDays int          `yaml:"retention_days"`
	WebDAV        WebDAVConfig `yaml:"webdav"`
}

// WebDAVConfig WebDAV 配置
type WebDAVConfig struct {
	URL        string `yaml:"url"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	RemotePath string `yaml:"remote_path"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret     string `yaml:"jwt_secret"`
	TokenExpiry   int    `yaml:"token_expiry"`   // 小时
	RefreshExpiry int    `yaml:"refresh_expiry"` // 小时
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        8443,
			CertFile:    "/etc/webvmm/cert.pem",
			KeyFile:     "/etc/webvmm/key.pem",
			AutoGenCert: true,
		},
		Database: DatabaseConfig{
			Path: "/var/lib/webvmm/data.db",
		},
		Backup: BackupConfig{
			Enabled:       true,
			Schedule:      "0 2 * * *",
			RetentionDays: 7,
			WebDAV: WebDAVConfig{
				RemotePath: "/webvmm_backups",
			},
		},
		Log: LogConfig{
			Level: "info",
			Path:  "/var/log/webvmm/app.log",
		},
		Auth: AuthConfig{
			JWTSecret:     "",
			TokenExpiry:   24,
			RefreshExpiry: 168, // 7天
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，返回默认配置
			return cfg, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// EnsureDataDirs 确保数据目录存在
func (c *Config) EnsureDataDirs() error {
	dirs := []string{
		filepath.Dir(c.Database.Path),
		filepath.Dir(c.Log.Path),
		filepath.Dir(c.Server.CertFile),
	}

	for _, dir := range dirs {
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
			}
		}
	}

	return nil
}
