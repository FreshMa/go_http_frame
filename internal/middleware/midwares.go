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
