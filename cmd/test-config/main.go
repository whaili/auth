package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("=========================================")
	fmt.Println("Bearer Token Service V2 - 配置验证")
	fmt.Println("=========================================")
	fmt.Println("")

	// 1. 账户查询配置
	accountFetcherMode := os.Getenv("ACCOUNT_FETCHER_MODE")
	if accountFetcherMode == "" {
		accountFetcherMode = "local"
	}

	fmt.Println("📌 账户查询配置 (AccountFetcher):")
	fmt.Printf("   模式: %s\n", accountFetcherMode)

	if accountFetcherMode == "external" {
		apiURL := os.Getenv("EXTERNAL_ACCOUNT_API_URL")
		apiToken := os.Getenv("EXTERNAL_ACCOUNT_API_TOKEN")
		if apiToken == "" {
			apiToken = "(未设置)"
		}
		fmt.Printf("   API URL: %s\n", apiURL)
		fmt.Printf("   API Token: %s\n", maskToken(apiToken))
		fmt.Println("   ✅ 使用外部 API 查询账户信息")
	} else {
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			mongoURI = "mongodb://localhost:27017"
		}
		fmt.Printf("   MongoDB URI: %s\n", mongoURI)
		fmt.Println("   ✅ 使用本地 MongoDB 查询账户信息")
	}
	fmt.Println("")

	// 2. UID 映射器配置
	mapperMode := os.Getenv("QINIU_UID_MAPPER_MODE")
	if mapperMode == "" {
		mapperMode = "simple"
	}

	fmt.Println("📌 七牛 UID 映射配置 (QiniuUIDMapper):")
	fmt.Printf("   模式: %s\n", mapperMode)

	if mapperMode == "database" {
		autoCreate := os.Getenv("QINIU_UID_AUTO_CREATE") == "true"
		fmt.Printf("   自动创建账户: %v\n", autoCreate)
		fmt.Println("   ✅ 使用数据库映射（查询或创建）")
	} else {
		fmt.Println("   映射规则: qiniu_{uid}")
		fmt.Println("   ✅ 使用简单映射（直接拼接）")
	}
	fmt.Println("")

	// 3. HMAC 配置
	tolerance := os.Getenv("HMAC_TIMESTAMP_TOLERANCE")
	if tolerance == "" {
		tolerance = "15m"
	}

	fmt.Println("📌 HMAC 签名配置:")
	fmt.Printf("   时间戳容忍度: %s\n", tolerance)
	fmt.Println("")

	// 4. 场景推荐
	fmt.Println("=========================================")
	fmt.Println("💡 当前配置适用场景:")
	fmt.Println("=========================================")

	if accountFetcherMode == "local" && mapperMode == "simple" {
		fmt.Println("✓ 开发环境")
		fmt.Println("✓ 独立部署的服务")
		fmt.Println("✓ 快速原型验证")
	} else if accountFetcherMode == "external" && mapperMode == "database" {
		fmt.Println("✓ 生产环境（推荐）")
		fmt.Println("✓ 使用共用账户系统")
		fmt.Println("✓ 需要完整的账户管理")
	} else if accountFetcherMode == "external" && mapperMode == "simple" {
		fmt.Println("✓ 混合模式")
		fmt.Println("✓ HMAC 用户来自外部系统")
		fmt.Println("✓ Qstub 用户临时访问")
	} else {
		fmt.Println("✓ 自定义配置")
	}

	fmt.Println("")
	fmt.Println("=========================================")
	fmt.Println("✅ 配置验证完成")
	fmt.Println("=========================================")
}

func maskToken(token string) string {
	if token == "(未设置)" {
		return token
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
