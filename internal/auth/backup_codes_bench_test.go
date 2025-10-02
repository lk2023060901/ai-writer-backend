package auth

import (
	"testing"
)

// BenchmarkGenerateBackupCode 基准测试：生成单个备用恢复码
func BenchmarkGenerateBackupCode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateBackupCode()
		if err != nil {
			b.Fatalf("GenerateBackupCode() error = %v", err)
		}
	}
}

// BenchmarkGenerateBackupCodes 基准测试：批量生成备用恢复码
func BenchmarkGenerateBackupCodes(b *testing.B) {
	benchmarks := []struct {
		name  string
		count int
	}{
		{"生成 8 个恢复码", 8},
		{"生成 10 个恢复码", 10},
		{"生成 20 个恢复码", 20},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := GenerateBackupCodes(bm.count)
				if err != nil {
					b.Fatalf("GenerateBackupCodes() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkVerifyBackupCode 基准测试：验证备用恢复码
func BenchmarkVerifyBackupCode(b *testing.B) {
	// 预生成测试数据
	plainCodes, backupCodes, err := GenerateBackupCodes(8)
	if err != nil {
		b.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	benchmarks := []struct {
		name      string
		inputCode string
	}{
		{"验证第 1 个恢复码", plainCodes[0]},
		{"验证第 4 个恢复码", plainCodes[3]},
		{"验证最后一个恢复码", plainCodes[7]},
		{"验证错误的恢复码", "0000-0000-0000-0000"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = VerifyBackupCode(backupCodes, bm.inputCode)
			}
		})
	}
}

// BenchmarkVerifyBackupCode_WorstCase 基准测试：最坏情况（验证最后一个恢复码）
func BenchmarkVerifyBackupCode_WorstCase(b *testing.B) {
	plainCodes, backupCodes, err := GenerateBackupCodes(20)
	if err != nil {
		b.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	// 验证最后一个恢复码（需要遍历整个数组）
	lastCode := plainCodes[19]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = VerifyBackupCode(backupCodes, lastCode)
	}
}

// BenchmarkMarkBackupCodeAsUsed 基准测试：标记恢复码为已使用
func BenchmarkMarkBackupCodeAsUsed(b *testing.B) {
	usedIP := "192.168.1.100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, backupCodes, err := GenerateBackupCodes(8)
		if err != nil {
			b.Fatalf("GenerateBackupCodes() error = %v", err)
		}
		b.StartTimer()

		_ = MarkBackupCodeAsUsed(backupCodes, 0, &usedIP)
	}
}

// BenchmarkCountRemainingBackupCodes 基准测试：统计剩余恢复码
func BenchmarkCountRemainingBackupCodes(b *testing.B) {
	benchmarks := []struct {
		name     string
		total    int
		usedIndices []int
	}{
		{"8 个恢复码，全部可用", 8, []int{}},
		{"8 个恢复码，使用 4 个", 8, []int{0, 2, 4, 6}},
		{"20 个恢复码，使用 10 个", 20, []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 18}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// 预生成测试数据
			_, backupCodes, err := GenerateBackupCodes(bm.total)
			if err != nil {
				b.Fatalf("GenerateBackupCodes() error = %v", err)
			}

			usedIP := "192.168.1.1"
			for _, idx := range bm.usedIndices {
				_ = MarkBackupCodeAsUsed(backupCodes, idx, &usedIP)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = CountRemainingBackupCodes(backupCodes)
			}
		})
	}
}

// BenchmarkFormatBackupCode 基准测试：格式化恢复码
func BenchmarkFormatBackupCode(b *testing.B) {
	hexString := "a3f29d7c4e1b8a6f"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatBackupCode(hexString)
	}
}

// BenchmarkFullFlow 基准测试：完整流程（生成 → 验证 → 标记）
func BenchmarkFullFlow(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 1. 生成恢复码
		plainCodes, backupCodes, err := GenerateBackupCodes(8)
		if err != nil {
			b.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		// 2. 验证第 3 个恢复码
		index, valid, err := VerifyBackupCode(backupCodes, plainCodes[2])
		if err != nil {
			b.Fatalf("VerifyBackupCode() error = %v", err)
		}
		if !valid {
			b.Fatalf("VerifyBackupCode() valid = false")
		}

		// 3. 标记为已使用
		usedIP := "192.168.1.100"
		err = MarkBackupCodeAsUsed(backupCodes, index, &usedIP)
		if err != nil {
			b.Fatalf("MarkBackupCodeAsUsed() error = %v", err)
		}

		// 4. 统计剩余数量
		_ = CountRemainingBackupCodes(backupCodes)
	}
}

// BenchmarkParallelGenerate 基准测试：并发生成恢复码
func BenchmarkParallelGenerate(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := GenerateBackupCode()
			if err != nil {
				b.Fatalf("GenerateBackupCode() error = %v", err)
			}
		}
	})
}

// BenchmarkParallelVerify 基准测试：并发验证恢复码
func BenchmarkParallelVerify(b *testing.B) {
	// 预生成测试数据
	plainCodes, backupCodes, err := GenerateBackupCodes(8)
	if err != nil {
		b.Fatalf("GenerateBackupCodes() error = %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = VerifyBackupCode(backupCodes, plainCodes[0])
		}
	})
}
