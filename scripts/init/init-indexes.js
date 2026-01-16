// ========================================
// Bearer Token Service V2 - MongoDB 索引初始化脚本
// ========================================
// 功能：创建所有集合的索引，确保高性能查询和数据一致性
// 执行方式：通过 init-db.sh 自动调用，或手动执行：
//   mongosh mongodb://localhost:27017/token_service_v2 init-indexes.js
// ========================================

print("=====================================");
print("开始创建索引...");
print("=====================================");
print("");

// ========================================
// 1. accounts 集合索引
// ========================================
print("📊 创建 accounts 集合索引...");

try {
    // 1.1 email 唯一索引
    db.accounts.createIndex(
        { email: 1 },
        { unique: true, name: "idx_email_unique" }
    );
    print("  ✅ 创建 email 唯一索引");

    // 1.2 access_key 唯一索引
    db.accounts.createIndex(
        { access_key: 1 },
        { unique: true, name: "idx_access_key_unique" }
    );
    print("  ✅ 创建 access_key 唯一索引");

    // 1.3 status 索引（用于查询活跃账户）
    db.accounts.createIndex(
        { status: 1 },
        { name: "idx_status" }
    );
    print("  ✅ 创建 status 索引");

    // 1.4 qiniu_uid 唯一稀疏索引（可选字段）
    db.accounts.createIndex(
        { qiniu_uid: 1 },
        { unique: true, sparse: true, name: "idx_qiniu_uid_unique_sparse" }
    );
    print("  ✅ 创建 qiniu_uid 唯一稀疏索引");

    // 1.5 created_at 索引（用于排序和分页）
    db.accounts.createIndex(
        { created_at: -1 },
        { name: "idx_created_at" }
    );
    print("  ✅ 创建 created_at 索引");

    print("✅ accounts 集合索引创建完成");
} catch (e) {
    print("⚠️  accounts 集合索引创建警告: " + e.message);
}

print("");

// ========================================
// 2. tokens 集合索引
// ========================================
print("📊 创建 tokens 集合索引...");

try {
    // 2.1 token 唯一索引（核心索引）
    db.tokens.createIndex(
        { token: 1 },
        { unique: true, name: "idx_token_unique" }
    );
    print("  ✅ 创建 token 唯一索引");

    // 2.2 租户隔离复合索引（最重要！）
    db.tokens.createIndex(
        { account_id: 1, is_active: 1 },
        { name: "idx_account_active" }
    );
    print("  ✅ 创建 account_id + is_active 复合索引（租户隔离）");

    // 2.3 租户查询优化索引
    db.tokens.createIndex(
        { account_id: 1, created_at: -1 },
        { name: "idx_account_created" }
    );
    print("  ✅ 创建 account_id + created_at 复合索引（查询优化）");

    // 2.4 expires_at 索引（用于清理过期 Token）
    db.tokens.createIndex(
        { expires_at: 1 },
        { name: "idx_expires_at" }
    );
    print("  ✅ 创建 expires_at 索引（过期清理）");

    // 2.5 last_used_at 索引（用于统计分析）
    db.tokens.createIndex(
        { last_used_at: -1 },
        { sparse: true, name: "idx_last_used_at" }
    );
    print("  ✅ 创建 last_used_at 索引（统计分析）");

    print("✅ tokens 集合索引创建完成");
} catch (e) {
    print("⚠️  tokens 集合索引创建警告: " + e.message);
}

print("");

// ========================================
// 3. audit_logs 集合索引
// ========================================
print("📊 创建 audit_logs 集合索引...");

try {
    // 3.1 租户隔离 + 时间排序复合索引（最常用）
    db.audit_logs.createIndex(
        { account_id: 1, timestamp: -1 },
        { name: "idx_account_timestamp" }
    );
    print("  ✅ 创建 account_id + timestamp 复合索引");

    // 3.2 按操作类型查询索引
    db.audit_logs.createIndex(
        { account_id: 1, action: 1 },
        { name: "idx_account_action" }
    );
    print("  ✅ 创建 account_id + action 复合索引");

    // 3.3 按资源 ID 查询索引
    db.audit_logs.createIndex(
        { account_id: 1, resource_id: 1 },
        { name: "idx_account_resource" }
    );
    print("  ✅ 创建 account_id + resource_id 复合索引");

    // 3.4 时间范围查询索引
    db.audit_logs.createIndex(
        { timestamp: -1 },
        { name: "idx_timestamp" }
    );
    print("  ✅ 创建 timestamp 索引");

    // 3.5 TTL 索引（自动删除 90 天前的日志）
    db.audit_logs.createIndex(
        { timestamp: 1 },
        {
            expireAfterSeconds: 90 * 24 * 60 * 60,  // 90 天
            name: "idx_timestamp_ttl"
        }
    );
    print("  ✅ 创建 timestamp TTL 索引（90天自动删除）");

    print("✅ audit_logs 集合索引创建完成");
} catch (e) {
    print("⚠️  audit_logs 集合索引创建警告: " + e.message);
}

print("");

// ========================================
// 4. 验证索引创建结果
// ========================================
print("=====================================");
print("📋 索引创建结果汇总:");
print("=====================================");
print("");

print("accounts 集合索引:");
db.accounts.getIndexes().forEach(function(idx) {
    print("  - " + idx.name);
});
print("");

print("tokens 集合索引:");
db.tokens.getIndexes().forEach(function(idx) {
    print("  - " + idx.name);
});
print("");

print("audit_logs 集合索引:");
db.audit_logs.getIndexes().forEach(function(idx) {
    print("  - " + idx.name);
});
print("");

print("=====================================");
print("✅ 所有索引创建完成！");
print("=====================================");
