# Web 服务

这是一个使用 Gin 框架的 Go 语言 Web 服务，对外提供一个符合 RESTful 结构的接口，接收一个 JSON 格式的数据，返回一个 JSON 格式的数据。

## 接口

- 接口地址：/api/v1/getPassword
- 请求方式：POST
- 请求参数：
  - JSON 格式的数据，包含两个字段：username，类型为 string，password，类型为 string
- 返回参数：
  - JSON 格式的数据，包含一个字段：result，类型为 string，base64 编码后的密码

## 示例

请求：

```json
{
  "username": "admin",
  "password": "123456"
}
```

响应：

```json
{
  "result": "YW5kcm9pZDoxMjM0NTY="
}
```

## 构建和运行

1. 克隆仓库到本地
2. 进入项目目录
3. 构建 Docker 镜像
  docker build -t my-web-service .
4. 运行 Docker 容器
  docker run -p 8080:8080 my-web-service

