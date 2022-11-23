package register

import (
	"context"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func CreateEtcdCli(option Option) (*clientv3.Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   option.Endpoints,   // 节点
		DialTimeout: option.DialTimeout, // 超过5秒钟连不上超时
	})
	return cli, err
}

func RegEtcdService(cli *clientv3.Client, serviceName string, host string, port int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := cli.Put(ctx, serviceName, fmt.Sprintf("%s:%d", host, port))
	return err
}

func GetEtcdValue(cli *clientv3.Client, serviceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	v, err := cli.Get(ctx, serviceName)
	if err != nil {
		return "", err
	}
	kvs := v.Kvs
	if len(kvs) == 0 {
		return "", errors.New("no value")
	}
	return string(kvs[0].Value), err
}

type GrEtcdRegister struct {
	cli *clientv3.Client
}

func (r *GrEtcdRegister) CreateCli(option Option) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   option.Endpoints,   // 节点
		DialTimeout: option.DialTimeout, // 超过5秒钟连不上超时
	})
	r.cli = cli
	return err
}

/**
 * RegisterService
 * @Author：Jack-Z
 * @Description: 依据服务名称注册服务
 * @receiver r
 * @param serviceName
 * @param host
 * @param port
 * @return error
 */
func (r *GrEtcdRegister) RegisterService(serviceName string, host string, port int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := r.cli.Put(ctx, serviceName, fmt.Sprintf("%s:%d", host, port))
	return err
}

func (r *GrEtcdRegister) GetValue(serviceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	v, err := r.cli.Get(ctx, serviceName)
	if err != nil {
		return "", err
	}
	kvs := v.Kvs
	if len(kvs) == 0 {
		return "", errors.New("no value")
	}
	return string(kvs[0].Value), err
}

func (r *GrEtcdRegister) Close() error {
	return r.cli.Close()
}
