package main

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type CommonResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func SignUp(c *Context) {
	user := &User{}
	if err := c.ReadJson(user); err != nil {
		log.Printf("read json failed, err:%v\n", err)
		c.W.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("sign up success\n")

	rsp := &CommonResponse{
		Code: 0,
		Msg:  "success",
	}
	if err := c.WriteJson(http.StatusOK, rsp); err != nil {
		log.Printf("write error:%v\n", err)
	}
}

func List(c *Context) {
	log.Printf("list all user\n")
	users := []User{
		{
			Name: "one",
			Age:  10,
		},
		{
			Name: "two",
			Age:  20,
		},
	}

	q := c.R.URL.Query()
	delay := q.Get("delay")
	delayMs := 0
	if len(delay) > 0 {
		delayMs, _ = strconv.Atoi(delay)
	}

	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	rsp := &CommonResponse{
		Code: 0,
		Msg:  "success",
		Data: users,
	}
	c.WriteJson(http.StatusOK, rsp)
}
