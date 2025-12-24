package repo

import (
	"context"
	"errors"

	"bearer-token-service.v1/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokensCollection = "tokens"
	adminCollection  = "admin_users"
)

// MongoTokenRepository 实现 Token 存储接口
type MongoTokenRepository struct {
	tokens *mongo.Collection
}

func NewMongoTokenRepository(db *mongo.Database) *MongoTokenRepository {
	return &MongoTokenRepository{
		tokens: db.Collection(tokensCollection),
	}
}

func (r *MongoTokenRepository) CreateToken(ctx context.Context, token *models.Token) error {
	_, err := r.tokens.InsertOne(ctx, token)
	return err
}

func (r *MongoTokenRepository) GetToken(ctx context.Context, tokenStr string) (*models.Token, error) {
	var token models.Token
	err := r.tokens.FindOne(ctx, bson.M{"token": tokenStr}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *MongoTokenRepository) ListTokens(ctx context.Context) ([]models.Token, error) {
	var tokens []models.Token
	cursor, err := r.tokens.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *MongoTokenRepository) UpdateTokenStatus(ctx context.Context, id string, isActive bool) (*models.Token, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var updatedToken models.Token
	err = r.tokens.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"is_active": isActive}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedToken)

	if err != nil {
		return nil, err
	}
	return &updatedToken, nil
}

func (r *MongoTokenRepository) DeleteToken(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.tokens.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

// MongoAdminRepository 实现管理员用户存储
type MongoAdminRepository struct {
	admin *mongo.Collection
}

func NewMongoAdminRepository(db *mongo.Database) *MongoAdminRepository {
	return &MongoAdminRepository{
		admin: db.Collection(adminCollection),
	}
}

func (r *MongoAdminRepository) InitializeAdmin(ctx context.Context) error {
	// 检查是否已存在管理员
	count, err := r.admin.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // 管理员已存在
	}

	// 创建默认管理员
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("adminpassword"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := models.AdminUser{
		Username: "admin",
		Password: string(hashedPassword),
	}

	_, err = r.admin.InsertOne(ctx, admin)
	return err
}

func (r *MongoAdminRepository) VerifyAdmin(ctx context.Context, username, password string) (bool, error) {
	var admin models.AdminUser
	err := r.admin.FindOne(ctx, bson.M{"username": username}).Decode(&admin)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password))
	return err == nil, nil
}
