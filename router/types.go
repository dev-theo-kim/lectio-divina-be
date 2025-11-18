package router

import (
	"strconv"
	"time"
)

const (
	CtxBody = "body"
	Conn    = "conn"

	timeout = 550 * time.Second
)

type Config struct {
	Port         any      `toml:"port"`
	Origins      []string `toml:"origins"`
	AllowHeaders []string `toml:"allow_headers"`
	IsProduction bool     `toml:"is_production"`
	Limit        int64    `toml:"limit"` // limit size of request data (megabytes)
}

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type Method int

const (
	GET Method = 1 + iota
	POST
	PUT
	DELETE
	PATCH
	WS
)

func (m Method) String() string {
	switch m {
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case DELETE:
		return "DELETE"
	case PATCH:
		return "PATCH"
	default:
		return strconv.Itoa(int(m))
	}
}
