#!/usr/bin/env python3
import hmac
import hashlib
import base64
import requests
import json
from datetime import datetime

# 配置
ACCESS_KEY = "AK_b1293071d5d3aad5580d6b97da50c006"
SECRET_KEY = "SK_623fd14a5d37bf72a966191e60e44cc62169fb65d4aed954b63d65635766f523"
BASE_URL = "http://localhost:8080"

# 请求参数
METHOD = "POST"
PATH = "/api/v2/tokens"
TIMESTAMP = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
BODY = {
    "description": "HMAC test token",
    "scope": ["cdn:refresh"],
    "expires_in_seconds": 7200
}
BODY_STR = json.dumps(BODY, separators=(',', ':'))

# 构建待签名字符串
string_to_sign = f"{METHOD}\n{PATH}\n{TIMESTAMP}\n{BODY_STR}"

# 计算签名
signature = base64.b64encode(
    hmac.new(SECRET_KEY.encode(), string_to_sign.encode(), hashlib.sha256).digest()
).decode()

print("=== HMAC 签名认证测试 ===")
print(f"Timestamp: {TIMESTAMP}")
print(f"Signature: {signature[:50]}...")
print()

# 发送请求
headers = {
    "Authorization": f"QINIU {ACCESS_KEY}:{signature}",
    "X-Qiniu-Date": TIMESTAMP,
    "Content-Type": "application/json"
}

response = requests.post(f"{BASE_URL}{PATH}", headers=headers, data=BODY_STR)

print(f"Status: {response.status_code}")
print(f"Response: {response.text}")
