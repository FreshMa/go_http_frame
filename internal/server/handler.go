package server

import (
	"log"
	"myserver/internal/ctx"
	"net/http"
	"strings"
	"sync"
)

type Routable interface {
	Route(method, path string, hs ...ctx.HandleFunc)
}

type Handler interface {
	http.Handler
	Routable
}

// MapBasedHandler
type MapBasedHandler struct {
	routes            sync.Map
	globalMiddlewares []ctx.HandleFunc
}

func NewMapBasedHandler(middlewares ...ctx.HandleFunc) *MapBasedHandler {
	wares := make([]ctx.HandleFunc, 0, len(middlewares))
	wares = append(wares, middlewares...)

	return &MapBasedHandler{
		globalMiddlewares: wares,
		//routes: make(map[string][]HandleFunc),
	}
}

func (h *MapBasedHandler) key(method, path string) string {
	return method + "_" + path
}

// Route 实现Router接口
func (h *MapBasedHandler) Route(method, path string, handlers ...ctx.HandleFunc) {
	k := h.key(method, path)
	hs := make([]ctx.HandleFunc, 0, len(handlers))
	for i := 0; i < len(handlers); i++ {
		hs = append(hs, handlers[i])
	}

	h.routes.Store(k, hs)
}

// ServeHTTP 实现http.Handler 接口
func (h *MapBasedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := ctx.NewContext(w, r)
	k := h.key(r.Method, r.URL.Path)

	hs, ok := h.routes.Load(k)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
		return
	}

	handlers := hs.([]ctx.HandleFunc)
	c.Hs = h.globalMiddlewares
	c.Hs = append(c.Hs, handlers...)
}

// TreeBasedHandler

type TreeBasedHandler struct {
	root              *Node
	globalMiddlewares []ctx.HandleFunc
}

func NewTreeBasedHandler(middlewares ...ctx.HandleFunc) *TreeBasedHandler {
	wares := make([]ctx.HandleFunc, 0, len(middlewares))
	wares = append(wares, middlewares...)
	return &TreeBasedHandler{
		root:              NewNode("/"),
		globalMiddlewares: wares,
	}
}

func (h *TreeBasedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlers := h.Query(h.root, r.Method, r.URL.Path)
	if handlers == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
		return
	}

	c := ctx.NewContext(w, r)
	c.Hs = h.globalMiddlewares
	c.Hs = append(c.Hs, handlers...)
	log.Printf("len of handler:%d\n", len(c.Hs))
	c.Next()
}

// 如果重复注册的话，这里其实不会报错，也不会生效
func (h *TreeBasedHandler) Route(method, path string, handlers ...ctx.HandleFunc) {
	cur := h.root
	paths := strings.Split(strings.Trim(path, "/"), "/")

	// 检查通配符 * 是否位于末尾
	wildcardPos := strings.Index(path, "*")
	if wildcardPos != -1 && wildcardPos != len(path)-1 {
		panic("illegal wildcard position, should appear only once and at the end of path")
	}

	for idx, p := range paths {
		subNode, ok := cur.Match(p, false)
		if ok {
			cur = subNode
		} else {
			// create
			h.createSubTree(cur, method, paths[idx:], handlers...)
			return
		}
	}
}

// 在root下建子树
func (h *TreeBasedHandler) createSubTree(root *Node, method string, path []string, handlers ...ctx.HandleFunc) {
	cur := root
	for _, p := range path {
		node := NewNode(p)
		cur.child = append(cur.child, node)
		cur = node
	}
	cur.method = method
	//log.Printf("pattern:%s, method:%s\n", cur.path, method)
	cur.isLeaf = true
	// 叶子结点添加handler
	for i := range handlers {
		cur.fns = append(cur.fns, handlers[i])
	}
}

func (h *TreeBasedHandler) Query(root *Node, method string, path string) []ctx.HandleFunc {
	paths := strings.Split(strings.Trim(path, "/"), "/")
	cur := root
	for _, p := range paths {
		log.Printf("p:%s\n", p)
		n, ok := cur.Match(p, true)
		if !ok {
			return nil
		}
		cur = n
	}
	if cur.method != method {
		//log.Printf("method not match\n")
		return nil
	}
	return cur.fns
}

type Node struct {
	path   string
	method string // 只有路由的最后一段才会赋值
	isLeaf bool   // 其实没有用到

	child []*Node
	fns   []ctx.HandleFunc
}

func NewNode(path string) *Node {
	return &Node{
		path:   path,
		child:  make([]*Node, 0, 4),
		isLeaf: false,
		fns:    nil,
	}
}

// 这里在寻找路由的时候需要支持通配符匹配
// 但是在添加路由的时候不需要支持
func (n *Node) Match(path string, enableWildcard bool) (*Node, bool) {
	// 首先肯定精确匹配，过程中如果匹配到了通配符，先记录下来
	var wildcardMatch *Node
	for _, ch := range n.child {
		if ch.path == path && ch.path != "*" {
			return ch, true
		}

		if enableWildcard && ch.path == "*" {
			wildcardMatch = ch
		}
	}
	return wildcardMatch, wildcardMatch != nil
}
