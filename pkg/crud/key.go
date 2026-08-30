package crud

import (
	"fmt"
	"strings"
)

// KeyBuilder 缓存 key 构建器。
type KeyBuilder struct {
	separator string
}

// NewKeyBuilder 创建 KeyBuilder，separator 默认 ":"。
func NewKeyBuilder() *KeyBuilder {
	return &KeyBuilder{separator: ":"}
}

// WithSeparator 设置分隔符。
func (b *KeyBuilder) WithSeparator(sep string) *KeyBuilder {
	b.separator = sep
	return b
}

// Primary 构建主键缓存 key。
// 格式: {prefix}:{table}:id:{primaryKey}
func (b *KeyBuilder) Primary(prefix, table string, pk any) string {
	return fmt.Sprintf("%s%s%s%sid%s%v", prefix, b.separator, table, b.separator, b.separator, pk)
}

// Field 构建单字段索引缓存 key。
// 格式: {prefix}:{table}:{field}:{value}
func (b *KeyBuilder) Field(prefix, table, field string, value any) string {
	return fmt.Sprintf("%s%s%s%s%s%s%v", prefix, b.separator, table, b.separator, field, b.separator, value)
}

// Composite 构建复合索引缓存 key。
// 格式: {prefix}:{table}:idx_{field1}|{field2}:{value1}|{value2}
func (b *KeyBuilder) Composite(prefix, table string, fields []string, values []any) string {
	if len(fields) != len(values) {
		panic(fmt.Sprintf("crud: Composite fields/values length mismatch: %d vs %d", len(fields), len(values)))
	}
	fieldPart := strings.Join(fields, "|")
	valParts := make([]string, len(values))
	for i, v := range values {
		valParts[i] = fmt.Sprintf("%v", v)
	}
	valuePart := strings.Join(valParts, "|")
	return fmt.Sprintf("%s%s%s%sidx_%s%s%s", prefix, b.separator, table, b.separator, fieldPart, b.separator, valuePart)
}

// MatchPrefix 匹配前缀，用于批量删除。
func (b *KeyBuilder) MatchPrefix(prefix, table string) string {
	return fmt.Sprintf("%s%s%s%s", prefix, b.separator, table, b.separator)
}
