package guid

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sony/sonyflake"
)

type Sf struct {
	client *sonyflake.Sonyflake
}

// NewSf 初始化一个 sonyflake
func NewSf(startTime time.Time, workerID uint16) *Sf {
	return &Sf{client: sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: startTime,
		MachineID: func() (uint16, error) {
			return workerID, nil
		},
		CheckMachineID: nil,
	})}
}

func (s *Sf) NextID() string {
	id, _ := s.client.NextID()
	return strconv.FormatInt(int64(id), 10)
}

func (s *Sf) WithPrefix(prefix string) string {
	return fmt.Sprintf("%v_%v", prefix, s.NextID())
}
