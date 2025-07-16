package client

import (
	"fmt"
)

type ClientError struct {
	err error
}

func (e ClientError) Error() string {
	return e.err.Error()
}

type AuthenticationError struct {
	err error
}

func (e AuthenticationError) Error() string {
	return e.err.Error()
}

type APIError struct {
	Msg string
	Err error
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Msg, e.Err.Error())
}

type errorMessage struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}
