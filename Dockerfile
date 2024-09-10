# 使用官方的Golang镜像作为基础镜像
FROM golang:1.22-alpine

# 设置工作目录
WORKDIR /app

# 将当前目录下的所有文件复制到工作目录中
COPY . .

# 构建应用程序
RUN go build -o main .

# 暴露端口
EXPOSE 8080

# 运行应用程序
CMD ["./main"]
