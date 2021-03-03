package mw

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var _ Controller = (*BaseController)(nil)

func Test_BaseController(t *testing.T) {
	type testCase struct {
		title  string
		method string
		mws    []func(w http.ResponseWriter) func (http.Handler) http.Handler
		out    string
	}

	cases := []testCase{
		{
			title:  "register middleware for HTTP method",
			method: http.MethodGet,
			mws: []func(w http.ResponseWriter) func(http.Handler) http.Handler{
				middlewareOne,
			},
			out:    "/mw1 prepare/mw1 before next/final handler/mw1 after next",
		},
		{
			title:  "add middleware to existing chain",
			method: http.MethodGet,
			mws: []func(w http.ResponseWriter) func(http.Handler) http.Handler{
				middlewareTwo, middlewareThree,
			},
			out: 	"/mw2 prepare/mw3 prepare/mw1 before next/mw2 before next/mw3 before next/final handler" +
					"/mw3 after next/mw2 after next/mw1 after next",
		},
		{
			title:  "get an empty middleware chain (by default)",
			method: http.MethodPost,
			mws: []func(w http.ResponseWriter) func(http.Handler) http.Handler{},
			out:    "/final handler",
		},
	}

	t.Run("Given a BaseController", func(t *testing.T) {
		controller := NewBaseController()
		for _, tc := range cases {
			t.Run(tc.title, func(t *testing.T) {
				w := httptest.NewRecorder()
				if len(tc.mws) > 0 {
					var mws []Middleware
					for _, mw := range tc.mws {
						mws = append(mws, mw(w))
					}
					controller.AddMiddleware(tc.method, mws...)
				}
				controller.Middleware(tc.method).Then(handlerFinal).ServeHTTP(w, nil)
				if w.Body.String() != tc.out {
					t.Errorf("handler output is expected to be %q but was %q", tc.out, w.Body.String())
				}
			})
		}
	})
}
