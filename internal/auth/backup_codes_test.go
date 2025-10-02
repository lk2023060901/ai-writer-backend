package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestGenerateBackupCode 测试单个备用恢复码生成
func TestGenerateBackupCode(t *testing.T) {
	plainCode, hashedCode, err := GenerateBackupCode()
	if err != nil {
		t.Fatalf("GenerateBackupCode() error = %v", err)
	}

	// 验证明文格式：xxxx-xxxx-xxxx-xxxx
	if len(plainCode) != 19 { // 16 字符 + 3 个分隔符
		t.Errorf("plainCode length = %d, want 19", len(plainCode))
	}

	parts := strings.Split(plainCode, "-")
	if len(parts) != 4 {
		t.Errorf("plainCode parts = %d, want 4", len(parts))
	}

	for i, part := range parts {
		if len(part) != 4 {
			t.Errorf("part[%d] length = %d, want 4", i, len(part))
		}
	}

	// 验证哈希格式（bcrypt 以 $2a$ 开头）
	if !strings.HasPrefix(hashedCode, "$2a$") {
		t.Errorf("hashedCode prefix = %s, want $2a$", hashedCode[:4])
	}

	// 验证哈希可以匹配明文（移除分隔符）
	plainCodeWithoutDashes := strings.ReplaceAll(plainCode, "-", "")
	err = bcrypt.CompareHashAndPassword([]byte(hashedCode), []byte(plainCodeWithoutDashes))
	if err != nil {
		t.Errorf("bcrypt.CompareHashAndPassword() error = %v", err)
	}
}

// TestGenerateBackupCodes 测试批量生成备用恢复码
func TestGenerateBackupCodes(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		wantError bool
	}{
		{"生成 8 个恢复码", 8, false},
		{"生成 10 个恢复码", 10, false},
		{"生成 1 个恢复码", 1, false},
		{"无效数量 0", 0, true},
		{"无效数量 -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainCodes, backupCodes, err := GenerateBackupCodes(tt.count)

			if tt.wantError {
				if err == nil {
					t.Errorf("GenerateBackupCodes() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateBackupCodes() error = %v", err)
			}

			if len(plainCodes) != tt.count {
				t.Errorf("plainCodes length = %d, want %d", len(plainCodes), tt.count)
			}

			if len(backupCodes) != tt.count {
				t.Errorf("backupCodes length = %d, want %d", len(backupCodes), tt.count)
			}

			// 验证所有恢复码唯一性
			seenCodes := make(map[string]bool)
			for i, code := range plainCodes {
				if seenCodes[code] {
					t.Errorf("duplicate code at index %d: %s", i, code)
				}
				seenCodes[code] = true

				// 验证每个 BackupCode 结构
				if backupCodes[i].Used {
					t.Errorf("backupCodes[%d].Used = true, want false", i)
				}
				if backupCodes[i].UsedAt != nil {
					t.Errorf("backupCodes[%d].UsedAt = %v, want nil", i, backupCodes[i].UsedAt)
				}
			}
		})
	}
}

// TestVerifyBackupCode 测试备用恢复码验证
func TestVerifyBackupCode(t *testing.T) {
	// 生成测试用的恢复码
	plainCodes, backupCodes, err := GenerateBackupCodes(5)
	if err != nil {
		t.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	tests := []struct {
		name       string
		inputCode  string
		wantIndex  int
		wantValid  bool
		wantError  bool
	}{
		{"正确的恢复码（索引 0）", plainCodes[0], 0, true, false},
		{"正确的恢复码（索引 2）", plainCodes[2], 2, true, false},
		{"大写格式", strings.ToUpper(plainCodes[1]), 1, true, false},
		{"带空格格式", strings.ReplaceAll(plainCodes[3], "-", " "), 3, true, false},
		{"错误的恢复码", "0000-0000-0000-0000", -1, false, false},
		{"格式错误的恢复码", "invalid", -1, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, valid, err := VerifyBackupCode(backupCodes, tt.inputCode)

			if tt.wantError {
				if err == nil {
					t.Errorf("VerifyBackupCode() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("VerifyBackupCode() error = %v", err)
			}

			if valid != tt.wantValid {
				t.Errorf("VerifyBackupCode() valid = %v, want %v", valid, tt.wantValid)
			}

			if tt.wantValid && index != tt.wantIndex {
				t.Errorf("VerifyBackupCode() index = %d, want %d", index, tt.wantIndex)
			}
		})
	}
}

// TestVerifyBackupCode_UsedCode 测试已使用的恢复码无法再次验证
func TestVerifyBackupCode_UsedCode(t *testing.T) {
	plainCodes, backupCodes, err := GenerateBackupCodes(3)
	if err != nil {
		t.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	// 标记第 1 个恢复码为已使用
	usedIP := "192.168.1.100"
	err = MarkBackupCodeAsUsed(backupCodes, 1, &usedIP)
	if err != nil {
		t.Fatalf("MarkBackupCodeAsUsed() error = %v", err)
	}

	// 尝试验证已使用的恢复码
	index, valid, err := VerifyBackupCode(backupCodes, plainCodes[1])
	if err != nil {
		t.Fatalf("VerifyBackupCode() error = %v", err)
	}

	if valid {
		t.Errorf("VerifyBackupCode() valid = true for used code, want false")
	}

	if index != -1 {
		t.Errorf("VerifyBackupCode() index = %d for used code, want -1", index)
	}

	// 验证未使用的恢复码仍然有效
	index, valid, err = VerifyBackupCode(backupCodes, plainCodes[0])
	if err != nil {
		t.Fatalf("VerifyBackupCode() error = %v", err)
	}

	if !valid {
		t.Errorf("VerifyBackupCode() valid = false for unused code, want true")
	}
}

// TestMarkBackupCodeAsUsed 测试标记恢复码为已使用
func TestMarkBackupCodeAsUsed(t *testing.T) {
	_, backupCodes, err := GenerateBackupCodes(3)
	if err != nil {
		t.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	usedIP := "203.0.113.42"

	tests := []struct {
		name      string
		index     int
		wantError bool
	}{
		{"有效索引 0", 0, false},
		{"有效索引 2", 2, false},
		{"无效索引 -1", -1, true},
		{"无效索引 999", 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MarkBackupCodeAsUsed(backupCodes, tt.index, &usedIP)

			if tt.wantError {
				if err == nil {
					t.Errorf("MarkBackupCodeAsUsed() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("MarkBackupCodeAsUsed() error = %v", err)
			}

			// 验证标记成功
			if !backupCodes[tt.index].Used {
				t.Errorf("backupCodes[%d].Used = false, want true", tt.index)
			}

			if backupCodes[tt.index].UsedAt == nil {
				t.Errorf("backupCodes[%d].UsedAt = nil, want timestamp", tt.index)
			}

			if backupCodes[tt.index].UsedIP == nil || *backupCodes[tt.index].UsedIP != usedIP {
				t.Errorf("backupCodes[%d].UsedIP = %v, want %s", tt.index, backupCodes[tt.index].UsedIP, usedIP)
			}
		})
	}
}

// TestCountRemainingBackupCodes 测试统计剩余恢复码
func TestCountRemainingBackupCodes(t *testing.T) {
	_, backupCodes, err := GenerateBackupCodes(5)
	if err != nil {
		t.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	// 初始状态：5 个可用
	remaining := CountRemainingBackupCodes(backupCodes)
	if remaining != 5 {
		t.Errorf("CountRemainingBackupCodes() = %d, want 5", remaining)
	}

	// 使用 2 个恢复码
	usedIP := "192.168.1.1"
	MarkBackupCodeAsUsed(backupCodes, 0, &usedIP)
	MarkBackupCodeAsUsed(backupCodes, 2, &usedIP)

	// 剩余 3 个
	remaining = CountRemainingBackupCodes(backupCodes)
	if remaining != 3 {
		t.Errorf("CountRemainingBackupCodes() = %d, want 3", remaining)
	}

	// 使用全部
	MarkBackupCodeAsUsed(backupCodes, 1, &usedIP)
	MarkBackupCodeAsUsed(backupCodes, 3, &usedIP)
	MarkBackupCodeAsUsed(backupCodes, 4, &usedIP)

	// 剩余 0 个
	remaining = CountRemainingBackupCodes(backupCodes)
	if remaining != 0 {
		t.Errorf("CountRemainingBackupCodes() = %d, want 0", remaining)
	}
}

// TestFormatBackupCode 测试格式化函数
func TestFormatBackupCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"标准 16 位", "a3f29d7c4e1b8a6f", "a3f2-9d7c-4e1b-8a6f"},
		{"全数字", "1234567890abcdef", "1234-5678-90ab-cdef"},
		{"长度不足", "abc", "abc"},
		{"长度超出", "a3f29d7c4e1b8a6f7b4e", "a3f29d7c4e1b8a6f7b4e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBackupCode(tt.input)
			if got != tt.want {
				t.Errorf("formatBackupCode() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestBackupCode_Uniqueness 测试恢复码唯一性（统计学测试）
func TestBackupCode_Uniqueness(t *testing.T) {
	const iterations = 20 // 减少迭代次数以避免 bcrypt 导致的超时
	seenCodes := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		plainCode, _, err := GenerateBackupCode()
		if err != nil {
			t.Fatalf("GenerateBackupCode() error = %v", err)
		}

		if seenCodes[plainCode] {
			t.Fatalf("duplicate code detected: %s", plainCode)
		}
		seenCodes[plainCode] = true
	}

	if len(seenCodes) != iterations {
		t.Errorf("generated %d codes, want %d", len(seenCodes), iterations)
	}
}
