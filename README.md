Moon，月球。是一个强大为了编写模块化Web应用而生的GO语言框架。

**重要说明**：这是一个即将改写的应用框架，非原创。

## 第一个应用

在你安装了GO语言和设置了G环境之后, 创建你的自己的`.go`文件, 这里我们假设它的名字叫做 `server.go`.

~~~ go
package main

import "https://github.com/yangxiaoyang/go-moon"

func main() {
  m := moon.Classic()
  m.Get("/", func() string {
    return "Hello world!"
  })
  m.Run()
}
~~~



最后运行你的服务:
~~~
go run server.go
~~~

这时你将会有一个moon的服务监听了, 地址是: `localhost:3000`.


## 核心 Moon
moon.Classic()提供了一些默认的方便Web开发的工具:
~~~ go
  m := moon.Classic()
  // ... middleware and routing goes here
  m.Run()
~~~

下面是moon核心已经包含的功能:
  * Request/Response Logging （请求/响应日志）
  * Panic Recovery （容错）
  * Static File serving （静态文件服务）
  * Routing （路由）

### 处理器
处理器是Dacheng的灵魂和核心所在. 一个处理器基本上可以是任何的函数:
~~~ go
m.Get("/", func() {
  println("hello world")
})
~~~

#### 返回值
当一个处理器返回结果的时候, Moon将会把返回值作为字符串写入到当前的http.ResponseWriter里面:
~~~ go
m.Get("/", func() string {
  return "hello world" // HTTP 200 : "hello world"
})
~~~

另外你也可以选择性的返回多一个状态码:
~~~ go
m.Get("/", func() (int, string) {
  return 418, "i'm a teapot" // HTTP 418 : "i'm a teapot"
})
~~~

#### 服务的注入
处理器是通过反射来调用的. Moon 通过*Dependency Injection* *（依赖注入）* 来为处理器注入参数列表. **这样使得Moon与Go语言的`http.HandlerFunc`接口完全兼容.**

如果你加入一个参数到你的处理器, Moon将会搜索它参数列表中的服务，并且通过类型判断来解决依赖关系:
~~~ go
m.Get("/", func(res http.ResponseWriter, req *http.Request) { // res 和 req 是通过Dacheng注入的
  res.WriteHeader(200) // HTTP 200
})
~~~

下面的这些服务已经被包含在核心Dacheng中:
  * [*log.Logger] - Martini的全局日志.
  * [moon.Context] - http request context （请求上下文）.
  * [moon.Params] - `map[string]string` of named params found by route matching. （名字和参数键值对的参数列表）
  * [moon.Routes] - Route helper service. （路由协助处理）
  * [http.ResponseWriter] - http Response writer interface. (响应结果的流接口)
  * [*http.Request] - http Request. （http请求)

### 路由
在Moon中, 路由是一个HTTP方法配对一个URL匹配模型. 每一个路由可以对应一个或多个处理器方法:
~~~ go
m.Get("/", func() {
  // 显示
})

m.Patch("/", func() {
  // 更新
})

m.Post("/", func() {
  // 创建
})

m.Put("/", func() {
  // 替换
})

m.Delete("/", func() {
  // 删除
})

m.Options("/", func() {
  // http 选项
})

m.NotFound(func() {
  // 处理 404
})
~~~

路由匹配的顺序是按照他们被定义的顺序执行的. 最先被定义的路由将会首先被用户请求匹配并调用.

路由模型可能包含参数列表, 可以通过moon.Params服务来获取:
~~~ go
m.Get("/hello/:name", func(params moon.Params) string {
  return "Hello " + params["name"]
})
~~~

路由匹配可以通过正则表达式或者glob的形式:
~~~ go
m.Get("/hello/**", func(params moon.Params) string {
  return "Hello " + params["_1"]
})
~~~

也可以这样使用正则表达式:
~~~go
m.Get("/hello/(?P<name>[a-zA-Z]+)", func(params moon.Params) string {
  return fmt.Sprintf ("Hello %s", params["name"])
})
~~~
有关正则表达式的更多信息请参见[Go官方文档](http://golang.org/pkg/regexp/syntax/).


路由处理器可以被相互叠加使用, 例如很有用的地方可以是在验证和授权的时候:
~~~ go
m.Get("/secret", authorize, func() {
  // 该方法将会在authorize方法没有输出结果的时候执行.
})
~~~

也可以通过 Group 方法, 将 route 编成一組.
~~~ go
m.Group("/books", func(r moon.Router) {
    r.Get("/:id", GetBooks)
    r.Post("/new", NewBook)
    r.Put("/update/:id", UpdateBook)
    r.Delete("/delete/:id", DeleteBook)
})
~~~

就像为 handler 增加 middleware 方法一样, 你也可以为一组 routes 增加 middleware.
~~~ go
m.Group("/books", func(r moon.Router) {
    r.Get("/:id", GetBooks)
    r.Post("/new", NewBook)
    r.Put("/update/:id", UpdateBook)
    r.Delete("/delete/:id", DeleteBook)
}, MyMiddleware1, MyMiddleware2)
~~~

### 服务
服务即是被注入到处理器中的参数. 你可以映射一个服务到 *全局* 或者 *请求* 的级别.


#### 全局映射
如果一个Moon实现了inject.Injector的接口, 那么映射成为一个服务就非常简单:
~~~ go
db := &MyDatabase{}
m := moon.Classic()
m.Map(db) // *MyDatabase 这个服务将可以在所有的处理器中被使用到.
// ...
m.Run()
~~~

#### 请求级别的映射
映射在请求级别的服务可以用moon.Context来完成:
~~~ go
func MyCustomLoggerHandler(c moon.Context, req *http.Request) {
  logger := &MyCustomLogger{req}
  c.Map(logger) // 映射成为了 *MyCustomLogger
}
~~~

#### 映射值到接口
关于服务最强悍的地方之一就是它能够映射服务到接口. 例如说, 假设你想要覆盖[http.ResponseWriter](http://godoc.org/net/http#ResponseWriter)成为一个对象, 那么你可以封装它并包含你自己的额外操作, 你可以如下这样来编写你的处理器:
~~~ go
func WrapResponseWriter(res http.ResponseWriter, c moon.Context) {
  rw := NewSpecialResponseWriter(res)
  c.MapTo(rw, (*http.ResponseWriter)(nil)) // 覆盖 ResponseWriter 成为我们封装过的 ResponseWriter
}
~~~

### 服务静态文件
moon.Classic() 默认会服务位于你服务器环境根目录下的"public"文件夹.
你可以通过加入moon.Static的处理器来加入更多的静态文件服务的文件夹.
~~~ go
m.Use(moon.Static("assets")) // 也会服务静态文件于"assets"的文件夹
~~~

## 中间件处理器
中间件处理器是工作于请求和路由之间的. 本质上来说和Moon其他的处理器没有分别. 你可以像如下这样添加一个中间件处理器到它的堆中:
~~~ go
m.Use(func() {
  // 做一些中间件该做的事情
})
~~~

你可以通过`Handlers`函数对中间件堆有完全的控制. 它将会替换掉之前的任何设置过的处理器:
~~~ go
m.Handlers(
  Middleware1,
  Middleware2,
  Middleware3,
)
~~~

中间件处理器可以非常好处理一些功能，像logging(日志), authorization(授权), authentication(认证), sessions(会话), error pages(错误页面), 以及任何其他的操作需要在http请求发生之前或者之后的:

~~~ go
// 验证api密匙
m.Use(func(res http.ResponseWriter, req *http.Request) {
  if req.Header.Get("X-API-KEY") != "secret123" {
    res.WriteHeader(http.StatusUnauthorized)
  }
})
~~~

### Next()
Context.Next()是一个可选的函数用于中间件处理器暂时放弃执行直到其他的处理器都执行完毕. 这样就可以很好的处理在http请求完成后需要做的操作.
~~~ go
// log 记录请求完成前后  (*译者注: 很巧妙，掌声鼓励.)
m.Use(func(c moon.Context, log *log.Logger){
  log.Println("before a request")

  c.Next()

  log.Println("after a request")
})
~~~

## Moon Env
一些handler使用环境变量 `moon.Env` 对开发环境和生产环境提供特殊功能. 推荐在生产环境设置环境变量 `MOON_ENV=production`.

## 我在哪里可以找到中间件资源?

Moon框架支持Martini所有的中间件，可以查看 [contrib](https://github.com/martini-contrib) 项目. 

* [acceptlang](https://github.com/martini-contrib/acceptlang) - 解析`Accept-Language` HTTP报头的处理器。
* [accessflags](https://github.com/martini-contrib/accessflags) - 启用访问控制处理器.
* [auth](https://github.com/martini-contrib/auth) - 认证处理器。
* [binding](https://github.com/martini-contrib/binding) - 映射/验证raw请求到结构体(structure)里的处理器。
* [cors](https://github.com/martini-contrib/cors) - 提供支持 CORS 的处理器。
* [csrf](https://github.com/martini-contrib/csrf) - 为应用提供CSRF防护。
* [encoder](https://github.com/martini-contrib/encoder) - 提供用于多种格式的数据渲染或内容协商的编码服务。
* [gzip](https://github.com/martini-contrib/gzip) - 通过giz方式压缩请求信息的处理器。
* [gorelic](https://github.com/martini-contrib/gorelic) - NewRelic 中间件
* [logstasher](https://github.com/martini-contrib/logstasher) - logstash日志兼容JSON中间件 
* [method](https://github.com/martini-contrib/method) - 通过请求头或表单域覆盖HTTP方法。
* [oauth2](https://github.com/martini-contrib/oauth2) - 基于 OAuth 2.0 的应用登录处理器。支持谷歌、Facebook和Github的登录。
* [permissions2](https://github.com/xyproto/permissions2) - 跟踪用户，登录状态和权限控制器
* [render](https://github.com/martini-contrib/render) - 渲染JSON和HTML模板的处理器。
* [secure](https://github.com/martini-contrib/secure) - 提供一些安全方面的速效方案。
* [sessions](https://github.com/martini-contrib/sessions) - 提供`Session`服务支持的处理器。
* [sessionauth](https://github.com/martini-contrib/sessionauth) - 提供简单的方式使得路由需要登录, 并在Session中处理用户登录
* [strip](https://github.com/martini-contrib/strip) - 用于过滤指定的URL前缀。
* [strip](https://github.com/martini-contrib/strip) - URL前缀剥离。
* [staticbin](https://github.com/martini-contrib/staticbin) - 从二进制数据中提供静态文件服务的处理器。
* [throttle](https://github.com/martini-contrib/throttle) - 请求速率调节中间件.
* [vauth](https://github.com/rafecolton/vauth) - 负责webhook认证的处理器(目前支持GitHub和TravisCI)。
* [web](https://github.com/martini-contrib/web) - hoisie web.go's Context

## 我如何整合到我现有的服务器中?

由于Moon实现了 `http.Handler`, 所以它可以很简单的应用到现有Go服务器的子集中. 例如说这是一段在Google App Engine中的示例:

~~~ go
package hello

import (
  "net/http"
  "https://github.com/yangxiaoyang/go-moon"
)

func init() {
  m := moon.Classic()
  m.Get("/", func() string {
    return "Hello world!"
  })
  http.Handle("/", m)
}
~~~

 