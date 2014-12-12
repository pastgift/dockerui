package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

    "math/rand"
    "time"
    "strconv"
    "fmt"
    "crypto/md5"
)

var (
	endpoint = flag.String("e", "/var/run/docker.sock", "Dockerd endpoint")
	addr     = flag.String("p", ":9000", "Address and port to serve dockerui")
	assets   = flag.String("a", ".", "Path to the assets")

    username = ""
    password = ""
    token    = ""
)

func GetMD5(s string) string {
    data := []byte(s)
    return md5.Sum(data)
}

func CreateRandomString() string {
    rand.Seed(time.Now().UnixNano())
    randInt := rand.Int63()
    randStr := strconv.FormatInt(randInt, 36)

    return randStr
}

func AuthenticateUser(r *http.Request) bool {
    cookie, err := r.Cookie("token")
    if err != nil || cookie.Value == "" || cookie.Value != token {
        fmt.Println("User not login")
        return false
    }
    return true
}

type DoLoginHandler struct {
}

func (f *DoLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Inputed UserName:", r.PostFormValue("username"))
    fmt.Println("Inputed Password:", r.PostFormValue("password"))
    if r.Method != "POST" || r.PostFormValue("username") != username || GetMD5(r.PostFormValue("password")) != password {
        http.Redirect(w, r, "/login.html", http.StatusFound)
    }

    token = CreateRandomString()

    cookie := http.Cookie{
        Name:"token",
        Value:token,
    }
    http.SetCookie(w, &cookie)

    http.Redirect(w, r, "/", http.StatusFound)
}

type AuthenticatedFileServer struct {
    BaseFileServerHandler http.Handler
}

func NewAuthenticatedFileServer(root http.FileSystem) *AuthenticatedFileServer {
    afs := AuthenticatedFileServer{}
    afs.BaseFileServerHandler = http.FileServer(root)

    return &afs
}

func (f *AuthenticatedFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Request:", r.URL.Path)

    if strings.HasPrefix(r.URL.Path, "/css/") || AuthenticateUser(r) == true {
        f.BaseFileServerHandler.ServeHTTP(w, r)
    } else {
        http.Redirect(w, r, "/login.html", http.StatusFound)
    }
}

type UnixHandler struct {
	path string
}

func (h *UnixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func createTcpHandler(e string) http.Handler {
	u, err := url.Parse(e)
	if err != nil {
		log.Fatal(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func createUnixHandler(e string) http.Handler {
	return &UnixHandler{e}
}

func createHandler(dir string, e string) http.Handler {
	var (
		mux         = http.NewServeMux()
		loginHandler = http.FileServer(http.Dir(dir))
        doLoginHandler = new(DoLoginHandler)
        aFileHandler = NewAuthenticatedFileServer(http.Dir(dir))
		h           http.Handler
	)

	if strings.Contains(e, "http") {
		h = createTcpHandler(e)
	} else {
		if _, err := os.Stat(e); err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("unix socket %s does not exist", e)
			}
			log.Fatal(err)
		}
		h = createUnixHandler(e)
	}

    mux.Handle("/login.html", loginHandler)
    mux.Handle("/dologin", doLoginHandler)
	mux.Handle("/dockerapi/", http.StripPrefix("/dockerapi", h))
	mux.Handle("/", aFileHandler)
	return mux
}

func initUser() {
    // TODO read user/password from shadow
    token = ""
}

func main() {
	flag.Parse()
    initUser()

	handler := createHandler(*assets, *endpoint)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
