package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bearer-token-service.v1/v2/auth"
	"bearer-token-service.v1/v2/config"
	"bearer-token-service.v1/v2/handlers"
	"bearer-token-service.v1/v2/ratelimit"
	"bearer-token-service.v1/v2/repository"
	"bearer-token-service.v1/v2/service"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log.Println("🚀 Bearer Token Service V2 - Starting...")

	// ========================================
	// 1. MongoDB 连接
	// ========================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// 验证连接
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("❌ MongoDB ping failed: %v", err)
	}
	log.Println("✅ Connected to MongoDB")

	// 数据库名称（优先级：环境变量 > URI 中的数据库名 > 默认值）
	dbName := os.Getenv("MONGO_DATABASE")
	if dbName == "" {
		// 尝试从 MONGO_URI 中解析数据库名
		// 格式: mongodb://user:pass@host:port/dbname?options
		dbName = extractDatabaseFromURI(mongoURI)

		// 如果还是没有，使用默认值
		if dbName == "" {
			dbName = "token_service_v2"
			log.Printf("⚠️  Warning: No database name specified in MONGO_URI or MONGO_DATABASE, using default: %s", dbName)
		} else {
			log.Printf("ℹ️  Using database from MONGO_URI: %s", dbName)
		}
	} else {
		log.Printf("ℹ️  Using database from MONGO_DATABASE env: %s", dbName)
	}
	db := client.Database(dbName)

	// ========================================
	// 2. 初始化 Repository 层
	// ========================================
	accountRepo := repository.NewMongoAccountRepository(db)
	tokenRepo := repository.NewMongoTokenRepository(db)
	auditRepo := repository.NewMongoAuditLogRepository(db)

	// 创建索引（可通过环境变量跳过，用于多实例负载均衡部署）
	skipIndexCreation := os.Getenv("SKIP_INDEX_CREATION") == "true"

	if skipIndexCreation {
		log.Println("⏭️  Skipping index creation (SKIP_INDEX_CREATION=true)")
		log.Println("ℹ️  Ensure indexes are created by running: scripts/init/init-db.sh")
	} else {
		log.Println("📊 Creating database indexes...")
		if err := accountRepo.CreateIndexes(context.Background()); err != nil {
			log.Printf("⚠️  Warning: Failed to create account indexes: %v", err)
		}
		if err := tokenRepo.CreateIndexes(context.Background()); err != nil {
			log.Printf("⚠️  Warning: Failed to create token indexes: %v", err)
		}
		if err := auditRepo.CreateIndexes(context.Background()); err != nil {
			log.Printf("⚠️  Warning: Failed to create audit log indexes: %v", err)
		}
		log.Println("✅ Database indexes created")
	}

	// ========================================
	// 3. 初始化 Service 层
	// ========================================
	tokenService := service.NewTokenService(tokenRepo, auditRepo)
	validationService := service.NewValidationService(tokenRepo)
	_ = service.NewAuditService(auditRepo) // 预留用于未来的审计日志查询

	log.Println("✅ Services initialized")

	// ========================================
	// 4. 初始化 Handler 层
	// ========================================
	tokenHandler := handlers.NewTokenHandler(tokenService)
	validationHandler := handlers.NewValidationHandler(validationService)

	log.Println("✅ Handlers initialized")

	// ========================================
	// 5. 创建 QiniuStub 认证中间件
	// ========================================
	// 配置七牛 UID 映射器
	var qiniuUIDMapper auth.QiniuUIDMapper
	mapperMode := os.Getenv("QINIU_UID_MAPPER_MODE") // "simple" 或 "database"

	if mapperMode == "database" {
		// 数据库模式（查询或创建映射关系）
		autoCreate := os.Getenv("QINIU_UID_AUTO_CREATE") == "true"
		qiniuUIDMapper = auth.NewDatabaseQiniuUIDMapper(accountRepo, autoCreate)
		log.Printf("✅ Using DatabaseQiniuUIDMapper (autoCreate=%v)", autoCreate)
	} else {
		// 简单模式（默认）：直接转换为 qiniu_{uid}
		qiniuUIDMapper = auth.NewSimpleQiniuUIDMapper()
		log.Println("✅ Using SimpleQiniuUIDMapper (format: qiniu_{uid})")
	}

	// 创建 QiniuStub 认证中间件
	qstubMiddleware := auth.NewQstubAuthMiddleware(qiniuUIDMapper)
	log.Println("✅ QiniuStub authentication middleware initialized")

	// ========================================
	// 6. 初始化限流中间件（可选）
	// ========================================
	rateLimitConfig := config.LoadRateLimitConfig()

	// 创建限流器
	limiter := ratelimit.NewMemoryLimiter()

	// 创建限流管理器
	rateLimitManager := ratelimit.NewRateLimitManager(limiter, ratelimit.RateLimitConfig{
		AppLimit:           rateLimitConfig.GetAppRateLimit(),
		EnableAppLimit:     rateLimitConfig.EnableAppLimit,
		EnableAccountLimit: rateLimitConfig.EnableAccountLimit,
		EnableTokenLimit:   rateLimitConfig.EnableTokenLimit,
	})

	// 创建限流中间件
	rateLimitMiddleware := ratelimit.NewMiddleware(rateLimitManager, accountRepo, tokenRepo)

	// 打印限流配置状态
	if rateLimitConfig.EnableAppLimit {
		log.Printf("✅ Application rate limit ENABLED: %d req/min, %d req/hour, %d req/day",
			rateLimitConfig.AppLimitPerMinute,
			rateLimitConfig.AppLimitPerHour,
			rateLimitConfig.AppLimitPerDay)
	} else {
		log.Println("ℹ️  Application rate limit DISABLED (set ENABLE_APP_RATE_LIMIT=true to enable)")
	}

	if rateLimitConfig.EnableAccountLimit {
		log.Println("✅ Account rate limit ENABLED (configured per account)")
	} else {
		log.Println("ℹ️  Account rate limit DISABLED (set ENABLE_ACCOUNT_RATE_LIMIT=true to enable)")
	}

	if rateLimitConfig.EnableTokenLimit {
		log.Println("✅ Token rate limit ENABLED (configured per token)")
	} else {
		log.Println("ℹ️  Token rate limit DISABLED (set ENABLE_TOKEN_RATE_LIMIT=true to enable)")
	}

	// ========================================
	// 7. 设置路由
	// ========================================
	router := mux.NewRouter()

	// 应用全局限流中间件（如果启用）
	if rateLimitConfig.EnableAppLimit {
		router.Use(rateLimitMiddleware.AppLimitMiddleware)
	}

	// 应用账户层限流中间件（如果启用）
	if rateLimitConfig.EnableAccountLimit {
		router.Use(rateLimitMiddleware.AccountLimitMiddleware)
	}

	// 健康检查
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// Token 管理（需要 QiniuStub 认证）
	router.HandleFunc("/api/v2/tokens", qstubMiddleware.Authenticate(tokenHandler.CreateToken)).Methods("POST")
	router.HandleFunc("/api/v2/tokens", qstubMiddleware.Authenticate(tokenHandler.ListTokens)).Methods("GET")
	router.HandleFunc("/api/v2/tokens/{id}", qstubMiddleware.Authenticate(tokenHandler.GetTokenInfo)).Methods("GET")
	router.HandleFunc("/api/v2/tokens/{id}/status", qstubMiddleware.Authenticate(tokenHandler.UpdateTokenStatus)).Methods("PUT")
	router.HandleFunc("/api/v2/tokens/{id}", qstubMiddleware.Authenticate(tokenHandler.DeleteToken)).Methods("DELETE")
	router.HandleFunc("/api/v2/tokens/{id}/stats", qstubMiddleware.Authenticate(tokenHandler.GetTokenStats)).Methods("GET")

	// Token 验证（使用 Bearer Token 认证）
	// 为 Token 层限流包装验证 handler
	var validateTokenHandler http.Handler = http.HandlerFunc(validationHandler.ValidateToken)
	if rateLimitConfig.EnableTokenLimit {
		// 提取 Token 到上下文，然后应用 Token 限流
		validateTokenHandler = extractTokenMiddleware(rateLimitMiddleware.TokenLimitMiddleware(validateTokenHandler))
	}
	router.Handle("/api/v2/validate", validateTokenHandler).Methods("POST")

	log.Println("✅ Routes configured")

	// ========================================
	// 8. 启动服务器
	// ========================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🌐 Server starting on http://localhost:%s", port)
	log.Printf("📖 API Documentation: /root/src/auth/bearer-token-service.v2/docs/api/API.md")
	log.Println("")
	log.Println("✨ Bearer Token Service V2 is ready!")
	log.Println("")

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

// ========================================
// 辅助函数：从 MongoDB URI 中提取数据库名
// ========================================
// extractDatabaseFromURI 从 MongoDB 连接字符串中提取数据库名
// 支持格式:
//   - mongodb://host:port/dbname
//   - mongodb://user:pass@host:port/dbname
//   - mongodb://host1:port1,host2:port2/dbname?options
func extractDatabaseFromURI(uri string) string {
	// 移除协议前缀
	uri = strings.TrimPrefix(uri, "mongodb://")
	uri = strings.TrimPrefix(uri, "mongodb+srv://")

	// 移除认证信息（user:pass@）
	if atIndex := strings.Index(uri, "@"); atIndex != -1 {
		uri = uri[atIndex+1:]
	}

	// 查找第一个 / 后的数据库名
	if slashIndex := strings.Index(uri, "/"); slashIndex != -1 {
		dbPart := uri[slashIndex+1:]

		// 移除查询参数（?后的内容）
		if questionIndex := strings.Index(dbPart, "?"); questionIndex != -1 {
			dbPart = dbPart[:questionIndex]
		}

		// 返回数据库名（如果不为空）
		dbName := strings.TrimSpace(dbPart)
		if dbName != "" {
			return dbName
		}
	}

	return ""
}

// ========================================
// 辅助中间件：从 Authorization 头提取 Token 到上下文
// ========================================
func extractTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 提取 Bearer Token
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenValue := strings.TrimPrefix(authHeader, "Bearer ")
			// 设置到上下文
			r = ratelimit.SetTokenToContext(r, tokenValue)
		}
		next.ServeHTTP(w, r)
	})
}
