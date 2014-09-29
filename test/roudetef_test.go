
package main

import (
	"net/http"
	def "github.com/nvlled/roudetef"
	//"fmt"
	"testing"
	"net/http/httptest"
	"net/http/cookiejar"
	//"github.com/gorilla/mux"
	"io/ioutil"
	//"fmt"
	//"strings"
	//"path/filepath"
)

func TestPaths(t *testing.T) {
	routeDef := routeDefinition()
	table := routeDef.Table()
	expected := []def.Entry{
		def.Entry{"home-path",		"/"},
		def.Entry{"sudo-path",		"/sudo"},
		def.Entry{"admin-path",		"/admin"},
		def.Entry{"login-path",		"/login"},
		def.Entry{"logout-path",	"/logout"},
		def.Entry{"broke-path",		"/broke"},
		def.Entry{"submit-path",	"/submit"},
		def.Entry{"a-path",			"/a"},
		def.Entry{"b-path",			"/a/b"},
		def.Entry{"c-path",			"/a/b/c"},
		def.Entry{"d-path",			"/a/d"},
	}
	routeDef.Print()
	if !sameTable(table, expected) {
		t.Fail()
	}
}

func TestHook(t *testing.T) {
	root, _ := createHandler()

	var hooked bool
	def.Attach(root.Get("a-path"), func(_ *http.Request) {
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

func TestRequest(t *testing.T) {
	root, _ := createHandler()
	server := httptest.NewServer(root)
	c := createClient()

	resp,_ := request(c, "GET",  server.URL+"/submit/")
	if resp.StatusCode != http.StatusOK {
		t.Fail()
	}
	resp,_ = request(c, "POST",  server.URL+"/submit/")
	if resp.StatusCode == http.StatusOK {
		t.Fail()
	}
	// route requires a header "X:123"
	resp,_ = request(c, "POST",  server.URL+"/submit/", "X", "123")
	if resp.StatusCode != http.StatusOK {
		t.Fail()
	}
}

func routeNames(def *def.RouteDef) []string {
	return []string{
		"home-path",
		"a-path",
		"b-path",
		"c-path",
		"d-path",
	}
}

func request(c *http.Client, method, path string, headers ...string) (*http.Response, string) {
	request,_ := http.NewRequest(method, path, nil)
	i := 0
	for i < len(headers) - 1 {
		key := headers[i]
		val := headers[i+1]
		request.Header[key] = []string{val}
		i += 2
	}
	resp,_ := c.Do(request)
	return resp, body(resp)
}

func get(c *http.Client, path string) string {
	resp, _ := c.Get(path)
	return body(resp)
}

func post(c *http.Client, path string) string {
	resp, _ := c.Post(path, "", nil)
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

func sameTable(t1 []def.Entry, t2 []def.Entry) bool {
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




