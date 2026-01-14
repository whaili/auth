package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"bearer-token-service.v1/v2/interfaces"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	tokensCollection = "tokens"
)

// TokenCache Token 缓存接口（避免循环依赖）
type TokenCache interface {
	GetByTokenValue(ctx context.Context, tokenValue string) (*interfaces.Token, error)
	GetByID(ctx context.Context, tokenID string) (*interfaces.Token, error)
	InvalidateByTokenValue(ctx context.Context, tokenValue string) error
	InvalidateByID(ctx context.Context, tokenID string) error
}

// MongoTokenRepository MongoDB 实现的 Token 存储库（带租户隔离）
type MongoTokenRepository struct {
	collection *mongo.Collection
	cache      TokenCache // 可选的缓存层
}

// NewMongoTokenRepository 创建 Token 存储库实例
func NewMongoTokenRepository(db *mongo.Database) *MongoTokenRepository {
	return &MongoTokenRepository{
		collection: db.Collection(tokensCollection),
	}
}

// SetCache 设置缓存层（依赖注入）
func (r *MongoTokenRepository) SetCache(cache TokenCache) {
	r.cache = cache
}

// GetByTokenValueDirect 直接从 MongoDB 查询（不经过缓存，供缓存层回调使用）
func (r *MongoTokenRepository) GetByTokenValueDirect(ctx context.Context, tokenValue string) (*interfaces.Token, error) {
	var token interfaces.Token
	err := r.collection.FindOne(ctx, bson.M{"token": tokenValue}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// GetByIDDirect 直接从 MongoDB 查询（不经过缓存，供缓存层回调使用）
func (r *MongoTokenRepository) GetByIDDirect(ctx context.Context, tokenID string) (*interfaces.Token, error) {
	var token interfaces.Token
	err := r.collection.FindOne(ctx, bson.M{"_id": tokenID}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// Create 创建新 Token
func (r *MongoTokenRepository) Create(ctx context.Context, token *interfaces.Token) error {
	// 生成 Token ID
	if token.ID == "" {
		token.ID = "tk_" + generateRandomID(16)
	}

	// 生成 Token 值
	if token.Token == "" {
		tokenValue, err := generateTokenValue(token.Prefix)
		if err != nil {
			return err
		}
		token.Token = tokenValue
	}

	// 设置创建时间
	token.CreatedAt = time.Now()

	// 初始化使用统计
	token.TotalRequests = 0

	_, err := r.collection.InsertOne(ctx, token)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("token already exists")
		}
		return err
	}

	return nil
}

// GetByID 根据 ID 查询 Token
func (r *MongoTokenRepository) GetByID(ctx context.Context, tokenID string) (*interfaces.Token, error) {
	// 如果配置了缓存，优先从缓存读取
	if r.cache != nil {
		return r.cache.GetByID(ctx, tokenID)
	}

	// 降级到 MongoDB 直查
	var token interfaces.Token
	err := r.collection.FindOne(ctx, bson.M{"_id": tokenID}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// GetByTokenValue 根据 token 值查询 Token
func (r *MongoTokenRepository) GetByTokenValue(ctx context.Context, tokenValue string) (*interfaces.Token, error) {
	// 如果配置了缓存，优先从缓存读取
	if r.cache != nil {
		return r.cache.GetByTokenValue(ctx, tokenValue)
	}

	// 降级到 MongoDB 直查
	var token interfaces.Token
	err := r.collection.FindOne(ctx, bson.M{"token": tokenValue}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// ListByAccountID 查询账户的所有 Tokens（租户隔离）
func (r *MongoTokenRepository) ListByAccountID(ctx context.Context, accountID string, activeOnly bool, limit, offset int) ([]interfaces.Token, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100 // 最大 100 条
	}

	// 构建查询条件 - 关键：租户隔离
	filter := bson.M{
		"account_id": accountID, // 强制只查询该账户的 Tokens
	}

	if activeOnly {
		filter["is_active"] = true
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tokens []interfaces.Token
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// CountByAccountID 统计账户的 Token 数量
func (r *MongoTokenRepository) CountByAccountID(ctx context.Context, accountID string, activeOnly bool) (int64, error) {
	filter := bson.M{
		"account_id": accountID, // 租户隔离
	}

	if activeOnly {
		filter["is_active"] = true
	}

	return r.collection.CountDocuments(ctx, filter)
}

// UpdateStatus 更新 Token 状态
func (r *MongoTokenRepository) UpdateStatus(ctx context.Context, tokenID string, isActive bool) error {
	// 先查询获取 token_value（用于失效缓存）
	var token interfaces.Token
	err := r.collection.FindOne(ctx, bson.M{"_id": tokenID}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("token not found")
		}
		return err
	}

	// 更新状态
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": tokenID},
		bson.M{
			"$set": bson.M{
				"is_active": isActive,
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("token not found")
	}

	// 失效两个缓存键（token:id 和 token:val）
	if r.cache != nil {
		_ = r.cache.InvalidateByID(ctx, tokenID)
		_ = r.cache.InvalidateByTokenValue(ctx, token.Token)
	}

	return nil
}

// Delete 删除 Token
func (r *MongoTokenRepository) Delete(ctx context.Context, tokenID string) error {
	// 先查询获取 token_value（用于失效缓存）
	var token interfaces.Token
	err := r.collection.FindOne(ctx, bson.M{"_id": tokenID}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("token not found")
		}
		return err
	}

	// 删除 MongoDB
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": tokenID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("token not found")
	}

	// 失效两个缓存键
	if r.cache != nil {
		_ = r.cache.InvalidateByID(ctx, tokenID)
		_ = r.cache.InvalidateByTokenValue(ctx, token.Token)
	}

	return nil
}

// IncrementUsage 增加使用次数
func (r *MongoTokenRepository) IncrementUsage(ctx context.Context, tokenID string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": tokenID},
		bson.M{
			"$inc": bson.M{"total_requests": 1},
			"$set": bson.M{"last_used_at": time.Now()},
		},
	)
	return err
}

// UpdateLastUsed 更新最后使用时间
func (r *MongoTokenRepository) UpdateLastUsed(ctx context.Context, tokenID string, lastUsedAt time.Time) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": tokenID},
		bson.M{
			"$set": bson.M{"last_used_at": lastUsedAt},
		},
	)
	return err
}

// DeleteExpired 删除过期的 Tokens
func (r *MongoTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now()

	// 删除已过期的 Token（expires_at < now）
	result, err := r.collection.DeleteMany(
		ctx,
		bson.M{
			"expires_at": bson.M{
				"$exists": true,
				"$lt":     now,
			},
		},
	)

	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// CreateIndexes 创建索引
func (r *MongoTokenRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			// 租户隔离的核心索引
			Keys: bson.D{
				{Key: "account_id", Value: 1},
				{Key: "is_active", Value: 1},
			},
		},
		{
			// 过期清理索引
			Keys: bson.D{{Key: "expires_at", Value: 1}},
		},
		{
			// 查询优化
			Keys: bson.D{
				{Key: "account_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// VerifyTokenOwnership 验证 Token 是否属于指定账户（租户隔离检查）
func (r *MongoTokenRepository) VerifyTokenOwnership(ctx context.Context, tokenID string, accountID string) (bool, error) {
	count, err := r.collection.CountDocuments(
		ctx,
		bson.M{
			"_id":        tokenID,
			"account_id": accountID,
		},
	)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ========================================
// 辅助函数
// ========================================

// generateTokenValue 生成 Token 值
// 如果提供自定义 prefix，格式为 prefix-XXXXX；否则使用默认前缀 sk-XXXXX
func generateTokenValue(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// 如果没有提供自定义前缀，使用默认前缀（已包含分隔符）
	if prefix == "" {
		return interfaces.TokenPrefix + hex.EncodeToString(b), nil
	}

	// 自定义前缀加分隔符
	return prefix + "-" + hex.EncodeToString(b), nil
}

// generateRandomID 生成随机 ID
func generateRandomID(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
