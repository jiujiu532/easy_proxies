# Easy Proxies 增强版 ✅ 已完成

> 最后更新: 2026-01-30

## 已实现功能

### 1.1 Web 端订阅管理
- [x] 添加订阅链接
- [x] 编辑订阅链接
- [x] 删除订阅链接
- [x] 刷新单个订阅
- [x] 查看订阅下的所有节点

### 1.2 智能分组
- [x] 按订阅来源分组
- [x] 按地区分组（自动识别国家/地区）
- [x] 按延迟分组（低≤100ms / 中100-300ms / 高>300ms）

### 1.3 节点状态管理
- [x] 启用/禁用单个节点
- [x] 拉黑节点（临时/永久）
- [x] 批量操作

### 1.4 动态测速分组
- [x] 定时测速（可配置间隔，默认30分钟）
- [x] 自动重新分组

### 1.5 按分组导出 API
- [x] `/api/proxy/get` - 获取一个可用代理
- [x] `/api/proxy/get?latency=low` - 获取低延迟节点
- [x] `/api/proxy/get?region=US` - 获取指定地区节点
- [x] `/api/proxy/get?sub=订阅名` - 获取指定订阅的节点
- [x] 支持多条件组合过滤

### 1.6 轮询策略增强
- [x] sequential - 顺序轮询（原版）
- [x] random - 随机选择
- [x] latency_first - 延迟优先
- [x] weighted - 加权轮询

### 1.7 API 认证
- [x] 可选的 API Key 保护

---

## 2. 数据结构设计

### 2.1 订阅表 (Subscription)
```go
type Subscription struct {
    ID              string    `json:"id"`
    Name            string    `json:"name"`
    URL             string    `json:"url"`
    Enabled         bool      `json:"enabled"`
    RefreshInterval string    `json:"refresh_interval"` // e.g., "1h", "30m"
    LastRefreshAt   time.Time `json:"last_refresh_at"`
    NodeCount       int       `json:"node_count"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

### 2.2 节点增强 (NodeConfig 扩展)
```go
type NodeConfigEnhanced struct {
    config.NodeConfig
    SubscriptionID   string    `json:"subscription_id,omitempty"`   // 所属订阅
    SubscriptionName string    `json:"subscription_name,omitempty"` // 订阅名称
    Region           string    `json:"region,omitempty"`            // 地区代码
    RegionName       string    `json:"region_name,omitempty"`       // 地区名称
    Latency          int64     `json:"latency"`                     // 当前延迟(ms)
    LatencyLevel     string    `json:"latency_level"`               // low/medium/high
    Status           string    `json:"status"`                      // enabled/disabled/blacklisted
    LastCheckAt      time.Time `json:"last_check_at"`
}
```

### 2.3 配置扩展
```yaml
# 新增配置项
subscriptions:
  - id: "sub_001"
    name: "机场A"
    url: "https://example.com/subscribe"
    enabled: true
    refresh_interval: "1h"

pool:
  mode: latency_first  # sequential | random | latency_first | weighted
  failure_threshold: 3
  blacklist_duration: 10m

latency_groups:
  low_threshold: 100     # ≤100ms 为低延迟
  medium_threshold: 300  # ≤300ms 为中延迟

auto_speedtest:
  enabled: true
  interval: 30m

api_auth:
  enabled: false
  key: ""  # API Key，为空则不需要认证
```

---

## 3. 新增 API 端点

### 3.1 订阅管理
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | /api/subscriptions | 获取所有订阅 |
| POST | /api/subscriptions | 添加订阅 |
| PUT | /api/subscriptions/:id | 更新订阅 |
| DELETE | /api/subscriptions/:id | 删除订阅 |
| POST | /api/subscriptions/:id/refresh | 刷新单个订阅 |

### 3.2 节点状态管理
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | /api/nodes/:tag/disable | 禁用节点 |
| POST | /api/nodes/:tag/enable | 启用节点 |
| POST | /api/nodes/:tag/blacklist | 拉黑节点 |

### 3.3 代理池 API
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | /api/proxy/get | 获取一个代理 |
| GET | /api/proxy/list | 获取代理列表 |

### 3.4 分组查询
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | /api/groups/latency | 按延迟分组 |
| GET | /api/groups/region | 按地区分组 |
| GET | /api/groups/subscription | 按订阅分组 |

---

## 4. 实现步骤

### Phase 1: 数据结构和存储
1. 创建 `internal/store/store.go` - 数据存储层
2. 创建 `internal/store/subscription.go` - 订阅存储
3. 修改 `internal/config/config.go` - 扩展配置

### Phase 2: 订阅管理
1. 创建 `internal/subscription/subscription.go` - 订阅服务
2. 修改 `internal/monitor/server.go` - 添加订阅 API

### Phase 3: 节点增强
1. 创建 `internal/node/region.go` - 地区识别
2. 创建 `internal/node/latency.go` - 延迟分组
3. 修改节点管理逻辑

### Phase 4: 代理池 API
1. 创建 `internal/proxypool/pool.go` - 代理池服务
2. 添加轮询策略
3. 添加 API 端点

### Phase 5: Web 界面
1. 修改 `internal/monitor/assets/index.html` - 添加新界面

---

## 5. 文件修改清单

### 新增文件
- `internal/store/store.go`
- `internal/store/subscription.go`
- `internal/node/region.go`
- `internal/node/latency.go`
- `internal/proxypool/pool.go`

### 修改文件
- `internal/config/config.go`
- `internal/monitor/server.go`
- `internal/monitor/manager.go`
- `internal/monitor/assets/index.html`
- `internal/subscription/manager.go`
- `config.example.yaml`
