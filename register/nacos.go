package register

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func CreateNacosClient() (naming_client.INamingClient, error) {
	// 创建clientConfig的另一种方式
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(""), // 当namespace是public时，此处填空字符串。
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)
	// 创建serverConfig的另一种方式
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			"127.0.0.1",
			8848,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
	}

	// 创建服务发现客户端
	// 创建服务发现客户端的另一种方式 (推荐)
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}
	return namingClient, nil
}

func RegService(namingClient naming_client.INamingClient, serviceName string, host string, port int) error {
	_, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "shanghai"},
		// ClusterName: "cluster-a", // 默认值DEFAULT
		// GroupName:   "group-a",   // 默认值DEFAULT_GROUP
	})

	return err
}

func GetInstance(namingClient naming_client.INamingClient, serviceName string) (string, uint64, error) {
	// SelectOneHealthyInstance将会按加权随机轮询的负载均衡策略返回一个健康的实例
	// 实例必须满足的条件：health=true,enable=true and weight>0
	instance, err := namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		// GroupName:   "group-a",             // 默认值DEFAULT_GROUP
		// Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
	})
	if err != nil {
		return "", uint64(0), err
	}
	return instance.Ip, instance.Port, nil
}

// CreateCli(option Option)  error
//	RegisterService(serviceName string, host string, port int) error
//	GetValue(serviceName string) (string, error)
//	Close() error

type GrNacosRegister struct {
	cli naming_client.INamingClient
}

func (r *GrNacosRegister) CreateCli(option Option) error {
	// 创建clientConfig的另一种方式
	// clientConfig := *constant.NewClientConfig(
	//	constant.WithNamespaceId(""), //当namespace是public时，此处填空字符串。
	//	constant.WithTimeoutMs(5000),
	//	constant.WithNotLoadCacheAtStart(true),
	//	constant.WithLogDir("/tmp/nacos/log"),
	//	constant.WithCacheDir("/tmp/nacos/cache"),
	//	constant.WithLogLevel("debug"),
	// )

	// 创建服务发现客户端
	// 创建服务发现客户端的另一种方式 (推荐)
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  option.NacosClientConfig,
			ServerConfigs: option.NacosServerConfig,
		},
	)
	if err != nil {
		return err
	}
	r.cli = namingClient
	return nil
}

func (r *GrNacosRegister) RegisterService(serviceName string, host string, port int) error {
	_, err := r.cli.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "shanghai"},
		// ClusterName: "cluster-a", // 默认值DEFAULT
		// GroupName:   "group-a",   // 默认值DEFAULT_GROUP
	})
	return err
}
func (r *GrNacosRegister) GetValue(serviceName string) (string, error) {
	instance, err := r.cli.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		// GroupName:   "group-a",             // 默认值DEFAULT_GROUP
		// Clusters:    []string{"cluster-a"}, // 默认值DEFAULT
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", instance.Ip, instance.Port), nil
}

func (r *GrNacosRegister) Close() error {
	return nil
}
