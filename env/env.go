package env

import (
	"os"
	"strconv"
)

// Get Var of ENV
func GetEnvVar(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	} else {
		return val
	}
}

// Get Var of ENV As Int64
func AsInt(key string, bitSize int, fallback int64) int64 {
	val := GetEnvVar(key, "")
	if val == "" {
		return fallback
	}
	v, e := strconv.ParseInt(val, 0, bitSize)
	if e == nil {
		return v
	}
	return fallback
}
