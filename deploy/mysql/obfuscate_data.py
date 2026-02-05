#!/usr/bin/env python3
"""数据混淆脚本 - 混淆 MySQL 导出文件中的敏感数据"""

import sys
import re
import hashlib
import random

def obfuscate_email(email):
    """混淆邮箱地址 - 保留域名，混淆用户名"""
    if not email or email == 'NULL':
        return email
    parts = email.split('@')
    if len(parts) == 2:
        username_hash = hashlib.md5(parts[0].encode()).hexdigest()[:8]
        return f"test_{username_hash}@{parts[1]}"
    return email

def obfuscate_username(username):
    """混淆用户名"""
    if not username or username == 'NULL':
        return username
    name_hash = hashlib.md5(username.encode()).hexdigest()[:6]
    return f"测试用户_{name_hash}"

def obfuscate_phone(phone):
    """混淆手机号 - 保留前3位和后4位"""
    if not phone or phone == 'NULL':
        return phone
    if len(phone) == 11 and phone.isdigit():
        return phone[:3] + '****' + phone[-4:]
    return phone

def obfuscate_ip(ip):
    """混淆IP地址 - 保留前两段，后两段随机化"""
    if not ip or ip == 'NULL':
        return ip
    parts = ip.split('.')
    if len(parts) == 4:
        try:
            return f"{parts[0]}.{parts[1]}.{random.randint(1,254)}.{random.randint(1,254)}"
        except:
            return ip
    return ip

def obfuscate_password_hash(password):
    """混淆密码 - 使用固定测试密码hash或SHA256格式"""
    if not password or password == 'NULL' or len(password) < 10:
        return password
    # 返回一个固定的测试密码hash
    return hashlib.sha256(b'test_password_123').hexdigest()

def obfuscate_insert_statement(line):
    """混淆 INSERT 语句中的敏感数据"""
    if not line.startswith('INSERT INTO'):
        return line

    # 1. 混淆邮箱地址
    line = re.sub(
        r"'([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})'",
        lambda m: f"'{obfuscate_email(m.group(1))}'",
        line
    )

    # 2. 混淆长哈希值（可能是密码，64个字符的十六进制）
    line = re.sub(
        r"'([a-f0-9]{64})'",
        lambda m: f"'{obfuscate_password_hash(m.group(1))}'",
        line
    )

    # 3. 混淆IP地址
    line = re.sub(
        r"'(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})'",
        lambda m: f"'{obfuscate_ip(m.group(1))}'",
        line
    )

    # 4. 混淆手机号（中国手机号格式）
    line = re.sub(
        r"'(1[3-9]\d{9})'",
        lambda m: f"'{obfuscate_phone(m.group(1))}'",
        line
    )

    return line

def main():
    if len(sys.argv) != 3:
        print("Usage: obfuscate_data.py <input_file> <output_file>", file=sys.stderr)
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    try:
        with open(input_file, 'r', encoding='utf-8') as fin, open(output_file, 'w', encoding='utf-8') as fout:
            for line in fin:
                line = line.rstrip('\n')
                # 只处理 INSERT 语句，其他行（包括注释、建表语句等）直接输出
                if line.startswith('INSERT INTO'):
                    line = obfuscate_insert_statement(line)
                fout.write(line + '\n')
        print(f"Successfully obfuscated data from {input_file} to {output_file}")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
