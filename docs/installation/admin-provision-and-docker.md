# 管理员用户开通接口与本地 Docker 镜像说明

本文档说明两部分内容：

1. 新增的管理员接口：
   - 按用户名创建用户，并返回完整 key
   - 按用户名查询该用户当前 key
2. 本地构建 `new-api` Docker 镜像，并保留为后续可直接推送的标签：
   - `qianlan/new-api:latest`

## 1. 接口概览

新增了两个管理员接口：

- `POST /api/user/provision`
- `POST /api/user/query_keys`

路由位置：

- `router/api-router.go`

控制器位置：

- `controller/user_admin_token.go`

服务实现位置：

- `service/user_provision.go`

## 2. 鉴权方式

这两个接口挂在 `AdminAuth()` 下，不是普通用户 token 鉴权。

调用时需要满足以下任一条件：

- 已登录后台，携带管理员会话 Cookie
- 使用管理员 `access_token`

并且无论哪种方式，都必须额外带上请求头：

```text
New-Api-User: <管理员用户ID>
```

如果使用 `access_token`，请求头示例：

```text
Authorization: <admin_access_token>
New-Api-User: <admin_user_id>
Content-Type: application/json
```

## 3. 创建用户并返回完整 key

### 接口

```http
POST /api/user/provision
```

### 入参

```json
{
  "username": "alice"
}
```

### 默认值

- 登录密码：`123456`
- 用户额度：`10000`
- key 名称：`test`
- key 额度：`10000`

### 返回示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "user_id": 12,
    "username": "alice",
    "password": "123456",
    "quota": 10000,
    "token_id": 34,
    "token_name": "test",
    "key": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

### curl 示例

```bash
curl -X POST 'http://127.0.0.1:3000/api/user/provision' \
  -H 'Authorization: YOUR_ADMIN_ACCESS_TOKEN' \
  -H 'New-Api-User: YOUR_ADMIN_USER_ID' \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice"
  }'
```

## 4. 按用户名查询 key

### 接口

```http
POST /api/user/query_keys
```

### 入参

```json
{
  "username": "alice"
}
```

### 返回说明

- `key`：该用户最新创建的 key
- `tokens`：该用户当前全部 key，返回完整 key，不做掩码

### 返回示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "user_id": 12,
    "username": "alice",
    "key": "latest_full_key",
    "tokens": [
      {
        "id": 35,
        "name": "test",
        "key": "latest_full_key",
        "remain_quota": 10000,
        "created_time": 1746691200
      },
      {
        "id": 34,
        "name": "old-key",
        "key": "old_full_key",
        "remain_quota": 5000,
        "created_time": 1746690000
      }
    ]
  }
}
```

### curl 示例

```bash
curl -X POST 'http://127.0.0.1:3000/api/user/query_keys' \
  -H 'Authorization: YOUR_ADMIN_ACCESS_TOKEN' \
  -H 'New-Api-User: YOUR_ADMIN_USER_ID' \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice"
  }'
```

## 5. 本地构建 Docker 镜像

仓库根目录直接执行：

```bash
docker build -t qianlan/new-api:latest .
```

这个标签已经适合后续直接推送：

```bash
docker push qianlan/new-api:latest
```

前提是你已经完成：

```bash
docker login
```

## 6. 验证镜像是否构建成功

```bash
docker images qianlan/new-api:latest
```

或者：

```bash
docker image inspect qianlan/new-api:latest
```

## 7. 本地启动示例

如果用 SQLite，本地建议挂载 `/data`：

```bash
docker run -d \
  --name new-api-local \
  -p 3000:3000 \
  -v "$(pwd)/data:/data" \
  -e SESSION_SECRET='replace-this-session-secret' \
  -e CRYPTO_SECRET='replace-this-crypto-secret' \
  qianlan/new-api:latest
```

启动后可访问：

```text
http://127.0.0.1:3000
```

## 8. 备注

- 正式推送前，建议先本地启动镜像验证页面和接口是否正常。
- 如果后续你希望把接口文档继续并入 `docs/openapi/api.json`，可以再补一版 OpenAPI 描述。
