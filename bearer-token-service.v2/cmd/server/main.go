package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bearer-token-service.v1/v2/auth"
	"bearer-token-service.v1/v2/handlers"
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
	accountService := service.NewAccountService(accountRepo, auditRepo)
	tokenService := service.NewTokenService(tokenRepo, auditRepo)
	validationService := service.NewValidationService(tokenRepo)
	_ = service.NewAuditService(auditRepo) // 预留用于未来的审计日志查询

	log.Println("✅ Services initialized")

	// ========================================
	// 4. 初始化 Handler 层
	// ========================================
	accountHandler := handlers.NewAccountHandler(accountService)
	tokenHandler := handlers.NewTokenHandler(tokenService)
	validationHandler := handlers.NewValidationHandler(validationService)

	log.Println("✅ Handlers initialized")

	// ========================================
	// 5. 创建认证中间件（统一认证：HMAC + Qstub）
	// ========================================
	// 5.1 配置 AccountFetcher（账户查询方式）
	var accountFetcher auth.AccountFetcher
	accountFetcherMode := os.Getenv("ACCOUNT_FETCHER_MODE") // "local" 或 "external"

	if accountFetcherMode == "external" {
		// 外部 API 模式（用于共用数据库场景）
		externalAPIURL := os.Getenv("EXTERNAL_ACCOUNT_API_URL")
		externalAPIToken := os.Getenv("EXTERNAL_ACCOUNT_API_TOKEN")

		if externalAPIURL == "" {
			log.Fatal("❌ EXTERNAL_ACCOUNT_API_URL is required when ACCOUNT_FETCHER_MODE=external")
		}

		accountFetcher = NewExternalAccountFetcher(externalAPIURL, externalAPIToken)
		log.Printf("✅ Using External AccountFetcher (API: %s)", externalAPIURL)
	} else {
		// 本地 MongoDB 模式（默认）
		accountFetcher = &MongoAccountFetcher{repo: accountRepo}
		log.Println("✅ Using Local MongoDB AccountFetcher")
	}

	// 5.2 配置七牛 UID 映射器
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

	// 5.3 创建统一认证中间件
	timestampTolerance := 15 * time.Minute
	if toleranceStr := os.Getenv("HMAC_TIMESTAMP_TOLERANCE"); toleranceStr != "" {
		if duration, err := time.ParseDuration(toleranceStr); err == nil {
			timestampTolerance = duration
		}
	}

	unifiedMiddleware := auth.NewUnifiedAuthMiddleware(accountFetcher, qiniuUIDMapper, timestampTolerance)
	log.Printf("✅ Unified authentication middleware initialized (HMAC + Qstub, tolerance=%v)", timestampTolerance)

	// ========================================
	// 6. 设置路由
	// ========================================
	router := mux.NewRouter()

	// 健康检查
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// 账户管理（不需要认证的注册接口）
	router.HandleFunc("/api/v2/accounts/register", accountHandler.Register).Methods("POST")

	// 账户管理（需要认证：支持 HMAC 或 Qstub）
	router.HandleFunc("/api/v2/accounts/me", unifiedMiddleware.Authenticate(accountHandler.GetAccountInfo)).Methods("GET")
	router.HandleFunc("/api/v2/accounts/regenerate-sk", unifiedMiddleware.Authenticate(accountHandler.RegenerateSecretKey)).Methods("POST")

	// Token 管理（需要认证：支持 HMAC 或 Qstub）
	router.HandleFunc("/api/v2/tokens", unifiedMiddleware.Authenticate(tokenHandler.CreateToken)).Methods("POST")
	router.HandleFunc("/api/v2/tokens", unifiedMiddleware.Authenticate(tokenHandler.ListTokens)).Methods("GET")
	router.HandleFunc("/api/v2/tokens/{id}", unifiedMiddleware.Authenticate(tokenHandler.GetTokenInfo)).Methods("GET")
	router.HandleFunc("/api/v2/tokens/{id}/status", unifiedMiddleware.Authenticate(tokenHandler.UpdateTokenStatus)).Methods("PUT")
	router.HandleFunc("/api/v2/tokens/{id}", unifiedMiddleware.Authenticate(tokenHandler.DeleteToken)).Methods("DELETE")
	router.HandleFunc("/api/v2/tokens/{id}/stats", unifiedMiddleware.Authenticate(tokenHandler.GetTokenStats)).Methods("GET")

	// Token 验证（使用 Bearer Token 认证）
	router.HandleFunc("/api/v2/validate", validationHandler.ValidateToken).Methods("POST")

	// 审计日志（需要 HMAC 认证）
	// TODO: 实现 AuditHandler

	log.Println("✅ Routes configured")

	// ========================================
	// 7. 启动服务器
	// ========================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🌐 Server starting on http://localhost:%s", port)
	log.Printf("📖 API Documentation: /root/src/auth/bearer-token-service.v1/v2/API.md")
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
// MongoAccountFetcher 实现 auth.AccountFetcher 接口（本地 MongoDB）
// ========================================
type MongoAccountFetcher struct {
	repo *repository.MongoAccountRepository
}

func (f *MongoAccountFetcher) GetAccountByAccessKey(ctx context.Context, accessKey string) (*auth.AccountInfo, error) {
	account, err := f.repo.GetByAccessKey(ctx, accessKey)
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, nil
	}

	return &auth.AccountInfo{
		ID:        account.ID,
		Email:     account.Email,
		AccessKey: account.AccessKey,
		SecretKey: account.SecretKey, // 已加密的 SecretKey
		Status:    account.Status,
	}, nil
}

// ========================================
// ExternalAccountFetcher 实现 auth.AccountFetcher 接口（外部 API）
// 用于查询共用的外部账户系统
// ========================================
type ExternalAccountFetcher struct {
	apiBaseURL string
	apiToken   string
	httpClient *http.Client
}

func NewExternalAccountFetcher(apiBaseURL, apiToken string) *ExternalAccountFetcher {
	return &ExternalAccountFetcher{
		apiBaseURL: apiBaseURL,
		apiToken:   apiToken,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (f *ExternalAccountFetcher) GetAccountByAccessKey(ctx context.Context, accessKey string) (*auth.AccountInfo, error) {
	// 构建请求 URL
	url := f.apiBaseURL + "/api/accounts?access_key=" + accessKey

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 添加认证头（如果配置了 API Token）
	if f.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+f.apiToken)
	}

	// 发送请求
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // 账户不存在
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external account API returned status %s", resp.Status)
	}

	// 解析响应
	var result struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
		Status    string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &auth.AccountInfo{
		ID:        result.ID,
		Email:     result.Email,
		AccessKey: result.AccessKey,
		SecretKey: result.SecretKey,
		Status:    result.Status,
	}, nil
}
