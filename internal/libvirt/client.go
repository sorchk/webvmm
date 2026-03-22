package libvirt

import (
	"encoding/xml"
	"fmt"
	"log"
	"sync"

	"github.com/libvirt/libvirt-go"
)

// Client Libvirt 客户端
type Client struct {
	conn *libvirt.Connect
	mu   sync.RWMutex
}

// VMInfo 虚拟机信息
type VMInfo struct {
	UUID   string
	Name   string
	Status string
	VCPU   uint
	Memory uint64
}

// DomainXML Domain XML 结构
type DomainXML struct {
	XMLName xml.Name `xml:"domain"`
	Type    string   `xml:"type,attr"`
	Name    string   `xml:"name"`
	UUID    string   `xml:"uuid"`
	Memory  struct {
		Value uint64 `xml:",chardata"`
		Unit  string `xml:"unit,attr"`
	} `xml:"memory"`
	VCPU struct {
		Value uint `xml:",chardata"`
	} `xml:"vcpu"`
	OS struct {
		Type struct {
			Arch    string `xml:"arch,attr"`
			Machine string `xml:"machine,attr"`
			Value   string `xml:",chardata"`
		} `xml:"type"`
	} `xml:"os"`
	Devices struct {
		Disks []struct {
			Type   string `xml:"type,attr"`
			Device string `xml:"device,attr"`
			Driver struct {
				Name string `xml:"name,attr"`
				Type string `xml:"type,attr"`
			} `xml:"driver"`
			Source struct {
				File   string `xml:"file,attr"`
				Dev    string `xml:"dev,attr"`
				Pool   string `xml:"pool,attr"`
				Volume string `xml:"volume,attr"`
			} `xml:"source"`
			Target struct {
				Dev string `xml:"dev,attr"`
				Bus string `xml:"bus,attr"`
			} `xml:"target"`
		} `xml:"disk"`
		Interfaces []struct {
			Type string `xml:"type,attr"`
			MAC  struct {
				Address string `xml:"address,attr"`
			} `xml:"mac"`
			Source struct {
				Network string `xml:"network,attr"`
				Bridge  string `xml:"bridge,attr"`
			} `xml:"source"`
			Model struct {
				Type string `xml:"type,attr"`
			} `xml:"model"`
		} `xml:"interface"`
		Graphics []struct {
			Type     string `xml:"type,attr"`
			Port     int    `xml:"port,attr"`
			AutoPort string `xml:"autoport,attr"`
			Listen   string `xml:"listen,attr"`
			Password string `xml:"passwd,attr"`
		} `xml:"graphics"`
	} `xml:"devices"`
}

var (
	client     *Client
	clientOnce sync.Once
)

// GetClient 获取 Libvirt 客户端单例
func GetClient() (*Client, error) {
	var initErr error
	clientOnce.Do(func() {
		conn, err := libvirt.NewConnect("qemu:///system")
		if err != nil {
			initErr = fmt.Errorf("连接 libvirt 失败: %w", err)
			return
		}
		client = &Client{conn: conn}
	})
	if initErr != nil {
		return nil, initErr
	}
	return client, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		_, err := c.conn.Close()
		return err
	}
	return nil
}

// ListDomains 列出所有虚拟机
func (c *Client) ListDomains() ([]VMInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 获取活动的虚拟机
	activeDomains, err := c.conn.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("获取活动虚拟机列表失败: %w", err)
	}

	// 获取所有定义的虚拟机
	definedDomains, err := c.conn.ListDefinedDomains()
	if err != nil {
		return nil, fmt.Errorf("获取定义虚拟机列表失败: %w", err)
	}

	var vms []VMInfo

	// 处理活动的虚拟机
	for _, domID := range activeDomains {
		dom, err := c.conn.LookupDomainById(domID)
		if err != nil {
			continue
		}

		info, err := dom.GetInfo()
		if err != nil {
			dom.Free()
			continue
		}

		name, err := dom.GetName()
		if err != nil {
			dom.Free()
			continue
		}

		uuid, err := dom.GetUUIDString()
		if err != nil {
			dom.Free()
			continue
		}

		vms = append(vms, VMInfo{
			UUID:   uuid,
			Name:   name,
			Status: getDomainStatus(info.State),
			VCPU:   info.NrVirtCpu,
			Memory: info.Memory,
		})
		dom.Free()
	}

	// 处理非活动的虚拟机
	for _, domName := range definedDomains {
		dom, err := c.conn.LookupDomainByName(domName)
		if err != nil {
			continue
		}

		info, err := dom.GetInfo()
		if err != nil {
			dom.Free()
			continue
		}

		uuid, err := dom.GetUUIDString()
		if err != nil {
			dom.Free()
			continue
		}

		vms = append(vms, VMInfo{
			UUID:   uuid,
			Name:   domName,
			Status: getDomainStatus(info.State),
			VCPU:   info.NrVirtCpu,
			Memory: info.Memory,
		})
		dom.Free()
	}

	return vms, nil
}

// getDomainStatus 获取虚拟机状态字符串
func getDomainStatus(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return "running"
	case libvirt.DOMAIN_BLOCKED:
		return "blocked"
	case libvirt.DOMAIN_PAUSED:
		return "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		return "stopped"
	case libvirt.DOMAIN_CRASHED:
		return "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "suspended"
	default:
		return "unknown"
	}
}

// ListDomainsOld 列出所有虚拟机 (旧方法保留备用)
func (c *Client) ListDomainsOld() ([]VMInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	domains, err := c.conn.ListAllDomains(0)
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机列表失败: %w", err)
	}

	var vms []VMInfo
	for _, dom := range domains {
		info, err := dom.GetInfo()
		if err != nil {
			log.Printf("获取虚拟机信息失败: %v", err)
			continue
		}

		name, err := dom.GetName()
		if err != nil {
			log.Printf("获取虚拟机名称失败: %v", err)
			continue
		}

		uuid, err := dom.GetUUIDString()
		if err != nil {
			log.Printf("获取虚拟机 UUID 失败: %v", err)
			continue
		}

		vms = append(vms, VMInfo{
			UUID:   uuid,
			Name:   name,
			Status: getDomainStatus(info.State),
			VCPU:   info.NrVirtCpu,
			Memory: info.Memory,
		})

		dom.Free()
	}

	return vms, nil
}

// GetDomain 获取单个虚拟机
func (c *Client) GetDomain(uuid string) (*libvirt.Domain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return nil, fmt.Errorf("查找虚拟机失败: %w", err)
	}

	return dom, nil
}

// GetDomainXML 获取虚拟机 XML
func (c *Client) GetDomainXML(uuid string) (*DomainXML, error) {
	dom, err := c.GetDomain(uuid)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	xmlStr, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("获取 XML 失败: %w", err)
	}

	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlStr), &domainXML); err != nil {
		return nil, fmt.Errorf("解析 XML 失败: %w", err)
	}

	return &domainXML, nil
}

// StartDomain 启动虚拟机
func (c *Client) StartDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Create()
}

// StopDomain 停止虚拟机 (ACPI 关机)
func (c *Client) StopDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Shutdown()
}

// ForceStopDomain 强制停止虚拟机
func (c *Client) ForceStopDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Destroy()
}

// RebootDomain 重启虚拟机
func (c *Client) RebootDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Reboot(0)
}

// SuspendDomain 挂起虚拟机
func (c *Client) SuspendDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Suspend()
}

// ResumeDomain 恢复虚拟机
func (c *Client) ResumeDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Resume()
}

// DefineDomain 定义虚拟机
func (c *Client) DefineDomain(xmlStr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.conn.DomainDefineXML(xmlStr)
	if err != nil {
		return fmt.Errorf("定义虚拟机失败: %w", err)
	}

	return nil
}

// UndefineDomain 取消定义虚拟机
func (c *Client) UndefineDomain(uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dom, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("查找虚拟机失败: %w", err)
	}
	defer dom.Free()

	return dom.Undefine()
}

// GetDomainInfo 获取虚拟机详细信息
func (c *Client) GetDomainInfo(uuid string) (*VMInfo, error) {
	dom, err := c.GetDomain(uuid)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	info, err := dom.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机信息失败: %w", err)
	}

	name, err := dom.GetName()
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机名称失败: %w", err)
	}

	status := "unknown"
	switch info.State {
	case libvirt.DOMAIN_RUNNING:
		status = "running"
	case libvirt.DOMAIN_BLOCKED:
		status = "blocked"
	case libvirt.DOMAIN_PAUSED:
		status = "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		status = "shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		status = "stopped"
	case libvirt.DOMAIN_CRASHED:
		status = "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		status = "suspended"
	}

	return &VMInfo{
		UUID:   uuid,
		Name:   name,
		Status: status,
		VCPU:   info.NrVirtCpu,
		Memory: info.Memory,
	}, nil
}

// GetVNCPort 获取 VNC 端口
func (c *Client) GetVNCPort(uuid string) (int, error) {
	dom, err := c.GetDomain(uuid)
	if err != nil {
		return 0, err
	}
	defer dom.Free()

	xmlStr, err := dom.GetXMLDesc(0)
	if err != nil {
		return 0, fmt.Errorf("获取 XML 失败: %w", err)
	}

	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlStr), &domainXML); err != nil {
		return 0, fmt.Errorf("解析 XML 失败: %w", err)
	}

	for _, graphics := range domainXML.Devices.Graphics {
		if graphics.Type == "vnc" && graphics.Port != -1 {
			return graphics.Port, nil
		}
	}

	return 0, fmt.Errorf("VNC 端口未找到")
}
