package utils

import (
	"math/rand/v2" // Go 1.22+ 推荐使用 v2
	"strings"
	"time"
)

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// 使用包级别的随机数生成器（更安全）
var rnd = rand.New(rand.NewPCG(
	uint64(time.Now().UnixNano()),
	uint64(time.Now().UnixNano()<<1),
))

// GenerateShortCode 生成6位随机短码（推荐）
func GenerateShortCode() string {
	return GenerateShortCodeWithLength(6)
}

// GenerateShortCodeWithLength 可指定长度
func GenerateShortCodeWithLength(length int) string {
	if length <= 0 {
		length = 6
	}

	var sb strings.Builder
	sb.Grow(length) // 预分配空间，提升性能

	for i := 0; i < length; i++ {
		sb.WriteByte(base62Chars[rnd.IntN(len(base62Chars))])
	}
	return sb.String()
}
