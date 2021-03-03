package mw

import (
	"fmt"
	"net/http"
	"time"
)

// DefaultTimeFormat contains default date/time layout to be used across middlewares
// of the package.
var DefaultTimeFormat = time.RFC1123

// MiddlewareFunc is a classic middleware - decides itself whether to call next
// handler or return an HTTP error directly to the client and breaks the chain.
type MiddlewareFunc func(http.ResponseWriter, *http.Request, http.Handler)

// Middleware in itself simply takes a http.Handler as the first parameter, wraps
// it and returns a new http.Handler for the server to call (wraps handlers with
// closures according to the principle of Russian doll).
type Middleware func(http.Handler) http.Handler

// New is a Middleware constructor func. The call without arguments returns an
// empty Middleware.
//
// Usage (all examples below are equal):
//	- mw.New().Use(mwOne, mwTwo, mwThree).Then(handler)
// 	- mw.New(mwOne).Use(mwTwo, mwThree).Then(handler)
// 	- mw.New(mwOne, mwTwo, mwThree).Then(handler)
func New(middlewares ...Middleware) Middleware {
	return Middleware(func(handler http.Handler) http.Handler {
		return handler
	}).Use(middlewares...)
}

// Use transforms provided middleware function(s) (including current one) into
// a single middleware func.
func (mw Middleware) Use(middlewares ...Middleware) Middleware {
	for _, next := range middlewares {
		mw = func(curr, next Middleware) Middleware {
			return func(handler http.Handler) http.Handler {
				var nextHandler http.Handler
				currHandler := curr(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextHandler.ServeHTTP(w, r)
				}))
				nextHandler = next(handler)
				return currHandler
			}
		}(mw, next)
	}
	return mw
}

// Then injects handler into middleware chain.
func (mw Middleware) Then(final http.Handler) http.Handler {
	return mw(final)
}

// blobHandler will be called anyway (if request reached your final handler),
// it is just a blob used to prevent calling ServeHTTP() method from panic since
// final handler is being wrapped with anonymous Middleware func. It has been
// declared as a variable intentionally - to be able to define a custom handler.
var blobHandler = func(w http.ResponseWriter, r *http.Request) {}

// Chain builds a http.Handler from passed arguments. It accepts different
// kinds of argument types:
// 	- MiddlewareFunc
// 	- func(http.ResponseWriter, *http.Request, http.Handler)
// 	- Middleware
// 	- func(http.Handler) http.Handler
// 	- http.Handler
// 	- func(w http.ResponseWriter, r *http.Request)
//
// Keep in mind:
//
// - by passing http.Handler/http.HandlerFunc instead of Middleware you lose
// control over the next Middleware (no way to cancel it), but if you only need
// to put something to the context (and do not have any logic after calling "next")
// there is no sense to build Middleware func around
//
// - even if you do not pass any handlers blobHandler will be executed.
func Chain(handlers ...interface{}) http.Handler {
	// fake handler in order to wrap last handler call "next"
	var f http.Handler = http.HandlerFunc(blobHandler)
	// apply middleware/handlers from the last to the first one
	for i := len(handlers) - 1; i >= 0; i-- {
		switch t := handlers[i].(type) {
		// build the handler from classic middleware func
		case func(http.ResponseWriter, *http.Request, http.Handler):
			f = func(curr MiddlewareFunc, next http.Handler) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					curr(w, r, next)
				}
			}(t, f)
		case MiddlewareFunc:
			f = func(curr MiddlewareFunc, next http.Handler) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					curr(w, r, next)
				}
			}(t, f)
		// wrap existing handler (or blobHandler) with a Middleware/func
		case func(http.Handler) http.Handler:
			f = t(f)
		case Middleware:
			f = t(f)
		// ordinary functions can also be provided as arguments, in such case they
		// will be called via adapter http.HandlerFunc
		case func(w http.ResponseWriter, r *http.Request):
			f = func(curr, next http.Handler) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					curr.ServeHTTP(w, r)
					// due to the blobHandler next will never be nil
					next.ServeHTTP(w, r)
				}
			}(http.HandlerFunc(t), f)
		// since http.HandlerFunc implements http.Handler interface we can use type
		// http.Handler for both of them
		case http.Handler:
			f = func(curr, next http.Handler) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					curr.ServeHTTP(w, r)
					next.ServeHTTP(w, r)
				}
			}(t, f)
		default:
			// everything else is not supported
			panic(fmt.Sprintf("unsupported argument type \"%T\"", t))
		}
	}
	return f
}
