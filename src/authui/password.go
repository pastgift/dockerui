package authui

import (
    "net/http"
    "io/ioutil"
    "fmt"
)

type DoPasswordHandler struct {
}

func NewDoPasswordHandler() *DoPasswordHandler {
    return &DoPasswordHandler{}
}

func (h *DoPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var (
        hasErr     = false
    )

    if r.Method != "POST" || AuthenticateUser(w, r) == false {
        hasErr = true
    }

    if  r.PostFormValue("ui_username") != UIUsername || GetMD5(r.PostFormValue("ui_password")) != UIPassword {
        hasErr = true
    }

    if r.PostFormValue("ui_newpassword") != r.PostFormValue("ui_newpassword2") {
        hasErr = true
    }

    // If error occured, return to `/password.html`
    if hasErr {
        http.Redirect(w, r, "/password.html", http.StatusFound)
        return
    }

    // If no error occured, return to homepage 
    UIUsername = r.PostFormValue("ui_newusername")
    UIPassword = GetMD5(r.PostFormValue("ui_newpassword"))

    userInfo  := fmt.Sprintf("%s\n%s", UIUsername, UIPassword)
    ioutil.WriteFile("shadow", []byte(userInfo), 0644)

    http.Redirect(w, r, HOME_PAGE_PATH, http.StatusFound)
}
