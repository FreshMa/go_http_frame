package ctx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type HandleFunc func(c *Context)

type Hook func(ctx context.Context) error

type Context struct {
	W   http.ResponseWriter
	R   *http.Request
	Hs  []HandleFunc
	idx int
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		W:   w,
		R:   r,
		idx: -1,
	}
}

func (c *Context) ReadJson(data interface{}) error {
	buf, err := io.ReadAll(c.R.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(buf, data); err != nil {
		return err
	}

	return nil
}

func (c *Context) WriteJson(code int, data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}

	c.W.WriteHeader(code)
	c.W.Write(buf)
	return nil
}

// 参考gin
func (c *Context) Next() {
	c.idx++
	for c.idx < len(c.Hs) {
		c.Hs[c.idx](c)
		c.idx++
	}
}

func (c *Context) Abort() {
	c.Hs = nil
	c.idx = -1
}
