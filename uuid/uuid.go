package uuid

import (
	"github.com/google/uuid"
)

var (
	U uuid.UUID
)

func GetUUID() string {
	return U.String()
}

func InitUUID() {
	// 可使用预定义的命名空间，如 uuid.NamespaceDNS 或自定义 uuid
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // 示例命名空间（DNS）

	// 定义名称（Name）
	name := "example.com"

	// 生成 uuid v5
	U = uuid.NewSHA1(namespace, []byte(name))
}
