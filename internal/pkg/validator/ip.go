package validator

import (
	"net"
	"strings"
)

// IsValidIP 验证 IP 地址格式（支持 IPv4 和 IPv6）
func IsValidIP(ip string) bool {
	if ip == "" {
		return false
	}
	return net.ParseIP(ip) != nil
}

// NormalizeIP 规范化 IP 地址
// 移除 IPv6 的 zone identifier (例如 fe80::1%eth0 -> fe80::1)
func NormalizeIP(ip string) string {
	if idx := strings.IndexByte(ip, '%'); idx != -1 {
		return ip[:idx]
	}
	return ip
}

// GetIPOrDefault 获取有效IP或返回默认值
func GetIPOrDefault(ip, defaultIP string) string {
	normalized := NormalizeIP(ip)
	if IsValidIP(normalized) {
		return normalized
	}
	return defaultIP
}
