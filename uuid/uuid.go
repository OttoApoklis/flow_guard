package uuid

import (
	"github.com/google/uuid"
	"strconv"
	"time"
)

// GetRandomUUID 返回一个随机UUID（v4）
func GetRandomUUID() string {
	return uuid.New().String()
}

// GetNamespacedUUIDWithTimestamp 返回基于命名空间 + 动态名称（带时间戳）的UUID（v5）
// 确保每次调用生成的UUID都不一样
func GetNamespacedUUIDWithTimestamp() string {
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	name := "example.com-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}
