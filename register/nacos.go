package register

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type GrNacosRegister struct {
	cli naming_client.INamingClient
}

/**
 * CreateCli
 * @Author：Jack-Z
 * @Description: 创建nacos客户端
 * @receiver r
 * @param option
 * @return error
 */
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

/**
 * RegisterService
 * @Author：Jack-Z
 * @Description: 注册服务
 * @receiver r
 * @param serviceName
 * @param host
 * @param port
 * @return error
 */
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

/**
 * GetValue
 * @Author：Jack-Z
 * @Description: 通过服务名称获取一个实例
 * @receiver r
 * @param serviceName
 * @return string
 * @return error
 */
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
