## go http frame

一个基础的http server 框架

### 目标

1. 提供基础的抽象，包括server、handler、hook、middleware等
2. 提供基础的路由匹配功能，包括通配符(`*`)和参数匹配(`:id`)
3. 提供基础的优雅关闭，需要实现：
    1. 拒绝新请求：使用middleware
    2. 完成当前剩余请求：使用hook和middleware共同实现
    3. 回收资源
    4. 关闭服务：使用hook实现
    5. 超时强制关闭
4. 提供一个简单清晰的项目layout

#### server

server抽象为一个服务，需要提供基础的路由、启动、关闭功能

```go
type Server interface{
    Route(method, path string, f HandleFunc)
    Start(port string)
    Shutdown(ctx context.Context) error
}
```
#### handler

handler负责路由的注册、选择，需要实现 http.Handler 接口，也就是提供一个 ServeHTTP 函数

```go
type Handler interface{
    Route(method, path string, f HandleFunc)
    http.Handler
}
```

#### hook

负责在服务退出的时候执行一些操作，参数是context，可以实现超时控制

用来实现优雅退出

```go
type Hook func(ctx context.Context) error

```

#### middleware

仿照gin，提供相同函数签名的中间件，并且需要显式调用 `c.Next()`

```go
type HandleFunc func(c *Context)
```
