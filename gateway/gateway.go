package gateway

import "net/http"

// 网关配置
type GWConfig struct {
	Name        string                  // 网管名称
	Path        string                  // 路径
	Host        string                  // ip地址
	Port        int                     // 端口
	Header      func(req *http.Request) //请求的header
	ServiceName string                  //服务名称
}
