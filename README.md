# 独角数卡迁移工具

从老版独角数卡 (dujiaoka) 迁移数据到 [dujiao-next](https://github.com/dujiao-next/dujiao-next) 的命令行工具。

## 功能特性

- 🔄 迁移分类、商品、卡密数据
- 🀄 中文名称自动转拼音生成 slug（基于 go-pinyin）
- 🔤 UTF-8 编码正确处理，中文零乱码
- 📷 支持本地图片自动上传迁移
- 🔁 增量迁移，跳过已存在数据，可重复运行
- 🏷️ slug 冲突自动加后缀重试
- 📦 卡密批量导入（默认 500 条/批）
- ⚙️ 支持命令行参数和 YAML 配置文件两种方式
- 🐳 支持 Docker 编译，无需本地安装 Go 环境

## 快速开始

### 从源码编译

```bash
git clone https://github.com/luoyanglang/dujiao-migrate.git
cd dujiao-migrate
go build -o dujiao-migrate main.go
```

### Docker 编译（无需本地 Go 环境）

```bash
docker run --rm -v $(pwd):/app -w /app golang:1.21 sh -c 'go mod tidy && go build -o dujiao-migrate main.go'
```

### 下载预编译二进制

从 [Releases](https://github.com/luoyanglang/dujiao-migrate/releases) 页面下载对应平台的二进制文件。

## 使用方法

### 命令行参数方式

```bash
./dujiao-migrate \
  --old-host 127.0.0.1 \
  --old-port 3306 \
  --old-user root \
  --old-password your_password \
  --old-database dujiaoka \
  --new-api http://127.0.0.1:8080/api/v1/admin \
  --new-user admin \
  --new-password admin123
```

### 配置文件方式

```bash
# 生成示例配置
./dujiao-migrate --generate-config > config.yaml

# 编辑配置后执行
./dujiao-migrate --config config.yaml
```

### 图片迁移

如果老版站点在同一台服务器上，可以指定站点路径自动上传图片：

```bash
./dujiao-migrate \
  --old-host 127.0.0.1 \
  --old-user root \
  --old-password your_password \
  --old-database dujiaoka \
  --new-api http://127.0.0.1:8080/api/v1/admin \
  --new-user admin \
  --new-password admin123 \
  --old-site-path /www/wwwroot/dujiaoka
```

工具会自动在 `public/`、`public/storage/` 等目录下查找图片文件并上传到新版 API。

## 配置文件示例

```yaml
# 老版数据库配置
old_db:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_password"
  database: "dujiaoka"
  charset: "utf8mb4"

# 新版 API 配置
new_api:
  base_url: "http://127.0.0.1:8080/api/v1/admin"
  username: "admin"
  password: "admin123"

# 迁移选项
options:
  retry_times: 3
  retry_delay: 1
  skip_existing: true
  migrate_cards: true
  only_active: true
  batch_size: 500
  old_site_path: ""
```

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--config` | 配置文件路径 | - |
| `--generate-config` | 生成示例配置文件 | - |
| `--version` | 显示版本信息 | - |
| `--old-driver` | 数据库驱动 (mysql/postgres/sqlite) | mysql |
| `--old-host` | 数据库主机 | - |
| `--old-port` | 数据库端口 | - |
| `--old-user` | 数据库用户名 | - |
| `--old-password` | 数据库密码 | - |
| `--old-database` | 数据库名 | - |
| `--new-api` | 新版 API 地址 | - |
| `--new-user` | 管理员用户名 | - |
| `--new-password` | 管理员密码 | - |
| `--old-site-path` | 老版站点路径（图片迁移） | - |
| `--no-skip` | 不跳过已存在的数据 | false |
| `--no-cards` | 不迁移卡密 | false |

## 迁移流程

1. 连接老版 MySQL 数据库
2. 登录新版 dujiao-next 管理后台 API
3. 迁移分类 → 中文名自动转拼音 slug
4. 迁移商品 → 关联分类、处理标签/图片/表单配置
5. 迁移卡密 → 批量导入
6. 输出统计报告

## 项目结构

```
dujiao-migrate/
├── main.go                     # 入口
├── go.mod
├── go.sum
├── config.example.yaml         # 配置示例
├── Dockerfile
├── Makefile
└── internal/
    ├── api/client.go           # API 客户端（登录、创建、上传）
    ├── config/config.go        # 配置管理
    ├── database/database.go    # 数据库连接
    ├── migrator/migrator.go    # 迁移核心逻辑
    ├── models/models.go        # 数据模型
    └── utils/utils.go          # 工具函数（拼音转换等）
```

## 注意事项

- 迁移前请备份新版数据库
- 建议先在测试环境验证
- 支持多次运行，自动跳过已存在数据
- 大量卡密导入可能需要较长时间

## 许可证

GPL-3.0 License

## 作者

狼哥 ([@luoyanglang](https://github.com/luoyanglang)) | Telegram: [@luoyanglang](https://t.me/luoyanglang)
