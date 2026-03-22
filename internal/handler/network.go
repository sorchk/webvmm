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

// NetworkHandler 网络处理器
type NetworkHandler struct{}

// NewNetworkHandler 创建网络处理器
func NewNetworkHandler() *NetworkHandler {
	return &NetworkHandler{}
}

// ListNetworks 列出所有虚拟网络
func (h *NetworkHandler) ListNetworks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var networks []models.VirtualNetwork
	var total int64

	query := database.DB.Model(&models.VirtualNetwork{})
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&networks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询虚拟网络失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"networks":  networks,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetNetwork 获取单个虚拟网络
func (h *NetworkHandler) GetNetwork(c *gin.Context) {
	id := c.Param("id")

	var network models.VirtualNetwork
	if err := database.DB.First(&network, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "虚拟网络不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询虚拟网络失败"})
		return
	}

	c.JSON(http.StatusOK, network)
}

// CreateNetworkRequest 创建虚拟网络请求
type CreateNetworkRequest struct {
	Name      string `json:"name" binding:"required"`
	Mode      string `json:"mode" binding:"required"` // nat, bridge, isolated
	Bridge    string `json:"bridge"`
	Subnet    string `json:"subnet"`
	DHCPStart string `json:"dhcp_start"`
	DHCPEnd   string `json:"dhcp_end"`
	AutoStart bool   `json:"auto_start"`
}

// CreateNetwork 创建虚拟网络
func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	// 检查名称是否已存在
	var count int64
	database.DB.Model(&models.VirtualNetwork{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "虚拟网络名称已存在"})
		return
	}

	// 验证模式
	if req.Mode != "nat" && req.Mode != "bridge" && req.Mode != "isolated" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的网络模式"})
		return
	}

	network := models.VirtualNetwork{
		Name:      req.Name,
		Mode:      req.Mode,
		Bridge:    req.Bridge,
		Subnet:    req.Subnet,
		DHCPStart: req.DHCPStart,
		DHCPEnd:   req.DHCPEnd,
		AutoStart: req.AutoStart,
		IsActive:  false,
	}

	if err := database.DB.Create(&network).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建虚拟网络失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "create_network", "virtual_network", strconv.FormatUint(uint64(network.ID), 10), "success", "创建虚拟网络: "+network.Name)

	c.JSON(http.StatusCreated, network)
}

// UpdateNetworkRequest 更新虚拟网络请求
type UpdateNetworkRequest struct {
	DHCPStart string `json:"dhcp_start"`
	DHCPEnd   string `json:"dhcp_end"`
	AutoStart *bool  `json:"auto_start"`
}

// UpdateNetwork 更新虚拟网络
func (h *NetworkHandler) UpdateNetwork(c *gin.Context) {
	id := c.Param("id")

	var network models.VirtualNetwork
	if err := database.DB.First(&network, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "虚拟网络不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询虚拟网络失败"})
		return
	}

	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if req.DHCPStart != "" {
		network.DHCPStart = req.DHCPStart
	}
	if req.DHCPEnd != "" {
		network.DHCPEnd = req.DHCPEnd
	}
	if req.AutoStart != nil {
		network.AutoStart = *req.AutoStart
	}

	if err := database.DB.Save(&network).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新虚拟网络失败"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)
	h.logAudit(user, middleware.GetClientIP(c), "update_network", "virtual_network", strconv.FormatUint(uint64(network.ID), 10), "success", "更新虚拟网络: "+network.Name)

	c.JSON(http.StatusOK, network)
}

// DeleteNetwork 删除虚拟网络
func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var network models.VirtualNetwork
	if err := database.DB.First(&network, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "虚拟网络不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询虚拟网络失败"})
		return
	}

	if err := database.DB.Delete(&network).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除虚拟网络失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "delete_network", "virtual_network", strconv.FormatUint(uint64(network.ID), 10), "success", "删除虚拟网络: "+network.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟网络已删除"})
}

// logAudit 记录审计日志
func (h *NetworkHandler) logAudit(user *models.User, ip, action, resourceType, resourceID, status, message string) {
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
