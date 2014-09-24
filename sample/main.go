package main

import (
	"net/http"
	def "github.com/nvlled/roudetef"
)

var home	= aHandler
var login	= aHandler
var logout	= aHandler
var submit	= aHandler
var a = aHandler
var b = aHandler
var c = aHandler
var d = aHandler
var logSomething = aHook
var requireLogin = aGuard

func main() {
	routeDef := def.SRoute(
		"/", home, "home-path",

		def.SRoute("/login",  login, "login-path"),
		def.SRoute("/logout", logout, "logout-path"),
		def.SRoute("/submit", submit, "submit-path"),
		def.Route(
			"/a", def.H(a), "a-path",
			def.Hooks(logSomething),
			def.Guards(requireLogin),

			def.SRoute(
				"/b", b, "b-path",
				def.SRoute("/c", c, "c-path"),
			),
			def.SRoute(
				"/d", d, "d-path",
			),
		),
	)

	routeDef.Print()
}

var aGuard = def.Guard{
	Reject: func(h *http.Request) bool {
		// do some checkin'
		return true
	},
	Handler: func(w http.ResponseWriter, h *http.Request) {
		// This code is run when Reject returns true
	},
}

func aHandler(w http.ResponseWriter, h *http.Request) {
	// handle the request
}

func aHook(h *http.Request) {
	// do something
}





