package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	jsoniter "github.com/json-iterator/go"
)

type JSON map[string]interface{}

func (j JSON) GormDataType() string {
	return "json"
}

func (j *JSON) Scan(value interface{}) error {
	if bytes, ok := value.([]byte); ok {
		result := make(map[string]interface{})
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		err := json.Unmarshal(bytes, &result)
		if err != nil {
			return err
		}
		*j = result
		return nil
	}
	if result, ok := value.(map[string]interface{}); ok {
		*j = result
		return nil
	}
	return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(j)
}

type IDList []int64

func (j IDList) GormDataType() string {
	return "json"
}

func (j *IDList) Scan(value interface{}) error {
	if bytes, ok := value.([]byte); ok {
		result := make([]int64, 0)
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		err := json.Unmarshal(bytes, &result)
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
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(j)
}
