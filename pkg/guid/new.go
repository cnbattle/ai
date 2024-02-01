package guid

import (
	"errors"
	"time"
)

var SONYFLAKE = "sonyflake"
var IDGEN = "idgen"
var UUID = "uuid"
var TYPEID = "typeid"

// New 初始化一个id 生成器
func New(engine string, startTime time.Time, workerID uint16) (Id, error) {
	switch engine {
	case IDGEN:
		return NewID(startTime, workerID), nil
	case SONYFLAKE:
		return NewSf(startTime, workerID), nil
	case UUID:
		return NewUUIDGen(), nil
	case TYPEID:
		return NewTypeIDGen(), nil
	}
	return nil, errors.New("engine does not support")
}
