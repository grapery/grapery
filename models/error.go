package models

import (
	"encoding/json"
	_ "fmt"
)

type Result struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Result) Error() string {
	data, _ := json.Marshal(e)
	return string(data)
}

func (e *Result) Byte() []byte {
	data, _ := json.Marshal(e)
	return data
}
