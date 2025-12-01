#!/bin/bash
# 生成开发测试用的自签名证书
# Usage: ./scripts/gen-dev-certs.sh

set -e

CERT_DIR="${1:-/etc/edr/certs}"
DAYS=365

echo "=== 生成 EDR 开发证书 ==="
echo "证书目录: $CERT_DIR"

# 创建目录
sudo mkdir -p "$CERT_DIR"

# 生成 CA 私钥和证书
echo "1. 生成 CA 证书..."
sudo openssl genrsa -out "$CERT_DIR/ca.key" 4096
sudo openssl req -new -x509 -days $DAYS -key "$CERT_DIR/ca.key" -out "$CERT_DIR/ca.crt" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=EDR-POC/OU=Dev/CN=EDR-CA"

# 生成服务器私钥和 CSR
echo "2. 生成服务器证书..."
sudo openssl genrsa -out "$CERT_DIR/server.key" 2048
sudo openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=EDR-POC/OU=Dev/CN=localhost"

# 创建扩展配置文件
sudo tee "$CERT_DIR/server_ext.cnf" > /dev/null << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
DNS.3 = edr-cloud
DNS.4 = api-gateway
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# 使用 CA 签名服务器证书
sudo openssl x509 -req -days $DAYS -in "$CERT_DIR/server.csr" \
    -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
    -out "$CERT_DIR/server.crt" -extfile "$CERT_DIR/server_ext.cnf"

# 生成客户端证书 (用于 Agent mTLS)
echo "3. 生成客户端证书..."
sudo openssl genrsa -out "$CERT_DIR/client.key" 2048
sudo openssl req -new -key "$CERT_DIR/client.key" -out "$CERT_DIR/client.csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=EDR-POC/OU=Agent/CN=edr-agent"

sudo tee "$CERT_DIR/client_ext.cnf" > /dev/null << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

sudo openssl x509 -req -days $DAYS -in "$CERT_DIR/client.csr" \
    -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
    -out "$CERT_DIR/client.crt" -extfile "$CERT_DIR/client_ext.cnf"

# 设置权限
sudo chmod 644 "$CERT_DIR"/*.crt
sudo chmod 600 "$CERT_DIR"/*.key
sudo chmod 644 "$CERT_DIR"/*.cnf 2>/dev/null || true

# 清理 CSR 文件
sudo rm -f "$CERT_DIR"/*.csr

echo ""
echo "=== 证书生成完成 ==="
echo "CA 证书:     $CERT_DIR/ca.crt"
echo "服务器证书:  $CERT_DIR/server.crt"
echo "服务器私钥:  $CERT_DIR/server.key"
echo "客户端证书:  $CERT_DIR/client.crt"
echo "客户端私钥:  $CERT_DIR/client.key"
echo ""
echo "验证证书:"
echo "  openssl x509 -in $CERT_DIR/server.crt -text -noout"
