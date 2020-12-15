package config

import "time"

type RpcConfig = map[string]interface{}

const (
	DefaultApiTimeout = 30 * time.Second
)
