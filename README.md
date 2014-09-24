roudetef
========

A route definition library built on top of gorilla's mux.

## Installation
```
go get https://github.com/nvlled/roudetef
```

## Usage
Routes can be simply defined using Route, which is
a function that takes 5 arguments and optional subroutes:
    Route(path, handler, routeName, hooks, guards, subroutes...)

```
import def "net/http"

...

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
```
If there aren't any hooks or guards for a subroute,
then SRoute can be used instead, which is a function
that is similar to Route but doesn't take hooks and guards:
    SRoute(path, handler, routeName, subroutes...)

See [sample file](sample/main.go)






