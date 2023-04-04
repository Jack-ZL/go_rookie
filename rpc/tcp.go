package rpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jack-ZL/go_rookie/register"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"io"
	"log"
	"net"
	"reflect"
	"sync/atomic"
	"time"
)

/**
客户端：
1、连接服务端
2、发送请求数据（编码）二进制数据，通过网络发送
3、等待回复，接收到响应（解码）
*/

/** 服务端
1、启动服务
2、接收请求（解码），根据请求调用对应的服务，得到响应数据
3、将响应数据发送给客户端（编码）
*/

/**
 * Serializer
 * @Description: 序列化接口定义
 */
type Serializer interface {
	Serialize(data any) ([]byte, error)
	DeSerialize(data []byte, target any) error
}

/**
 * CompressInterface
 * @Description: 解压缩接口
 */
type CompressInterface interface {
	Compress([]byte) ([]byte, error)
	UnCompress([]byte) ([]byte, error)
}

type SerializerType byte
type CompressType byte

const (
	Gzip CompressType = iota
)

const (
	Gob SerializerType = iota
	ProtoBuff
)
const (
	MagicNumber byte = 0x1d // 魔法数
	Version          = 0x01 // 版本
)

type MessageType byte

// Gob协议
type GobSerializer struct{}
type ProtobufSerializer struct{}
type GzipCompress struct{}

/**
 * Serialize
 * @Author：Jack-Z
 * @Description: Gob协议-序列化
 * @receiver c
 * @param data
 * @return []byte
 * @return error
 */
func (c GobSerializer) Serialize(data any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

/**
 * Deserialize
 * @Author：Jack-Z
 * @Description: Gob协议-反序列化
 * @receiver c
 * @param data
 * @param target
 * @return error
 */
func (c GobSerializer) DeSerialize(data []byte, target any) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	return decoder.Decode(target)
}

/**
 * Serialize
 * @Author：Jack-Z
 * @Description:ProtoBuff协议-序列化
 * @receiver c
 * @param data
 * @return []byte
 * @return error
 */
func (c ProtobufSerializer) Serialize(data any) ([]byte, error) {
	marshal, err := proto.Marshal(data.(proto.Message))
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

/**
 * DeSerialize
 * @Author：Jack-Z
 * @Description: ProtoBuff协议-反序列化
 * @receiver c
 * @param data
 * @param target
 * @return error
 */
func (c ProtobufSerializer) DeSerialize(data []byte, target any) error {
	message := target.(proto.Message)
	return proto.Unmarshal(data, message)
}

/**
 * Compress
 * @Author：Jack-Z
 * @Description: 压缩数据
 * @receiver c
 * @param data
 * @return []byte
 * @return error
 */
func (c GzipCompress) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

/**
 * UnCompress
 * @Author：Jack-Z
 * @Description: 将数据解压
 * @receiver c
 * @param data
 * @return []byte
 * @return error
 */
func (c GzipCompress) UnCompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	// 从 Reader 中读取出数据
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const (
	msgRequest MessageType = iota
	msgResponse
	msgPing
	msgPong
)

type Header struct {
	MagicNumber   byte
	Version       byte
	FullLength    int32
	MessageType   MessageType
	CompressType  CompressType
	SerializeType SerializerType
	RequestId     int64
}

type GrRpcMessage struct {
	Header *Header // 头
	Data   any     // 消息体
}

type GrRpcRequest struct {
	RequestId   int64
	ServiceName string // 服务名
	MethodName  string // 方法名
	Args        []any  // 请求参数
}

type GrRpcResponse struct {
	RequestId     int64
	Code          int16
	Msg           string
	CompressType  CompressType   // 压缩类型
	SerializeType SerializerType // 序列化类型
	Data          any            // 返回的数据
}

type G2RpcServer interface {
	Register(name string, service interface{}) // 注册服务
	Run()                                      // 运行服务
	Stop()                                     // 暂停服务
}

type GrTcpServer struct {
	host           string
	port           int
	listen         net.Listener
	serviceMap     map[string]any
	RegisterType   string //注册类型：nacos或etcd
	RegisterOption register.Option
	RegisterCli    register.GrRegister
	LimiterTimeOut time.Duration // 限流超时时间
	Limiter        *rate.Limiter // 限流器
}

func NewTcpServer(host string, port int) (*GrTcpServer, error) {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	m := &GrTcpServer{serviceMap: make(map[string]any)}
	m.listen = listen
	m.port = port
	m.host = host
	return m, nil
}

func (s *GrTcpServer) SetLimiter(limit, cap int) {
	s.Limiter = rate.NewLimiter(rate.Limit(limit), cap)
}

/**
 * Register
 * @Author：Jack-Z
 * @Description: 注册服务
 * @receiver s
 * @param name
 * @param service
 */
func (s *GrTcpServer) Register(name string, service interface{}) {
	t := reflect.TypeOf(service)
	if t.Kind() != reflect.Pointer {
		panic("service must be pointer")
	}
	s.serviceMap[name] = service

	err := s.RegisterCli.CreateCli(s.RegisterOption)
	if err != nil {
		panic(err)
	}
	err = s.RegisterCli.RegisterService(name, s.host, s.port)
	if err != nil {
		panic(err)
	}
}

type GrTcpConn struct {
	conn    net.Conn
	rspChan chan *GrRpcResponse
}

/**
 * Send
 * @Author：Jack-Z
 * @Description: 服务端发送数据
 * @receiver c
 * @param rsp
 * @return error
 */
func (c GrTcpConn) Send(rsp *GrRpcResponse) error {
	if rsp.Code != 200 {
		// 进行默认的数据发送
	}
	// 编码 发送出去
	headers := make([]byte, 17)
	// magic number
	headers[0] = MagicNumber
	// version
	headers[1] = Version
	// full length
	// 消息类型
	headers[6] = byte(msgResponse)
	// 压缩类型
	headers[7] = byte(rsp.CompressType)
	// 序列化
	headers[8] = byte(rsp.SerializeType)
	// 请求id
	binary.BigEndian.PutUint64(headers[9:], uint64(rsp.RequestId))
	// 编码 先序列化 在压缩
	se := loadSerializer(rsp.SerializeType)
	var body []byte
	var err error
	if rsp.SerializeType == ProtoBuff {
		pRsp := &Response{}
		pRsp.SerializeType = int32(rsp.SerializeType)
		pRsp.CompressType = int32(rsp.CompressType)
		pRsp.Code = int32(rsp.Code)
		pRsp.Msg = rsp.Msg
		pRsp.RequestId = rsp.RequestId
		// value, err := structpb.
		//	log.Println(err)
		m := make(map[string]any)
		marshal, _ := json.Marshal(rsp.Data)
		_ = json.Unmarshal(marshal, &m)
		value, err := structpb.NewStruct(m)
		log.Println(err)
		pRsp.Data = structpb.NewStructValue(value)
		body, err = se.Serialize(pRsp)
	} else {
		body, err = se.Serialize(rsp)
	}
	if err != nil {
		return err
	}
	com := loadCompress(rsp.CompressType)
	body, err = com.Compress(body)
	if err != nil {
		return err
	}
	fullLen := 17 + len(body)
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen))

	_, err = c.conn.Write(headers[:])
	if err != nil {
		return err
	}
	_, err = c.conn.Write(body[:])
	if err != nil {
		return err
	}
	return nil
}

/**
 * Stop
 * @Author：Jack-Z
 * @Description: 终止服务
 * @receiver s
 */
func (s *GrTcpServer) Stop() {
	err := s.listen.Close()
	if err != nil {
		log.Println(err)
	}
}

/**
 * Run
 * @Author：Jack-Z
 * @Description: 运行服务
 * @receiver s
 */
func (s *GrTcpServer) Run() {
	for {
		conn, err := s.listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		msConn := &GrTcpConn{conn: conn, rspChan: make(chan *GrRpcResponse, 1)}
		// 1. 一直接收数据 解码工作 请求业务获取结果 发送到rspChan
		// 2. 获得结果 编码 发送数据
		go s.readHandle(msConn)
		go s.writeHandle(msConn)
	}
}

func (s *GrTcpServer) readHandle(conn *GrTcpConn) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("readHandle recover ", err)
			conn.conn.Close()
		}
	}()

	// 在这加一个限流
	ctx, cancel := context.WithTimeout(context.Background(), s.LimiterTimeOut)
	defer cancel()
	err2 := s.Limiter.WaitN(ctx, 1)
	if err2 != nil {
		rsp := &GrRpcResponse{}
		rsp.Code = 700 // 被限流的错误
		rsp.Msg = err2.Error()
		conn.rspChan <- rsp
		return
	}

	// 接收数据
	// 解码
	msg, err := decodeFrame(conn.conn)
	if err != nil {
		rsp := &GrRpcResponse{}
		rsp.Code = 500
		rsp.Msg = err.Error()
		conn.rspChan <- rsp
		return
	}
	if msg.Header.MessageType == msgRequest {
		if msg.Header.SerializeType == ProtoBuff {
			req := msg.Data.(*Request)
			rsp := &GrRpcResponse{RequestId: req.RequestId}
			rsp.SerializeType = msg.Header.SerializeType
			rsp.CompressType = msg.Header.CompressType
			serviceName := req.ServiceName
			service, ok := s.serviceMap[serviceName]
			if !ok {
				rsp := &GrRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service found").Error()
				conn.rspChan <- rsp
				return
			}
			methodName := req.MethodName
			method := reflect.ValueOf(service).MethodByName(methodName)
			if method.IsNil() {
				rsp := &GrRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service method found").Error()
				conn.rspChan <- rsp
				return
			}
			// 调用方法
			args := make([]reflect.Value, len(req.Args))
			for i := range req.Args {
				of := reflect.ValueOf(req.Args[i].AsInterface())
				of = of.Convert(method.Type().In(i))
				args[i] = of
			}
			result := method.Call(args)

			results := make([]any, len(result))
			for i, v := range result {
				results[i] = v.Interface()
			}
			err, ok := results[len(result)-1].(error)
			if ok {
				rsp.Code = 500
				rsp.Msg = err.Error()
				conn.rspChan <- rsp
				return
			}
			rsp.Code = 200
			rsp.Data = results[0]
			conn.rspChan <- rsp
		} else {
			req := msg.Data.(*GrRpcRequest)
			rsp := &GrRpcResponse{RequestId: req.RequestId}
			rsp.SerializeType = msg.Header.SerializeType
			rsp.CompressType = msg.Header.CompressType
			serviceName := req.ServiceName
			service, ok := s.serviceMap[serviceName]
			if !ok {
				rsp := &GrRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service found").Error()
				conn.rspChan <- rsp
				return
			}
			methodName := req.MethodName
			method := reflect.ValueOf(service).MethodByName(methodName)
			if method.IsNil() {
				rsp := &GrRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service method found").Error()
				conn.rspChan <- rsp
				return
			}
			// 调用方法
			args := req.Args
			var valuesArg []reflect.Value
			for _, v := range args {
				valuesArg = append(valuesArg, reflect.ValueOf(v))
			}
			result := method.Call(valuesArg)

			results := make([]any, len(result))
			for i, v := range result {
				results[i] = v.Interface()
			}
			err, ok := results[len(result)-1].(error)
			if ok {
				rsp.Code = 500
				rsp.Msg = err.Error()
				conn.rspChan <- rsp
				return
			}
			rsp.Code = 200
			rsp.Data = results[0]
			conn.rspChan <- rsp
		}
	}
}

/**
 * writeHandler
 * @Author：Jack-Z
 * @Description: 写入数据发送
 * @receiver s
 * @param conn
 */
func (s *GrTcpServer) writeHandle(conn *GrTcpConn) {
	select {
	case rsp := <-conn.rspChan:
		defer conn.conn.Close()
		// 发送数据
		err := conn.Send(rsp)
		if err != nil {
			log.Println(err)
		}

	}
}

/**
 * SetRegister
 * @Author：Jack-Z
 * @Description: 设置注册类型和option
 * @receiver s
 * @param registerType
 * @param option
 */
func (s *GrTcpServer) SetRegister(registerType string, option register.Option) {
	s.RegisterType = registerType
	s.RegisterOption = option
	if registerType == "nacos" {
		s.RegisterCli = &register.GrNacosRegister{}
	}
	if registerType == "etcd" {
		s.RegisterCli = &register.GrEtcdRegister{}
	}
}

func decodeFrame(conn net.Conn) (*GrRpcMessage, error) {
	// 1+1+4+1+1+1+8=17
	headers := make([]byte, 17)
	_, err := io.ReadFull(conn, headers)
	if err != nil {
		return nil, err
	}
	mn := headers[0]
	if mn != MagicNumber {
		return nil, errors.New("magic number error")
	}
	// version
	vs := headers[1]
	// full length
	// 网络传输 大端
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6]))
	// messageType
	messageType := headers[6]
	// 压缩类型
	compressType := headers[7]
	// 序列化类型
	seType := headers[8]
	// 请求id
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))

	msg := &GrRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn
	msg.Header.Version = vs
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(messageType)
	msg.Header.CompressType = CompressType(compressType)
	msg.Header.SerializeType = SerializerType(seType)
	msg.Header.RequestId = requestId

	// body
	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}
	// 编码的 先序列化 后 压缩
	// 解码的时候 先解压缩，反序列化
	compress := loadCompress(CompressType(compressType))
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.UnCompress(body)
	if compress == nil {
		return nil, err
	}
	serializer := loadSerializer(SerializerType(seType))
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	if MessageType(messageType) == msgRequest {
		if SerializerType(seType) == ProtoBuff {
			req := &Request{}
			err := serializer.DeSerialize(body, req)
			if err != nil {
				return nil, err
			}
			msg.Data = req
		} else {
			req := &GrRpcRequest{}
			err := serializer.DeSerialize(body, req)
			if err != nil {
				return nil, err
			}
			msg.Data = req
		}
		return msg, nil
	}
	if MessageType(messageType) == msgResponse {
		if SerializerType(seType) == ProtoBuff {
			rsp := &Response{}
			err := serializer.DeSerialize(body, rsp)
			if err != nil {
				return nil, err
			}
			msg.Data = rsp
		} else {
			rsp := &GrRpcResponse{}
			err := serializer.DeSerialize(body, rsp)
			if err != nil {
				return nil, err
			}
			msg.Data = rsp
		}

		return msg, nil
	}
	return nil, errors.New("no message type")
}

/**
 * loadSerializer
 * @Author：Jack-Z
 * @Description: 加载序列化类型
 * @param serializeType
 * @return Serializer
 */
func loadSerializer(serializerType SerializerType) Serializer {
	switch serializerType {
	case Gob:
		return GobSerializer{}
	case ProtoBuff:
		return ProtobufSerializer{}
	}
	return nil
}

/**
 * loadCompression
 * @Author：Jack-Z
 * @Description: 加载压缩类型
 * @param compressType
 * @return CompressInterface
 */
func loadCompress(compressType CompressType) CompressInterface {
	switch compressType {
	case Gzip:
		return GzipCompress{}
	}
	return nil
}

type GrRpcClient interface {
	Connect() error
	Invoke(context context.Context, serviceName string, methodName string, args []any) (any, error)
	Close() error
}

type GrTcpClient struct {
	conn        net.Conn
	option      TcpClientOption
	ServiceName string
	RegisterCli register.GrRegister
}
type TcpClientOption struct {
	Retries           int
	ConnectionTimeout time.Duration
	SerializeType     SerializerType
	CompressType      CompressType
	Host              string
	Port              int
	RegisterType      string
	RegisterOption    register.Option
	RegisterCli       register.GrRegister
}

var DefaultOption = TcpClientOption{
	Host:              "127.0.0.1",
	Port:              9222,
	Retries:           3,
	ConnectionTimeout: 5 * time.Second,
	SerializeType:     Gob,
	CompressType:      Gzip,
}

func NewTcpClient(option TcpClientOption) *GrTcpClient {
	return &GrTcpClient{option: option}
}

/**
 * Connect
 * @Author：Jack-Z
 * @Description: tcp客户端连接
 * @receiver c
 * @return error
 */
func (c *GrTcpClient) Connect() error {
	var addr string
	err := c.RegisterCli.CreateCli(c.option.RegisterOption)
	if err != nil {
		panic(err)
	}
	addr, err = c.RegisterCli.GetValue(c.ServiceName)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialTimeout("tcp", addr, c.option.ConnectionTimeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *GrTcpClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

var reqId int64

func (c *GrTcpClient) Invoke(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	// 包装 request对象 编码 发送即可
	req := &GrRpcRequest{}
	req.RequestId = atomic.AddInt64(&reqId, 1)
	req.ServiceName = serviceName
	req.MethodName = methodName
	req.Args = args

	headers := make([]byte, 17)
	// magic number
	headers[0] = MagicNumber
	// version
	headers[1] = Version
	// full length
	// 消息类型
	headers[6] = byte(msgRequest)
	// 压缩类型
	headers[7] = byte(c.option.CompressType)
	// 序列化
	headers[8] = byte(c.option.SerializeType)
	// 请求id
	binary.BigEndian.PutUint64(headers[9:], uint64(req.RequestId))

	serializer := loadSerializer(c.option.SerializeType)
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	var body []byte
	var err error
	if c.option.SerializeType == ProtoBuff {
		pReq := &Request{}
		pReq.RequestId = atomic.AddInt64(&reqId, 1)
		pReq.ServiceName = serviceName
		pReq.MethodName = methodName
		listValue, err := structpb.NewList(args)
		if err != nil {
			return nil, err
		}
		pReq.Args = listValue.Values
		body, err = serializer.Serialize(pReq)
	} else {
		body, err = serializer.Serialize(req)
	}

	if err != nil {
		return nil, err
	}
	compress := loadCompress(c.option.CompressType)
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.Compress(body)
	if err != nil {
		return nil, err
	}
	fullLen := 17 + len(body)
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen))

	_, err = c.conn.Write(headers[:])
	if err != nil {
		return nil, err
	}

	_, err = c.conn.Write(body[:])
	if err != nil {
		return nil, err
	}
	rspChan := make(chan *GrRpcResponse)
	go c.readHandle(rspChan)
	rsp := <-rspChan
	return rsp, nil
}

/**
 * readHandle
 * @Author：Jack-Z
 * @Description: 客户端读取数据
 * @receiver c
 * @param rspChan
 */
func (c *GrTcpClient) readHandle(rspChan chan *GrRpcResponse) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("GrTcpClient readHandle recover: ", err)
			c.conn.Close()
		}
	}()
	for {
		msg, err := decodeFrame(c.conn)
		if err != nil {
			log.Println("未解析出任何数据")
			rsp := &GrRpcResponse{}
			rsp.Code = 500
			rsp.Msg = err.Error()
			rspChan <- rsp
			return
		}
		// 根据请求
		if msg.Header.MessageType == msgResponse {
			if msg.Header.SerializeType == ProtoBuff {
				rsp := msg.Data.(*Response)
				asInterface := rsp.Data.AsInterface()
				marshal, _ := json.Marshal(asInterface)
				rsp1 := &GrRpcResponse{}
				json.Unmarshal(marshal, rsp1)
				rspChan <- rsp1
			} else {
				rsp := msg.Data.(*GrRpcResponse)
				rspChan <- rsp
			}
			return
		}
	}
}

func (c *GrTcpClient) decodeFrame(conn net.Conn) (*GrRpcMessage, error) {
	// 1+1+4+1+1+1+8=17
	headers := make([]byte, 17)
	_, err := io.ReadFull(conn, headers)
	if err != nil {
		return nil, err
	}
	mn := headers[0]
	if mn != MagicNumber {
		return nil, errors.New("magic number error")
	}
	// version
	vs := headers[1]
	// full length
	// 网络传输 大端
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6]))
	// messageType
	messageType := headers[6]
	// 压缩类型
	compressType := headers[7]
	// 序列化类型
	seType := headers[8]
	// 请求id
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))

	msg := &GrRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn
	msg.Header.Version = vs
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(messageType)
	msg.Header.CompressType = CompressType(compressType)
	msg.Header.SerializeType = SerializerType(seType)
	msg.Header.RequestId = requestId

	// body
	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}
	// 编码的 先序列化 后 压缩
	// 解码的时候 先解压缩，反序列化
	compress := loadCompress(CompressType(compressType))
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.UnCompress(body)
	if compress == nil {
		return nil, err
	}
	serializer := loadSerializer(SerializerType(seType))
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	if MessageType(messageType) == msgRequest {
		req := &GrRpcRequest{}
		err := serializer.DeSerialize(body, req)
		if err != nil {
			return nil, err
		}
		msg.Data = req
		return msg, nil
	}
	if MessageType(messageType) == msgResponse {
		rsp := &GrRpcResponse{}
		err := serializer.DeSerialize(body, rsp)
		if err != nil {
			return nil, err
		}
		msg.Data = rsp
		return msg, nil
	}
	return nil, errors.New("no message type")
}

type GrTcpClientProxy struct {
	client *GrTcpClient
	option TcpClientOption
}

func NewGrTcpClientProxy(option TcpClientOption) *GrTcpClientProxy {
	return &GrTcpClientProxy{option: option}
}

func (p *GrTcpClientProxy) Call(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	client := NewTcpClient(p.option)
	client.ServiceName = serviceName
	if p.option.RegisterType == "nacos" {
		client.RegisterCli = &register.GrNacosRegister{}
	}
	if p.option.RegisterType == "etcd" {
		client.RegisterCli = &register.GrEtcdRegister{}
	}
	p.client = client
	err := client.Connect()
	if err != nil {
		return nil, err
	}
	for i := 0; i < p.option.Retries; i++ {
		result, err := client.Invoke(ctx, serviceName, methodName, args)
		if err != nil {
			if i >= p.option.Retries-1 {
				log.Println(errors.New("already retry all time"))
				client.Close()
				return nil, err
			}
			// 睡眠一小会
			continue
		}
		client.Close()
		return result, nil
	}
	return nil, errors.New("retry time is 0")
}
