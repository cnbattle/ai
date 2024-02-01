package guid

import (
	"fmt"
	"strconv"
	"time"

	"github.com/yitter/idgenerator-go/idgen"
)

type IdGen struct {
	client *idgen.DefaultIdGenerator
}

// NewID 初始化一个 id gen
func NewID(startTime time.Time, workerID uint16) *IdGen {
	return &IdGen{client: idgen.NewDefaultIdGenerator(&idgen.IdGeneratorOptions{
		Method:            1,
		WorkerId:          workerID,
		BaseTime:          startTime.UnixMilli(),
		WorkerIdBitLength: 6,
		SeqBitLength:      6,
		MaxSeqNumber:      0,
		MinSeqNumber:      5,
		TopOverCostCount:  2000,
	})}
}

func (s *IdGen) NextID() string {
	return strconv.FormatInt(s.client.NewLong(), 10)
}

func (s *IdGen) WithPrefix(prefix string) string {
	return fmt.Sprintf("%v_%v", prefix, s.NextID())
}
