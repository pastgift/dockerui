package main // import "github.com/crosbymichael/dockerui"

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"

//Deleted by pastgift
//	"net/url"
//	"strings"

    "fmt"
    "authui"
)

var (
	endpoint = flag.String("e", "/var/run/docker.sock", "Dockerd endpoint")
	addr     = flag.String("p", ":9000", "Address and port to serve dockerui")
	assets   = flag.String("a", ".", "Path to the assets")
)

type UnixHandler struct {
	path string
}

//Modified by pastgift: Add user authentication
func (h *UnixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Avoid unauthenticated access from AngularJS app
    fmt.Println(">>> UnixHandler <<<")
    fmt.Println("Request:", r.URL.Path)
    for i, v := range r.Cookies() {
        fmt.Printf("Cookies[%d]: %s\n", i, v)
    }

    if authui.AuthenticateUser(w, r) == false {
        http.Redirect(w, r, "/login.html", http.StatusFound)
        return
    }

	conn, err := net.Dial("unix", h.path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	c := httputil.NewClientConn(conn, nil)
	defer c.Close()

	res, err := c.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer res.Body.Close()

	copyHeader(w.Header(), res.Header)
	if _, err := io.Copy(w, res.Body); err != nil {
		log.Println(err)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

//Deleted by pastgift:
// Only for local management
//func createTcpHandler(e string) http.Handler {
//	u, err := url.Parse(e)
//	if err != nil {
//		log.Fatal(err)
//	}
//	return httputil.NewSingleHostReverseProxy(u)
//}

func createUnixHandler(e string) http.Handler {
	return &UnixHandler{e}
}

//Modified by pastgift
func createHandler(dir string, e string) http.Handler {
	var (
		mux               = http.NewServeMux()
		apiHandler          http.Handler
		fileHandler       = http.FileServer(http.Dir(dir))
        doLoginHandler    = authui.NewDoLoginHandler()
        doPasswordHandler = authui.NewDoPasswordHandler()
        authFileHandler   = authui.NewAuthenticatedFileServer(http.Dir(dir))
	)

//	if strings.Contains(e, "http") {
//		apiHandler = createTcpHandler(e)
//	} else {
		if _, err := os.Stat(e); err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("unix socket %s does not exist", e)
			}
			log.Fatal(err)
		}
		apiHandler = createUnixHandler(e)
//	}

    // `login.html` does not need authentication
    mux.Handle("/login.html", fileHandler)

    mux.Handle("/dopassword", doPasswordHandler)
    mux.Handle("/dologin",    doLoginHandler)

	mux.Handle("/dockerapi/", http.StripPrefix("/dockerapi", apiHandler))
	mux.Handle("/",           authFileHandler)
	return mux
}

func main() {
	flag.Parse()

    //Add by pastgift
    authui.InitUser()

	handler := createHandler(*assets, *endpoint)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
