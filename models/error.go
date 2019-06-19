package models

import (
	"encoding/json"
	_ "fmt"
)

type Err struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Area    string `json:"area,omitempty"`
}

func (e *Err) Error() string {
	data, _ := json.Marshal(e)
	return string(data)
}
