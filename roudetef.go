
package roudetef

import (
	ht "net/http"
	"github.com/gorilla/mux"
	"path/filepath"
	"strings"
	"fmt"
)

// TODO:
// Abstract URL generation from mux
// Rename MapRoute to Map
// Move others to def subpackage
// Use reflection to get function name when printing the routing table
// Add accept (in addition to reject) on guards
// Interchange order of route name and transformer arguments

type Entry struct {
	Name string
	Path string
}

type Guard struct {
	Reject func(*ht.Request)bool
	Handler ht.HandlerFunc
}

type Hook func(req *ht.Request)

type RouteDef struct {
	path string
	//handler ht.HandlerFunc
	transformer Transformer
	name string // assumed to be same as the path if ommited?
	guards []Guard
	hooks []Hook
	subroutes []*RouteDef
	parent *RouteDef
}

func (r *RouteDef) Name() string {
	return r.name
}

func (r *RouteDef) AddTransformer(t Transformer) Transformer {
	t2 := Group(r.transformer, t)
	r.transformer = t2
	return t2
}

func Attach(r *mux.Route, hook Hook) {
	r.MatcherFunc(func(r *ht.Request, m *mux.RouteMatch) bool {
		hook(r)
		return true
	})
}

func Ward(r *mux.Route, guard Guard) {
	r.MatcherFunc(func(r *ht.Request, m *mux.RouteMatch) bool {
		if guard.Reject(r) {

			//suppose path have Guards(requireLogin, requireAdmin)
			// problem: requireAdmin takes priority
			// requireLogin should be done first

			// solution: reverse the order or guards when Guards(...) is called
			// subproblem: all guards are needlessly checked

			// solution2: type assertion

			// where to put state....

			// handler is already set to the normal handler
			m.Handler = guard.Handler

			//if m.Handler == nil {
			//	m.Handler = guard.Handler
			//} else {
			//	// note of possible performance bottleneck
			//	m.Handler = combine(m.Handler, guard.Handler)
			//}
		}
		return true
	})
}

func Guards(guards ...Guard) []Guard {
	var reversed []Guard
	for _, g := range guards {
		reversed = append(reversed, g)
	}
	return reversed
}

func Hooks(hooks ...Hook) []Hook {
	return hooks
}

func Route(path string, t interface{}, name string, hooks []Hook,
	guards []Guard, subroutes ...*RouteDef) *RouteDef {

	var transformer Transformer

	switch t := t.(type) {
		case func(w ht.ResponseWriter, r *ht.Request):
			transformer = H(ht.HandlerFunc(t))
		case ht.HandlerFunc:
			transformer = H(t)
		case Transformer:
			transformer = t
	}

	r := &RouteDef{
		path: path,
		//handler: handler,
		transformer: transformer,
		name: name,
		hooks: hooks,
		guards: guards,
		subroutes: subroutes,
	}
	for _, sub := range subroutes {
		sub.parent = r
	}
	return r
}

func SRoute(path string, t interface{},
	name string, subroutes ...*RouteDef) *RouteDef {
		return Route(path, t, name, Hooks(), Guards(), subroutes...)
}

func (r *RouteDef) MapRoute(f func(r *RouteDef)) *RouteDef {
	return MapRoute(r, f)
}

func (r *RouteDef) BuildRouter(base *mux.Router) *mux.Router {
	return BuildRouter(r, base)
}

func (r *RouteDef) Print() {
	PrintRouteDef(r)
}

func MapRoute(r *RouteDef, f func(r *RouteDef)) *RouteDef {
	f(r)
	for _, sub := range r.subroutes {
		MapRoute(sub, f)
	}
	return r
}

func BuildRouter(routeDef *RouteDef, base *mux.Router) *mux.Router {
	route := base.PathPrefix(routeDef.path).Name(routeDef.name)
	for _, hook := range  routeDef.hooks {
		Attach(route, hook)
	}
	for _, g := range  routeDef.guards {
		Ward(route, g)
	}

	//router.Handle("/", routeDef.handler)
	router := route.Subrouter()
	t := routeDef.transformer
	t.Transform(router.Path("/"))

	for _, subroute := range  routeDef.subroutes {
		BuildRouter(subroute, router)
	}
	return base
}

func (r *RouteDef) FullPath() string {
	var paths []string
	// A bit inefficient, but it'll do
	for r != nil {
		paths = append([]string{r.path}, paths...)
		r = r.parent
	}
	return filepath.Join(paths...)
}

func(r *RouteDef) String() string {
	var lines []string
	r.MapRoute(func(sub *RouteDef) {
		line := fmt.Sprintf("%-15v\t%v", sub.name, sub.FullPath())
		lines = append(lines, line)
	})
	return strings.Join(lines, "\n")
}

func(r *RouteDef) Table() []Entry {
	var table []Entry
	r.MapRoute(func(sub *RouteDef) {
		entry := Entry{sub.name, sub.FullPath()}
		table = append(table, entry)
	})
	return table
}

func PrintRouteDef(routeDef *RouteDef) {
	fmt.Println(routeDef.String())
}

type Transformer interface {
	Transform(*mux.Route)
}

type TransformerFunc func(*mux.Route)

type Ts []Transformer

func (ts Ts) Transform(r *mux.Route) {
	sub := r.Subrouter()
	sub.StrictSlash(true)
	for _, t := range ts {
		t.Transform(sub.Path("/"))
	}
}

func (transform TransformerFunc) Transform(r *mux.Route) {
	transform(r)
}

func H(handler ht.HandlerFunc) Transformer {
	return TransformerFunc(func(r *mux.Route) {
		r.HandlerFunc(handler)
	})
}

func Schemes(schemes ...string) Transformer {
	return TransformerFunc(func(r *mux.Route) {
		r.Schemes(schemes...)
	})
}

func Headers(pairs ...string) Transformer {
	return TransformerFunc(func(r *mux.Route) {
		r.Headers(pairs...)
	})
}

func Methods(methods ...string) Transformer {
	return TransformerFunc(func(r *mux.Route) {
		r.Methods(methods...)
	})
}

func Group(transformers ...Transformer) Transformer {
	var f TransformerFunc
	f = func(r *mux.Route) {
		for _, t := range transformers {
			t.Transform(r)
		}
	}
	return f
}

var GET = TransformerFunc(func(r *mux.Route) {  r.Methods("GET") })
var POST = TransformerFunc(func(r *mux.Route) { r.Methods("POST") })

func combine(h1 ht.Handler, h2 ht.Handler) ht.Handler{
	if h1 == nil { return h2 }
	if h2 == nil { return h1 }
	return ht.HandlerFunc(func(w ht.ResponseWriter, r *ht.Request) {
		h1.ServeHTTP(w, r)
		h2.ServeHTTP(w, r)
	})
}


//var GhostGuard = Guard { Reject: false }

//func Sortie(guards []Guard) Guard {
//	var sortie Guard
//	for _, g := range guards {
//		combine(sortie, g)
//	}
//	return sortie
//}




