
package main

import (
	"net/http"
	"nvlled/rut"
	//"fmt"
	"testing"
	"net/http/httptest"
	"net/http/cookiejar"
	//"github.com/gorilla/mux"
	"io/ioutil"
	"fmt"
	//"strings"
	//"path/filepath"
)

func TestPaths(t *testing.T) {
	def := routeDefinition()
	table := def.Table()
	expected := []rut.Entry{
		rut.Entry{"home-path",		"/"},
		rut.Entry{"login-path",		"/login"},
		rut.Entry{"logout-path",	"/logout"},
		rut.Entry{"broke-path",		"/broke"},
		rut.Entry{"a-path",			"/a"},
		rut.Entry{"b-path",			"/a/b"},
		rut.Entry{"c-path",			"/a/b/c"},
		rut.Entry{"d-path",			"/a/d"},
	}
	if !sameTable(table, expected) {
		t.Fail()
	}
}

func TestHook(t *testing.T) {
	root, _ := createHandler()

	var hooked bool
	rut.Attach(root.Get("a-path"), func(_ *http.Request) {
		hooked = true
	})

	server := httptest.NewServer(root)
	http.Get(server.URL+"/login")

	hooked = false
	http.Get(server.URL+"/a/")
	if !hooked { t.Fail() }

	hooked = false
	http.Get(server.URL+"/a/b/")
	if !hooked { t.Fail() }

	hooked = false
	http.Get(server.URL+"/login")
	if hooked { t.Fail() }
}

func TestGuard(t *testing.T) {
	root, _ := createHandler()
	server := httptest.NewServer(root)
	client := createClient()

	// path /a and its subpaths /a/b, /a/b/c, etc...
	// are protected and requires a login
	resp,_ := client.Get(server.URL+"/a")
	if resp.StatusCode == http.StatusOK { t.Fail() }
	resp,_ = client.Get(server.URL+"/a/b")
	if resp.StatusCode == http.StatusOK { t.Fail() }
	resp,_ = client.Get(server.URL+"/a/b/c")
	if resp.StatusCode == http.StatusOK { t.Fail() }
	resp,_ = client.Get(server.URL+"/a/d")
	if resp.StatusCode == http.StatusOK { t.Fail() }

	// super login
	resp,_ = client.Get(server.URL+"/login")

	resp,_ = client.Get(server.URL+"/a")
	if resp.StatusCode != http.StatusOK { t.Fail() }
	resp,_ = client.Get(server.URL+"/a/b")
	if resp.StatusCode != http.StatusOK { t.Fail() }
	resp,_ = client.Get(server.URL+"/a/b/c")
	if resp.StatusCode != http.StatusOK { t.Fail() }
	resp,_ = client.Get(server.URL+"/a/d")
	if resp.StatusCode != http.StatusOK { t.Fail() }
}

func TestRouting(t *testing.T) {
	root, def := createHandler()
	server := httptest.NewServer(root)
	c := createClient()

	base := server.URL
	for _, name := range routeNames(def) {
		msg, ok := message[name]
		if !ok {
			continue
		}

		path,err := root.Get(name).URL()
		if err != nil { panic(err) }

		c.Get(base+"/login") // ensure access to all path
		resp := get(c, base+path.String())

		//fmt.Printf("%v: %v == %v\n", name, resp, msg)
		if resp != msg {
			t.Fail()
		}
	}
}

func routeNames(def *rut.RouteDef) []string {
	var names []string
	def.MapRoute(func(r *rut.RouteDef) {
		names = append(names, r.Name())
	})
	return names
}

func get(c *http.Client, path string) string {
	resp, _ := c.Get(path)
	return body(resp)
}

func body(resp *http.Response) string {
	s,_ := ioutil.ReadAll(resp.Body)
	return string(s)
}

func createClient() *http.Client {
	c := new(http.Client)
	c.Jar,_ = cookiejar.New(nil)
	return c
}

func sameTable(t1 []rut.Entry, t2 []rut.Entry) bool {
	if len(t1) != len(t2) {
		return false
	}
	for i, _ := range t1 {
		if t1[i] != t2[i] {
			return false
		}
	}
	return true
}


