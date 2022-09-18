package token

import (
	"github.com/Jack-ZL/go_rookie"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

type JwtHandler struct {
	Alg            string           // jwt的加密算法
	TimeOut        time.Duration    // 过期时间
	RefreshTimeOut time.Duration    // refreshToken的过期时间
	TimeFunc       func() time.Time // 时间函数
	Key            []byte           // 加密密钥
	PrivateKey     string           // 私钥
	SendCookie     bool             // 是否发送存储到cookie
	Authenticator  func(ctx *go_rookie.Context) (map[string]any, error)
	CookieName     string // cookie缓存键名
	CookieMaxAge   int64  // cookie有效期
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
}

type JwtResponse struct {
	Token        string
	RefreshToken string
}

/**
 * LoginHandler
 * @Author：Jack-Z
 * @Description: jwt-登录认证：用户认证->用户id生成jwt，并保存到cooki或直接返回
 * @receiver j
 * @param ctx
 * @return *JwtResponse
 * @return error
 */
func (j *JwtHandler) LoginHandler(ctx *go_rookie.Context) (*JwtResponse, error) {
	data, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err
	}

	if j.Alg == "" {
		j.Alg = "HS256"
	}
	// part-A
	method := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(method)

	// part-B
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for k, v := range data {
			claims[k] = v
		}
	}

	if j.TimeFunc == nil {
		j.TimeFunc = func() time.Time {
			return time.Now()
		}
	}

	expire := j.TimeFunc().Add(j.TimeOut)

	claims["exp"] = expire.Unix() // 过期时间
	claims["iat"] = j.TimeFunc().Unix()

	// part-C
	var tokenString string
	var tokenError error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenError = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenError = token.SignedString(j.Key)
	}

	if tokenError != nil {
		return nil, tokenError
	}

	jr := &JwtResponse{
		Token: tokenString,
	}

	refreshToken, err := j.refreshToken(token)
	if err != nil {
		return nil, err
	}
	jr.RefreshToken = refreshToken
	if j.SendCookie {
		// 发送到cookie存储
		if j.CookieName == "" {
			j.CookieName = "gr_token"
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFunc().Unix()

		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	return jr, nil
}

/**
 * usingPublicKeyAlgo
 * @Author：Jack-Z
 * @Description: 加密方式判断
 * @receiver j
 * @return bool
 */
func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

/**
 * refreshToken
 * @Author：Jack-Z
 * @Description: refreshToken的生成
 * @receiver j
 * @param token
 * @return string
 * @return error
 */
func (j *JwtHandler) refreshToken(token *jwt.Token) (string, error) {
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = j.TimeFunc().Add(j.RefreshTimeOut)

	var tokenString string
	var tokenError error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenError = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenError = token.SignedString(j.Key)
	}

	if tokenError != nil {
		return "", tokenError
	}
	return tokenString, nil
}
