package go_rookie

import (
	"encoding/base64"
	"net/http"
)

type Accounts struct {
	UnAuthHandler func(ctx *Context)
	Users         map[string]string
	Realm         string
}

func (a *Accounts) BasicAuth(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			a.unAuthHandler(ctx)
			return
		}
		pwd, exist := a.Users[username]
		if !exist {
			a.unAuthHandler(ctx)
			return
		}

		if password != pwd {
			a.unAuthHandler(ctx)
			return
		}

		ctx.Set("user", username)
		next(ctx)
	}
}

func (a *Accounts) unAuthHandler(ctx *Context) {
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.Header().Set("WWW-Authenticate", a.Realm)
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}

func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
