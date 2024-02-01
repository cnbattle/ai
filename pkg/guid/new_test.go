package guid

import (
	"fmt"
	"testing"
	"time"
)

func TestNewIdGen(t *testing.T) {
	id, _ := New(IDGEN, time.Date(2022, 12, 12, 12, 12, 12, 12, time.UTC), 1)
	for i := 0; i < 10; i++ {
		fmt.Println(id.NextID())
	}
	for i := 0; i < 10; i++ {
		fmt.Println(id.WithPrefix("aif"))
	}
}

func TestNewSonyflake(t *testing.T) {
	id, _ := New(SONYFLAKE, time.Date(2022, 12, 12, 12, 12, 12, 12, time.UTC), 1)
	for i := 0; i < 10; i++ {
		fmt.Println(id.NextID())
	}
	for i := 0; i < 10; i++ {
		fmt.Println(id.WithPrefix("aif"))
	}
}

func TestUUID(t *testing.T) {
	id, _ := New(UUID, time.Date(2022, 12, 12, 12, 12, 12, 12, time.UTC), 1)
	for i := 0; i < 10; i++ {
		fmt.Println(id.NextID())
	}
	for i := 0; i < 10; i++ {
		fmt.Println(id.WithPrefix("aif"))
	}
}

func TestTypeID(t *testing.T) {
	id, _ := New(TYPEID, time.Date(2022, 12, 12, 12, 12, 12, 12, time.UTC), 1)
	for i := 0; i < 10; i++ {
		fmt.Println(id.NextID())
	}
	for i := 0; i < 10; i++ {
		fmt.Println(id.WithPrefix("ai"))
	}
}
