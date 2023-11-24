package ai

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Total struct {
	Total float64
}

// DbDate 自定义时间 2006-01-02
type DbDate time.Time

func (t *DbDate) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var err error
	//前端接收的时间字符串
	str := string(data)
	//去除接收的str收尾多余的"
	timeStr := strings.Trim(str, "\"")
	t1, err := time.Parse("2006-01-02", timeStr)
	*t = DbDate(t1)
	return err
}

func (t *DbDate) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%v\"", time.Time(*t).Format("2006-01-02"))
	return []byte(formatted), nil
}

func (t *DbDate) Value() (driver.Value, error) {
	// DbDate 转换成 time.Time 类型
	tTime := time.Time(*t)
	return tTime.Format("2006-01-02"), nil
}

func (t *DbDate) Scan(v interface{}) error {
	switch vt := v.(type) {
	case time.Time:
		// 字符串转成 time.Time 类型
		*t = DbDate(vt)
	default:
		return errors.New("类型处理错误")
	}
	return nil
}

func (t *DbDate) String() string {
	return fmt.Sprintf("hhh:%s", time.Time(*t).String())
}

// DbTime 自定义时间 2006-01-02 15:04:05
type DbTime time.Time

func (t *DbTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var err error
	//前端接收的时间字符串
	str := string(data)
	//去除接收的str收尾多余的"
	timeStr := strings.Trim(str, "\"")
	t1, err := time.Parse("2006-01-02 15:04:05", timeStr)
	*t = DbTime(t1)
	return err
}

func (t *DbTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%v\"", time.Time(*t).Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

func (t *DbTime) Value() (driver.Value, error) {
	// DbTime 转换成 time.Time 类型
	tTime := time.Time(*t)
	return tTime.Format("2006-01-02 15:04:05"), nil
}

func (t *DbTime) Scan(v interface{}) error {
	switch vt := v.(type) {
	case time.Time:
		// 字符串转成 time.Time 类型
		*t = DbTime(vt)
	default:
		return errors.New("类型处理错误")
	}
	return nil
}

func (t *DbTime) String() string {
	return fmt.Sprintf("hhh:%s", time.Time(*t).String())
}
