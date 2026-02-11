# sysstat-web

一个基于 Go 的 Web 应用，用于可视化 sysstat（sar）数据。提供实时系统监控图表，包括 CPU、内存、磁盘、网络、负载和 I/O 统计。

## 功能特性

- **实时数据可视化**：通过 Web 界面展示系统性能指标
- **时间序列图表**：支持历史数据查询和趋势分析
- **认证保护**：密码保护的访问控制
- **RESTful API**：提供 JSON API 接口
- **时区支持**：可配置时区偏移

## 安装

### 前置要求

- Go 1.25 或更高版本
- sysstat 工具（sar 命令）
- Linux 系统（sysstat 数据通常在 `/var/log/sysstat`）

### 构建

```bash
git clone <repository-url>
cd sysstat-web
go build -o sysstat-web
```

## 配置

复制示例配置文件并修改：

```bash
cp config_example.json config.json
```

编辑 `config.json`：

```json
{
  "sysstat_dir": "/var/log/sysstat",
  "port": "8080",
  "password": "your_secure_password",
  "tz_offset_hours": 8
}
```

配置说明：
- `sysstat_dir`: sysstat 数据文件目录
- `port`: Web 服务器监听端口
- `password`: 访问密码
- `tz_offset_hours`: 时区偏移（小时）

## 运行

```bash
./sysstat-web
```

或使用环境变量指定配置文件：

```bash
CONFIG_FILE=/path/to/config.json ./sysstat-web
```

访问 `http://localhost:<port>` 并使用配置中的密码登录。

## API 接口

### 获取最新数据
```
GET /api/latest
```

### 获取时间序列数据
```
GET /api/timeseries?hours=24
```

### 刷新数据
```
POST /api/refresh
```

## 数据采样

项目包含采样脚本 `scripts/sample_sar.sh`，用于定期收集 sysstat 数据：

```bash
./scripts/sample_sar.sh [output_file]
```

## 开发

### 项目结构

- `main.go`: 程序入口
- `handlers.go`: HTTP 处理器和路由
- `config.go`: 配置加载
- `sampler.go`: 数据采样逻辑
- `sar.go`: sar 数据解析
- `timeseries.go`: 时间序列处理
- `utils.go`: 工具函数
- `web/`: 前端静态文件

### 依赖

无外部 Go 依赖，仅使用标准库。

## 许可证

本项目采用 GPL v2 许可证。详见 [LICENSE](LICENSE) 文件。