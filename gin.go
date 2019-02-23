// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin/render"
)

const defaultMultipartMemory = 32 << 20 // 32 MB

var (
	default404Body   = []byte("404 page not found")
	default405Body   = []byte("405 method not allowed")
	defaultAppEngine bool
)

// HandlerFunc defines the handler used by gin middleware as return value.
type HandlerFunc func(*Context)

// HandlersChain defines a HandlerFunc array.
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. 
// ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

// RouteInfo represents a request route's specification which contains method and path and its handler.
type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	HandlerFunc HandlerFunc
}

// RoutesInfo defines a RouteInfo array.
type RoutesInfo []RouteInfo

// Engine is the framework's instance, it contains the muxer, middleware and configuration settings.
// Create an instance of Engine, by using New() or Default()
type Engine struct {


	// Engine 继承于 RouterGroup，由于“路由”和“引擎”毕竟是两个逻辑，使用继承的方式有利于代码逻辑分离。
	RouterGroup

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.

    // 如果true，当前路由匹配失败但将路径最后的 / 去掉时匹配成功时	，自动匹配后者。
    // 比如：请求是 /foo/ 但没有命中，而存在 /foo，
    // 对 get method 请求，客户端会被301重定向到 /foo
    // 对于其他 method 请求，客户端会被307重定向到 /foo
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.


 	// 如果true，在没有handler被注册来处理当前请求时，router将尝试修复当前请求路径
    // 逻辑为：
    // 		1. 移除前面的 ../ 或者 //
    //  	2. 对新的路径进行大小写不敏感的查询
    // 如果找到了handler，请求会被301或307重定向
    // 比如： /FOO 和 /..//FOO 会被重定向到 /foo
    // 备注：RedirectTrailingSlash 参数和这个参数独立
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool

	// #726 #755 If enabled, it will thrust some headers starting with
	// 'X-AppEngine...' for better integration with that PaaS.
	AppEngine bool

	// If enabled, the url.RawPath will be used to find parameters.
	UseRawPath bool

	// If true, the path value will be unescaped.
	// If UseRawPath is false (by default), the UnescapePathValues effectively is true,
	// as url.Path gonna be used, which is already unescaped.
	UnescapePathValues bool

	// Value of 'maxMemory' param that is given to http.Request's ParseMultipartForm
	// method call.
	MaxMultipartMemory int64

	delims           render.Delims
	secureJsonPrefix string
	HTMLRender       render.HTMLRender
	FuncMap          template.FuncMap
	allNoRoute       HandlersChain
	allNoMethod      HandlersChain
	noRoute          HandlersChain
	noMethod         HandlersChain
	pool             sync.Pool
	trees            methodTrees
}

var _ IRouter = &Engine{}





//初始化：
// 1. gin.New() 初始化得到一个 *gin.Engine, 这个 Engine 是不带任何 middleware 的;
// 2. gin.Default() 初始化会在空的 Engine 加上了 Logger 和 Recovery 这俩 middleware。



// New returns a new blank Engine instance without any middleware attached.
// By default the configuration is:
// - RedirectTrailingSlash:  true
// - RedirectFixedPath:      false
// - HandleMethodNotAllowed: false
// - ForwardedByClientIP:    true
// - UseRawPath:             false
// - UnescapePathValues:     true
func New() *Engine {
	debugPrintWARNINGNew()
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		FuncMap:                template.FuncMap{},
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		ForwardedByClientIP:    true,
		AppEngine:              defaultAppEngine,
		UseRawPath:             false,
		UnescapePathValues:     true,
		MaxMultipartMemory:     defaultMultipartMemory,
		// trees 是一个多维切片，每个请求方法(Get/Post/Put/...)都有对应的 methodTree
		trees:                  make(methodTrees, 0, 9),
		delims:                 render.Delims{Left: "{{", Right: "}}"},
		secureJsonPrefix:       "while(1);",
	}

	//反向关系保存到RouterGroup中
	engine.RouterGroup.engine = engine

	// engine.pool 使用了 go 标准库里的 sync/pool，实现了 Gin Context 的临时对象池，
	// 便于重用 Context，以减少 golang gc 垃圾回收的压力，提升框架性能。
	engine.pool.New = func() interface{} {
		 // 分配新context
		return engine.allocateContext()
	}
	return engine
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default() *Engine {
	debugPrintWARNINGDefault()
	// 创建 gin 实例
	engine := New()
	// 加上 Logger 和 Recovery 中间件
	engine.Use(Logger(), Recovery())
	return engine
}


func (engine *Engine) allocateContext() *Context {
	return &Context{
		engine: engine,
	}
}

// Delims sets template left and right delims and returns a Engine instance.
func (engine *Engine) Delims(left, right string) *Engine {
	engine.delims = render.Delims{Left: left, Right: right}
	return engine
}

// SecureJsonPrefix sets the secureJsonPrefix used in Context.SecureJSON.
func (engine *Engine) SecureJsonPrefix(prefix string) *Engine {
	engine.secureJsonPrefix = prefix
	return engine
}

// LoadHTMLGlob loads HTML files identified by glob pattern
// and associates the result with HTML renderer.
func (engine *Engine) LoadHTMLGlob(pattern string) {
	left := engine.delims.Left
	right := engine.delims.Right
	templ := template.Must(template.New("").Delims(left, right).Funcs(engine.FuncMap).ParseGlob(pattern))

	if IsDebugging() {
		debugPrintLoadTemplate(templ)
		engine.HTMLRender = render.HTMLDebug{Glob: pattern, FuncMap: engine.FuncMap, Delims: engine.delims}
		return
	}

	engine.SetHTMLTemplate(templ)
}

// LoadHTMLFiles loads a slice of HTML files
// and associates the result with HTML renderer.
func (engine *Engine) LoadHTMLFiles(files ...string) {
	if IsDebugging() {
		engine.HTMLRender = render.HTMLDebug{Files: files, FuncMap: engine.FuncMap, Delims: engine.delims}
		return
	}

	templ := template.Must(template.New("").Delims(engine.delims.Left, engine.delims.Right).Funcs(engine.FuncMap).ParseFiles(files...))
	engine.SetHTMLTemplate(templ)
}

// SetHTMLTemplate associate a template with HTML renderer.
func (engine *Engine) SetHTMLTemplate(templ *template.Template) {
	if len(engine.trees) > 0 {
		debugPrintWARNINGSetHTMLTemplate()
	}

	engine.HTMLRender = render.HTMLProduction{Template: templ.Funcs(engine.FuncMap)}
}

// SetFuncMap sets the FuncMap used for template.FuncMap.
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.FuncMap = funcMap
}

// NoRoute adds handlers for NoRoute. It return a 404 code by default.
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// NoMethod sets the handlers called when... TODO.
func (engine *Engine) NoMethod(handlers ...HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

// Use attachs a global middleware to the router. 
// ie. the middleware attached though Use() will be included in the handlers chain 
// for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.

// 注册 midlleware：

// Use() 方法其实主要是调用 RouterGroup 中的 Use() 方法，主要功能是注册 Gin 框架的中间件。
// 接下来的 r.GET("/ping", func(c *gin.Context) { })}) 和 Use 类似，是将路由注册到 Gin 的 RouterGroup 中
func (engine *Engine) Use(middleware ...HandlerFunc) IRoutes {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
	return engine
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}


// 添加router信息
func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	// 常规检查
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(method != "", "HTTP method can not be empty")
	assert1(len(handlers) > 0, "there must be at least one handler")
	debugPrintRoute(method, path, handlers)

	// 维护engine.trees
	root := engine.trees.get(method)
	if root == nil {
		root = new(node)
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}

	// 核心，后面一起来讲
	root.addRoute(path, handlers)
}



// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (engine *Engine) Routes() (routes RoutesInfo) {
	for _, tree := range engine.trees {
		routes = iterate("", tree.method, routes, tree.root)
	}
	return routes
}

func iterate(path, method string, routes RoutesInfo, root *node) RoutesInfo {
	path += root.path
	if len(root.handlers) > 0 {
		handlerFunc := root.handlers.Last()
		routes = append(routes, RouteInfo{
			Method:      method,
			Path:        path,
			Handler:     nameOfFunction(handlerFunc),
			HandlerFunc: handlerFunc,
		})
	}
	for _, child := range root.children {
		routes = iterate(path, method, routes, child)
	}
	return routes
}





// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.


// 执行主逻辑
func (engine *Engine) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()
	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)


	//注意，这里engine需要实现 Handler 接口（https://golang.org/pkg/net/http/#Handler）：
	// type Handler interface {
    //     ServeHTTP(ResponseWriter, *Request)
	// }
	//
	//ServeHTTP的方法传递的两个参数，一个是Request，一个是ResponseWriter，
	//Engine中的ServeHTTP的方法就是要对这两个对象进行读取或者写入操作。
	err = http.ListenAndServe(address, engine)
	return
}

// RunTLS attaches the router to a http.Server and starts listening and serving HTTPS (secure) requests.
// It is a shortcut for http.ListenAndServeTLS(addr, certFile, keyFile, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()

	err = http.ListenAndServeTLS(addr, certFile, keyFile, engine)
	return
}

// RunUnix attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (ie. a file).
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunUnix(file string) (err error) {
	debugPrint("Listening and serving HTTP on unix:/%s", file)
	defer func() { debugPrintError(err) }()

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()
	os.Chmod(file, 0777)
	err = http.Serve(listener, engine)
	return
}

// RunFd attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified file descriptor.
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunFd(fd int) (err error) {
	debugPrint("Listening and serving HTTP on fd@%d", fd)
	defer func() { debugPrintError(err) }()

	f := os.NewFile(uintptr(fd), fmt.Sprintf("fd@%d", fd))
	listener, err := net.FileListener(f)
	if err != nil {
		return
	}
	defer listener.Close()
	err = http.Serve(listener, engine)
	return
}



// ServeHTTP conforms to the http.Handler interface.

// 1. 从 engine 的 Context pool 里取出一个 Context
// 2. 将 Context 的 writestream 和 Request 都设置成需要处理的 HTTP 请求的
// 3. 调用 reset() 方法来重置 Context，这样就完成了 Context 的初始化
// 4. 调用 engine.handleHTTPRequest(c) 来处理 HTTP 请求了（核心）
// 5. HTTP请求处理结束之后调用 engine.pool.Put(c) 将 Context 对象放回临时对象池中，以便后续复用。

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 从上下文对象池中获取一个上下文对象
	c := engine.pool.Get().(*Context)
	// 初始化上下文对象，因为从对象池取出来的数据，有脏数据，故要初始化。
	c.writermem.reset(w)
	c.Request = req
	c.reset()
	// 通过请求 method 找到 engine.trees 中对应的树，然后在树中查找对应的路由, 执行相关的 handlers。
	engine.handleHTTPRequest(c)
	// 将Context对象扔回对象池了
	engine.pool.Put(c)
}

// HandleContext re-enter a context that has been rewritten.
// This can be done by setting c.Request.URL.Path to your new target.
// Disclaimer: You can loop yourself to death with this, use wisely.
func (engine *Engine) HandleContext(c *Context) {
	oldIndexValue := c.index
	c.reset()
	engine.handleHTTPRequest(c)
	c.index = oldIndexValue
}

func (engine *Engine) handleHTTPRequest(c *Context) {


	httpMethod := c.Request.Method
	path := c.Request.URL.Path
	unescape := false

	// 看是否使用 RawPath
	if engine.UseRawPath && len(c.Request.URL.RawPath) > 0 {
		path = c.Request.URL.RawPath
		unescape = engine.UnescapePathValues
	}

	
	t := engine.trees
	// 遍历路由树
	for i, tl := 0, len(t); i < tl; i++ {
		// 根据 http method 得到对应的路由子树
		if t[i].method != httpMethod {
			continue
		}
		// 树根
		root := t[i].root
		// 根据path查找 handlers
		handlers, params, tsr := root.getValue(path, c.Params, unescape)
		// handlers 存在
		if handlers != nil {
			c.handlers = handlers
			c.Params = params
			// (核心逻辑) 从第一个 handler 开始链式调用
			c.Next()
			// 写 Header
			c.writermem.WriteHeaderNow()
			return
		}

		 // 若 handlers 不存在且 method 不是 CONNECT 且 path 不是 /
		if httpMethod != "CONNECT" && path != "/" {
			// 如果配置是需要尾重定向，执行尾重定向
			if tsr && engine.RedirectTrailingSlash {
				redirectTrailingSlash(c)
				return
			}
			// 如果不需要尾重定向但是配置了重定向固定 path, 重定向到固定 path
			if engine.RedirectFixedPath && redirectFixedPath(c, root, engine.RedirectFixedPath) {
				return
			}
		}
		break
	}

	// 如果是因为 HTTP method 有误，且配置了 HandleMethodNotAllowed 为 true，则处理如下处理
	if engine.HandleMethodNotAllowed {
		for _, tree := range engine.trees {
			if tree.method == httpMethod {
				continue
			}
			// 路由中存在 method 不一样但是 path 和 params 匹配的路由，则返回 405 Method Not Allowed
			if handlers, _, _ := tree.root.getValue(path, nil, unescape); handlers != nil {
				c.handlers = engine.allNoMethod
				serveError(c, http.StatusMethodNotAllowed, default405Body)
				return
			}
		}
	}

	// 如果找不到匹配 route，返回404
	c.handlers = engine.allNoRoute
	serveError(c, http.StatusNotFound, default404Body)
}

var mimePlain = []string{MIMEPlain}

func serveError(c *Context, code int, defaultMessage []byte) {
	c.writermem.status = code
	c.Next()
	if c.writermem.Written() {
		return
	}
	if c.writermem.Status() == code {
		c.writermem.Header()["Content-Type"] = mimePlain
		_, err := c.Writer.Write(defaultMessage)
		if err != nil {
			debugPrint("cannot write message to writer during serve error: %v", err)
		}
		return
	}
	c.writermem.WriteHeaderNow()
	return
}



//尾重定向
func redirectTrailingSlash(c *Context) {
	req := c.Request
	path := req.URL.Path
	code := http.StatusMovedPermanently // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = http.StatusTemporaryRedirect
	}

	req.URL.Path = path + "/"
	if length := len(path); length > 1 && path[length-1] == '/' {
		req.URL.Path = path[:length-1]
	}
	debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
	http.Redirect(c.Writer, req, req.URL.String(), code)
	c.writermem.WriteHeaderNow()
}


//固定路由重定向
func redirectFixedPath(c *Context, root *node, trailingSlash bool) bool {
	req := c.Request
	path := req.URL.Path

	if fixedPath, ok := root.findCaseInsensitivePath(cleanPath(path), trailingSlash); ok {
		code := http.StatusMovedPermanently // Permanent redirect, request with GET method
		if req.Method != "GET" {
			code = http.StatusTemporaryRedirect
		}
		req.URL.Path = string(fixedPath)
		debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
		http.Redirect(c.Writer, req, req.URL.String(), code)
		c.writermem.WriteHeaderNow()
		return true
	}
	return false
}
