package ai

import (
	"time"

	"github.com/spf13/cast"
)

// ToBool casts an interface to a bool type.
func ToBool(i interface{}) bool {
	return cast.ToBool(i)
}

// ToTime casts an interface to a time.Time type.
func ToTime(i interface{}) time.Time {
	return cast.ToTime(i)
}

func ToTimeInDefaultLocation(i interface{}, location *time.Location) time.Time {
	return cast.ToTimeInDefaultLocation(i, location)
}

// ToDuration casts an interface to a time.Duration type.
func ToDuration(i interface{}) time.Duration {
	return cast.ToDuration(i)
}

// ToFloat64 casts an interface to a float64 type.
func ToFloat64(i interface{}) float64 {
	return cast.ToFloat64(i)
}

// ToFloat32 casts an interface to a float32 type.
func ToFloat32(i interface{}) float32 {
	return cast.ToFloat32(i)
}

// ToInt64 casts an interface to an int64 type.
func ToInt64(i interface{}) int64 {
	return cast.ToInt64(i)
}

// ToInt32 casts an interface to an int32 type.
func ToInt32(i interface{}) int32 {
	return cast.ToInt32(i)
}

// ToInt16 casts an interface to an int16 type.
func ToInt16(i interface{}) int16 {
	return cast.ToInt16(i)
}

// ToInt8 casts an interface to an int8 type.
func ToInt8(i interface{}) int8 {
	return cast.ToInt8(i)
}

// ToInt casts an interface to an int type.
func ToInt(i interface{}) int {
	return cast.ToInt(i)
}

// ToUint casts an interface to a uint type.
func ToUint(i interface{}) uint {
	return cast.ToUint(i)
}

// ToUint64 casts an interface to a uint64 type.
func ToUint64(i interface{}) uint64 {
	return cast.ToUint64(i)
}

// ToUint32 casts an interface to a uint32 type.
func ToUint32(i interface{}) uint32 {
	return cast.ToUint32(i)
}

// ToUint16 casts an interface to a uint16 type.
func ToUint16(i interface{}) uint16 {
	return cast.ToUint16(i)
}

// ToUint8 casts an interface to a uint8 type.
func ToUint8(i interface{}) uint8 {
	return cast.ToUint8(i)
}

// ToString casts an interface to a string type.
func ToString(i interface{}) string {
	return cast.ToString(i)
}

// ToStringMapString casts an interface to a map[string]string type.
func ToStringMapString(i interface{}) map[string]string {
	return cast.ToStringMapString(i)
}

// ToStringMapStringSlice casts an interface to a map[string][]string type.
func ToStringMapStringSlice(i interface{}) map[string][]string {
	return cast.ToStringMapStringSlice(i)
}

// ToStringMapBool casts an interface to a map[string]bool type.
func ToStringMapBool(i interface{}) map[string]bool {
	return cast.ToStringMapBool(i)
}

// ToStringMapInt casts an interface to a map[string]int type.
func ToStringMapInt(i interface{}) map[string]int {
	return cast.ToStringMapInt(i)
}

// ToStringMapInt64 casts an interface to a map[string]int64 type.
func ToStringMapInt64(i interface{}) map[string]int64 {
	return cast.ToStringMapInt64(i)
}

// ToStringMap casts an interface to a map[string]interface{} type.
func ToStringMap(i interface{}) map[string]interface{} {
	return cast.ToStringMap(i)
}

// ToSlice casts an interface to a []interface{} type.
func ToSlice(i interface{}) []interface{} {
	return cast.ToSlice(i)
}

// ToBoolSlice casts an interface to a []bool type.
func ToBoolSlice(i interface{}) []bool {
	return cast.ToBoolSlice(i)
}

// ToStringSlice casts an interface to a []string type.
func ToStringSlice(i interface{}) []string {
	return cast.ToStringSlice(i)
}

// ToIntSlice casts an interface to a []int type.
func ToIntSlice(i interface{}) []int {
	return cast.ToIntSlice(i)
}

// ToDurationSlice casts an interface to a []time.Duration type.
func ToDurationSlice(i interface{}) []time.Duration {
	return cast.ToDurationSlice(i)
}
