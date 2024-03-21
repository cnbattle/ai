package uarand

import (
	"fmt"
	"testing"
)

func TestGetRandom(t *testing.T) {
	for k := 0; k < len(UserAgents)*10; k++ {
		fmt.Println(GetRandom())
	}
}
