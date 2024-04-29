package guid

import (
	"go.jetify.com/typeid"
)

type TypeIDGen struct {
}

// NewTypeIDGen 初始化一个 TypeIDGen
func NewTypeIDGen() *TypeIDGen {
	return &TypeIDGen{}
}

func (s *TypeIDGen) NextID() string {
	tid, _ := typeid.WithPrefix("aif")
	return tid.String()
}

func (s *TypeIDGen) WithPrefix(prefix string) string {
	tid, _ := typeid.WithPrefix(prefix)
	return tid.String()
}
