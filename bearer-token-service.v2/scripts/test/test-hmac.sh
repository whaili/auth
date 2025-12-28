#!/bin/bash
# HMAC 签名认证测试

ACCESS_KEY="AK_b1293071d5d3aad5580d6b97da50c006"
SECRET_KEY="SK_623fd14a5d37bf72a966191e60e44cc62169fb65d4aed954b63d65635766f523"

METHOD="POST"
PATH="/api/v2/tokens"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY='{"description":"HMAC test token","scope":["cdn:refresh"],"expires_in_seconds":7200}'

# 构建待签名字符串
STRING_TO_SIGN="${METHOD}"$'\n'"${PATH}"$'\n'"${TIMESTAMP}"$'\n'"${BODY}"

# 计算签名
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)

echo "=== 测试 3: HMAC 签名认证 ==="
echo "Timestamp: $TIMESTAMP"
echo "Signature: ${SIGNATURE:0:50}..."
echo ""

curl -s -X POST "http://localhost:8080${PATH}" \
  -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Qiniu-Date: ${TIMESTAMP}" \
  -H "Content-Type: application/json" \
  -d "$BODY"

echo ""
