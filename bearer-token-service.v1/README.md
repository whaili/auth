这是一个Bearer Token Service实现，Token Service与业务后端架构分离。

```
_____________________         _______________            _____________
|                              |         |                      |           |                  |
|    tokenservice      |<--> |    backend    |  <--> |    portal     |
|                              |         |                      |           |                  |
|____________________|         |______________|           | ___________ |
                                                                                    ^
                                                                                     |
                                                                                    v
                                                                          _______________
                                                                          |       users      |
                                                                          |______________|
```

# HTTP API 接口说明
1. 创建 Token: `POST /api/tokens`
2. 验证 Token: `GET /api/validate`
3. 获取 Token 列表: `GET /api/tokens`
4. 启用/禁用 Token: `PUT /api/tokens/{id}/status`
5. 删除 Token: `DELETE /api/tokens/{id}`

# 测试使用说明
1. 安装依赖:
```shell
go mod tidy
```

2. 启动 MongoDB 服务:
```shell
docker run -d -p 27017:27017 --name mongodb mongo:latest
```

3. 运行token-service服务:
```shell
export MONGO_URI="mongodb://localhost:27017"
export PORT="8080"
go run main.go
```

打开另一个命令行窗口:
```shell
# 创建新令牌（管理员）
curl -v -u admin:adminpassword -H "Content-Type: application/json" http://127.0.0.1:8080/api/tokens --data '{"description":"abc","expires_in_days":365}'
```

```shell
# 验证令牌（用户）
curl -v -H "Authorization: Bearer <TOKEN-VALUE>" http://127.0.0.1:8080/api/validate
```

```shell
# 列举令牌（管理员）
curl -v -u admin:adminpassword -H "Content-Type: application/json" http://127.0.0.1:8080/api/tokens
```

```shell
# 停用/启用令牌（管理员）
curl -v -u admin:adminpassword -XPUT -H "Content-Type: application/json" http://127.0.0.1:8080/api/tokens/<ID-OF-A-TOKEN>/status --data '{"is_active": false}'
```

```shell
# 删除令牌（管理员）
curl -v -u admin:adminpassword -XDELETE -H "Content-Type: application/json" http://127.0.0.1:8080/api/tokens/<ID-OF-A-TOKEN> --data '{"is_active": false}'
```


4. 默认管理员凭据:
* 账号: `admin`
* 密码: `adminpassword`
