package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"bearer-token-service.v1/v2/interfaces"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	accountsCollection = "accounts"
)

// MongoAccountRepository MongoDB 实现的账户存储库
type MongoAccountRepository struct {
	collection *mongo.Collection
}

// NewMongoAccountRepository 创建账户存储库实例
func NewMongoAccountRepository(db *mongo.Database) *MongoAccountRepository {
	return &MongoAccountRepository{
		collection: db.Collection(accountsCollection),
	}
}

// Create 创建新账户
func (r *MongoAccountRepository) Create(ctx context.Context, account *interfaces.Account) error {
	// 生成 MongoDB ObjectID
	if account.ID == "" {
		account.ID = primitive.NewObjectID().Hex()
	}

	// 设置创建时间
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	// 生成 AccessKey
	if account.AccessKey == "" {
		accessKey, err := generateAccessKey()
		if err != nil {
			return err
		}
		account.AccessKey = accessKey
	}

	// SecretKey 应该已经加密（由 Service 层处理）
	// 这里只负责存储

	_, err := r.collection.InsertOne(ctx, account)
	if err != nil {
		// 检查唯一索引冲突
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("email or access_key already exists")
		}
		return err
	}

	return nil
}

// GetByAccessKey 根据 AccessKey 查询账户
func (r *MongoAccountRepository) GetByAccessKey(ctx context.Context, accessKey string) (*interfaces.Account, error) {
	var account interfaces.Account
	err := r.collection.FindOne(ctx, bson.M{"access_key": accessKey}).Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// GetByEmail 根据邮箱查询账户
func (r *MongoAccountRepository) GetByEmail(ctx context.Context, email string) (*interfaces.Account, error) {
	var account interfaces.Account
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// GetByID 根据 ID 查询账户
func (r *MongoAccountRepository) GetByID(ctx context.Context, id string) (*interfaces.Account, error) {
	var account interfaces.Account
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// UpdateSecretKey 更新 SecretKey
func (r *MongoAccountRepository) UpdateSecretKey(ctx context.Context, accountID string, newSecretKey string) error {
	// newSecretKey 应该已经加密（由 Service 层处理）
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": accountID},
		bson.M{
			"$set": bson.M{
				"secret_key": newSecretKey,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("account not found")
	}

	return nil
}

// UpdateStatus 更新账户状态
func (r *MongoAccountRepository) UpdateStatus(ctx context.Context, accountID string, status string) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": accountID},
		bson.M{
			"$set": bson.M{
				"status":     status,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("account not found")
	}

	return nil
}

// List 列出所有账户（管理员功能）
func (r *MongoAccountRepository) List(ctx context.Context, limit, offset int) ([]interfaces.Account, error) {
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accounts []interfaces.Account
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// Count 统计账户数量
func (r *MongoAccountRepository) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

// CreateIndexes 创建索引
func (r *MongoAccountRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "access_key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// ========================================
// 辅助函数
// ========================================

// generateAccessKey 生成 AccessKey
func generateAccessKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return interfaces.AccessKeyPrefix + hex.EncodeToString(b), nil
}

// GenerateSecretKey 生成 SecretKey（明文）
func GenerateSecretKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return interfaces.SecretKeyPrefix + hex.EncodeToString(b), nil
}

// HashSecretKey 不再加密 SecretKey，直接返回明文
// HMAC 签名验证需要明文 SecretKey，不能使用单向哈希
// 安全性依赖：MongoDB 传输层加密(TLS) + 访问控制 + 网络隔离
func HashSecretKey(secretKey string) (string, error) {
	// 直接返回明文 SecretKey（用于 HMAC 签名验证）
	return secretKey, nil
}

// VerifySecretKey 验证 SecretKey（明文比较）
func VerifySecretKey(storedSecretKey, plainSecretKey string) bool {
	// 直接比较明文
	return storedSecretKey == plainSecretKey
}
