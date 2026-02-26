package model

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
)

type JSON map[string]any

func (j JSON) GormDataType() string {
	return "json"
}

func (j *JSON) Scan(value any) error {
	if bytes, ok := value.([]byte); ok {
		result := make(map[string]any)
		err := sonic.Unmarshal(bytes, &result)
		if err != nil {
			return err
		}
		*j = result
		return nil
	}
	if result, ok := value.(map[string]any); ok {
		*j = result
		return nil
	}
	return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return sonic.Marshal(j)
}

type IDList []int64

func (j IDList) GormDataType() string {
	return "json"
}

func (j *IDList) Scan(value any) error {
	if bytes, ok := value.([]byte); ok {
		result := make([]int64, 0)
		err := sonic.Unmarshal(bytes, &result)
		if err != nil {
			return err
		}
		*j = result
		return nil
	}
	if result, ok := value.([]int64); ok {
		*j = result
		return nil
	}
	return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
}

func (j IDList) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return sonic.Marshal(j)
}
