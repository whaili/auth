package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/qiniu/bearer-token-service/v2/auth"
	"github.com/qiniu/bearer-token-service/v2/cache"
	"github.com/qiniu/bearer-token-service/v2/config"
	"github.com/qiniu/bearer-token-service/v2/handlers"
	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/qiniu/bearer-token-service/v2/observability"
	"github.com/qiniu/bearer-token-service/v2/pkg/mysql"
	"github.com/qiniu/bearer-token-service/v2/ratelimit"
	"github.com/qiniu/bearer-token-service/v2/repository"
	"github.com/qiniu/bearer-token-service/v2/service"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// ========================================
	// 0. 初始化日志系统
	// ========================================
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text"
	}
	logFile := os.Getenv("LOG_FILE") // 日志文件路径，如 "/app/logs/service.log"

	if logFile != "" {
		if err := observability.InitLoggerWithFile(logLevel, logFormat, logFile); err != nil {
			// 如果文件日志初始化失败，回退到 stdout
			observability.InitLogger(logLevel, logFormat, nil)
			slog.Warn("Failed to initialize file logger, using stdout only",
				slog.String("error", err.Error()),
				slog.String("log_file", logFile))
		} else {
			defer observability.CloseLogger()
			slog.Info("Logging to file", slog.String("log_file", logFile))
		}
	} else {
		observability.InitLogger(logLevel, logFormat, nil)
	}

	slog.Info("Bearer Token Service V2 starting...")

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
		slog.Error("Failed to connect to MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer client.Disconnect(ctx)

	// 验证连接
	if err := client.Ping(ctx, nil); err != nil {
		slog.Error("MongoDB ping failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("Connected to MongoDB")

	// 数据库名称（优先级：环境变量 > URI 中的数据库名 > 默认值）
	dbName := os.Getenv("MONGO_DATABASE")
	if dbName == "" {
		// 尝试从 MONGO_URI 中解析数据库名
		// 格式: mongodb://user:pass@host:port/dbname?options
		dbName = extractDatabaseFromURI(mongoURI)

		// 如果还是没有，使用默认值
		if dbName == "" {
			dbName = "token_service_v2"
			slog.Warn("No database name specified, using default", slog.String("database", dbName))
		} else {
			slog.Info("Using database from MONGO_URI", slog.String("database", dbName))
		}
	} else {
		slog.Info("Using database from MONGO_DATABASE env", slog.String("database", dbName))
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
		slog.Info("Skipping index creation (SKIP_INDEX_CREATION=true)")
	} else {
		slog.Info("Creating database indexes...")
		if err := accountRepo.CreateIndexes(context.Background()); err != nil {
			slog.Warn("Failed to create account indexes", slog.String("error", err.Error()))
		}
		if err := tokenRepo.CreateIndexes(context.Background()); err != nil {
			slog.Warn("Failed to create token indexes", slog.String("error", err.Error()))
		}
		if err := auditRepo.CreateIndexes(context.Background()); err != nil {
			slog.Warn("Failed to create audit log indexes", slog.String("error", err.Error()))
		}
		slog.Info("Database indexes created")
	}

	// ========================================
	// 3. 初始化用户信息查询（UserInfoRepository）
	// 支持两种方式：qconfapi RPC（推荐）或 MySQL 直接查询（备选）
	// ========================================
	var userInfoRepo interfaces.UserInfoRepository

	// 优先尝试 qconfapi
	qconfConfig := config.LoadQconfConfig()
	if qconfConfig.IsValid() {
		slog.Info("Initializing qconfapi connection...")
		qconfapiClient, err := repository.InitQconfClient(qconfConfig.ToQconfapiConfig())
		if err != nil {
			slog.Error("Failed to initialize qconfapi", slog.String("error", err.Error()))
			slog.Warn("qconfapi initialization failed, will try MySQL as fallback")
		} else {
			slog.Info("Connected to qconfapi",
				slog.Int("master_hosts_count", len(qconfConfig.MasterHosts)),
				slog.String("access_key", qconfConfig.AccessKey[:8]+"..."))

			// 创建 RPC UserInfoRepository
			userInfoRepo = repository.NewRPCUserInfoRepository(qconfapiClient)
			slog.Info("UserInfoRepository initialized (qconfapi RPC)")
		}
	} else if !qconfConfig.Enabled {
		slog.Info("qconfapi disabled (set QCONF_ENABLED=true to enable)")
	}

	// 如果 qconfapi 未配置或失败，尝试使用 MySQL
	if userInfoRepo == nil {
		mysqlConfig := config.LoadMySQLConfig()
		mysqlEnabled := os.Getenv("MYSQL_ENABLED") == "true"

		if mysqlEnabled {
			slog.Info("Initializing MySQL connection...")
			var mysqlClient *mysql.Client
			var err error
			mysqlClient, err = mysql.NewClient(mysqlConfig)
			if err != nil {
				slog.Error("Failed to connect to MySQL", slog.String("error", err.Error()))
				slog.Warn("MySQL connection failed, /api/v2/validateu will return user_info: null")
			} else {
				defer mysqlClient.Close()
				slog.Info("Connected to MySQL",
					slog.String("host", mysqlConfig.Host),
					slog.Int("port", mysqlConfig.Port),
					slog.String("database", mysqlConfig.Database))

				// 创建 MySQL UserInfoRepository
				userInfoRepo = repository.NewMySQLUserInfoRepository(mysqlClient)
				slog.Info("UserInfoRepository initialized (MySQL)")
			}
		} else {
			slog.Info("MySQL disabled (set MYSQL_ENABLED=true to enable extended user info)")
		}
	}

	// ========================================
	// 4. 初始化 Redis 和缓存层（可选）
	// ========================================
	redisConfig := cache.LoadRedisConfig()

	if redisConfig.Enabled {
		slog.Info("Initializing Redis cache...")

		// 创建 Redis 客户端
		redisClient, err := cache.NewRedisClient(
			redisConfig.Addr,
			redisConfig.Password,
			redisConfig.DB,
			redisConfig.PoolSize,
			redisConfig.MinIdleConns,
			redisConfig.MaxRetries,
		)
		if err != nil {
			slog.Error("Failed to connect to Redis", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer redisClient.Close()

		slog.Info("Connected to Redis", slog.String("addr", redisConfig.Addr))

		// 初始化 Token 缓存
		tokenCache := cache.NewTokenCache(redisClient, tokenRepo, redisConfig.TokenCacheTTL)

		// 注入缓存到 Repository
		tokenRepo.SetCache(tokenCache)

		slog.Info("Redis cache enabled", slog.Duration("token_cache_ttl", redisConfig.TokenCacheTTL))
	} else {
		slog.Info("Redis cache disabled (set REDIS_ENABLED=true to enable)")
	}

	// ========================================
	// 5. 初始化 Service 层
	// ========================================
	tokenService := service.NewTokenService(tokenRepo, auditRepo)

	// 根据是否有 UserInfoRepository 创建不同的 ValidationService
	var validationService interfaces.ValidationService
	if userInfoRepo != nil {
		validationService = service.NewValidationServiceWithUserInfo(tokenRepo, userInfoRepo)
		slog.Info("ValidationService initialized with UserInfo support")
	} else {
		validationService = service.NewValidationService(tokenRepo)
		slog.Info("ValidationService initialized (basic mode)")
	}

	_ = service.NewAuditService(auditRepo) // 预留用于未来的审计日志查询

	slog.Info("Services initialized")

	// ========================================
	// 5. 初始化 Handler 层
	// ========================================
	tokenHandler := handlers.NewTokenHandler(tokenService)
	validationHandler := handlers.NewValidationHandler(validationService)

	slog.Info("Handlers initialized")

	// ========================================
	// 6. 创建 QiniuStub 认证中间件
	// ========================================
	// 配置七牛 UID 映射器
	var qiniuUIDMapper auth.QiniuUIDMapper
	mapperMode := os.Getenv("QINIU_UID_MAPPER_MODE") // "simple" 或 "database"

	if mapperMode == "database" {
		// 数据库模式（查询或创建映射关系）
		autoCreate := os.Getenv("QINIU_UID_AUTO_CREATE") == "true"
		qiniuUIDMapper = auth.NewDatabaseQiniuUIDMapper(accountRepo, autoCreate)
		slog.Info("Using DatabaseQiniuUIDMapper", slog.Bool("auto_create", autoCreate))
	} else {
		// 简单模式（默认）：直接转换为 qiniu_{uid}
		qiniuUIDMapper = auth.NewSimpleQiniuUIDMapper()
		slog.Info("Using SimpleQiniuUIDMapper (format: qiniu_{uid})")
	}

	// 创建 QiniuStub 认证中间件
	qstubMiddleware := auth.NewQstubAuthMiddleware(qiniuUIDMapper)
	slog.Info("QiniuStub authentication middleware initialized")

	// ========================================
	// 7. 初始化限流中间件（可选）
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
		slog.Info("Application rate limit enabled",
			slog.Int("per_minute", rateLimitConfig.AppLimitPerMinute),
			slog.Int("per_hour", rateLimitConfig.AppLimitPerHour),
			slog.Int("per_day", rateLimitConfig.AppLimitPerDay))
	} else {
		slog.Info("Application rate limit disabled (set ENABLE_APP_RATE_LIMIT=true to enable)")
	}

	if rateLimitConfig.EnableAccountLimit {
		slog.Info("Account rate limit enabled")
	} else {
		slog.Info("Account rate limit disabled (set ENABLE_ACCOUNT_RATE_LIMIT=true to enable)")
	}

	if rateLimitConfig.EnableTokenLimit {
		slog.Info("Token rate limit enabled")
	} else {
		slog.Info("Token rate limit disabled (set ENABLE_TOKEN_RATE_LIMIT=true to enable)")
	}

	// ========================================
	// 8. 设置路由
	// ========================================
	router := mux.NewRouter()

	// 可观测性中间件（最外层）
	router.Use(observability.RequestTrackingMiddleware)
	router.Use(observability.MetricsMiddleware)

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

	// Prometheus metrics 端点
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

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

	// Token 验证（扩展版，包含用户信息）
	var validateTokenUHandler http.Handler = http.HandlerFunc(validationHandler.ValidateTokenU)
	if rateLimitConfig.EnableTokenLimit {
		// 提取 Token 到上下文，然后应用 Token 限流
		validateTokenUHandler = extractTokenMiddleware(rateLimitMiddleware.TokenLimitMiddleware(validateTokenUHandler))
	}
	router.Handle("/api/v2/validateu", validateTokenUHandler).Methods("POST")

	slog.Info("Routes configured")

	// ========================================
	// 9. 启动服务器
	// ========================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Bearer Token Service V2 is ready",
		slog.String("port", port),
		slog.String("metrics_endpoint", "/metrics"),
		slog.String("health_endpoint", "/health"))

	if err := http.ListenAndServe(":"+port, router); err != nil {
		slog.Error("Server failed to start", slog.String("error", err.Error()))
		os.Exit(1)
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
