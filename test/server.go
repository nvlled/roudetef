
package main

import (
	ht "net/http"
	def "github.com/nvlled/roudetef"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	//"github.com/gorilla/context"
	"fmt"
	"log"
)

var store = sessions.NewCookieStore([]byte("supersecretpassword"))
var sessionName = "kalapato"

var message = map[string]string{
	"home-path": `
	<p>hello</p>
	links:
	<ul>
		<li><a href='/admin'>admin</a></li>
		<li><a href='/login'>login</a></li>
		<li><a href='/logout'>logout</a></li>
		<li><a href='/sudo'>sudo</a></li>
		<li><a href='/a'>a path</a></li>
		<li><a href='/a/b'>b path</a></li>
		<li><a href='/a/b/c'>c path</a></li>
		<li><a href='/a/d'>d path</a></li>
		<li><a href='/submit'>Submit personal info</a></li>
	</ul>
	`,
	"login-path": "You are now alive",
	"logout-path": "You are now dead",
	"a-path": "This is a path. You now have diarrhea.",
	"b-path": "This is b path",
	"c-path": "This is c path",
	"d-path": "This is d path",
	"broke-path": "You were born",
	"protected-path": "login required",
	"admin-path": "localhost: ~#",
	"submit-path": "Submission received and trashed",
	"sudo-path": "You are now logged in as root\nlocalhost: ~# ",
}

func home(w ht.ResponseWriter, r *ht.Request) {
	w.Header()["Content-Type"] = []string{"text/html"}
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
	delete(s.Values, "admin")
	delete(s.Values, "hasDiarrhea")
	s.Save(r, w)
	fmt.Fprint(w, message["logout-path"])
}

func a(w ht.ResponseWriter, r *ht.Request) {
	s,_ := store.Get(r, sessionName)
	s.Values["hasDiarrhea"] = "yep"
	s.Save(r, w)
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

func sudo(w ht.ResponseWriter, r *ht.Request) {
	s,_ := store.Get(r, sessionName)
	s.Values["admin"] = "yep"
	s.Save(r, w)
	fmt.Fprint(w, message["sudo-path"])
}

func admin(w ht.ResponseWriter, r *ht.Request) {
	fmt.Fprint(w, message["admin-path"])
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

func notLoggedIn(r *ht.Request) bool {
	s,_ := store.Get(r, sessionName)
	return s.Values["username"] == nil
}

func notAdmin(r *ht.Request) bool {
	s,_ := store.Get(r, sessionName)
	return s.Values["admin"] == nil
}

func noDiarrhea(r *ht.Request) bool {
	s,_ := store.Get(r, sessionName)
	return s.Values["hasDiarrhea"] == nil
}

func catchError(handler ht.HandlerFunc) ht.HandlerFunc {
	return func(w ht.ResponseWriter, r *ht.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprintf(w, "An error occured: %v", err)
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
		fmt.Fprint(w, "You must be root to continue")
	},
}

var requireDiarrhea = def.Guard{
	Reject: noDiarrhea,
	Handler: func(w ht.ResponseWriter, r *ht.Request) {
		w.Header()["Content-Type"] = []string{"text/html"}
		w.WriteHeader(ht.StatusUnauthorized)
		fmt.Fprint(w, "You must have a diarrhea to continue\n",
			"Get one <a href='/a'>here</a>")
	},
}


func routeDefinition() *def.RouteDef {
	return def.SRoute(
		"/", home, "home-path",

		def.Route(
			"/sudo",   sudo, "sudo-path",
			def.Hooks(), def.Guards(requireLogin),
		),
		def.Route(
			"/admin",   admin, "admin-path",
			def.Hooks(), def.Guards(requireAdmin),
		),

		def.SRoute("/login",  login, "login-path"),
		def.SRoute("/logout", logout, "logout-path"),
		def.SRoute("/broke",  catchError(broke), "broke-path"),
		def.SRoute("/submit", postSubmit, "submit-path"),
		def.Route(
			"/a", a, "a-path",
			def.Hooks(),
			def.Guards(requireLogin),

			def.SRoute(
				"/b", b, "b-path",
				def.SRoute("/c", c, "c-path"),
			),
			def.Route(
				"/d", d, "d-path",
				def.Hooks(), def.Guards(requireDiarrhea),
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
	port := "7070"
	log.Println("Server listening at ", port)
	ht.ListenAndServe(":"+port, handler)
}

