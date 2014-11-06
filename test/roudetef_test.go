
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

func TestMap(t *testing.T) {
	routeDef1 := routeDefinition()
    routeDef2 := routeDef1.Map(func(r def.RouteDef) def.RouteDef {
        r.Name = "test-"+r.Name
        return r
    })

    name := "test-a-path"
    subroute := routeDef1.Search(name)
    if subroute != nil {
        t.Error("original route should not be modified")
    }

    subroute = routeDef2.Search("test-a-path")
    if subroute.Name != name {
        t.Error("wrong search route result")
    }
    if subroute == nil {
        t.Error("route mapping failed")
    }
    if subroute.Name != name {
        t.Error("wrong search route result")
    }
    subroute.Map(func(r def.RouteDef) def.RouteDef {
        r.Name = "***"+r.Name
        return r
    })

    subroute = routeDef1.Search("test-a-path")
    if subroute != nil {
        t.Error("original route should not be modified")
    }
    subroute = routeDef2.Search("test-a-path")
    if subroute == nil {
        t.Error("route mapping failed")
    }
    if subroute.Search("test-login-path") != nil {
        t.Error("search should not work upstream")
    }
}

func TestPaths(t *testing.T) {
	routeDef := routeDefinition()
	table := routeDef.Table()
	expected := []def.Entry{
		def.Entry{"home-path",		"/",	   "ANY"},
		def.Entry{"sudo-path",		"/sudo",   "ANY"},
		def.Entry{"admin-path",		"/admin",  "ANY"},
		def.Entry{"login-path",		"/login",  "ANY"},
		def.Entry{"logout-path",	"/logout", "ANY"},
		def.Entry{"broke-path",		"/broke",  "ANY"},
		def.Entry{"submit-get",		"/submit", "GET"},
		def.Entry{"submit-post",	"/submit", "POST"},
		def.Entry{"a-path",		"/a",      "ANY"},
		def.Entry{"b-path",		"/a/b",    "ANY"},
		def.Entry{"c-path",		"/a/b/c",  "ANY"},
		def.Entry{"d-path",		"/a/d",    "ANY"},
	}
	routeDef.Print()
	if !sameTable(table, expected) {
		t.Fail()
	}
}

func TestPathGeneration(t *testing.T) {
	routeDef := routeDefinition()
    urlfor := routeDef.CreateUrlFn()
    for _,entry := range routeDef.Table() {
        if url,_ := urlfor(entry.Name); url != entry.Path {
            t.Error("wrong path for " + entry.Name + ": " + entry.Path)
        }
    }
    if _,err := urlfor("x-path"); err == nil {
        t.Error("error expected")
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
