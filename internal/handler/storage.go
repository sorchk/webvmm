package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/middleware"
	"github.com/webvmm/webvmm/internal/models"
	"gorm.io/gorm"
)

// StorageHandler 存储处理器
type StorageHandler struct{}

// NewStorageHandler 创建存储处理器
func NewStorageHandler() *StorageHandler {
	return &StorageHandler{}
}

// ListStoragePools 列出所有存储池
func (h *StorageHandler) ListStoragePools(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var pools []models.StoragePool
	var total int64

	query := database.DB.Model(&models.StoragePool{})
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pools).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询存储池失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pools":     pools,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetStoragePool 获取单个存储池
func (h *StorageHandler) GetStoragePool(c *gin.Context) {
	id := c.Param("id")

	var pool models.StoragePool
	if err := database.DB.First(&pool, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "存储池不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询存储池失败"})
		return
	}

	// 获取存储卷
	var volumes []models.StorageVolume
	database.DB.Where("pool_id = ?", pool.ID).Find(&volumes)

	c.JSON(http.StatusOK, gin.H{
		"pool":    pool,
		"volumes": volumes,
	})
}

// CreateStoragePoolRequest 创建存储池请求
type CreateStoragePoolRequest struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"` // dir, fs, logical
	Path      string `json:"path"`
	AutoStart bool   `json:"auto_start"`
}

// CreateStoragePool 创建存储池
func (h *StorageHandler) CreateStoragePool(c *gin.Context) {
	var req CreateStoragePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	// 检查名称是否已存在
	var count int64
	database.DB.Model(&models.StoragePool{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "存储池名称已存在"})
		return
	}

	pool := models.StoragePool{
		Name:      req.Name,
		Type:      req.Type,
		Path:      req.Path,
		AutoStart: req.AutoStart,
		IsActive:  false,
	}

	if err := database.DB.Create(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建存储池失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "create_pool", "storage_pool", strconv.FormatUint(uint64(pool.ID), 10), "success", "创建存储池: "+pool.Name)

	c.JSON(http.StatusCreated, pool)
}

// DeleteStoragePool 删除存储池
func (h *StorageHandler) DeleteStoragePool(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var pool models.StoragePool
	if err := database.DB.First(&pool, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "存储池不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询存储池失败"})
		return
	}

	// 检查是否有存储卷
	var volumeCount int64
	database.DB.Model(&models.StorageVolume{}).Where("pool_id = ?", pool.ID).Count(&volumeCount)
	if volumeCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "存储池中还有存储卷，请先删除"})
		return
	}

	if err := database.DB.Delete(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除存储池失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "delete_pool", "storage_pool", strconv.FormatUint(uint64(pool.ID), 10), "success", "删除存储池: "+pool.Name)

	c.JSON(http.StatusOK, gin.H{"message": "存储池已删除"})
}

// ListVolumes 列出存储卷
func (h *StorageHandler) ListVolumes(c *gin.Context) {
	poolID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var volumes []models.StorageVolume
	var total int64

	query := database.DB.Model(&models.StorageVolume{}).Where("pool_id = ?", poolID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&volumes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询存储卷失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"volumes":   volumes,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateVolumeRequest 创建存储卷请求
type CreateVolumeRequest struct {
	Name   string `json:"name" binding:"required"`
	Format string `json:"format"`                                 // qcow2, raw
	Size   int64  `json:"size" binding:"required,min=1073741824"` // 最小 1GB
}

// CreateVolume 创建存储卷
func (h *StorageHandler) CreateVolume(c *gin.Context) {
	poolID := c.Param("id")

	var req CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	// 检查存储池是否存在
	var pool models.StoragePool
	if err := database.DB.First(&pool, poolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "存储池不存在"})
		return
	}

	// 设置默认格式
	if req.Format == "" {
		req.Format = "qcow2"
	}

	volume := models.StorageVolume{
		PoolID:     pool.ID,
		Name:       req.Name,
		Path:       pool.Path + "/" + req.Name + "." + req.Format,
		Format:     req.Format,
		Capacity:   req.Size,
		Allocation: 0,
	}

	if err := database.DB.Create(&volume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建存储卷失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "create_volume", "storage_volume", strconv.FormatUint(uint64(volume.ID), 10), "success", "创建存储卷: "+volume.Name)

	c.JSON(http.StatusCreated, volume)
}

// DeleteVolume 删除存储卷
func (h *StorageHandler) DeleteVolume(c *gin.Context) {
	poolID := c.Param("id")
	volumeID := c.Param("volume_id")
	user, _ := middleware.GetCurrentUser(c)

	var volume models.StorageVolume
	if err := database.DB.Where("id = ? AND pool_id = ?", volumeID, poolID).First(&volume).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "存储卷不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询存储卷失败"})
		return
	}

	if err := database.DB.Delete(&volume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除存储卷失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "delete_volume", "storage_volume", strconv.FormatUint(uint64(volume.ID), 10), "success", "删除存储卷: "+volume.Name)

	c.JSON(http.StatusOK, gin.H{"message": "存储卷已删除"})
}

// logAudit 记录审计日志
func (h *StorageHandler) logAudit(user *models.User, ip, action, resourceType, resourceID, status, message string) {
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
