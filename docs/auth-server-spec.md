# Flowmi CLI Login 流程详解 — Auth Server 实现指南

本文档详细描述了 flowmi CLI 的 OAuth2 PKCE 登录流程，目的是让另一个 agent 据此生成正确的 auth server 端代码。

---

## 1. 整体时序图

```
CLI                          Browser                    Auth Server
 |                              |                           |
 |-- 生成 PKCE (verifier+challenge) --|                     |
 |-- 生成 state -------------------|                        |
 |-- 启动 localhost:PORT/callback --|                        |
 |-- 构造 authorize URL ---------->|                        |
 |                              |--- GET /authorize ------->|
 |                              |<-- 显示登录页面 ------------|
 |                              |--- 用户登录+授权 --------->|
 |                              |<-- 302 Redirect ----------|
 |                              |     Location: http://127.0.0.1:PORT/callback?code=XXX&state=YYY
 |<-- 收到 code+state ----------|                           |
 |-- 验证 state 一致 -----------|                            |
 |-- POST /token ---------------------------------------------->|
 |<-- { access_token, refresh_token, ... } --------------------|
 |-- 保存 token 到 ~/.config/flowmi/credentials.toml          |
```

---

## 2. Auth Server 需要实现的两个端点

### 2.1 `GET /authorize` — 授权端点

**用途**: CLI 打开浏览器访问此 URL，用户在这里完成登录和授权。

**接收的 Query 参数**:

| 参数 | 类型 | 示例值 | 说明 |
|------|------|--------|------|
| `client_id` | string | `flowmi-cli` | 固定值，唯一客户端标识 |
| `redirect_uri` | string | `http://127.0.0.1:54321/callback` | CLI 本地回调地址，端口动态 |
| `response_type` | string | `code` | 固定值，OAuth2 授权码模式 |
| `state` | string | base64url 编码的 16 字节随机值 | CSRF 保护，必须原样回传 |
| `code_challenge` | string | base64url(SHA256(verifier)) | PKCE challenge |
| `code_challenge_method` | string | `S256` | 固定值，PKCE 使用 SHA256 |

**Server 处理逻辑**:
1. 验证 `client_id` 是否为已注册客户端（当前仅 `flowmi-cli`）
2. 验证 `response_type` = `code`
3. 验证 `code_challenge_method` = `S256`
4. 保存 `code_challenge`、`redirect_uri`、`state` 关联到本次请求（用于后续 /token 验证）
5. 显示登录/授权页面给用户
6. 用户登录成功后，生成 authorization code
7. 302 重定向到 `redirect_uri`，附加 `code` 和 `state` 参数

**成功响应**: HTTP 302 重定向
```
HTTP/1.1 302 Found
Location: http://127.0.0.1:54321/callback?code=AUTHORIZATION_CODE&state=ORIGINAL_STATE
```

**错误响应**: 重定向回 `redirect_uri` 并附加 `error` 参数
```
HTTP/1.1 302 Found
Location: http://127.0.0.1:54321/callback?error=access_denied
```

**关键约束**:
- `redirect_uri` 的 scheme 必须是 `http`，host 必须是 `127.0.0.1`，path 必须是 `/callback`，端口是动态的
- `state` 值必须原样回传，CLI 会严格比对
- authorization code 应该是一次性的，且有短暂有效期（建议 5-10 分钟）

---

### 2.2 `POST /token` — 令牌交换端点

**用途**: CLI 用 authorization code 换取 access_token 和 refresh_token。

**请求格式**:
- **Method**: `POST`
- **Content-Type**: `application/x-www-form-urlencoded`
- **无 Authorization header**（公开客户端，不使用 client_secret）

**请求 Body 参数**:

| 参数 | 类型 | 示例值 | 说明 |
|------|------|--------|------|
| `grant_type` | string | `authorization_code` | 固定值 |
| `code` | string | 授权码 | /authorize 生成的一次性授权码 |
| `code_verifier` | string | base64url 编码的 32 字节随机值 | PKCE 原始验证器 |
| `redirect_uri` | string | `http://127.0.0.1:54321/callback` | 必须与 /authorize 请求时相同 |
| `client_id` | string | `flowmi-cli` | 固定值 |

**Server 处理逻辑**:
1. 验证 `grant_type` = `authorization_code`
2. 验证 `client_id` = `flowmi-cli`
3. 查找并验证 `code` 有效且未被使用
4. 验证 `redirect_uri` 与授权时记录的一致
5. **PKCE 验证**: 计算 `base64url(SHA256(code_verifier))`，对比 /authorize 时保存的 `code_challenge`
6. 作废 authorization code（一次性使用）
7. 生成 access_token 和 refresh_token
8. 返回 JSON 响应

**PKCE 验证伪代码**:
```
computed_challenge = base64url_encode(SHA256(code_verifier))
if computed_challenge != stored_code_challenge:
    return 400 error
```

注意: base64url 编码是 **不带 padding** 的（即不带尾部 `=` 号），对应 Go 的 `base64.RawURLEncoding`。

**成功响应**: HTTP 200
```json
{
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g...",
    "token_type": "Bearer",
    "expires_in": 3600
}
```

**JSON 字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_token` | string | 用于 API 认证的 Bearer token |
| `refresh_token` | string | 用于刷新 access_token |
| `token_type` | string | 固定值 `Bearer` |
| `expires_in` | int | access_token 有效期（秒） |

**错误响应**: 非 200 状态码。CLI 只检查 `resp.StatusCode != 200`，不解析错误 body。但建议返回标准 OAuth2 错误格式:
```json
{
    "error": "invalid_grant",
    "error_description": "Authorization code has expired"
}
```

---

## 3. PKCE 细节

CLI 端的 PKCE 生成方式（server 验证时需要理解）:

```go
// code_verifier: 32 字节随机数 → base64url (无 padding)
buf := make([]byte, 32)  // 32 bytes of crypto/rand
verifier = base64.RawURLEncoding.EncodeToString(buf)
// verifier 长度 = 43 字符

// code_challenge: SHA256(verifier 字符串) → base64url (无 padding)
h := sha256.Sum256([]byte(verifier))  // 注意: hash 的是 verifier 的字符串形式，不是原始字节
challenge = base64.RawURLEncoding.EncodeToString(h[:])
// challenge 长度 = 43 字符
```

**验证时**:
```
server 在 /authorize 时保存: code_challenge
server 在 /token 时收到: code_verifier

验证逻辑:
  SHA256(code_verifier) → base64url 编码 → 对比 code_challenge
```

---

## 4. 关键配置

| 项目 | 值 |
|------|-----|
| 默认 Auth Server URL | `https://auth.flowmi.ai` |
| Client ID | `flowmi-cli` |
| 公开客户端 | 是（无 client_secret） |
| 授权端点路径 | `/authorize` |
| 令牌端点路径 | `/token` |
| redirect_uri 格式 | `http://127.0.0.1:{dynamic_port}/callback` |
| PKCE method | `S256` |
| CLI 超时 | 2 分钟 |

---

## 5. CLI 端回调服务器行为

Server 实现者需要了解 CLI 回调端如何工作（以确保正确对接）:

- 监听地址: `127.0.0.1:0`（系统分配随机端口）
- 回调路径: `/callback`
- 接受的 query 参数: `code`、`state`、`error`
- 成功时: 收到 `code` + `state`，返回 200 + 成功 HTML 页面
- 失败时: 缺少 `code` 或收到 `error` 参数，返回 400 + 错误 HTML 页面
- 回调服务器只处理一次请求后即关闭

---

## 6. Token 存储

CLI 将 token 保存到 `~/.config/flowmi/credentials.toml`（或 `$XDG_CONFIG_HOME/flowmi/credentials.toml`），权限 0600:

```toml
access_token = "eyJhbGciOiJSUzI1NiIs..."
refresh_token = "dGhpcyBpcyBhIHJlZnJlc2g..."
```

应用设置（如 `auth_server_url`）保存在同目录下的 `config.toml`:

```toml
auth_server_url = "https://auth.flowmi.ai"
```

只保存 `access_token` 和 `refresh_token`，不保存 `token_type` 和 `expires_in`。

---

## 7. 源文件参考

| 文件 | 说明 |
|------|------|
| `internal/auth/oauth.go` | PKCE、state 生成，回调服务器，token 交换，authorize URL 构造 |
| `internal/auth/oauth_test.go` | 各函数的单元测试，含 mock server 示例 |
| `cmd/login.go` | login 命令入口，串联整个流程 |
| `cmd/login_test.go` | 集成测试，含完整 mock auth server 示例 |
| `cmd/root.go` | 配置初始化，`auth_server_url` 默认值 |
