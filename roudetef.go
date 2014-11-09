package roudetef

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"math"
	ht "net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// TODO:
// Add accept (in addition to reject) on guards
// Interchange order of route name and transformer arguments
// Remove API leaks
// Allow route def serialization

// code convention note:
// Since quotes aren't allowed in identifiers,
// _ (underscore) is used instead.
// e.g.
// somevar  := blah()
// somevar_ := wah(somevar)

type Entry struct {
	Name    string
	Path    string
	Methods string
}

type Guard struct {
	Reject  func(*ht.Request) bool
	Handler ht.HandlerFunc
}

type pathod struct {
	path    string
	methods []string
}

type HandlerT struct {
	handler     ht.HandlerFunc
	transformer Transformer
}

type Hook func(req *ht.Request)

type RouteDef struct {
	Name        string
	Path        string
	methods     []string
	Handler     ht.HandlerFunc
	transformer Transformer
	guards      []Guard
	hooks       []Hook
	parent      *RouteDef
	subroutes   []*RouteDef
}

type ReRouteDef struct {
	destName   string
	pathPrefix string
	namePrefix string
	guards     []Guard
	hooks      []Hook
}

// solution for safely emulating union/variant types
type SubRouteDef interface {
	SubRouteDef()
}

func (r *RouteDef) SubRouteDef()   {}
func (r *ReRouteDef) SubRouteDef() {}

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

func Route(pathmethod interface{}, handlerT interface{}, name string, hooks []Hook,
	guards []Guard, subroutes ...SubRouteDef) *RouteDef {

	var path string
	var methods []string
	switch t := pathmethod.(type) {
	case string:
		path = t
	case pathod:
		{
			path = t.path
			methods = t.methods
		}
	default:
		panic("Invalid path argument")
	}

	var handler ht.HandlerFunc
	var transformer Transformer
	switch t := handlerT.(type) {
	case ht.Handler:
		handler = func(w ht.ResponseWriter, r *ht.Request) {
			t.ServeHTTP(w, r)
		}
	case func(ht.ResponseWriter, *ht.Request):
		handler = t
	case HandlerT:
		{
			handler = t.handler
			transformer = t.transformer
		}
	default:
		panic("Invalid handlerT argument")
	}

	r := &RouteDef{
		Path:        path,
		methods:     methods,
		Handler:     handler,
		transformer: transformer,
		Name:        name,
		hooks:       hooks,
		guards:      guards,
	}
	r.subroutes = expandReRoutes(r, subroutes)

	for _, sub := range r.subroutes {
		sub.parent = r
	}
	return r
}

func SRoute(pathmethod interface{}, handlerT interface{},
	name string, subroutes ...SubRouteDef) *RouteDef {
	return Route(pathmethod, handlerT, name, Hooks(), Guards(), subroutes...)
}

func ReRoute(pathPrefix string, namePrefix string, destName string,
	hooks []Hook, guards []Guard) *ReRouteDef {
	return &ReRouteDef{
		pathPrefix: pathPrefix,
		namePrefix: namePrefix,
		destName:   destName,
		hooks:      hooks,
		guards:     guards,
	}
}

func ReSRoute(pathPrefix string, namePrefix string, destName string) *ReRouteDef {
	return ReRoute(pathPrefix, namePrefix, destName, Hooks(), Guards())
}

func (r *RouteDef) Search(name string) *RouteDef {
	return SearchRoute(r, name)
}

func (r *RouteDef) Iter(f func(r *RouteDef)) *RouteDef {
	return IterRoute(r, f)
}

func (r *RouteDef) Map(f func(r RouteDef) RouteDef) *RouteDef {
	return MapRoute(r, f)
}

func (r *RouteDef) BuildRouter(base *mux.Router) *mux.Router {
	return BuildRouter(r, base)
}

func (r *RouteDef) BuildNewRouter() *mux.Router {
	base := mux.NewRouter()
	base.StrictSlash(true)
	return BuildRouter(r, base)
}

func (r *RouteDef) Print() {
	PrintRouteDef(r)
}

func SearchRoute(r *RouteDef, name string) *RouteDef {
	var result *RouteDef
	if r.Name == name {
		result = r
	} else {
		for _, sub := range r.subroutes {
			result = SearchRoute(sub, name)
			if result != nil {
				break
			}
		}
	}
	return result
}

func IterRoute(r *RouteDef, f func(r *RouteDef)) *RouteDef {
	f(r)
	for _, sub := range r.subroutes {
		IterRoute(sub, f)
	}
	return r
}

func MapRoute(r *RouteDef, f func(r RouteDef) RouteDef) *RouteDef {
	r_ := f(*r) // nil exception?
	var subroutes []*RouteDef
	for _, sub := range r_.subroutes {
		sub := MapRoute(sub, f)
		sub.parent = &r_
		subroutes = append(subroutes, sub)
	}
	r_.subroutes = subroutes
	return &r_
}

func BuildRouter(routeDef *RouteDef, base *mux.Router) *mux.Router {
	route := base.PathPrefix(routeDef.Path).Name(routeDef.Name)

	for _, hook := range routeDef.hooks {
		Attach(route, hook)
	}

	Ward(route, routeDef.guards...)

	if routeDef.Handler != nil {
		route.HandlerFunc(routeDef.Handler)
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
		if routeDef.Handler != nil {
			router.HandleFunc("/", routeDef.Handler)
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
		paths = append([]string{r.Path}, paths...)
		r = r.parent
	}
	return filepath.Join(paths...)
}

func (r *RouteDef) String() string {
	var lines []string
	col1Len := 0
	col2Len := 0
	r.Iter(func(sub *RouteDef) {
		ms := stringMethods(sub.methods)
		col1Len = int(math.Max(float64(col1Len), float64(len(ms))))
		col2Len = int(math.Max(float64(col2Len), float64(len(sub.FullPath()))))
	})
	r.Iter(func(sub *RouteDef) {
		ms := stringMethods(sub.methods)
		fmts := fmt.Sprintf("%%-%vv  %%-%vv %%v", col1Len, col2Len)
		line := fmt.Sprintf(fmts, ms, sub.FullPath(), sub.Name)
		lines = append(lines, line)
	})
	return strings.Join(lines, "\n")
}

func (r *RouteDef) Table() []Entry {
	var table []Entry
	r.Iter(func(sub *RouteDef) {
		entry := Entry{sub.Name, sub.FullPath(), stringMethods(sub.methods)}
		table = append(table, entry)
	})
	return table
}

func (r *RouteDef) CreateUrlFn(returnErrOpt ...bool) UrlFn {
	routes := r.BuildNewRouter()
	return CreateUrlFn(routes, returnErrOpt...)
}

type UrlFn func(name string, params ...string) (string, error)

func CreateUrlFn(routes *mux.Router, returnErrOpt ...bool) UrlFn {
	returnErr := true
	if len(returnErrOpt) > 0 {
		returnErr = returnErrOpt[0]
	}

	var __ = func(urlpath *url.URL, err error) (string, error) {
		if err != nil {
			if returnErr {
				return "", err
			}
			// embed error in the url
			return url.QueryEscape(fmt.Sprint("(%s)", err.Error())), nil
		}
		return urlpath.String(), nil
	}

	return func(name string, params ...string) (string, error) {
		r := routes.Get(name)
		if r != nil {
			urlpath, err := r.URL(params...)
			return __(urlpath, err)
		}
		return __(nil, errors.New("invalid route name"))
	}
}

func PrintRouteDef(routeDef *RouteDef) {
	fmt.Println(routeDef.String())
	fmt.Println("----------")
}

func rootRoute(routeDef *RouteDef) *RouteDef {
	root := routeDef
	for root.parent != nil {
		root = root.parent
	}
	return root
}

func expandReRoutes(base *RouteDef, routes []SubRouteDef) []*RouteDef {
	var routes_ []*RouteDef
	for _, r := range routes {
		var reroute *ReRouteDef

		switch t := r.(type) {
		case *RouteDef:
			routes_ = append(routes_, t)
			continue
		case *ReRouteDef:
			reroute = t
		}

		rebase := searchRoute(reroute.destName, routes)

		if rebase == nil {
			println("Route not found:", reroute.destName,
				"Re-route Must be the same level as the destination.")
			continue
		}
		var temp RouteDef
		temp = *rebase
		temp.Path = filepath.Join(reroute.pathPrefix, temp.Path)

		rebase = temp.Map(func(route RouteDef) RouteDef {
			route.Name = reroute.namePrefix + "-" + route.Name
			return route
		})

		rebase.hooks = append(rebase.hooks, reroute.hooks...)
		rebase.guards = append(rebase.guards, reroute.guards...)
		routes_ = append(routes_, rebase)
	}
	return routes_
}

func searchRoute(name string, routes []SubRouteDef) *RouteDef {
	for _, r := range routes {
		var routeDef *RouteDef

		switch t := r.(type) {
		case *RouteDef:
			routeDef = t
		default:
			continue
		}
		if routeDef.Name == name {
			return routeDef
		}
	}
	return nil
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

func With(handler ht.HandlerFunc, ts ...Transformer) HandlerT {
	return HandlerT{handler, Group(ts...)}
}

func Group(transformers ...Transformer) Transformer {
	var ts []Transformer
	for _, t := range transformers {
		if t == nil {
			continue
		}
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

func Methods(methods ...string) func(string) pathod {
	return func(path string) pathod {
		return pathod{
			path:    path,
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
