# CoreX JWT认证模块使用指南

## 🎯 概述

CoreX的JWT认证模块提供了完整的JWT令牌生成、验证、中间件集成和黑名单管理功能。基于高性能的`golang-jwt/jwt/v5`库，支持HMAC-SHA256签名算法，适用于微服务架构中的身份认证和授权。

## 🚀 快速开始

### 1. 初始化认证系统

在应用启动时初始化认证系统（通常在main.go中）：

```go
import (
    "github.com/ArtisanCloud/CoreX/config"
    "github.com/ArtisanCloud/CoreX/pkg/auth"
)

func main() {
    // 加载配置
    cfg, err := config.Load("config/example.yaml")
    if err != nil {
        panic(err)
    }
    
    // 设置JWT密钥
    auth.SetJWTSecret([]byte(cfg.Auth.JWTSecret))
    
    // 或者使用Init()函数自动初始化
    if err := auth.Init(); err != nil {
        panic(err)
    }
}
```

### 2. 在Gin中使用JWT中间件

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/ArtisanCloud/CoreX/pkg/auth"
)

func SetupRouter() *gin.Engine {
    r := gin.Default()
    
    // 公开路由
    r.GET("/health", healthHandler)
    r.POST("/auth/login", loginHandler)
    
    // 受保护的路由组
    protected := r.Group("/api")
    protected.Use(auth.JwtMiddleware(
        "admin",                    // 期望的受众
        []string{"user:read"},      // 必需的权限范围
        auth.SampleCallback,        // 可选的回调函数
    ))
    
    protected.GET("/users", getUsersHandler)
    protected.POST("/users", createUserHandler)
    
    return r
}
```

## 📝 核心功能详解

### 1. JWT令牌生成

#### 基本令牌生成

```go
import (
    "time"
    "github.com/ArtisanCloud/CoreX/pkg/auth"
)

func generateToken() {
    token, err := auth.GenerateJWT(
        "tenant-001",           // 租户ID
        "user-123",            // 用户ID
        "web",                 // 平台标识
        "admin",               // 受众
        "user:read,user:write", // 权限范围
        24*time.Hour,          // 有效期
        []byte("your-secret"), // JWT密钥
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println("生成的JWT令牌:", token)
}
```

#### 带JTI的令牌生成（支持撤销）

```go
import "github.com/google/uuid"

func generateTokenWithJTI() {
    jti := uuid.New().String() // 生成唯一ID
    
    token, err := auth.GenerateJWTWithJTI(
        "tenant-001",           // 租户ID
        "user-123",            // 用户ID
        "web",                 // 平台标识
        "admin",               // 受众
        "user:read,user:write", // 权限范围
        jti,                   // JWT ID
        24*time.Hour,          // 有效期
        []byte("your-secret"), // JWT密钥
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("生成的JWT令牌: %s\nJTI: %s\n", token, jti)
}
```

### 2. JWT中间件使用

#### 基本中间件配置

```go
// 简单的JWT验证中间件
authMiddleware := auth.JwtMiddleware(
    "admin",                // 期望受众
    []string{"api:access"}, // 必需权限
    nil,                    // 无回调函数
)

// 应用到路由
router.Use(authMiddleware)
```

#### 带回调函数的中间件

```go
// 自定义认证回调函数
func customAuthCallback(ctx context.Context, claims *auth.CoreXClaims) error {
    // 可以在这里进行额外的验证逻辑
    tenantID := claims.TenantID
    
    // 检查租户是否有效
    if !isValidTenant(tenantID) {
        return fmt.Errorf("无效的租户: %s", tenantID)
    }
    
    // 记录认证事件
    logAuthEvent(claims)
    
    return nil
}

// 使用自定义回调的中间件
authMiddleware := auth.JwtMiddleware(
    "admin",
    []string{"api:access"},
    customAuthCallback,
)
```

### 3. 从上下文获取认证信息

在处理函数中获取JWT中的信息：

```go
func userHandler(c *gin.Context) {
    ctx := c.Request.Context()
    
    // 获取基本信息
    tenantID := auth.GetTenantID(ctx)
    userID := auth.GetSubject(ctx)
    platform := auth.GetPlatform(ctx)
    scope := auth.GetScope(ctx)
    traceID := auth.GetTraceID(ctx)
    
    // 获取完整的JWT声明
    claims := auth.GetJWTClaims(ctx)
    if claims != nil {
        fmt.Printf("完整声明: %+v\n", claims)
    }
    
    c.JSON(200, gin.H{
        "tenant_id": tenantID,
        "user_id":   userID,
        "platform":  platform,
        "scope":     scope,
        "trace_id":  traceID,
    })
}
```

### 4. JWT令牌撤销（黑名单）

#### 撤销令牌

```go
import "time"

func revokeToken(jti string) {
    // 将令牌加入黑名单，设置TTL为令牌的剩余有效期
    auth.Revoke(jti, 24*time.Hour)
    
    fmt.Printf("令牌 %s 已被撤销\n", jti)
}
```

#### 检查令牌是否被撤销

```go
func checkTokenStatus(jti string) {
    if auth.IsRevoked(jti) {
        fmt.Printf("令牌 %s 已被撤销\n", jti)
    } else {
        fmt.Printf("令牌 %s 仍然有效\n", jti)
    }
}
```

#### 监控黑名单大小

```go
func monitorBlacklist() {
    size := auth.GetBlacklistSize()
    fmt.Printf("当前黑名单中有 %d 个令牌\n", size)
}
```

## ⚙️ 配置说明

### 统一配置文件

在 `config/example.yaml` 中配置JWT认证：

```yaml
auth:
  jwt_secret: "K8mN2pQ7rS9tU4vW6xY1zA3bC5dE8fG0"  # 32位安全密钥
  expected_audience: "admin"                        # 默认受众
  required_scopes: ["flow:execute"]                 # 默认权限范围
  token_ttl_hours: 24                              # 默认令牌有效期(小时)
```

### 环境变量配置

```bash
# 新格式环境变量
export CORE_X_AUTH_JWT_SECRET="your-32-char-secret-key"
export CORE_X_AUTH_EXPECTED_AUDIENCE="admin"
export CORE_X_AUTH_REQUIRED_SCOPES="user:read,user:write"
export CORE_X_AUTH_TOKEN_TTL_HOURS=24

# 兼容旧格式
export CORE_X_JWT_SECRET="your-secret-key"
```

## 🔒 JWT声明结构

CoreX使用自定义的JWT声明结构：

```go
type CoreXClaims struct {
    TenantID string           `json:"tenant_id"` // 租户ID
    Platform string           `json:"platform"`  // 平台标识(web/mobile/api)
    Scope    string           `json:"scope"`     // 权限范围(逗号分隔)
    Audience jwt.ClaimStrings `json:"aud"`       // 受众
    Subject  string           `json:"sub"`       // 主体(通常是用户ID)
    jwt.RegisteredClaims                         // 标准JWT声明
}
```

### JWT令牌示例

解码后的JWT载荷：

```json
{
  "tenant_id": "tenant-001",
  "platform": "web",
  "scope": "user:read,user:write,admin:access",
  "aud": ["admin"],
  "sub": "user-123",
  "iss": "corex-auth",
  "exp": 1703980800,
  "iat": 1703894400,
  "nbf": 1703894400,
  "jti": "550e8400-e29b-41d4-a716-446655440000"
}
```

## 🌟 最佳实践

### 1. 安全建议

#### JWT密钥管理
```go
// ✅ 推荐：使用强密钥
const MinSecretLength = 32
if len(secret) < MinSecretLength {
    return fmt.Errorf("JWT密钥长度至少需要%d个字符", MinSecretLength)
}

// ✅ 推荐：从环境变量读取密钥
secret := os.Getenv("CORE_X_AUTH_JWT_SECRET")
if secret == "" {
    return fmt.Errorf("必须设置CORE_X_AUTH_JWT_SECRET环境变量")
}
```

#### 权限范围设计
```go
// ✅ 推荐：细粒度权限
scopes := []string{
    "user:read",      // 读取用户信息
    "user:write",     // 修改用户信息
    "admin:access",   // 管理员访问
    "api:call",       // API调用权限
}

// ❌ 不推荐：过于宽泛的权限
scopes := []string{"*", "all", "admin"}
```

### 2. 错误处理

```go
func handleAuthError(c *gin.Context, err error) {
    switch {
    case strings.Contains(err.Error(), "token is expired"):
        c.JSON(401, gin.H{
            "error": "令牌已过期",
            "code":  "TOKEN_EXPIRED",
        })
    case strings.Contains(err.Error(), "权限不足"):
        c.JSON(403, gin.H{
            "error": "权限不足",
            "code":  "INSUFFICIENT_PERMISSIONS",
        })
    default:
        c.JSON(401, gin.H{
            "error": "认证失败",
            "code":  "AUTH_FAILED",
        })
    }
}
```

### 3. 令牌刷新策略

```go
func refreshTokenIfNeeded(claims *auth.CoreXClaims) (string, error) {
    // 如果令牌在1小时内过期，生成新令牌
    if time.Until(claims.ExpiresAt.Time) < time.Hour {
        return auth.GenerateJWT(
            claims.TenantID,
            claims.Subject,
            claims.Platform,
            claims.Audience[0],
            claims.Scope,
            24*time.Hour,
            jwtSecret,
        )
    }
    return "", nil
}
```

### 4. 多租户支持

```go
func tenantAwareHandler(c *gin.Context) {
    tenantID := auth.GetTenantID(c.Request.Context())
    
    // 基于租户ID进行数据隔离
    users, err := getUsersByTenant(tenantID)
    if err != nil {
        c.JSON(500, gin.H{"error": "获取用户失败"})
        return
    }
    
    c.JSON(200, gin.H{
        "tenant_id": tenantID,
        "users":     users,
    })
}
```

## 🔧 高级用法

### 1. 自定义中间件

```go
func CustomJWTMiddleware(config JWTConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 自定义令牌提取逻辑
        token := extractTokenFromRequest(c)
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "缺少认证令牌"})
            return
        }
        
        // 验证令牌
        claims, err := validateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "无效令牌"})
            return
        }
        
        // 检查黑名单
        if auth.IsRevoked(claims.ID) {
            c.AbortWithStatusJSON(401, gin.H{"error": "令牌已被撤销"})
            return
        }
        
        // 注入上下文
        ctx := injectClaimsToContext(c.Request.Context(), claims)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
    }
}
```

### 2. 批量令牌管理

```go
// 批量撤销用户的所有令牌
func RevokeAllUserTokens(userID string, ttl time.Duration) {
    // 这需要维护用户令牌映射表
    tokens := getUserTokens(userID)
    for _, jti := range tokens {
        auth.Revoke(jti, ttl)
    }
}

// 批量清理过期令牌
func BatchCleanup() {
    auth.CleanupExpired()
    fmt.Printf("清理完成，当前黑名单大小: %d\n", auth.GetBlacklistSize())
}
```

### 3. 令牌统计和监控

```go
func getAuthStats() map[string]interface{} {
    return map[string]interface{}{
        "blacklist_size":    auth.GetBlacklistSize(),
        "active_tokens":     getActiveTokenCount(),
        "expired_tokens":    getExpiredTokenCount(),
        "revoked_tokens":    auth.GetBlacklistSize(),
    }
}
```

## 🧪 测试示例

### 1. 单元测试

```go
func TestJWTGeneration(t *testing.T) {
    secret := []byte("test-secret-key-32-characters-long")
    
    token, err := auth.GenerateJWT(
        "test-tenant",
        "test-user",
        "web",
        "admin",
        "test:access",
        time.Hour,
        secret,
    )
    
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
    
    // 验证令牌
    parsedToken, err := jwt.ParseWithClaims(token, &auth.CoreXClaims{}, func(token *jwt.Token) (interface{}, error) {
        return secret, nil
    })
    
    assert.NoError(t, err)
    assert.True(t, parsedToken.Valid)
    
    claims := parsedToken.Claims.(*auth.CoreXClaims)
    assert.Equal(t, "test-tenant", claims.TenantID)
    assert.Equal(t, "test-user", claims.Subject)
}
```

### 2. 集成测试

```go
func TestJWTMiddleware(t *testing.T) {
    // 设置测试环境
    auth.SetJWTSecret([]byte("test-secret-key-32-characters-long"))
    
    // 创建测试路由
    r := gin.New()
    r.Use(auth.JwtMiddleware("admin", []string{"test:access"}, nil))
    r.GET("/protected", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "success"})
    })
    
    // 生成测试令牌
    token, _ := auth.GenerateJWT(
        "test-tenant", "test-user", "web", "admin", "test:access",
        time.Hour, []byte("test-secret-key-32-characters-long"),
    )
    
    // 测试请求
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    r.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}
```

## 📊 性能考虑

### 1. 黑名单优化

```go
// 使用Redis作为分布式黑名单存储
type RedisBlacklist struct {
    client *redis.Client
}

func (r *RedisBlacklist) Revoke(jti string, ttl time.Duration) error {
    return r.client.Set(context.Background(), "bl:"+jti, "1", ttl).Err()
}

func (r *RedisBlacklist) IsRevoked(jti string) bool {
    result := r.client.Get(context.Background(), "bl:"+jti)
    return result.Err() != redis.Nil
}
```

### 2. 令牌缓存

```go
// 缓存已验证的令牌以提高性能
var tokenCache = sync.Map{}

func validateTokenWithCache(tokenStr string) (*auth.CoreXClaims, error) {
    // 检查缓存
    if cached, ok := tokenCache.Load(tokenStr); ok {
        if claims, ok := cached.(*auth.CoreXClaims); ok {
            if time.Now().Before(claims.ExpiresAt.Time) {
                return claims, nil
            }
            tokenCache.Delete(tokenStr) // 清理过期缓存
        }
    }
    
    // 验证令牌
    claims, err := validateToken(tokenStr)
    if err != nil {
        return nil, err
    }
    
    // 缓存结果
    tokenCache.Store(tokenStr, claims)
    return claims, nil
}
```

## 🎯 总结

CoreX的JWT认证模块提供了：

- ✅ **完整的JWT支持**：生成、验证、中间件集成
- ✅ **灵活的权限控制**：受众验证、权限范围检查
- ✅ **令牌撤销机制**：基于JTI的黑名单管理
- ✅ **多租户支持**：租户隔离和上下文注入
- ✅ **高性能设计**：内存黑名单、令牌缓存
- ✅ **生产就绪**：错误处理、监控、测试覆盖
- ✅ **易于集成**：Gin中间件、统一配置

现在您可以在CoreX项目中安全、高效地使用JWT认证功能了！🎉

## 📚 相关文档

- [Logger使用指南](./logger_guide.md)
- [配置系统文档](../config/example.yaml)
- [事件总线文档](../pkg/event_bus/)