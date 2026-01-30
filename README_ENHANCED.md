# Easy Proxies Enhanced

基于 [jasonwong1991/easy_proxies](https://github.com/jasonwong1991/easy_proxies) 的增强版本。

## ✨ 新增功能

### 1. Web 端订阅管理
- 添加/编辑/删除订阅链接
- 刷新单个订阅
- 查看订阅下的所有节点

### 2. 智能分组
- **按订阅来源分组** - 区分不同机场的节点
- **按地区分组** - 自动识别节点所在国家/地区（支持中英文和 emoji 旗帜）
- **按延迟分组** - 低延迟(≤100ms) / 中延迟(100-300ms) / 高延迟(>300ms)

### 3. 节点状态管理
- 启用/禁用单个节点
- 拉黑节点（临时/永久）
- 批量操作

### 4. 动态测速分组
- 定时自动测速（可配置间隔，默认30分钟）
- 测速后自动重新分组

### 5. 代理池 API（重点功能）

#### 获取单个代理
```
GET /api/proxy/get
GET /api/proxy/get?latency=low          # 只获取低延迟节点
GET /api/proxy/get?region=US            # 只获取美国节点
GET /api/proxy/get?sub=机场A            # 只获取指定订阅的节点
GET /api/proxy/get?latency=low&region=JP # 组合过滤
GET /api/proxy/get?format=json          # 返回 JSON 格式
GET /api/proxy/get?key=你的API密钥       # API 认证
```

返回示例（纯文本）：
```
http://127.0.0.1:24001
```

返回示例（JSON）：
```json
{
  "proxy": "http://127.0.0.1:24001",
  "name": "🇯🇵 日本节点",
  "region": "JP",
  "latency": 85,
  "latency_level": "low"
}
```

#### 获取代理列表
```
GET /api/proxy/list
GET /api/proxy/list?latency=low&limit=10
```

#### 获取统计信息
```
GET /api/proxy/stats
```

### 6. 轮询策略增强

| 模式 | 说明 |
|------|------|
| `sequential` | 顺序轮询（原版） |
| `random` | 随机选择 |
| `latency_first` | 延迟优先（推荐）- 优先使用低延迟节点 |
| `weighted` | 加权轮询 - 低延迟节点权重高 |

### 7. API 认证
可选的 API Key 保护，防止代理池被滥用。

---

## 🚀 快速开始

### Docker 部署

```bash
docker run -d \
  --name easy-proxies \
  -p 9090:9090 \
  -p 2323:2323 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/data:/app/data \
  easy-proxies:enhanced
```

### 配置示例

```yaml
mode: pool
log_level: info

management:
  enabled: true
  listen: 0.0.0.0:9090
  probe_target: www.apple.com:80
  password: ""  # Web 管理密码

pool:
  mode: latency_first  # 延迟优先模式
  failure_threshold: 3
  blacklist_duration: 10m

# 延迟分组配置
latency_groups:
  low_threshold: 100      # ≤100ms 为低延迟
  medium_threshold: 300   # ≤300ms 为中延迟

# 自动测速
auto_speedtest:
  enabled: true
  interval: 30m

# API 认证（可选）
api_auth:
  enabled: false
  key: "your-secret-key"

# 订阅配置
subscriptions:
  - name: "机场A"
    url: "https://example.com/subscribe"
    enabled: true
    refresh_interval: 1h
```

---

## 📖 与 Grok2API 配合使用

在 Grok2API 的 `Proxy Pool URL` 中填入：

```
# 使用所有可用节点
http://你的服务器:9090/api/proxy/get

# 只使用高速节点（推荐）
http://你的服务器:9090/api/proxy/get?latency=low

# 只使用日本节点
http://你的服务器:9090/api/proxy/get?region=JP

# 带 API Key
http://你的服务器:9090/api/proxy/get?latency=low&key=你的密钥
```

---

## 🔌 API 端点完整列表

### 代理池 API
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/proxy/get` | 获取一个可用代理 |
| GET | `/api/proxy/list` | 获取代理列表 |
| GET | `/api/proxy/stats` | 获取统计信息 |

### 订阅管理 API
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/subscriptions` | 获取所有订阅 |
| POST | `/api/subscriptions` | 添加订阅 |
| GET | `/api/subscriptions/:id` | 获取单个订阅 |
| PUT | `/api/subscriptions/:id` | 更新订阅 |
| DELETE | `/api/subscriptions/:id` | 删除订阅 |
| POST | `/api/subscriptions/:id/refresh` | 刷新订阅 |
| POST | `/api/subscriptions/:id/toggle` | 启用/禁用订阅 |

### 节点状态 API
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/api/nodes/status/:name/enable` | 启用节点 |
| POST | `/api/nodes/status/:name/disable` | 禁用节点 |
| POST | `/api/nodes/status/:name/blacklist` | 拉黑节点 |

### 分组查询 API
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/groups/latency` | 按延迟分组 |
| GET | `/api/groups/region` | 按地区分组 |
| GET | `/api/groups/subscription` | 按订阅分组 |

---

## 📝 更新日志

### v2.0.0 (Enhanced)
- ✨ 新增 Web 端订阅管理
- ✨ 新增智能分组（按订阅/地区/延迟）
- ✨ 新增节点状态管理
- ✨ 新增代理池 API（按条件筛选）
- ✨ 新增多种轮询策略
- ✨ 新增 API 认证
- ✨ 新增动态测速自动分组
- 🔧 优化延迟优先轮询算法
- 🐛 修复各种问题

---

## 🙏 致谢

- [jasonwong1991/easy_proxies](https://github.com/jasonwong1991/easy_proxies) - 原项目
- [sagernet/sing-box](https://github.com/sagernet/sing-box) - 代理核心
