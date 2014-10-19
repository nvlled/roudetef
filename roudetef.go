
package roudetef

import (
	ht "net/http"
	"github.com/gorilla/mux"
	"path/filepath"
	"strings"
	"fmt"
)

// TODO:
// Add accept (in addition to reject) on guards
// Interchange order of route name and transformer arguments
// Remove API leaks
// Allow route def serialization

type Entry struct {
	Name string
	Path string
	Methods string
}

type Guard struct {
	Reject func(*ht.Request)bool
	Handler ht.HandlerFunc
}

type pathod struct {
	path string
	methods []string
}

type Hook func(req *ht.Request)

type RouteDef struct {
	path string
	methods []string
	handler ht.HandlerFunc
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

func Ward(r *mux.Route, guards ...Guard) {
	r.MatcherFunc(func(r *ht.Request, m *mux.RouteMatch) bool {
		for _, g := range guards {
			if g.Reject(r) {
				m.Handler = g.Handler
				break
			}
		}
		return true
	})
}

func Guards(guards ...Guard) []Guard {
	return guards
}

func Hooks(hooks ...Hook) []Hook {
	return hooks
}

func Route(pathmethod interface{}, t interface{}, name string, hooks []Hook,
	guards []Guard, subroutes ...*RouteDef) *RouteDef {

	var handler ht.HandlerFunc
	var transformer Transformer

	var path string
	var methods []string

	switch t := pathmethod.(type) {
		case string: path = t
		case pathod: {
			path = t.path
			methods = t.methods
		}
	}

	switch t := t.(type) {
		case func(w ht.ResponseWriter, r *ht.Request):
			handler = t
		case ht.HandlerFunc:
			handler = t
		case Transformer:
			transformer = t
	}

	r := &RouteDef{
		path: path,
		methods: methods,
		name: name,
		hooks: hooks,
		guards: guards,
		subroutes: subroutes,
	}
	if handler != nil { r.handler = handler }
	if transformer != nil { r.transformer = transformer }

	for _, sub := range subroutes {
		sub.parent = r
	}
	return r
}

func SRoute(path interface{}, t interface{},
	name string, subroutes ...*RouteDef) *RouteDef {
		return Route(path, t, name, Hooks(), Guards(), subroutes...)
}

func (r *RouteDef) Map(f func(r *RouteDef)) *RouteDef {
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

	Ward(route, routeDef.guards...)

	if routeDef.handler != nil {
		route.HandlerFunc(routeDef.handler)
	}

	if routeDef.methods != nil {
		route.Methods(routeDef.methods...)
	}

	t := routeDef.transformer
	if t != nil {
		t.Transform(route)
	}

	subroutes := routeDef.subroutes
	if len(subroutes) > 0 {
		// The handler func for a path prefix
		// fails when subrouter is called.
		// Call subrouter() only when there are no
		// subroutes.
		router := route.Subrouter()
		if routeDef.handler != nil {
			router.HandleFunc("/", routeDef.handler)
		}
		for _, subroute := range routeDef.subroutes {
			BuildRouter(subroute, router)
		}
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
	r.Map(func(sub *RouteDef) {
		ms := stringMethods(sub.methods)
		line := fmt.Sprintf("%-15v\t%v\t%s", sub.name, sub.FullPath(), ms)
		lines = append(lines, line)
	})
	return strings.Join(lines, "\n")
}

func(r *RouteDef) Table() []Entry {
	var table []Entry
	r.Map(func(sub *RouteDef) {
		entry := Entry{sub.name, sub.FullPath(), stringMethods(sub.methods)}
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

func Group(transformers ...Transformer) Transformer {
	var ts []Transformer
	for _, t := range transformers {
		if t == nil { continue }
		ts = append(ts, t)
	}

	var f TransformerFunc
	f = func(r *mux.Route) {
		for _, t := range ts {
			t.Transform(r)
		}
	}
	return f
}

func Methods(methods ...string) func(string)pathod {
	return func(path string) pathod {
		return pathod{
			path: path,
			methods: methods,
		}
	}
}

var GET = Methods("GET")
var POST = Methods("POST")
var HEAD = Methods("HEAD")
//...I'll add the others later


func stringMethods(methods []string) string {
	if methods == nil {
		return "ANY"
	}
	return strings.Join(methods, ",")
}


