package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BackupCodeLength 备用恢复码长度（16位十六进制 = 64 bits 熵）
	BackupCodeLength = 16
	// BackupCodeCount 生成的备用恢复码数量
	BackupCodeCount = 8
	// BackupCodeBcryptCost bcrypt 哈希成本（推荐 12）
	BackupCodeBcryptCost = 12
)

// BackupCode 备用恢复码结构
type BackupCode struct {
	Hash   string     `json:"hash"`             // bcrypt 哈希值
	Used   bool       `json:"used"`             // 是否已使用
	UsedAt *time.Time `json:"used_at,omitempty"` // 使用时间
	UsedIP *string    `json:"used_ip,omitempty"` // 使用 IP
}

// GenerateBackupCode 生成单个 16 位十六进制备用恢复码
// 格式：xxxx-xxxx-xxxx-xxxx（如 a3f2-9d7c-4e1b-8a6f）
// 返回：明文码（带格式化）、bcrypt 哈希值、错误
func GenerateBackupCode() (plainCode string, hashedCode string, err error) {
	// 生成 8 字节（64 bits）随机数
	randomBytes := make([]byte, BackupCodeLength/2)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 转为 16 位十六进制字符串（无分隔符）
	hexString := hex.EncodeToString(randomBytes)

	// 生成 bcrypt 哈希（基于无分隔符的原始字符串）
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(hexString), BackupCodeBcryptCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash backup code: %w", err)
	}

	// 格式化为 xxxx-xxxx-xxxx-xxxx（仅用于显示）
	plainCode = formatBackupCode(hexString)

	return plainCode, string(hashedBytes), nil
}

// GenerateBackupCodes 生成多个备用恢复码
// 返回：明文码数组、BackupCode 结构数组（含哈希）、错误
func GenerateBackupCodes(count int) (plainCodes []string, backupCodes []BackupCode, err error) {
	if count <= 0 {
		return nil, nil, fmt.Errorf("count must be greater than 0")
	}

	plainCodes = make([]string, 0, count)
	backupCodes = make([]BackupCode, 0, count)

	for i := 0; i < count; i++ {
		plain, hashed, err := GenerateBackupCode()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate backup code %d: %w", i+1, err)
		}

		plainCodes = append(plainCodes, plain)
		backupCodes = append(backupCodes, BackupCode{
			Hash: hashed,
			Used: false,
		})
	}

	return plainCodes, backupCodes, nil
}

// VerifyBackupCode 验证备用恢复码
// backupCodes: 数据库中的 BackupCode 数组
// inputCode: 用户输入的明文恢复码
// 返回：匹配的索引、是否有效、错误
func VerifyBackupCode(backupCodes []BackupCode, inputCode string) (matchIndex int, valid bool, err error) {
	// 移除可能的空格和分隔符，转为小写
	inputCode = strings.ToLower(inputCode)
	inputCode = strings.ReplaceAll(inputCode, " ", "")
	inputCode = strings.ReplaceAll(inputCode, "-", "")

	for i, code := range backupCodes {
		// 跳过已使用的恢复码
		if code.Used {
			continue
		}

		// 验证 bcrypt 哈希
		err := bcrypt.CompareHashAndPassword([]byte(code.Hash), []byte(inputCode))
		if err == nil {
			return i, true, nil
		}
	}

	return -1, false, nil
}

// MarkBackupCodeAsUsed 标记备用恢复码为已使用
// backupCodes: BackupCode 数组（会被修改）
// index: 要标记的索引
// usedIP: 使用者的 IP 地址（可选）
func MarkBackupCodeAsUsed(backupCodes []BackupCode, index int, usedIP *string) error {
	if index < 0 || index >= len(backupCodes) {
		return fmt.Errorf("invalid backup code index: %d", index)
	}

	now := time.Now()
	backupCodes[index].Used = true
	backupCodes[index].UsedAt = &now
	backupCodes[index].UsedIP = usedIP

	return nil
}

// CountRemainingBackupCodes 统计剩余可用的备用恢复码数量
func CountRemainingBackupCodes(backupCodes []BackupCode) int {
	count := 0
	for _, code := range backupCodes {
		if !code.Used {
			count++
		}
	}
	return count
}

// formatBackupCode 格式化为 xxxx-xxxx-xxxx-xxxx
func formatBackupCode(hexString string) string {
	if len(hexString) != BackupCodeLength {
		return hexString
	}

	parts := []string{
		hexString[0:4],
		hexString[4:8],
		hexString[8:12],
		hexString[12:16],
	}

	return strings.Join(parts, "-")
}

// GenerateRandomToken 生成随机 token（用于邮箱验证、密码重置等）
func GenerateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
