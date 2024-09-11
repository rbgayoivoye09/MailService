#!/bin/bash

# 检查是否有go.mod文件
if [ ! -f go.mod ]; then
    echo "Error: go.mod file not found in the current directory."
    exit 1
fi

# 从go.mod文件中提取Go版本
GO_VERSION=$(grep "^go " go.mod | awk '{print $2}')

if [ -z "$GO_VERSION" ]; then
    echo "Error: Unable to find Go version in go.mod file."
    exit 1
fi

echo "Detected Go version: $GO_VERSION"

# 构建下载URL
DOWNLOAD_URL="https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"

# 下载Go，显示进度条
echo "Downloading Go $GO_VERSION..."
wget --progress=bar:force:noscroll $DOWNLOAD_URL -O go.tar.gz

if [ $? -ne 0 ]; then
    echo "Error: Failed to download Go $GO_VERSION."
    rm go.tar.gz
    exit 1
fi

# 移除旧版本的Go（如果存在）
sudo rm -rf /usr/local/go

# 安装新版本的Go
echo "Installing Go $GO_VERSION..."
sudo tar -C /usr/local -xzf go.tar.gz

if [ $? -ne 0 ]; then
    echo "Error: Failed to install Go $GO_VERSION."
    rm go.tar.gz
    exit 1
fi

# 清理下载的压缩包
rm go.tar.gz

# 更新PATH环境变量
echo "Updating PATH..."
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile

# 使环境变量更改立即生效
source /etc/profile

echo "Go $GO_VERSION has been successfully installed."
echo "Please run 'source /etc/profile' or log out and log back in for the PATH changes to take effect."

# 验证安装
go version
