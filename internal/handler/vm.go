package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/webvmm/webvmm/internal/database"
	"github.com/webvmm/webvmm/internal/libvirt"
	"github.com/webvmm/webvmm/internal/middleware"
	"github.com/webvmm/webvmm/internal/models"
	"gorm.io/gorm"
)

// VMHandler 虚拟机处理器
type VMHandler struct {
	libvirtClient *libvirt.Client
}

// NewVMHandler 创建虚拟机处理器
func NewVMHandler(libvirtClient *libvirt.Client) *VMHandler {
	return &VMHandler{libvirtClient: libvirtClient}
}

// ListVMs 列出所有虚拟机
func (h *VMHandler) ListVMs(c *gin.Context) {
	user, _ := middleware.GetCurrentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	status := c.Query("status")

	var vms []models.VM
	var total int64

	query := database.DB.Model(&models.VM{}).Preload("Owner")

	// 非管理员只能看到自己的虚拟机
	if user.Role != models.RoleAdmin {
		query = query.Where("owner_id = ?", user.ID)
	}

	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&vms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询虚拟机失败"})
		return
	}

	// 同步 libvirt 状态
	for i := range vms {
		if info, err := h.libvirtClient.GetDomainInfo(vms[i].UUID); err == nil {
			vms[i].Status = info.Status
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"vms":       vms,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetVM 获取单个虚拟机详情
func (h *VMHandler) GetVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.Preload("Owner").First(&vm, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询虚拟机失败"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此虚拟机"})
		return
	}

	// 获取 libvirt 状态
	if info, err := h.libvirtClient.GetDomainInfo(vm.UUID); err == nil {
		vm.Status = info.Status
	}

	// 获取 XML 配置
	xmlDef, _ := h.libvirtClient.GetDomainXML(vm.UUID)

	c.JSON(http.StatusOK, gin.H{
		"vm":      vm,
		"xml_def": xmlDef,
	})
}

// CreateVMRequest 创建虚拟机请求
type CreateVMRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	VCPU        int    `json:"vcpu" binding:"required,min=1,max=64"`
	Memory      int64  `json:"memory" binding:"required,min=524288"`        // 最小 512MB
	DiskSize    int64  `json:"disk_size" binding:"required,min=1073741824"` // 最小 1GB
	DiskBus     string `json:"disk_bus"`
	Network     string `json:"network"`
	ISOPath     string `json:"iso_path"`
	AutoStart   bool   `json:"auto_start"`
}

// CreateVM 创建虚拟机
func (h *VMHandler) CreateVM(c *gin.Context) {
	var req CreateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	// 生成 UUID
	vmUUID := uuid.New().String()

	// 设置默认值
	if req.DiskBus == "" {
		req.DiskBus = "virtio"
	}
	if req.Network == "" {
		req.Network = "default"
	}

	// 生成 Domain XML
	xmlStr := h.generateDomainXML(req, vmUUID)

	// 在 libvirt 中定义虚拟机
	if err := h.libvirtClient.DefineDomain(xmlStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建虚拟机失败: " + err.Error()})
		return
	}

	// 保存到数据库
	vm := models.VM{
		UUID:        vmUUID,
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     user.ID,
		Status:      "stopped",
		VCPU:        req.VCPU,
		Memory:      req.Memory,
		AutoStart:   req.AutoStart,
		XMLDef:      xmlStr,
	}

	if err := database.DB.Create(&vm).Error; err != nil {
		// 回滚 libvirt 定义
		h.libvirtClient.UndefineDomain(vmUUID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存虚拟机失败"})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "create_vm", "vm", vm.UUID, "success", "创建虚拟机: "+vm.Name)

	c.JSON(http.StatusCreated, vm)
}

// generateDomainXML 生成 Domain XML
func (h *VMHandler) generateDomainXML(req CreateVMRequest, vmUUID string) string {
	memoryMB := req.Memory / 1024 / 1024

	return fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <currentMemory unit='MiB'>%d</currentMemory>
  <vcpu placement='static'>%d</vcpu>
  <os>
    <type arch='x86_64' machine='pc'>hvm</type>
    <boot dev='hd'/>
    <boot dev='cdrom'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <cpu mode='host-passthrough'/>
  <clock offset='utc'>
    <timer name='rtc' tickpolicy='catchup'/>
    <timer name='pit' tickpolicy='delay'/>
    <timer name='hpet' present='no'/>
  </clock>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <pm>
    <suspend-to-mem enabled='no'/>
    <suspend-to-disk enabled='no'/>
  </pm>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/%s.qcow2'/>
      <target dev='vda' bus='%s'/>
    </disk>
    %s
    <controller type='usb' index='0'/>
    <controller type='pci' index='0' model='pci-root'/>
    <interface type='network'>
      <mac address='52:54:00:00:00:01'/>
      <source network='%s'/>
      <model type='virtio'/>
    </interface>
    <graphics type='vnc' port='-1' autoport='yes' listen='127.0.0.1'>
      <listen type='address' address='127.0.0.1'/>
    </graphics>
    <video>
      <model type='qxl' ram='65536' vram='65536' vgamem='16384' heads='1' primary='yes'/>
    </video>
    <memballoon model='virtio'/>
  </devices>
</domain>`,
		req.Name,
		vmUUID,
		memoryMB,
		memoryMB,
		req.VCPU,
		vmUUID,
		req.DiskBus,
		h.generateISODiskXML(req.ISOPath),
		req.Network,
	)
}

// generateISODiskXML 生成 ISO 磁盘 XML
func (h *VMHandler) generateISODiskXML(isoPath string) string {
	if isoPath == "" {
		return ""
	}
	return fmt.Sprintf(`<disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='hda' bus='ide'/>
      <readonly/>
    </disk>`, isoPath)
}

// StartVM 启动虚拟机
func (h *VMHandler) StartVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	if err := h.libvirtClient.StartDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动虚拟机失败: " + err.Error()})
		return
	}

	vm.Status = "running"
	database.DB.Save(&vm)

	h.logAudit(user, middleware.GetClientIP(c), "start_vm", "vm", vm.UUID, "success", "启动虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已启动"})
}

// StopVM 停止虚拟机
func (h *VMHandler) StopVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	if err := h.libvirtClient.StopDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "停止虚拟机失败: " + err.Error()})
		return
	}

	vm.Status = "stopped"
	database.DB.Save(&vm)

	h.logAudit(user, middleware.GetClientIP(c), "stop_vm", "vm", vm.UUID, "success", "停止虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已停止"})
}

// ForceStopVM 强制停止虚拟机
func (h *VMHandler) ForceStopVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	if err := h.libvirtClient.ForceStopDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "强制停止虚拟机失败: " + err.Error()})
		return
	}

	vm.Status = "stopped"
	database.DB.Save(&vm)

	h.logAudit(user, middleware.GetClientIP(c), "force_stop_vm", "vm", vm.UUID, "success", "强制停止虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已强制停止"})
}

// RebootVM 重启虚拟机
func (h *VMHandler) RebootVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	if err := h.libvirtClient.RebootDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重启虚拟机失败: " + err.Error()})
		return
	}

	h.logAudit(user, middleware.GetClientIP(c), "reboot_vm", "vm", vm.UUID, "success", "重启虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已重启"})
}

// SuspendVM 挂起虚拟机
func (h *VMHandler) SuspendVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	if err := h.libvirtClient.SuspendDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "挂起虚拟机失败: " + err.Error()})
		return
	}

	vm.Status = "paused"
	database.DB.Save(&vm)

	h.logAudit(user, middleware.GetClientIP(c), "suspend_vm", "vm", vm.UUID, "success", "挂起虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已挂起"})
}

// ResumeVM 恢复虚拟机
func (h *VMHandler) ResumeVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	if err := h.libvirtClient.ResumeDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复虚拟机失败: " + err.Error()})
		return
	}

	vm.Status = "running"
	database.DB.Save(&vm)

	h.logAudit(user, middleware.GetClientIP(c), "resume_vm", "vm", vm.UUID, "success", "恢复虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已恢复"})
}

// DeleteVM 删除虚拟机
func (h *VMHandler) DeleteVM(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此虚拟机"})
		return
	}

	// 先停止虚拟机
	h.libvirtClient.ForceStopDomain(vm.UUID)

	// 取消定义
	if err := h.libvirtClient.UndefineDomain(vm.UUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除虚拟机失败: " + err.Error()})
		return
	}

	// 从数据库删除
	database.DB.Delete(&vm)

	h.logAudit(user, middleware.GetClientIP(c), "delete_vm", "vm", vm.UUID, "success", "删除虚拟机: "+vm.Name)

	c.JSON(http.StatusOK, gin.H{"message": "虚拟机已删除"})
}

// GetVNCConsole 获取 VNC 控制台信息
func (h *VMHandler) GetVNCConsole(c *gin.Context) {
	id := c.Param("id")
	user, _ := middleware.GetCurrentUser(c)

	var vm models.VM
	if err := database.DB.First(&vm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "虚拟机不存在"})
		return
	}

	// 权限检查
	if user.Role != models.RoleAdmin && vm.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此虚拟机"})
		return
	}

	// 获取 VNC 端口
	port, err := h.libvirtClient.GetVNCPort(vm.UUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 VNC 端口失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"port":    port,
		"ws_path": fmt.Sprintf("/api/v1/vms/%s/console", id),
	})
}

// logAudit 记录审计日志
func (h *VMHandler) logAudit(user *models.User, ip, action, resourceType, resourceID, status, message string) {
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
