#!/usr/bin/env python3
"""
Bearer Token Service V2 - 测试脚本
使用 Python + HMAC 签名认证
"""

import hmac
import hashlib
import base64
import json
import requests
from datetime import datetime
import time

BASE_URL = "http://localhost:8080"

def sign_request(method, uri, timestamp, body, secret_key):
    """生成 HMAC-SHA256 签名"""
    string_to_sign = f"{method}\n{uri}\n{timestamp}\n{body}"
    signature = hmac.new(
        secret_key.encode(),
        string_to_sign.encode(),
        hashlib.sha256
    ).digest()
    return base64.b64encode(signature).decode()

print("=" * 50)
print("Bearer Token Service V2 - Python 测试")
print("=" * 50)
print()

# 测试 1: 账户注册
print("测试 1: 账户注册")
print("-" * 50)

test_email = f"test-{int(time.time())}@example.com"
register_data = {
    "email": test_email,
    "company": "Test Company",
    "password": "TestPassword123"
}

response = requests.post(
    f"{BASE_URL}/api/v2/accounts/register",
    json=register_data
)

print(f"状态码: {response.status_code}")
print(f"响应: {response.text}")

if response.status_code == 201:
    account = response.json()
    ACCESS_KEY = account['access_key']
    SECRET_KEY = account['secret_key']
    ACCOUNT_ID = account['account_id']

    print(f"✅ 账户注册成功")
    print(f"  - Account ID: {ACCOUNT_ID}")
    print(f"  - Access Key: {ACCESS_KEY}")
    print(f"  - Secret Key: {SECRET_KEY[:20]}...")
else:
    print("❌ 账户注册失败")
    exit(1)

print()

# 测试 2: 创建 Token（1小时过期）
print("测试 2: 创建 Token（1小时过期 = 3600秒）")
print("-" * 50)

method = "POST"
uri = "/api/v2/tokens"
timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
body_data = {
    "description": "Test 1-hour token",
    "scope": ["storage:read"],
    "expires_in_seconds": 3600
}
body = json.dumps(body_data)

signature = sign_request(method, uri, timestamp, body, SECRET_KEY)

headers = {
    "Authorization": f"QINIU {ACCESS_KEY}:{signature}",
    "X-Qiniu-Date": timestamp,
    "Content-Type": "application/json"
}

response = requests.post(
    f"{BASE_URL}{uri}",
    headers=headers,
    data=body
)

print(f"状态码: {response.status_code}")
print(f"响应: {response.text}")

if response.status_code == 201:
    token_resp = response.json()
    BEARER_TOKEN = token_resp['token']
    TOKEN_ID = token_resp['token_id']

    print(f"✅ Token 创建成功（1小时过期）")
    print(f"  - Token ID: {TOKEN_ID}")
    print(f"  - Token: {BEARER_TOKEN[:30]}...")
    print(f"  - Expires At: {token_resp.get('expires_at', 'N/A')}")
else:
    print("❌ Token 创建失败")
    exit(1)

print()

# 测试 3: 创建 Token（90天过期）
print("测试 3: 创建 Token（90天过期 = 7776000秒）")
print("-" * 50)

timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
body_data = {
    "description": "Test 90-day token",
    "scope": ["storage:*", "cdn:refresh"],
    "expires_in_seconds": 7776000
}
body = json.dumps(body_data)

signature = sign_request(method, uri, timestamp, body, SECRET_KEY)

headers = {
    "Authorization": f"QINIU {ACCESS_KEY}:{signature}",
    "X-Qiniu-Date": timestamp,
    "Content-Type": "application/json"
}

response = requests.post(
    f"{BASE_URL}{uri}",
    headers=headers,
    data=body
)

print(f"状态码: {response.status_code}")
print(f"响应: {response.text}")

if response.status_code == 201:
    token_resp_90d = response.json()
    BEARER_TOKEN_90D = token_resp_90d['token']

    print(f"✅ Token 创建成功（90天过期）")
    print(f"  - Token: {BEARER_TOKEN_90D[:30]}...")
    print(f"  - Expires At: {token_resp_90d.get('expires_at', 'N/A')}")
else:
    print("❌ Token 创建失败")
    exit(1)

print()

# 测试 4: 验证 Token
print("测试 4: 验证 Token")
print("-" * 50)

response = requests.post(
    f"{BASE_URL}/api/v2/validate",
    headers={
        "Authorization": f"Bearer {BEARER_TOKEN}",
        "Content-Type": "application/json"
    },
    json={"required_scope": "storage:read"}
)

print(f"状态码: {response.status_code}")
print(f"响应: {response.text}")

if response.status_code == 200:
    validate_resp = response.json()
    if validate_resp.get('valid'):
        print(f"✅ Token 验证成功")
        print(f"  - Granted: {validate_resp.get('permission_check', {}).get('granted')}")
    else:
        print("❌ Token 验证失败")
        exit(1)
else:
    print("❌ Token 验证请求失败")
    exit(1)

print()

# 测试 5: 列出 Tokens
print("测试 5: 列出 Tokens")
print("-" * 50)

method = "GET"
uri = "/api/v2/tokens"  # 签名时只用 path，不包含 query
timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
body = ""

signature = sign_request(method, uri, timestamp, body, SECRET_KEY)

headers = {
    "Authorization": f"QINIU {ACCESS_KEY}:{signature}",
    "X-Qiniu-Date": timestamp
}

response = requests.get(
    f"{BASE_URL}{uri}?active_only=true",  # 请求时带上 query
    headers=headers
)

print(f"状态码: {response.status_code}")
print(f"响应: {response.text[:500]}...")

if response.status_code == 200:
    list_resp = response.json()
    total = list_resp.get('total', 0)
    print(f"✅ 列出 Tokens 成功（共 {total} 个）")
else:
    print("❌ 列出 Tokens 失败")
    exit(1)

print()

# 测试 6: 获取账户信息
print("测试 6: 获取账户信息")
print("-" * 50)

method = "GET"
uri = "/api/v2/accounts/me"
timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
body = ""

signature = sign_request(method, uri, timestamp, body, SECRET_KEY)

headers = {
    "Authorization": f"QINIU {ACCESS_KEY}:{signature}",
    "X-Qiniu-Date": timestamp
}

response = requests.get(
    f"{BASE_URL}{uri}",
    headers=headers
)

print(f"状态码: {response.status_code}")
print(f"响应: {response.text}")

if response.status_code == 200:
    account_resp = response.json()
    if account_resp.get('email') == test_email:
        print(f"✅ 获取账户信息成功")
    else:
        print("❌ 账户信息不匹配")
        exit(1)
else:
    print("❌ 获取账户信息失败")
    exit(1)

print()
print("=" * 50)
print("所有测试通过！✅")
print("=" * 50)
print()
print(f"测试账户信息:")
print(f"  - Email: {test_email}")
print(f"  - Access Key: {ACCESS_KEY}")
print(f"  - Secret Key: {SECRET_KEY}")
print()
print(f"生成的 Tokens:")
print(f"  1. 1小时 Token: {BEARER_TOKEN}")
print(f"  2. 90天 Token: {BEARER_TOKEN_90D}")
