#!/bin/bash

# 开发环境启动脚本 - 支持优雅退出

set -e

CONFIG_FILE="${1:-config.yaml}"
PID_FILE="/tmp/ai-writer-dev.pid"
LOG_FILE="/tmp/ai-writer-dev.log"

# 清理函数
cleanup() {
    echo ""
    echo "🛑 Shutting down server..."

    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            echo "Sending SIGTERM to process $PID..."
            kill -TERM "$PID"

            # 等待最多10秒
            for i in {1..10}; do
                if ! kill -0 "$PID" 2>/dev/null; then
                    echo "✅ Server stopped gracefully"
                    rm -f "$PID_FILE"
                    exit 0
                fi
                sleep 1
            done

            # 强制杀死
            echo "⚠️  Force killing process $PID..."
            kill -9 "$PID" 2>/dev/null || true
        fi
        rm -f "$PID_FILE"
    fi

    # 确保清理所有相关进程
    pkill -f "go run cmd/server/main.go" 2>/dev/null || true

    echo "👋 Cleanup complete"
    exit 0
}

# 注册信号处理
trap cleanup INT TERM EXIT

echo "🚀 Starting AI Writer in development mode..."
echo "📝 Config: $CONFIG_FILE"
echo "📄 Logs: $LOG_FILE"
echo "🔑 PID file: $PID_FILE"
echo ""
echo "Press Ctrl+C to stop"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 启动服务器（前台运行）
go run cmd/server/main.go -config="$CONFIG_FILE" 2>&1 | tee "$LOG_FILE" &
SERVER_PID=$!

# 保存 PID
echo $SERVER_PID > "$PID_FILE"

# 等待进程
wait $SERVER_PID
