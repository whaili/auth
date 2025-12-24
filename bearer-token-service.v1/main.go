package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"bearer-token-service.v1/handlers"
	"bearer-token-service.v1/repo"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// MongoDB 连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	// 初始化存储库
	tokenRepo := repo.NewMongoTokenRepository(client.Database("token_service"))
	adminRepo := repo.NewMongoAdminRepository(client.Database("token_service"))

	// 初始化默认管理员用户
	if err := adminRepo.InitializeAdmin(context.Background()); err != nil {
		log.Fatalf("Failed to initialize admin: %v", err)
	}

	// 初始化路由
	r := mux.NewRouter()

	// Token 路由
	tokenHandler := handlers.NewTokenHandler(*tokenRepo, *adminRepo)
	r.HandleFunc("/api/tokens", tokenHandler.CreateToken).Methods("POST")
	r.HandleFunc("/api/tokens", tokenHandler.ListTokens).Methods("GET")
	r.HandleFunc("/api/tokens/{id}/status", tokenHandler.ToggleTokenStatus).Methods("PUT")
	r.HandleFunc("/api/tokens/{id}", tokenHandler.DeleteToken).Methods("DELETE")

	// 验证路由
	r.HandleFunc("/api/validate", tokenHandler.ValidateToken).Methods("GET")

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
