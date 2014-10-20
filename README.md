roudetef
========

A route definition library built on top of [gorilla's mux](http://www.gorillatoolkit.org/pkg/mux).

## Installation
```
go get https://github.com/nvlled/roudetef
```

## Usage

### Defining the routes
Routes can be simply defined using Route, which is
a function that takes 5 arguments and optional subroutes:
```Route(path, handler, routeName, hooks, guards, subroutes...)```

```go
import def "github.com/nvlled/roudetef"

...
routeDef := def.SRoute(
    "/", home, "home-path",

    def.SRoute(def.GET("/login"),  login, "login-page"),
    def.SRoute(def.POST("/login"),  login, "login-submit"),
    def.SRoute("/logout", logout, "logout-path"),
    def.SRoute(def.POST("/submit"), submit, "submit-path"),
    def.Route(
        "/a", a, "a-path",
        def.Hooks(logSomething),
        def.Guards(requireLogin),

        def.SRoute(
            def.GET("/b"), b, "b-path",
            def.SRoute("/c", c, "c-path"),
        ),
        def.SRoute(
            def.Methods("HEAD", "PUT")("/d"), d, "d-path",
        ),
    ),
)
```
If there aren't any hooks or guards for a subroute,
then SRoute can be used instead, which is a function
that is similar to Route but doesn't take hooks and guards:

```SRoute(path, handler, routeName, subroutes...)```

The routeDef above results to:
```
home-path      	/	    ANY
login-page     	/login	GET
login-submit   	/login	POST
logout-path    	/logout	ANY
submit-path    	/submit	POST
a-path         	/a	    ANY
b-path         	/a/b	GET
c-path         	/a/b/c	ANY
d-path         	/a/d	HEAD,PUT
```
(See [sample file](sample/main.go))

### Specifying the http methods

Notice that the path argument to Route or SRoute
may either be a string or an abstract
type resulting from calls such as GET, POST, etc.

Just using a string as a path argumenta can match
any http methods.

Likewise, GET("/login") and POST("/login") results
to routes that match GET and POST methods respectively.

If more than one method is desired, the function Method
can be used, as seen in the d-path above:
```
def.Methods("HEAD", "PUT")("/d"), d, "d-path",
````
The invocation may seem odd, but it is for
both consistency's sake and the way
... arguments work in Go.

### Building the routes
After  the routes have been defined, the routes can be built by
invoking the method BuildRouter():
```go
router := routeDef.BuildRouter()
```

BuildRouter() returns a [*mux.Router](http://www.gorillatoolkit.org/pkg/mux#Router),
which means the routes can be further modified as needed.

### Serving http
If you are not familiar with mux, you can use the resulting router (which implements
the [http.Handler](http://golang.org/pkg/net/http/#Handler)) from Buildrouter() as an http handler:
```
http.ListenAndServe(":8080", routeDef.BuildRouter())
```

## Advanced usages
See the [server](test/server.go) in the test folder for a complete example using the library.

### Hooks
Hooks are simply functions of type ```func(*http.Request)``` that are run
before the handler for a given route. One or more hooks can be added to the route:
```
routeDef := def.Route(
	"/login", loginHandler, "login-path",
	def.Hooks(logRequest, setDB),
	def.Guards(),
)
```
In the example above, logRequest and setDB hooks are attached to login-path.
The order of execution of hooks starts from the leftmost to the rightmost,
e.g., logRequest first then setDB.


### Guards
Guards are created as follows:
```
var requireLogin = def.Guard{
	Reject: func(r *http.Request) bool { return true },
	Handler: func(w *http.ResponseWriter, r *http.Request) {
		fmt.Println(w, "login required")
	},
}

```
The guard created can then be used as such:
```
routeDef := def.SRoute(
	"/", homeHandler, "home-path",
	def.Route(
		"/user/{id}", userHandler, "user-path",
		def.Hooks(),
		def.Guards(requireLogin),
	),
)
```
The handler for a route will only execute if all the guards doesn't reject the request.
In the example above, the Reject function of requireLogin always reject the request.
In a more realistic example (again see the [test server](test/server.go)),
the guard will make decisions based on the request parameter.

Only one of the given guards' handler may execute:
```
def.SRoute(
	"/sample", sampleHandler, "sample-path",
	def.Hooks(),
	def.Guards(guardA, guardB, guardC),
```
In the code above, sample Handler will only execute when guards A, B and C
accept the request. The order of execution of guards is from left to right.


### More specific routes
Previously, the Route function was stated to have a signature
```Route(path, handler, routeName, hooks, guards, subroutes...)```
To be precise, handler can be a type of [http.Handler](http://golang.org/pkg/net/http/#Handler)
or a type created from With function which takes a mandatory argument of http.Handler,
and optional transformers.
````
With(http.Handler, ...Transformer)
````

Simply put, transformers add matchers to each [mux.Route](http://www.gorillatoolkit.org/pkg/mux#Route)
in the route definition. To make things more concrete, suppose
we want a route that matches only request with headers X=123
and with a scheme of GOPHER (for purely whimsical purposes).

The route can be defined as follows:
```
With, Headers, Schemes = def.With, def.Headers, def.Schemes

...

def.Route(
    "/login",
    With(loginHandler, Headers("X", "123"), Schemes("GOPHER")),
    "login-page"
),
...
```
Headers and Schemes are transformers that call
[mux.Route.Headers](http://www.gorillatoolkit.org/pkg/mux#Route.Methods) and
[mux.Route.Schemes](http://www.gorillatoolkit.org/pkg/mux#Route.Methods) internally.


