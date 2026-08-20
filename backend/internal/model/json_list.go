package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONList 字符串列表 JSON 类型，兼容 MySQL JSON 列。
type JSONList []string

// Value 实现 driver.Valuer。
// nil 或空列表序列化为 JSON 空数组 "[]"，而不是 SQL NULL，
// 避免 JSON 列出现 NULL 后被后续 Scan 复用同一接收变量造成跨行串场。
func (j JSONList) Value() (driver.Value, error) {
	if j == nil {
		j = JSONList{}
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner。
func (j *JSONList) Scan(v any) error {
	if v == nil {
		*j = JSONList{}
		return nil
	}
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = x
	case string:
		b = []byte(x)
	default:
		return errors.New("invalid json bytes")
	}
	return json.Unmarshal(b, j)
}
