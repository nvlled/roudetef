
package main

import (
	ht "net/http"
	def "nvlled/roudetef"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/context"
	"fmt"
)

var store = sessions.NewCookieStore([]byte("supersecretpassword"))
var sessionName = "kalapato"

var message = map[string]string{
	"home-path": "This is the homepage",
	"login-path": "You are now alive",
	"logout-path": "You are now dead",
	"a-path": "This is a path",
	"b-path": "This is b path",
	"c-path": "This is c path",
	"d-path": "This is d path",
	"broke-path": "You were born",
	"protected-path": "login required",
	"admin-path": "must be admin",
	"submit-path": "Submission received and trashed",
}

func home(w ht.ResponseWriter, r *ht.Request) {
	fmt.Fprint(w, message["home-path"])
}

func login(w ht.ResponseWriter, r *ht.Request) {
	s,_ := store.Get(r, sessionName)
	s.Values["username"] = "joe"
	s.Save(r, w)
	fmt.Fprint(w, message["login-path"])
}

func logout(w ht.ResponseWriter, r *ht.Request) {
	s,_ := store.Get(r, sessionName)
	delete(s.Values, "username")
	s.Save(r, w)
	fmt.Fprint(w, message["logout-path"])
}

func a(w ht.ResponseWriter, r *ht.Request) {
	fmt.Fprint(w, message["a-path"])
}

func b(w ht.ResponseWriter, r *ht.Request) {
	fmt.Fprint(w, message["b-path"])
}

func c(w ht.ResponseWriter, r *ht.Request) {
	fmt.Fprint(w, message["c-path"])
}

func d(w ht.ResponseWriter, r *ht.Request) {
	fmt.Fprint(w, message["d-path"])
}

func broke(w ht.ResponseWriter, r *ht.Request) {
	panic(message["broke-path"])
}

var postSubmit = def.Ts{
	def.Group(
		def.GET,
		def.H(func (w ht.ResponseWriter, r *ht.Request) {
				fmt.Fprintln(w, "POST to submit; Need header X=123")
			}),
	),
	def.Group(
		def.Headers("X", "123"),
		def.POST,
		def.H(func (w ht.ResponseWriter, r *ht.Request) {
				fmt.Fprintln(w, "submission successful")
			}),
	),
}

func beAdmin(r *ht.Request) {
	context.Set(r, "admin", "yep")
}

func giveDiarrhea(r *ht.Request) {
	context.Set(r, "hasDiarrhea", "yep")
}

func notLoggedIn(r *ht.Request) bool {
	s,_ := store.Get(r, sessionName)
	return s.Values["username"] == nil
}

func notAdmin(r *ht.Request) bool {
	return context.Get(r, "admin") == nil
}

func catchError(handler ht.HandlerFunc) ht.HandlerFunc {
	return func(w ht.ResponseWriter, r *ht.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprintf(w, "%v", err)
			}
		}()
		handler(w, r)
	}
}

var requireLogin = def.Guard{
	Reject: notLoggedIn,
	Handler: func(w ht.ResponseWriter, r *ht.Request) {
		w.WriteHeader(ht.StatusUnauthorized)
		fmt.Fprint(w, message["protected-path"])
	},
}

var requireAdmin = def.Guard{
	Reject: notAdmin,
	Handler: func(w ht.ResponseWriter, r *ht.Request) {
		w.WriteHeader(ht.StatusUnauthorized)
		fmt.Fprint(w, message["admin-path"])
	},
}

func routeDefinition() *def.RouteDef {
	return def.Route(
		"/", home, "home-path",
		def.Hooks(beAdmin),
		def.Guards(),

		def.SRoute("/login",  login, "login-path"),
		def.SRoute("/logout", logout, "logout-path"),
		def.SRoute("/broke",  catchError(broke), "broke-path"),
		def.SRoute("/submit", postSubmit, "submit-path"),
		def.Route(
			"/a", def.H(a), "a-path",
			def.Hooks(giveDiarrhea),
			def.Guards(requireLogin, requireAdmin),

			def.SRoute(
				"/b", b, "b-path",
				def.SRoute("/c", c, "c-path"),
			),
			def.SRoute(
				"/d", d, "d-path",
			),
		),
	)
}

func createHandler() (*mux.Router, *def.RouteDef) {
	root := mux.NewRouter()
	root.StrictSlash(true)
	routeDef := routeDefinition()
	root.Path("/admin").Handler(ht.HandlerFunc(c)).Methods("POST")

	def.BuildRouter(routeDef, root)
	return root, routeDef
}

func main() {
	handler,_ := createHandler()
	ht.ListenAndServe(":7070", handler)
}



