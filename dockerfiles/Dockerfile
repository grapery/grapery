# 使用多阶段构建
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制源代码和配置文件
COPY . .

# 设置 Go 环境变量
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

# 编译应用
RUN go build -o grapes-app .

# 使用轻量级基础镜像
FROM alpine:latest

# 安装基本工具和证书
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建应用目录
WORKDIR /app

# 从 builder 阶段复制编译好的程序
COPY --from=builder /app/grapes-app .

# 复制配置文件（假设配置文件在 configs 目录下）
COPY configs/ /app/configs/

# 暴露端口（根据实际需要修改）
EXPOSE 8080

# 运行应用
CMD ["./grapes-app"]