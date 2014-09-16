
package main

import (
	ht "net/http"
	"nvlled/rut"
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

func giveDiarrhea(r *ht.Request) {
	context.Set(r, "hasDiarrhea", "yep")
}

func notLoggedIn(r *ht.Request) bool {
	s,_ := store.Get(r, sessionName)
	return s.Values["username"] == nil
}

func notAdmin(r *ht.Request) bool {
	s,_ := store.Get(r, sessionName)
	_, ok := s.Values["admin"]
	return !ok
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

var requireLogin = rut.Guard{
	Reject: notLoggedIn,
	Handler: func(w ht.ResponseWriter, r *ht.Request) {
		w.WriteHeader(ht.StatusUnauthorized)
		fmt.Fprint(w, message["protected-path"])
	},
}

var requireAdmin = rut.Guard{
	Reject: notAdmin,
	Handler: func(w ht.ResponseWriter, r *ht.Request) {
		w.WriteHeader(ht.StatusUnauthorized)
		fmt.Fprint(w, message["admin-path"])
	},
}

func routeDefinition() *rut.RouteDef {
	return rut.SRoute(
		"/", home, "home-path",
		rut.SRoute("/login",  login, "login-path"),
		rut.SRoute("/logout", logout, "logout-path"),
		rut.SRoute("/broke",  catchError(broke), "broke-path"),
		rut.Route(
			"/a", a, "a-path",
			rut.Hooks(giveDiarrhea),
			rut.Guards(requireLogin, requireAdmin),
			rut.SRoute(
				"/b", b, "b-path",
				rut.SRoute("/c", c, "c-path"),
			),
			rut.SRoute("/d", d, "d-path"),
		),
	)
}

func createHandler() (*mux.Router, *rut.RouteDef) {
	root := mux.NewRouter()
	root.StrictSlash(true)
	def := routeDefinition()
	rut.BuildRouter(def, root)
	return root, def
}

func main() {
	handler,_ := createHandler()
	ht.ListenAndServe(":7070", handler)
}






