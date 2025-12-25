package main

import (
	"context"
	"log"
	"net/http"
	"os"
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

	db := client.Database("token_service_v2")

	// ========================================
	// 2. 初始化 Repository 层
	// ========================================
	accountRepo := repository.NewMongoAccountRepository(db)
	tokenRepo := repository.NewMongoTokenRepository(db)
	auditRepo := repository.NewMongoAuditLogRepository(db)

	// 创建索引
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
	// 5. 创建认证中间件
	// ========================================
	// 实现 AccountFetcher 接口
	accountFetcher := &AccountFetcherImpl{repo: accountRepo}

	// 创建 HMAC 认证中间件（15 分钟时间窗口）
	hmacMiddleware := auth.NewHMACMiddleware(accountFetcher, 15*time.Minute)

	log.Println("✅ Authentication middleware initialized")

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

	// 账户管理（需要 HMAC 认证）
	router.HandleFunc("/api/v2/accounts/me", hmacMiddleware.Authenticate(accountHandler.GetAccountInfo)).Methods("GET")
	router.HandleFunc("/api/v2/accounts/regenerate-sk", hmacMiddleware.Authenticate(accountHandler.RegenerateSecretKey)).Methods("POST")

	// Token 管理（需要 HMAC 认证）
	router.HandleFunc("/api/v2/tokens", hmacMiddleware.Authenticate(tokenHandler.CreateToken)).Methods("POST")
	router.HandleFunc("/api/v2/tokens", hmacMiddleware.Authenticate(tokenHandler.ListTokens)).Methods("GET")
	router.HandleFunc("/api/v2/tokens/{id}", hmacMiddleware.Authenticate(tokenHandler.GetTokenInfo)).Methods("GET")
	router.HandleFunc("/api/v2/tokens/{id}/status", hmacMiddleware.Authenticate(tokenHandler.UpdateTokenStatus)).Methods("PUT")
	router.HandleFunc("/api/v2/tokens/{id}", hmacMiddleware.Authenticate(tokenHandler.DeleteToken)).Methods("DELETE")
	router.HandleFunc("/api/v2/tokens/{id}/stats", hmacMiddleware.Authenticate(tokenHandler.GetTokenStats)).Methods("GET")

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
// AccountFetcherImpl 实现 auth.AccountFetcher 接口
// ========================================
type AccountFetcherImpl struct {
	repo *repository.MongoAccountRepository
}

func (f *AccountFetcherImpl) GetAccountByAccessKey(ctx context.Context, accessKey string) (*auth.AccountInfo, error) {
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
