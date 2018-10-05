// Package moon is a powerful package for quickly writing modular web applications/services in Golang.
//
// For a full guide visit http://github.com/go-moon/moon
//
//  package main
//
//  import ".../moon"
//
//  func main() {
//    m := moon.Classic()
//
//    m.Get("/", func() string {
//      return "Hello world!"
//    })
//
//    m.Run()
//  }
package moon

import (
	"log"
	"net/http"
	"os"
	"reflect"

	"moon/inject"
)

// Moon represents the top level web application. inject.Injector methods can be invoked to map services on a global level.
type Moon struct {
	inject.Injector
	handlers []Handler
	action   Handler
	logger   *log.Logger
}

// New creates a bare bones moon instance. Use this method if you want to have full control over the middleware that is used.
func New() *Moon {
	m := &Moon{Injector: inject.New(), action: func() {}, logger: log.New(os.Stdout, "[moon] ", 0)}
	m.Map(m.logger)
	m.Map(defaultReturnHandler())
	return m
}

// Handlers sets the entire middleware stack with the given Handlers. This will clear any current middleware handlers.
// Will panic if any of the handlers is not a callable function
func (m *Moon) Handlers(handlers ...Handler) {
	m.handlers = make([]Handler, 0)
	for _, handler := range handlers {
		m.Use(handler)
	}
}

// Action sets the handler that will be called after all the middleware has been invoked. This is set to moon.Router in a moon.Classic().
func (m *Moon) Action(handler Handler) {
	validateHandler(handler)
	m.action = handler
}

// Logger sets the logger
func (m *Moon) Logger(logger *log.Logger) {
	m.logger = logger
	m.Map(m.logger)
}

// Use adds a middleware Handler to the stack. Will panic if the handler is not a callable func. Middleware Handlers are invoked in the order that they are added.
func (m *Moon) Use(handler Handler) {
	validateHandler(handler)

	m.handlers = append(m.handlers, handler)
}

// ServeHTTP is the HTTP Entry point for a moon instance. Useful if you want to control your own HTTP server.
func (m *Moon) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.createContext(res, req).run()
}

// Run the http server on a given host and port.
func (m *Moon) RunOnAddr(addr string) {
	// TODO: Should probably be implemented using a new instance of http.Server in place of
	// calling http.ListenAndServer directly, so that it could be stored in the moon struct for later use.
	// This would also allow to improve testing when a custom host and port are passed.

	logger := m.Injector.Get(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
	logger.Printf("listening on %s (%s)\n", addr, Env)
	logger.Fatalln(http.ListenAndServe(addr, m))
}

// Run the http server. Listening on os.GetEnv("PORT") or 3000 by default.
func (m *Moon) Run() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	host := os.Getenv("HOST")

	m.RunOnAddr(host + ":" + port)
}

func (m *Moon) createContext(res http.ResponseWriter, req *http.Request) *context {
	c := &context{inject.New(), m.handlers, m.action, NewResponseWriter(res), 0}
	c.SetParent(m)
	c.MapTo(c, (*Context)(nil))
	c.MapTo(c.rw, (*http.ResponseWriter)(nil))
	c.Map(req)
	return c
}

// ClassicMoon represents a moon with some reasonable defaults. Embeds the router functions for convenience.
type ClassicMoon struct {
	*Moon
	Router
}

// Classic creates a classic moon with some basic default middleware - moon.Logger, moon.Recovery and moon.Static.
// Classic also maps moon.Routes as a service.
func Classic() *ClassicMoon {
	r := NewRouter()
	m := New()
	m.Use(Logger())
	m.Use(Recovery())
	m.Use(Static("public"))
	m.MapTo(r, (*Routes)(nil))
	m.Action(r.Handle)
	return &ClassicMoon{m, r}
}

// Handler can be any callable function. moon attempts to inject services into the handler's argument list.
// moon will panic if an argument could not be fullfilled via dependency injection.
type Handler interface{}

func validateHandler(handler Handler) {
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		panic("moon handler must be a callable func")
	}
}

// Context represents a request context. Services can be mapped on the request level from this interface.
type Context interface {
	inject.Injector
	// Next is an optional function that Middleware Handlers can call to yield the until after
	// the other Handlers have been executed. This works really well for any operations that must
	// happen after an http request
	Next()
	// Written returns whether or not the response for this context has been written.
	Written() bool
}

type context struct {
	inject.Injector
	handlers []Handler
	action   Handler
	rw       ResponseWriter
	index    int
}

func (c *context) handler() Handler {
	if c.index < len(c.handlers) {
		return c.handlers[c.index]
	}
	if c.index == len(c.handlers) {
		return c.action
	}
	panic("invalid index for context handler")
}

func (c *context) Next() {
	c.index += 1
	c.run()
}

func (c *context) Written() bool {
	return c.rw.Written()
}

func (c *context) run() {
	for c.index <= len(c.handlers) {
		_, err := c.Invoke(c.handler())
		if err != nil {
			panic(err)
		}
		c.index += 1

		if c.Written() {
			return
		}
	}
}
