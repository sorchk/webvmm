package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/webvmm/webvmm/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init(dbPath string, logLevel string) error {
	// 确保数据库目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 配置日志级别
	var gormLogLevel logger.LogLevel
	switch logLevel {
	case "debug":
		gormLogLevel = logger.Info
	case "info":
		gormLogLevel = logger.Warn
	case "warn":
		gormLogLevel = logger.Error
	case "error":
		gormLogLevel = logger.Error
	default:
		gormLogLevel = logger.Warn
	}

	// 打开数据库连接
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 启用 WAL 模式以提高并发性能
	if err := DB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return fmt.Errorf("启用 WAL 模式失败: %w", err)
	}

	// 自动迁移
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

// autoMigrate 自动迁移数据库表结构
func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.VM{},
		&models.VMDisk{},
		&models.VMNIC{},
		&models.Snapshot{},
		&models.StoragePool{},
		&models.StorageVolume{},
		&models.VirtualNetwork{},
		&models.AuditLog{},
		&models.SystemSetting{},
		&models.Backup{},
		&models.Task{},
	)
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// IsInitialized 检查系统是否已初始化
func IsInitialized() bool {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	return count > 0
}

// GetSetting 获取系统设置
func GetSetting(key string) (string, error) {
	var setting models.SystemSetting
	result := DB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", result.Error
	}
	return setting.Value, nil
}

// SetSetting 设置系统设置
func SetSetting(key, value string) error {
	setting := models.SystemSetting{
		Key: key,
	}
	result := DB.Where("key = ?", key).First(&setting)
	if result.Error == gorm.ErrRecordNotFound {
		setting.Value = value
		return DB.Create(&setting).Error
	}
	setting.Value = value
	return DB.Save(&setting).Error
}