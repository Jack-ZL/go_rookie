package register

import (
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"time"
)

type Option struct {
	Endpoints         []string      // 节点
	DialTimeout       time.Duration // 超时时间
	ServiceName       string        // 服务名称
	Host              string        // 域名
	Port              int           // 端口号
	NacosServerConfig []constant.ServerConfig
	NacosClientConfig *constant.ClientConfig
}

type GrRegister interface {
	CreateCli(option Option) error
	RegisterService(serviceName string, host string, port int) error
	GetValue(serviceName string) (string, error)
	Close() error
}
