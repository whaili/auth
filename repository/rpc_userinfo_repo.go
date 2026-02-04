package repository

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/qiniu/bearer-token-service/v2/pkg/qconfapi"
	"github.com/qiniu/xlog.v1"
)

// RPCUserInfoRepository qconfapi RPC 实现的用户信息存储库
type RPCUserInfoRepository struct {
	client *qconfapi.Client
}

// NewRPCUserInfoRepository 创建 RPC 用户信息存储库实例
func NewRPCUserInfoRepository(client *qconfapi.Client) *RPCUserInfoRepository {
	return &RPCUserInfoRepository{
		client: client,
	}
}

// GetUserInfoByUID 根据 UID 通过 qconfapi RPC 查询用户信息
func (r *RPCUserInfoRepository) GetUserInfoByUID(ctx context.Context, uid uint32) (*interfaces.UserInfo, error) {
	// 创建 xlog.Logger（qconfapi 需要）
	xl := xlog.NewWith(fmt.Sprintf("uid=%d", uid))

	// 调用 qconfapi 获取账户信息
	accountInfo, err := r.client.GetAccountInfo(xl, uid)
	if err != nil {
		return nil, fmt.Errorf("qconfapi GetAccountInfo failed: %w", err)
	}

	// 转换为 interfaces.UserInfo
	userInfo := &interfaces.UserInfo{
		UID:            accountInfo.Uid,
		Email:          accountInfo.Email,
		Username:       accountInfo.Username,
		Utype:          accountInfo.Utype,
		Activated:      accountInfo.Activated,
		DisabledType:   int(accountInfo.DisabledType),
		DisabledReason: accountInfo.DisabledReason,
		DisabledAt:     &accountInfo.DisabledAt,
		ParentUID:      accountInfo.ParentUid,
		CreatedAt:      int64(accountInfo.CreatedAt),
		UpdatedAt:      int64(accountInfo.UpdatedAt),
		LastLoginAt:    int64(accountInfo.LastLoginAt),
	}

	return userInfo, nil
}

// ========================================
// 用于初始化 qconfapi Client 的辅助函数
// ========================================

// InitQconfClient 初始化 qconfapi 客户端
// 配置参数从环境变量读取
func InitQconfClient(cfg *qconfapi.Config) (*qconfapi.Client, error) {
	// 验证必需配置
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("qconf AccessKey and SecretKey are required")
	}

	if len(cfg.MasterHosts) == 0 {
		return nil, fmt.Errorf("qconf MasterHosts is required")
	}

	// 创建 qconfapi 客户端
	client := qconfapi.New(cfg)

	return client, nil
}

// ParseQconfConfigFromEnv 从环境变量解析 qconfapi 配置
// 示例环境变量：
//   QCONF_ACCESS_KEY=your_ak
//   QCONF_SECRET_KEY=your_sk
//   QCONF_MASTER_HOSTS=host1:port1,host2:port2
//   QCONF_MC_HOSTS=mc1:port1,mc2:port2
//   QCONF_LC_EXPIRES_MS=300000
//   QCONF_LC_DURATION_MS=60000
//   QCONF_MC_RW_TIMEOUT_MS=1000
func ParseQconfConfigFromEnv() *qconfapi.Config {
	cfg := &qconfapi.Config{
		AccessKey: getEnvOrDefault("QCONF_ACCESS_KEY", ""),
		SecretKey: getEnvOrDefault("QCONF_SECRET_KEY", ""),

		// MasterHosts: 逗号分隔的主机列表
		MasterHosts: parseCommaSeparatedString(getEnvOrDefault("QCONF_MASTER_HOSTS", "")),

		// McHosts: Memcache 主机列表
		McHosts: parseCommaSeparatedString(getEnvOrDefault("QCONF_MC_HOSTS", "")),

		// 本地缓存配置
		LcacheExpires:     parseInt(getEnvOrDefault("QCONF_LC_EXPIRES_MS", "300000")),    // 5分钟
		LcacheDuration:    parseInt(getEnvOrDefault("QCONF_LC_DURATION_MS", "60000")),    // 1分钟
		LcacheChanBufSize: parseInt(getEnvOrDefault("QCONF_LC_CHAN_BUFSIZE", "1000")),

		// Memcache 配置
		McExpires:   int32(parseInt(getEnvOrDefault("QCONF_MC_EXPIRES_S", "300"))),      // 5分钟
		McRWTimeout: int64(parseInt(getEnvOrDefault("QCONF_MC_RW_TIMEOUT_MS", "1000"))), // 1秒
	}

	return cfg
}

// 辅助函数
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func parseCommaSeparatedString(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
