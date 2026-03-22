package models

import (
	"time"

	"gorm.io/gorm"
)

// Role 用户角色
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleAuditor Role = "auditor"
	RoleUser    Role = "user"
)

// User 用户模型
type User struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Username      string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email         string         `gorm:"uniqueIndex;size:100;not null" json:"email"`
	PasswordHash  string         `gorm:"size:255;not null" json:"-"`
	Role          Role           `gorm:"size:20;default:user" json:"role"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	TOTPSecret    string         `gorm:"size:100" json:"-"`
	TOTPEnabled   bool           `gorm:"default:false" json:"totp_enabled"`
	LastLoginAt   *time.Time     `json:"last_login_at"`
	LastLoginIP   string         `gorm:"size:45" json:"last_login_ip"`
	FailedLogins  int            `gorm:"default:0" json:"failed_logins"`
	LockedUntil   *time.Time     `json:"locked_until"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// VM 虚拟机模型
type VM struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UUID        string         `gorm:"uniqueIndex;size:36;not null" json:"uuid"`
	Name        string         `gorm:"index;size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	OwnerID     uint           `gorm:"index;not null" json:"owner_id"`
	Owner       *User          `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Status      string         `gorm:"size:20" json:"status"` // running, stopped, paused, error
	VCPU        int            `gorm:"default:1" json:"vcpu"`
	Memory      int64          `gorm:"default:1073741824" json:"memory"` // bytes
	AutoStart   bool           `gorm:"default:false" json:"auto_start"`
	XMLDef      string         `gorm:"type:text" json:"xml_def"`
	Tags        string         `gorm:"size:255" json:"tags"` // JSON array
}

// TableName 指定表名
func (VM) TableName() string {
	return "vms"
}

// VMDisk 虚拟机磁盘
type VMDisk struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VMID      uint      `gorm:"index;not null" json:"vm_id"`
	Path      string    `gorm:"size:255;not null" json:"path"`
	Size      int64     `json:"size"` // bytes
	Format    string    `gorm:"size:20;default:qcow2" json:"format"`
	Bus       string    `gorm:"size:20;default:virtio" json:"bus"`
	ReadOnly  bool      `gorm:"default:false" json:"read_only"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (VMDisk) TableName() string {
	return "vm_disks"
}

// VMNIC 虚拟机网卡
type VMNIC struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VMID      uint      `gorm:"index;not null" json:"vm_id"`
	MAC       string    `gorm:"size:17" json:"mac"`
	Network   string    `gorm:"size:100" json:"network"`
	Model     string    `gorm:"size:20;default:virtio" json:"model"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (VMNIC) TableName() string {
	return "vm_nics"
}

// Snapshot 快照模型
type Snapshot struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	VMID        uint      `gorm:"index;not null" json:"vm_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"size:500" json:"description"`
	ParentID    *uint     `json:"parent_id"`
	IsCurrent   bool      `gorm:"default:false" json:"is_current"`
	IsInternal  bool      `gorm:"default:true" json:"is_internal"`
}

// TableName 指定表名
func (Snapshot) TableName() string {
	return "snapshots"
}

// StoragePool 存储池
type StoragePool struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Name       string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Type       string    `gorm:"size:20;not null" json:"type"` // dir, fs, logical, etc.
	Path       string    `gorm:"size:255" json:"path"`
	Capacity   int64     `json:"capacity"`   // bytes
	Allocation int64     `json:"allocation"` // bytes
	Available  int64     `json:"available"`  // bytes
	IsActive   bool      `gorm:"default:false" json:"is_active"`
	AutoStart  bool      `gorm:"default:false" json:"auto_start"`
}

// TableName 指定表名
func (StoragePool) TableName() string {
	return "storage_pools"
}

// StorageVolume 存储卷
type StorageVolume struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	PoolID       uint      `gorm:"index;not null" json:"pool_id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Path         string    `gorm:"size:255;not null" json:"path"`
	Format       string    `gorm:"size:20" json:"format"`
	Capacity     int64     `json:"capacity"`     // bytes
	Allocation   int64     `json:"allocation"`   // bytes
	BackingFile  string    `gorm:"size:255" json:"backing_file"`
}

// TableName 指定表名
func (StorageVolume) TableName() string {
	return "storage_volumes"
}

// VirtualNetwork 虚拟网络
type VirtualNetwork struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Mode        string    `gorm:"size:20;not null" json:"mode"` // nat, bridge, isolated
	Bridge      string    `gorm:"size:100" json:"bridge"`
	Subnet      string    `gorm:"size:20" json:"subnet"`
	DHCPStart   string    `gorm:"size:15" json:"dhcp_start"`
	DHCPEnd     string    `gorm:"size:15" json:"dhcp_end"`
	IsActive    bool      `gorm:"default:false" json:"is_active"`
	AutoStart   bool      `gorm:"default:false" json:"auto_start"`
	XMLDef      string     `gorm:"type:text" json:"xml_def"`
}

// TableName 指定表名
func (VirtualNetwork) TableName() string {
	return "virtual_networks"
}

// AuditLog 审计日志
type AuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Username    string    `gorm:"size:50;index" json:"username"`
	IP          string    `gorm:"size:45" json:"ip"`
	Action      string    `gorm:"size:50;index;not null" json:"action"` // login, create_vm, start_vm, etc.
	ResourceType string   `gorm:"size:50;index" json:"resource_type"`   // vm, user, pool, network
	ResourceID   string   `gorm:"size:50;index" json:"resource_id"`
	Details     string    `gorm:"type:text" json:"details"` // JSON
	Status      string    `gorm:"size:20" json:"status"`    // success, failed
	Message     string    `gorm:"size:500" json:"message"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

// SystemSetting 系统设置
type SystemSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	Key       string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
}

// TableName 指定表名
func (SystemSetting) TableName() string {
	return "system_settings"
}

// Backup 备份记录
type Backup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Size        int64     `json:"size"`
	Path        string    `gorm:"size:255" json:"path"`
	Status      string    `gorm:"size:20" json:"status"` // success, failed
	Message     string    `gorm:"size:500" json:"message"`
	IsRemote    bool      `gorm:"default:false" json:"is_remote"`
}

// TableName 指定表名
func (Backup) TableName() string {
	return "backups"
}

// Task 异步任务
type Task struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Type      string    `gorm:"size:50;not null" json:"type"` // import, export, backup, clone
	Status    string    `gorm:"size:20;default:pending" json:"status"` // pending, running, completed, failed
	Progress  int       `gorm:"default:0" json:"progress"` // 0-100
	Message   string    `gorm:"size:500" json:"message"`
	UserID    uint      `gorm:"index" json:"user_id"`
	ResourceType string `gorm:"size:50" json:"resource_type"`
	ResourceID   uint   `json:"resource_id"`
	Result    string    `gorm:"type:text" json:"result"` // JSON result
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}