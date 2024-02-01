package guid

import (
	"fmt"

	"github.com/gofrs/uuid/v5"
)

type UUIDGen struct {
	gen *uuid.Gen
}

// NewUUIDGen 初始化一个 UUIDGen
func NewUUIDGen() *UUIDGen {
	return &UUIDGen{gen: uuid.NewGen()}
}

func (s *UUIDGen) NextID() string {
	id, _ := s.gen.NewV7()
	return id.String()
}

func (s *UUIDGen) WithPrefix(prefix string) string {
	return fmt.Sprintf("%v_%v", prefix, s.NextID())
}
