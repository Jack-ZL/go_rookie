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
	CreateCli(option Option) error                                   //创建客户端
	RegisterService(serviceName string, host string, port int) error //通过名称注册服务
	GetValue(serviceName string) (string, error)                     //通过服务名称获取一个实例
	Close() error                                                    //关闭客户端
}
