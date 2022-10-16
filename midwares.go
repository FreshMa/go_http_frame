package main

import (
	"log"
	"time"

	"github.com/google/uuid"
)

func Metric() HandleFunc {
	return func(c *Context) {
		uuidStr := uuid.NewString()
		log.Printf("[REQUEST] [%s] url:%s, method:%s\n",
			uuidStr,
			c.R.URL.Path,
			c.R.Method)

		start := time.Now().UnixMicro()
		c.Next()
		end := time.Now().UnixMicro()
		log.Printf("[COST] [%s] %d us\n", uuidStr, end-start)
	}
}
