package rpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

type G2rpcServer struct {
	listen     net.Listener                    // 监听端口
	grpcServer *grpc.Server                    // grpc服务
	registers  []func(grpcServer *grpc.Server) // 注册服务
	ops        []grpc.ServerOption
}

/**
 * NewG2rpcServer
 * @Author：Jack-Z
 * @Description: 创建服务端连接
 * @param address
 * @param ops
 * @return *G2rpcServer
 * @return error
 */
func NewG2rpcServer(address string, ops ...G2rpcOption) (*G2rpcServer, error) {
	listen, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	ms := &G2rpcServer{
		listen: listen,
	}
	for _, op := range ops {
		op.Apply(ms)
	}
	s := grpc.NewServer(ms.ops...)
	ms.grpcServer = s
	return ms, nil
}

/**
 * Run
 * @Author：Jack-Z
 * @Description: 启动服务端
 * @receiver s
 * @return error
 */
func (s *G2rpcServer) Run() error {
	for _, register := range s.registers {
		register(s.grpcServer)
	}
	return s.grpcServer.Serve(s.listen)
}

/**
 * Stop
 * @Author：Jack-Z
 * @Description: 终止服务端
 * @receiver s
 */
func (s *G2rpcServer) Stop() {
	s.grpcServer.Stop()
}

/**
 * Register
 * @Author：Jack-Z
 * @Description: 注册服务
 * @receiver s
 * @param register
 */
func (s *G2rpcServer) Register(register func(grpServer *grpc.Server)) {
	s.registers = append(s.registers, register)
}

type G2rpcOption interface {
	Apply(s *G2rpcServer)
}

type DefaultG2rpcOption struct {
	f func(s *G2rpcServer)
}

func (d DefaultG2rpcOption) Apply(s *G2rpcServer) {
	d.f(s)
}

func WithG2rpcOptions(options ...grpc.ServerOption) G2rpcOption {
	return DefaultG2rpcOption{f: func(s *G2rpcServer) {
		s.ops = append(s.ops, options...)
	}}
}

/**
 * G2rpcClient
 *  @Description: 客户端
 */
type G2rpcClient struct {
	Conn *grpc.ClientConn
}

/**
 * NewG2rpcClient
 * @Author：Jack-Z
 * @Description: 创建客户端连接
 * @param config
 * @return *G2rpcClient
 * @return error
 */
func NewG2rpcClient(config *G2rpcClientConfig) (*G2rpcClient, error) {
	var ctx = context.Background()
	var dialOptions = config.dialOptions

	if config.Block {
		// 阻塞
		if config.DialTimeout > time.Duration(0) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.DialTimeout)
			defer cancel()
		}
		dialOptions = append(dialOptions, grpc.WithBlock())
	}
	if config.KeepAlive != nil {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*config.KeepAlive))
	}
	conn, err := grpc.DialContext(ctx, config.Address, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &G2rpcClient{
		Conn: conn,
	}, nil
}

/**
 * G2rpcClientConfig
 *  @Description: 客户端连接配置
 */
type G2rpcClientConfig struct {
	Address     string        // 客户端连接地址
	Block       bool          // 是否阻塞
	DialTimeout time.Duration // 连接超时时间
	ReadTimeout time.Duration // 读取超时时间
	Direct      bool
	KeepAlive   *keepalive.ClientParameters
	dialOptions []grpc.DialOption
}

/**
 * DefaultG2rpcClientConfig
 * @Author：Jack-Z
 * @Description: 默认客户端连接配置
 * @return *G2rpcClientConfig
 */
func DefaultG2rpcClientConfig() *G2rpcClientConfig {
	return &G2rpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		DialTimeout: time.Second * 3,
		ReadTimeout: time.Second * 2,
		Block:       true,
	}
}
