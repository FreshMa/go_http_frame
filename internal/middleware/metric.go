package middleware

import (
	"log"
	"time"

	"myserver/internal/ctx"

	"github.com/google/uuid"
)

func Metric() ctx.HandleFunc {
	return func(c *ctx.Context) {
		uuidStr := uuid.NewString()
		log.Printf("[%s] [REQUEST] url:%s, method:%s\n",
			uuidStr,
			c.R.URL.Path,
			c.R.Method)

		start := time.Now().UnixNano() * 1e3
		c.Next()
		end := time.Now().UnixNano() * 1e3
		log.Printf("[%s] [COST] %d us\n", uuidStr, end-start)
	}
}
