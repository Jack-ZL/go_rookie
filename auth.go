package go_rookie

import (
	"encoding/base64"
	"net/http"
)

type Accounts struct {
	UnAuthHandler func(ctx *Context)
	Users         map[string]string // 用户名和密码的映射，用户名为键，密码为值
	Realm         string            // 认证领域，通常用于摘要认证（Digest Authentication）
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

// unAuthHandler 处理未授权的请求
func (a *Accounts) unAuthHandler(ctx *Context) {
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.Header().Set("WWW-Authenticate", a.Realm) // 摘要认证（Digest）
		ctx.W.WriteHeader(http.StatusUnauthorized)      // 401 Unauthorized
	}
}

func BasicAuth(username, password string) string {
	auth := username + ":" + password // 拼接用户名和密码作为认证信息的基础信息
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
