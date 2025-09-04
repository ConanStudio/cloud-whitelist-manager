# Cloud Whitelist Manager

## 概述

Cloud Whitelist Manager是一个用于自动检测主机公网IP变化并更新阿里云服务白名单的工具。当主机的公网IP发生变化时，该工具会自动将新的IP地址添加到阿里云ECS安全组、RDS MySQL、Redis和CLB的白名单中，同时删除旧的IP地址。

## 功能特性

- **自动IP检测**：支持多种方式获取公网IP（HTTP接口、网卡、命令行）
- **多服务支持**：支持ECS安全组、RDS白名单、Redis白名单、CLB白名单
- **自动更新**：IP变化时自动添加新IP并删除旧IP
- **灵活配置**：支持自定义检查间隔和多种IP获取方式
- **容器化部署**：提供Docker镜像便于部署

## 技术架构

该工具使用Go语言开发，具有以下特点：
- 高性能和低资源消耗
- 支持多种IP获取方式
- 与阿里云API集成
- 容器化部署支持

## 安装和部署

### 本地编译运行

1. 克隆代码库
2. 安装依赖：`go mod download`
3. 编译：`go build -o cloud-whitelist-manager ./cmd/cloud-whitelist-manager`
4. 运行：`./cloud-whitelist-manager --config config.yaml`

### Docker部署

构建镜像：
```bash
docker build -t cloud-whitelist-manager .
```

运行容器：
```bash
docker run -d \
  --name cloud-whitelist-manager \
  -v /path/to/your/config.yaml:/app/config.yaml \
  cloud-whitelist-manager
```

### Kubernetes部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloud-whitelist-manager
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cloud-whitelist-manager
  template:
    metadata:
      labels:
        app: cloud-whitelist-manager
    spec:
      containers:
      - name: cloud-whitelist-manager
        image: cloud-whitelist-manager:latest
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: cloud-whitelist-manager-config
```

## 配置说明

配置文件采用YAML格式，主要包含以下部分：

### 基本配置

- `interval`: 检查间隔（秒）
- `ip_source`: IP获取源（选择其中一种方式）
- `accounts`: 多阿里云账号配置列表

### IP获取源配置

支持三种方式获取IP，选择其中一种方式：

1. **HTTP方式**：
   ```yaml
   ip_source:
     type: http
     url: "http://ipinfo.io/ip"
     timeout: 10
     headers:
       User-Agent: "IP-Update-Tool"
   ```

2. **命令行方式**：
   ```yaml
   ip_source:
     type: command
     cmd: "curl -s ifconfig.me"
     timeout: 10
   ```

3. **网卡方式**：
   ```yaml
   ip_source:
     type: interface
     interface: "eth0"
     ipv6: false
   ```

### 阿里云配置

支持两种配置方式：

1. **单账号配置**（向后兼容）：
- `access_key_id`: 阿里云访问密钥ID
- `access_key_secret`: 阿里云访问密钥Secret
- `region_id`: 阿里云区域ID

2. **多账号配置**（推荐）：
在`accounts`列表中配置多个账号，每个账号包含：
- `name`: 账号名称
- `access_key_id`: 阿里云访问密钥ID
- `access_key_secret`: 阿里云访问密钥Secret
- `region_id`: 阿里云区域ID

#### ECS安全组配置
- `enabled`: 是否启用
- `security_groups`: 安全组列表，支持配置多个安全组，每个安全组包含：
  - `security_group_id`: 安全组ID
  - `port`: 端口
  - `priority`: 规则优先级

#### RDS配置
- `enabled`: 是否启用
- `instance_whitelists`: RDS实例白名单列表，支持配置多个实例，每个实例包含：
  - `instance_id`: RDS实例ID
  - `whitelist_name`: 白名单分组名称

#### Redis配置
- `enabled`: 是否启用
- `instance_whitelists`: Redis实例白名单列表，支持配置多个实例，每个实例包含：
  - `instance_id`: Redis实例ID
  - `whitelist_name`: 白名单分组名称

#### CLB配置
- `enabled`: 是否启用
- `load_balancer_whitelists`: CLB白名单列表，支持配置多个访问控制策略组，每个策略组包含：
  - `acl_id`: 访问控制策略组ID

## 使用说明

### 快速开始

1. 配置文件设置

首先，您需要编辑 [config.yaml](file:///home/conan/workspace/qoder/cloud-kit/config.yaml) 文件来配置Cloud Whitelist Manager：

```yaml
# 基本配置
interval: 300  # 检查间隔（秒）

# IP获取源配置（选择其中一种方式）
ip_source:
  type: http
  url: "http://ipinfo.io/ip"
  timeout: 10  # 超时时间（秒）
  headers:     # 自定义请求头
    User-Agent: "Cloud-Whitelist-Manager"

# 多账号配置（推荐使用）
accounts:
- name: "production-account"
  access_key_id: "your_production_access_key_id"
  access_key_secret: "your_production_access_key_secret"
  region_id: "cn-hangzhou"
  
  # ECS安全组配置（支持多个安全组）
  ecs:
    enabled: true
    security_groups:
      - security_group_id: "sg-production-web"
        port: 80
        priority: 100
      - security_group_id: "sg-production-web"
        port: 443
        priority: 101
      - security_group_id: "sg-production-ssh"
        port: 22
        priority: 102
    
  # RDS配置（支持多个实例白名单）
  rds:
    enabled: true
    instance_whitelists:
      - instance_id: "rm-production-mysql-primary"
        whitelist_name: "default"
      - instance_id: "rm-production-mysql-standby"
        whitelist_name: "default"
    
  # Redis配置（支持多个实例白名单）
  redis:
    enabled: true
    instance_whitelists:
      - instance_id: "r-production-redis-primary"
        whitelist_name: "default"
      - instance_id: "r-production-redis-standby"
        whitelist_name: "default"
    
  # CLB配置（支持多个访问控制策略组）
  clb:
    enabled: true
    load_balancer_whitelists:
      - acl_id: "acl-production-web"

# 单账号配置（向后兼容）
#aliyun:
#  access_key_id: "your_access_key_id"
#  access_key_secret: "your_access_key_secret"
#  region_id: "cn-hangzhou"
#  
#  # ECS安全组配置
#  ecs:
#    enabled: true
#    security_groups:
#      - security_group_id: "sg-xxxxxxxxx"
#        port: 22
#        priority: 100
#      
#  # RDS配置
#  rds:
#    enabled: true
#    instance_whitelists:
#      - instance_id: "rm-xxxxxxxxx"
#        whitelist_name: "default"
#      
#  # Redis配置
#  redis:
#    enabled: true
#    instance_whitelists:
#      - instance_id: "r-xxxxxxxxx"
#        whitelist_name: "default"
#      
#  # CLB配置
#  clb:
#    enabled: true
#    load_balancer_whitelists:
#      - acl_id: "acl-xxxxxxxxx"
```

### 2. 编译和运行

#### 本地运行
```bash
# 安装依赖
go mod download

# 编译
go build -o cloud-whitelist-manager ./cmd/cloud-whitelist-manager

# 运行
./cloud-whitelist-manager --config config.yaml
```

#### Docker运行
```bash
# 构建镜像
docker build -t cloud-whitelist-manager .

# 运行容器
docker run -d \
  --name cloud-whitelist-manager \
  -v $(pwd)/config.yaml:/app/config.yaml \
  cloud-whitelist-manager
```

## 扩展性设计

本工具在设计时已考虑扩展性，未来可以轻松支持：

1. **更多阿里云服务**：如OSS、CDN等
2. **其他云平台**：如AWS、Azure等
3. **更多IP获取方式**：根据需求添加

## 安全考虑

1. 建议使用最小权限的阿里云RAM用户
2. AccessKey信息建议通过环境变量或Kubernetes Secret配置
3. 容器以非root用户运行

## 配置说明

### 阿里云凭证配置

要使用本工具，您需要配置阿里云凭证：

1. **获取AccessKey**：
   - 登录阿里云控制台
   - 进入"访问控制" -> "用户" -> "用户管理"
   - 创建新用户或选择现有用户
   - 在"AccessKey"标签页中创建AccessKey

2. **权限要求**：
   - ECS管理权限：授权安全组相关操作
   - RDS管理权限：授权白名单相关操作
   - Redis管理权限：授权白名单相关操作
   - SLB管理权限：授权访问控制相关操作

### 资源ID获取

1. **安全组ID**：
   - 阿里云控制台 -> 云服务器ECS -> 网络与安全 -> 安全组
   - 找到对应的安全组，复制ID（以sg-开头）

2. **RDS实例ID**：
   - 阿里云控制台 -> 云数据库RDS -> 实例列表
   - 找到对应的实例，复制ID（以rm-开头）

3. **Redis实例ID**：
   - 阿里云控制台 -> 云数据库Redis -> 实例列表
   - 找到对应的实例，复制ID（以r-开头）

4. **CLB实例ID**：
   - 阿里云控制台 -> 负载均衡 -> 实例管理
   - 找到对应的实例，复制ID（以lb-开头）

## 故障排除

### 常见问题

1. **无法获取IP**：检查网络连接和IP获取源配置
2. **阿里云API调用失败**：检查AccessKey权限和区域配置
3. **白名单更新失败**：检查实例ID和白名单分组名称
4. **资源未找到错误**：确认配置文件中的资源ID是否正确，是否与AccessKey所在账号匹配

### 日志查看

工具使用logrus记录日志，可以通过Docker logs或kubectl logs查看运行日志。

## 贡献

欢迎提交Issue和Pull Request来改进这个工具。