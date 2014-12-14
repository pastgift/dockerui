package authui

import (
    "time"
    "net/http"
    "io/ioutil"
    "strings"
    "strconv"
    "math/rand"
    "crypto/md5"
    "fmt"
)

const (
    COOKIE_MAX_AGE = 60 * 60
    HOME_PAGE_PATH = "/index.html"
    LOGIN_PAGE_PATH = "/login.html"
)

var (
    UIUsername = ""
    UIPassword = ""
    UIToken    = ""
)

func GetMD5(s string) string {
    data    := []byte(s)
    md5Str := fmt.Sprintf("%x", md5.Sum(data))
    fmt.Println("MD5:", md5Str)
    return md5Str
}

func CreateRandomString() string {
    rand.Seed(time.Now().UnixNano())
    randInt := rand.Int63()
    randStr := strconv.FormatInt(randInt, 36)

    fmt.Println("RandomString Created:", randStr)
    return randStr
}

func AuthenticateUser(w http.ResponseWriter, r *http.Request) bool {
    cookie, err := r.Cookie("ui_token")
    if err != nil || cookie.Value == "" || cookie.Value != UIToken {
        fmt.Println("User not login:", r.URL.Path)
        return false
    }

    cookie.MaxAge = COOKIE_MAX_AGE
    cookie.Path   = "/"

    http.SetCookie(w, cookie)

    return true
}

func InitUser() {
    // Read user/password from shadow
    f, err := ioutil.ReadFile("shadow")
    if err != nil {
        // Default username/password is `admin`
        UIUsername = "admin"
        UIPassword = GetMD5("admin")
    } else {
        userInfo  := strings.Split(string(f), "\n")
        UIUsername = userInfo[0]
        UIPassword = userInfo[1]
    }

    UIToken = ""
}

type DoLoginHandler struct {
}

func NewDoLoginHandler() *DoLoginHandler {
    return &DoLoginHandler{}
}

func (h *DoLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Println(">>> DoLoginHandler <<<")
    fmt.Println("Inputed UserName:", r.PostFormValue("ui_username"))
    fmt.Println("Inputed Password:", r.PostFormValue("ui_password"))
    if r.Method != "POST" || r.PostFormValue("ui_username") != UIUsername || GetMD5(r.PostFormValue("ui_password")) != UIPassword {
        http.Redirect(w, r, "/login.html", http.StatusFound)
        return
    }

    UIToken = CreateRandomString()

    cookie := http.Cookie{
        Name  :"ui_token",
        Value : UIToken,
        Path  :"/",
        MaxAge:COOKIE_MAX_AGE,
    }
    http.SetCookie(w, &cookie)

    http.Redirect(w, r, HOME_PAGE_PATH, http.StatusFound)
}

type AuthenticatedFileServer struct {
    BaseFileServerHandler http.Handler
}

func NewAuthenticatedFileServer(root http.FileSystem) *AuthenticatedFileServer {
    afs := AuthenticatedFileServer{}
    afs.BaseFileServerHandler = http.FileServer(root)

    return &afs
}

func (h *AuthenticatedFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Println(">>> AuthenticatedFileServer <<<")
    fmt.Println("Request:", r.URL.Path)
    for i, v := range r.Cookies() {
        fmt.Printf("Cookies[%d]: %s\n", i, v)
    }

    // Avoid show page directly
    if r.URL.Path == "/" && AuthenticateUser(w, r) == false {
        http.Redirect(w, r, LOGIN_PAGE_PATH, http.StatusFound)
        return
    }

    if strings.HasPrefix(r.URL.Path, "/css/") || strings.HasPrefix(r.URL.Path, "/assets/css/") || AuthenticateUser(w, r) == true {
        h.BaseFileServerHandler.ServeHTTP(w, r)
    } else {
        http.Redirect(w, r, "/login.html", http.StatusFound)
    }
}