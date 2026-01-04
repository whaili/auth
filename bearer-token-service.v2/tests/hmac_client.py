#!/usr/bin/env python3
"""
HMAC 签名客户端 - 用于测试 Bearer Token Service V2 API

使用方法:
    python3 hmac_client.py create_token <AK> <SK> <description> <scope_json> <days>
    python3 hmac_client.py list_tokens <AK> <SK>
    python3 hmac_client.py get_token <AK> <SK> <token_id>
    python3 hmac_client.py delete_token <AK> <SK> <token_id>
"""

import sys
import json
import hmac
import hashlib
import base64
from datetime import datetime
import requests

BASE_URL = "http://localhost:8081"


class HMACClient:
    """HMAC 签名客户端"""

    def __init__(self, access_key, secret_key, base_url=BASE_URL):
        self.access_key = access_key
        self.secret_key = secret_key
        self.base_url = base_url

    def _sign(self, method, uri, timestamp, body):
        """生成 HMAC-SHA256 签名"""
        # 只使用 path 部分签名，不包含查询参数
        path = uri.split('?')[0]
        string_to_sign = f"{method}\n{path}\n{timestamp}\n{body}"
        signature = hmac.new(
            self.secret_key.encode(),
            string_to_sign.encode(),
            hashlib.sha256
        ).digest()
        return base64.b64encode(signature).decode()

    def _request(self, method, uri, body=""):
        """发送 HMAC 签名请求"""
        timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        signature = self._sign(method, uri, timestamp, body)

        headers = {
            "Authorization": f"QINIU {self.access_key}:{signature}",
            "X-Qiniu-Date": timestamp,
            "Content-Type": "application/json"
        }

        url = f"{self.base_url}{uri}"

        if method == "GET":
            response = requests.get(url, headers=headers)
        elif method == "POST":
            response = requests.post(url, headers=headers, data=body)
        elif method == "PUT":
            response = requests.put(url, headers=headers, data=body)
        elif method == "DELETE":
            response = requests.delete(url, headers=headers)
        else:
            raise ValueError(f"Unsupported method: {method}")

        return response

    def create_token(self, description, scope, expires_in_days, prefix=None):
        """创建 Token"""
        uri = "/api/v2/tokens"
        payload = {
            "description": description,
            "scope": scope,
            "expires_in_days": expires_in_days
        }
        if prefix:
            payload["prefix"] = prefix

        body = json.dumps(payload)
        response = self._request("POST", uri, body)
        return response.json()

    def list_tokens(self, active_only=False, limit=50):
        """列出 Tokens"""
        uri = f"/api/v2/tokens?active_only={str(active_only).lower()}&limit={limit}"
        response = self._request("GET", uri)
        return response.json()

    def get_token(self, token_id):
        """获取 Token 详情"""
        uri = f"/api/v2/tokens/{token_id}"
        response = self._request("GET", uri)
        return response.json()

    def update_token_status(self, token_id, is_active):
        """更新 Token 状态"""
        uri = f"/api/v2/tokens/{token_id}/status"
        body = json.dumps({"is_active": is_active})
        response = self._request("PUT", uri, body)
        return response.json()

    def delete_token(self, token_id):
        """删除 Token"""
        uri = f"/api/v2/tokens/{token_id}"
        response = self._request("DELETE", uri)
        return response.json()

    def get_token_stats(self, token_id):
        """获取 Token 统计"""
        uri = f"/api/v2/tokens/{token_id}/stats"
        response = self._request("GET", uri)
        return response.json()

    def get_account_info(self):
        """获取账户信息"""
        uri = "/api/v2/accounts/me"
        response = self._request("GET", uri)
        return response.json()

    def regenerate_secret_key(self):
        """重新生成 Secret Key"""
        uri = "/api/v2/accounts/regenerate-sk"
        response = self._request("POST", uri)
        return response.json()


def main():
    """命令行入口"""
    if len(sys.argv) < 2:
        print("Usage: hmac_client.py <command> [args...]")
        print("\nCommands:")
        print("  create_token <AK> <SK> <description> <scope_json> <days>")
        print("  list_tokens <AK> <SK>")
        print("  get_token <AK> <SK> <token_id>")
        print("  update_token_status <AK> <SK> <token_id> <true|false>")
        print("  delete_token <AK> <SK> <token_id>")
        print("  get_token_stats <AK> <SK> <token_id>")
        print("  get_account <AK> <SK>")
        print("  regenerate_sk <AK> <SK>")
        sys.exit(1)

    command = sys.argv[1]

    if command == "create_token":
        if len(sys.argv) < 7 or len(sys.argv) > 8:
            print("Usage: create_token <AK> <SK> <description> <scope_json> <days> [prefix]")
            sys.exit(1)

        ak, sk, desc, scope_json, days = sys.argv[2:7]
        prefix = sys.argv[7] if len(sys.argv) == 8 else None
        client = HMACClient(ak, sk)
        scope = json.loads(scope_json)
        result = client.create_token(desc, scope, int(days), prefix)
        print(json.dumps(result, separators=(",", ":")))

    elif command == "list_tokens":
        if len(sys.argv) != 4:
            print("Usage: list_tokens <AK> <SK>")
            sys.exit(1)

        ak, sk = sys.argv[2:4]
        client = HMACClient(ak, sk)
        result = client.list_tokens()
        print(json.dumps(result, separators=(",", ":")))

    elif command == "get_token":
        if len(sys.argv) != 5:
            print("Usage: get_token <AK> <SK> <token_id>")
            sys.exit(1)

        ak, sk, token_id = sys.argv[2:5]
        client = HMACClient(ak, sk)
        result = client.get_token(token_id)
        print(json.dumps(result, separators=(",", ":")))

    elif command == "update_token_status":
        if len(sys.argv) != 6:
            print("Usage: update_token_status <AK> <SK> <token_id> <true|false>")
            sys.exit(1)

        ak, sk, token_id, is_active = sys.argv[2:6]
        client = HMACClient(ak, sk)
        result = client.update_token_status(token_id, is_active.lower() == "true")
        print(json.dumps(result, separators=(",", ":")))

    elif command == "delete_token":
        if len(sys.argv) != 5:
            print("Usage: delete_token <AK> <SK> <token_id>")
            sys.exit(1)

        ak, sk, token_id = sys.argv[2:5]
        client = HMACClient(ak, sk)
        result = client.delete_token(token_id)
        print(json.dumps(result, separators=(",", ":")))

    elif command == "get_token_stats":
        if len(sys.argv) != 5:
            print("Usage: get_token_stats <AK> <SK> <token_id>")
            sys.exit(1)

        ak, sk, token_id = sys.argv[2:5]
        client = HMACClient(ak, sk)
        result = client.get_token_stats(token_id)
        print(json.dumps(result, separators=(",", ":")))

    elif command == "get_account":
        if len(sys.argv) != 4:
            print("Usage: get_account <AK> <SK>")
            sys.exit(1)

        ak, sk = sys.argv[2:4]
        client = HMACClient(ak, sk)
        result = client.get_account_info()
        print(json.dumps(result, separators=(",", ":")))

    elif command == "regenerate_sk":
        if len(sys.argv) != 4:
            print("Usage: regenerate_sk <AK> <SK>")
            sys.exit(1)

        ak, sk = sys.argv[2:4]
        client = HMACClient(ak, sk)
        result = client.regenerate_secret_key()
        print(json.dumps(result, separators=(",", ":")))

    else:
        print(f"Unknown command: {command}")
        sys.exit(1)


if __name__ == "__main__":
    main()
