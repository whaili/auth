package repository

import (
	"context"
	"time"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	auditLogsCollection = "audit_logs"
)

// MongoAuditLogRepository MongoDB 实现的审计日志存储库
type MongoAuditLogRepository struct {
	collection *mongo.Collection
}

// NewMongoAuditLogRepository 创建审计日志存储库实例
func NewMongoAuditLogRepository(db *mongo.Database) *MongoAuditLogRepository {
	return &MongoAuditLogRepository{
		collection: db.Collection(auditLogsCollection),
	}
}

// Create 创建审计日志
func (r *MongoAuditLogRepository) Create(ctx context.Context, log *interfaces.AuditLog) error {
	// 生成日志 ID
	if log.ID == "" {
		log.ID = primitive.NewObjectID().Hex()
	}

	// 设置时间戳
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	_, err := r.collection.InsertOne(ctx, log)
	return err
}

// ListByAccountID 查询账户的审计日志
func (r *MongoAuditLogRepository) ListByAccountID(ctx context.Context, accountID string, query *interfaces.AuditLogQuery) ([]interfaces.AuditLog, error) {
	// 构建查询条件 - 租户隔离
	filter := bson.M{
		"account_id": accountID,
	}

	// 可选过滤条件
	if query != nil {
		if query.Action != "" {
			filter["action"] = query.Action
		}

		if query.ResourceID != "" {
			filter["resource_id"] = query.ResourceID
		}

		if !query.StartTime.IsZero() || !query.EndTime.IsZero() {
			timeFilter := bson.M{}
			if !query.StartTime.IsZero() {
				timeFilter["$gte"] = query.StartTime
			}
			if !query.EndTime.IsZero() {
				timeFilter["$lte"] = query.EndTime
			}
			filter["timestamp"] = timeFilter
		}
	}

	// 分页参数
	limit := 50
	offset := 0
	if query != nil {
		if query.Limit > 0 {
			limit = query.Limit
			if limit > 200 {
				limit = 200 // 最大 200 条
			}
		}
		if query.Offset > 0 {
			offset = query.Offset
		}
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}}) // 最新的在前

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []interfaces.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

// CountByAccountID 统计账户的审计日志数量
func (r *MongoAuditLogRepository) CountByAccountID(ctx context.Context, accountID string, query *interfaces.AuditLogQuery) (int64, error) {
	filter := bson.M{
		"account_id": accountID,
	}

	// 可选过滤条件
	if query != nil {
		if query.Action != "" {
			filter["action"] = query.Action
		}

		if query.ResourceID != "" {
			filter["resource_id"] = query.ResourceID
		}

		if !query.StartTime.IsZero() || !query.EndTime.IsZero() {
			timeFilter := bson.M{}
			if !query.StartTime.IsZero() {
				timeFilter["$gte"] = query.StartTime
			}
			if !query.EndTime.IsZero() {
				timeFilter["$lte"] = query.EndTime
			}
			filter["timestamp"] = timeFilter
		}
	}

	return r.collection.CountDocuments(ctx, filter)
}

// DeleteOldLogs 删除旧日志（数据清理）
func (r *MongoAuditLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.collection.DeleteMany(
		ctx,
		bson.M{
			"timestamp": bson.M{
				"$lt": olderThan,
			},
		},
	)

	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// CreateIndexes 创建索引
func (r *MongoAuditLogRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			// 租户隔离 + 时间排序
			Keys: bson.D{
				{Key: "account_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			// 按操作类型查询
			Keys: bson.D{
				{Key: "account_id", Value: 1},
				{Key: "action", Value: 1},
			},
		},
		{
			// 按资源 ID 查询
			Keys: bson.D{
				{Key: "account_id", Value: 1},
				{Key: "resource_id", Value: 1},
			},
		},
		{
			// 时间范围查询
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
		{
			// TTL 索引：自动删除 90 天前的日志
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60), // 90 天
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
