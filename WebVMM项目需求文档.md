
WebVMM 产品需求文档 (PRD)
文档概述

1.1 产品简介
WebVMM 是一款基于 Web 的 KVM 虚拟化管理平台，旨在提供等同于或优于桌面客户端 virt-manager 的管理能力。它解决了传统工具依赖桌面环境、远程管理不便、权限控制粒度不足等痛点，为现代 IT 运维团队提供一个功能完备、安全可靠、易于部署的集中式虚拟化解决方案。

1.2 目标用户
系统管理员: 负责虚拟化平台的日常维护、资源分配和故障排查。
运维工程师: 需要远程管理虚拟机、监控资源使用情况。
审计员: 负责审查操作日志，确保合规性。
普通开发人员/测试人员: 需要自助创建、启动、停止测试用虚拟机（在授权范围内）。

1.3 核心价值
Web 化访问: 无需安装桌面客户端，通过浏览器即可在任何地点管理 KVM 虚拟化环境。
企业级安全: 细粒度的 RBAC 权限控制、TOTP 双因素认证、HTTPS 加密通信、完善的操作审计日志。
全生命周期管理: 覆盖虚拟机从创建、配置、运行、快照、克隆到销毁的全过程。
便捷部署: 单二进制文件交付，支持 systemd 服务化，开箱即用，自动初始化。
数据保护: 支持 OVA/OVF 标准格式的导入导出，以及基于 WebDAV 的数据库远程备份。

功能需求分析 (基于 virt-manager 对标)

通过对 virt-manager (Virtual Machine Manager) 的核心功能分析，WebVMM 将实现以下对应功能，并针对 Web 场景进行增强：
功能模块 | virt-manager 功能点 | WebVMM 实现方案与增强
:--- | :--- | :---
连接管理 | 本地 QEMU/KVM, SSH 远程连接 | 默认本地 libvirt (qemu:///system)。支持配置多个 Libvirt 连接（未来扩展），当前版本聚焦单机管理。
虚拟机概览 | 列表展示 (名称, 状态, CPU, 内存), 资源图表 | 仪表盘: 实时展示宿主机 CPU/内存/存储使用率，虚拟机状态统计 (运行中/停止/错误)。列表支持排序、筛选。
虚拟机创建 | 向导式创建 (ISO, PXE, 导入磁盘), 硬件定制 (CPU, 内存, 磁盘, 网络) | 创建向导: 支持上传 ISO (临时存储)、选择现有镜像、网络选择。硬件配置: 动态调整 vCPU, 内存，添加/移除磁盘 (qcow2/raw)，配置 Virtio 网卡。
虚拟机控制台 | VNC/Spice 嵌入式窗口 | Web 控制台: 集成 noVNC，通过 Go 后端 WebSocket 代理 libvirt VNC 流量，实现浏览器内无插件访问。支持全屏、剪贴板同步。
虚拟机操作 | 启动、关闭、重启、强制断电、暂停、恢复 | 操作按钮组: 提供启动、关机 (ACPI)、强制停止、重启、挂起、恢复等操作，带二次确认。
硬件配置 | 修改 CPU/内存，添加/删除磁盘、网卡、USB 设备 | 详情编辑页: 运行时热插拔支持 (取决于 Guest OS 和 libvirt 能力)。支持磁盘总线类型 (Virtio/IDE/SATA)、网络模型 (Virtio/e1000)、MAC 地址修改。
存储管理 | 存储池 (Storage Pool) 管理，卷 (Volume) 创建/上传/删除 | 存储池管理: 列出所有存储池 (dir, fs, netfs, logical 等)。卷管理: 创建新卷、上传镜像文件、删除卷、查看卷信息 (格式、容量、分配量)。
网络管理 | 虚拟网络 (NAT, Bridge, Isolated) 定义与配置 | 虚拟网络管理: 创建/编辑/删除虚拟网络。支持 NAT、隔离模式。显示关联的 DHCP 范围和转发规则。支持桥接物理网卡配置。
快照管理 | 创建、恢复、删除快照 (内部快照) | 快照树: 可视化展示快照层级。支持创建当前状态快照、恢复到指定快照、删除快照。支持快照元数据备注。
克隆/迁移 | 克隆虚拟机，迁移到另一台主机 | 克隆: 基于现有 VM 快速克隆 (重命名、重置 MAC)。冷迁移: 导出配置和磁盘，支持导入 (未来支持热迁移需共享存储)。
性能监控 | 实时 CPU、内存、磁盘 I/O、网络 I/O 图表 | 实时监控: 基于 libvirt 统计信息，前端使用 ECharts/Chart.js 绘制实时曲线图 (每秒刷新)。
XML 编辑 | 直接编辑 Domain XML | 高级设置: 提供只读/可编辑的 XML 视图，允许高级用户直接修改底层配置 (带语法校验)。
详细功能需求

3.1 用户认证与权限管理 (IAM)

3.1.1 角色定义
系统预置三种角色，基于 RBAC (Role-Based Access Control) 模型：
管理员 (Admin):
拥有系统最高权限。
管理所有虚拟机、存储、网络资源。
管理用户账户（创建、禁用、重置密码、分配角色）。
配置系统参数（证书、备份策略）。
查看所有审计日志。
审计员 (Auditor):
只读权限。
可查看所有虚拟机、存储、网络的配置和状态。
专属权限: 可查询、导出所有操作日志和审计日志。
不可执行任何变更操作（如开关机、创建、删除）。
用户 (User):
默认仅对自己创建的虚拟机拥有完全控制权。
可查看公共存储池和网络。
不可管理其他用户的虚拟机（除非管理员授权）。
不可修改系统级配置。

3.1.2 登录与安全
账号密码登录: 用户名/邮箱 + 密码。
TOTP 双因素认证 (2FA):
用户可在个人中心开启/关闭 TOTP。
开启流程：后端生成 Secret -> 展示 QR Code -> 用户绑定 Authenticator App -> 输入验证码验证 -> 启用。
登录时若开启 2FA，需输入动态验证码。
库选型：github.com/pquerna/otp。
会话管理: JWT Token 认证，支持 Refresh Token 机制，可配置会话超时时间。
密码策略: 最小长度 8 位，包含大小写字母、数字、特殊字符。

3.1.3 首次启动向导 (First Run Wizard)
触发条件: 数据库为空（无用户记录）。
流程:
访问任意页面重定向至 /setup。
表单输入：管理员用户名、邮箱、密码、确认密码。
提交后创建第一个 Admin 用户。
标记系统为“已初始化”，永久关闭该入口。
CLI 重置: 提供命令行工具 webvmm reset-admin，在服务停止状态下可重置 admin 密码或重新开启向导（需二次确认）。

3.2 虚拟机全生命周期管理
列表与搜索: 支持按名称、状态、IP、标签搜索。
创建虚拟机:
来源: 本地 ISO 上传、已有镜像卷、PXE (高级)、导入 OVF/OVA。
配置: 名称、描述、操作系统类型、vCPU、内存、磁盘（大小、总线、格式）、网络（选择虚拟网络、MAC 地址）、显卡（Virtio/VGA）。
云初始化: 支持注入 SSH Key、设置 root 密码、Hostname (Cloud-Init)。
运行控制: 启动、关机、重启、强制停止、挂起、恢复。
控制台访问: 点击“控制台”打开 noVNC 窗口，自动获取 VNC Ticket 并建立 WebSocket 连接。
编辑配置:
支持离线修改所有硬件参数。
支持在线热添加 CPU/内存/磁盘/网卡（需 Guest 支持）。
快照管理:
创建快照（支持内存快照/磁盘快照）。
快照列表展示（名称、创建时间、父节点）。
回滚到指定快照。
删除快照。
克隆: 基于现有 VM 创建副本，自动处理 MAC 地址冲突和磁盘复制。
删除: 软删除（移至回收站，可选）或硬删除（同时删除磁盘文件）。

3.3 导入导出功能 (OVA/OVF)
导出 (Export):
用户选择虚拟机 -> 点击“导出”。
后端调用 libvirt 获取 XML 配置，结合 qemu-img 转换/复制磁盘文件。
打包为标准 OVA (Tarball) 或 OVF (Descriptor + Disk files) 格式。
提供浏览器下载链接。
导入 (Import):
用户上传 OVA/OVF 文件包（或解压后的文件）。
后端解析 OVF Descriptor (XML)，校验磁盘文件完整性。
映射存储池和网络。
创建新的 Libvirt Domain 和存储卷。
显示导入进度条。
技术实现: 使用 Go 编写 OVF 解析器，调用 qemu-img convert 处理磁盘格式转换。

3.4 资源监控 (Storage & Network)
存储池管理:
列表展示所有 Storage Pool (Name, Type, Capacity, Allocation, Available)。
激活/停用存储池。
浏览存储卷 (Volumes): 创建、上传、下载、删除、格式化。
虚拟网络管理:
列表展示虚拟网络 (Name, Bridge, Forwarding Mode, DHCP Range)。
创建/编辑/删除虚拟网络 (NAT, Isolated, Bridge)。
查看关联的虚拟机。

3.5 操作日志与审计
日志内容:
用户操作: 登录、登出、修改密码、创建/删除用户。
资源变更: 虚拟机创建、启动、停止、配置修改、快照操作。
系统事件: 备份成功/失败、证书更新、服务重启。
字段: 时间戳、操作用户、IP 地址、操作对象 (Type+ID)、操作类型 (Create/Start/Edit)、详细信息 (JSON Diff)、结果 (Success/Fail)。
查询功能:
多维度筛选：用户、虚拟机名称、操作类型、日期范围、结果状态。
分页展示。
支持导出 CSV/Excel。

3.6 系统管理与备份
HTTPS 配置:
默认: 首次启动检测证书文件不存在，自动生成自签名证书 (RSA 2048, 有效期 1 年)。
自定义: 配置文件 config.yaml 中指定 cert_file 和 key_file 路径，重启生效。
数据库备份:
引擎: SQLite (单文件 webvmm.db)。
定时备份: 内置 Cron 调度器，支持配置备份频率 (如每天凌晨 2 点)。
WebDAV 支持:
配置 WebDAV 服务器地址、用户名、密码、远程路径。
备份时自动上传 .db 文件到远端，支持保留最近 N 份备份。
手动触发备份按钮。
服务管理:
提供 CLI 命令 webvmm install-service，自动生成 systemd unit 文件 (/etc/systemd/system/webvmm.service)。
自动创建配置目录 /etc/webvmm/ 和数据目录 /var/lib/webvmm/。
设置开机自启。

技术架构设计

4.1 技术栈
后端:
语言: Go (Golang) 1.21+
Web 框架: Gin
虚拟化库: github.com/libvirt/libvirt-go (绑定 C libvirt)
数据库: SQLite (gorm.io/gorm + sqlite)
认证: JWT (golang-jwt/jwt), TOTP (github.com/pquerna/otp)
WebSocket: github.com/gorilla/websocket (用于 noVNC 代理)
任务队列: Goroutine + Channel (轻量级异步任务，如导入导出、备份)
前端:
语言: TypeScript
框架: Vue 3 (Composition API)
UI 库: Naive UI
状态管理: Pinia
HTTP 客户端: Axios
控制台: noVNC (嵌入前端静态资源或通过后端代理)
基础设施:
OS: Linux (Ubuntu/CentOS/Rocky/Debian)
Hypervisor: KVM/QEMU
管理守护进程: Libvirtd

4.2 系统架构图 (逻辑)

graph TD
    User[用户浏览器] -->|HTTPS/WSS| Nginx[可选反向代理]
    Nginx -->|HTTPS/WSS| WebVMM[WebVMM Server (Go+Gin)]
    
    subgraph "WebVMM Server"
        Auth[认证模块 (JWT/TOTP)]
        API[RESTful API]
        WS_Proxy[WebSocket Proxy (noVNC)]
        Scheduler[定时任务 (备份)]
        DB[(SQLite)]
        Libvirt_Client[Libvirt Go Client]
    end
    
    WebVMM --> DB
    WS_Proxy -->|TCP| Libvirtd
    Libvirt_Client -->|Socket| Libvirtd
    
    subgraph "Host System"
        Libvirtd[Libvirtd Daemon]
        KVM[KVM/QEMU VMs]
        Storage[存储池/卷]
        Net[虚拟网络]
    end
    
    Libvirtd --> KVM
    Libvirtd --> Storage
    Libvirtd --> Net

4.3 关键模块设计

4.3.1 noVNC 控制台代理
由于浏览器无法直接连接 VNC (TCP)，且 noVNC 需要 WebSocket：
前端加载 noVNC 静态资源。
前端发起 WSS 请求到 WebVMM 后端 /api/console/ticket?vm_id=xxx。
后端验证用户权限，调用 Libvirt API 获取 VNC 监听端口和密码 (Ticket)。
后端作为 WebSocket 服务端，同时作为 TCP 客户端连接 localhost:VNC_Port。
双向转发数据流 (Browser <-> Go Server <-> VNC Server)。
安全: 仅在内存中维持连接，不记录 VNC 密码，连接关闭即销毁。

4.3.2 OVA/OVF 处理流程
导出:
接收请求，锁定 VM (防止变更)。
生成 OVF XML (基于 Libvirt XML 转换)。
调用 qemu-img convert -O qcow2 src dst (确保格式兼容)。
打包文件 (tar) 为 .ova。
流式传输给前端下载。
解锁 VM。
导入:
上传文件到临时目录。
解压/解析 OVF。
校验磁盘文件。
转换磁盘格式为目标存储池格式。
定义 Libvirt XML 并 Create Domain。
清理临时文件。

4.3.3 数据库备份 (WebDAV)
使用 golang.org/x/net/webdav 或第三方库实现 WebDAV Client。
备份任务：
SQLite WAL 模式下检查点 (Checkpoint) 确保数据落盘。
复制 .db 文件到临时带时间戳的文件。
建立 WebDAV 连接。
PUT 上传文件。
清理本地临时文件。
记录备份日志。

非功能需求

5.1 安全性
通信加密: 强制 HTTPS (TLS 1.2+)，WSS 用于控制台。
密码存储: 使用 bcrypt 加盐哈希存储。
防暴力破解: 登录失败 5 次锁定账号 15 分钟。
XSS/CSRF: 前端框架自动防御，后端设置 Secure Cookie，校验 Origin/Referer。
权限隔离: 所有 API 必须经过中间件鉴权，严格校验资源归属。

5.2 性能
响应时间: 普通 API < 200ms，控制台延迟 < 100ms (局域网)。
并发: 支持至少 50 个并发用户操作，20 个并发控制台会话。
资源占用: 空闲时内存占用 < 100MB。

5.3 可靠性
单点故障: 当前版本为单机部署，依赖宿主机稳定性。
数据一致性: SQLite 事务保证，备份机制防止数据丢失。
异常处理: 所有 Libvirt 操作需捕获错误，返回友好提示，避免 Panic。

5.4 易用性
界面: 简洁直观，符合运维习惯，深色/浅色模式切换。
反馈: 操作耗时任务（如创建、导入）需提供进度条或任务中心。
文档: 内置帮助文档链接，错误码说明。

部署与交付

6.1 交付物
单一可执行文件: webvmm (Linux amd64/arm64)。
Systemd 服务文件模板: 内置于二进制或通过 CLI 生成。
默认配置: config.yaml (自动生成)。
前端静态资源: 嵌入到二进制文件中 (embed.FS)，无需单独部署 Nginx。

6.2 安装流程
下载二进制文件: wget .../webvmm
赋予执行权限: chmod +x webvmm
安装服务: sudo ./webvmm install-service
自动创建 /etc/webvmm/config.yaml
自动创建 /var/lib/webvmm/data.db
自动创建 /var/log/webvmm/
生成 systemd unit 并 enable/start
防火墙开放端口 (默认 8443)。
浏览器访问 https://<IP>:8443。
完成首次管理员设置。

6.3 配置文件示例 (config.yaml)
server:
  port: 8443
  cert_file: /etc/webvmm/cert.pem
  key_file: /etc/webvmm/key.pem
  # 若文件不存在，自动生成的路径
  auto_gen_cert: true 

database:
  path: /var/lib/webvmm/data.db
  
backup:
  enabled: true
  schedule: "0 2 * * *" # Cron 表达式
  retention_days: 7
  webdav:
    url: "https://backup.example.com/dav"
    username: "admin"
    password: "secret" # 建议通过环境变量注入
    remote_path: "/webvmm_backups"

log:
  level: "info"
  path: "/var/log/webvmm/app.log"

6.4 命令行工具 (CLI)
./webvmm --help
./webvmm version
./webvmm install-service   # 安装为 systemd 服务
./webvmm uninstall-service # 卸载服务
./webvmm dev   # 开发测试使用这个命令启动服务
./webvmm reset-admin       # 重置 admin 密码 (需停止服务)
./webvmm backup-now        # 手动触发一次备份
./webvmm gen-cert          # 手动生成自签名证书

开发计划与里程碑
阶段 | 周期 | 主要任务 | 交付成果
:--- | :--- | :--- | :---
Phase 1: 核心框架 | 2 周 | 项目初始化，Gin+Vue 骨架，SQLite 集成，用户登录 (JWT), RBAC 基础，HTTPS 自签名 | 可运行的空壳，能登录，有 Admin 初始化向导
Phase 2: 虚拟机基础 | 3 周 | Libvirt 对接，VM 列表，详情，启动/停止，创建向导 (ISO), 基础控制台 (noVNC) | 可创建和管理 VM，能通过 Web 访问控制台
Phase 3: 资源与高级功能 | 3 周 | 存储池/卷管理，网络管理，快照，克隆，编辑配置，TOTP 2FA | 完整的 VM 生命周期管理，安全增强
Phase 4: 导入导出与备份 | 2 周 | OVF/OVA 解析与实现，WebDAV 备份模块，操作日志系统 | 数据迁移能力，灾备能力，审计能力
Phase 5: 优化与测试 | 2 周 | 性能优化，UI/UX 打磨，单元测试，集成测试，文档编写 | 生产就绪版本 (RC)
总计 | 12 周 | | WebVMM v1.0
风险与应对
风险点 | 描述 | 应对策略
:--- | :--- | :---
Libvirt 兼容性 | 不同 Linux 发行版 libvirt 版本差异可能导致 API 行为不一致 | 锁定最低 libvirt 版本要求，在 CI 中多环境测试，封装适配层。
noVNC 性能 | 高延迟网络下控制台体验差 | 优化 WebSocket 缓冲，支持调整图像质量/帧率，提供“只读”模式。
OVA 标准复杂性 | OVF 标准庞大，完全兼容困难 | 优先支持主流厂商 (VMware, VirtualBox) 导出的标准子集，明确文档说明支持范围。
SQLite 并发写 | 高并发写操作可能锁库 | 开启 WAL 模式，优化事务粒度，对于重度写入日志考虑异步批量写入。
安全风险 | Web 暴露面增加被攻击风险 | 严格代码审计，定期依赖扫描，默认防火墙策略，强制 2FA 推荐。
