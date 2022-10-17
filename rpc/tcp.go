package rpc

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
	"reflect"
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

type Serializer interface {
	Serialize(data any) ([]byte, error)
	Deserialize(data []byte, target any) error
}

// Gob协议
type GobSerializer struct{}

/**
 * Serialize
 * @Author：Jack-Z
 * @Description: 序列化
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
 * @Description: 反序列化
 * @receiver c
 * @param data
 * @param target
 * @return error
 */
func (c GobSerializer) Deserialize(data []byte, target any) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	return decoder.Decode(target)
}

type SerializeType byte

const (
	Gob SerializeType = iota
	ProtoBuff
)

// 解压缩接口
type CompressInterface interface {
	Compress([]byte) ([]byte, error)
	UnCompress([]byte) ([]byte, error)
}

// 解压缩类型
type CompressType byte

const (
	Gzip CompressType = iota
)

type GzipCompress struct {
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
	// 从reader中读取数据
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const (
	MagicNumber byte = 0x1d // 魔法数
	Version          = 0x01 // 版本
)

type MessageType byte

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
	SerializeType SerializeType
	RequestId     int64
}

type G2RpcMessage struct {
	Header *Header // 头
	Data   any     // 消息体
}

type G2RpcRequest struct {
	RequestId   int64
	ServiceName string // 服务名
	MethodName  string // 方法名
	Args        []any  // 请求参数
}

type G2RpcResponse struct {
	RequestId     int64
	Code          int16
	Msg           string
	CompressType  CompressType  // 压缩类型
	SerializeType SerializeType // 序列化类型
	Data          any           // 返回的数据
}

type G2RpcServer interface {
	Register(name string, service interface{}) // 注册服务
	Run()                                      // 运行服务
	Stop()                                     // 暂停服务
}

type G2TcpServer struct {
	listen     net.Listener
	serviceMap map[string]any
}

func NewTcpServer(addr string) (*G2TcpServer, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	m := &G2TcpServer{
		serviceMap: make(map[string]any),
	}
	m.listen = listen
	return m, nil
}

/**
 * Register
 * @Author：Jack-Z
 * @Description: 注册服务
 * @receiver s
 * @param name
 * @param service
 */
func (s *G2TcpServer) Register(name string, service interface{}) {
	t := reflect.TypeOf(service)
	if t.Kind() != reflect.Pointer {
		panic("service must be pointer")
	}
	s.serviceMap[name] = service
}

type G2TcpConn struct {
	conn    net.Conn
	rspChan chan *G2RpcResponse
}

/**
 * Send
 * @Author：Jack-Z
 * @Description: 服务端发送数据
 * @receiver c
 * @param rsp
 * @return error
 */
func (c G2TcpConn) Send(rsp *G2RpcResponse) error {
	if rsp.Code != 200 {
		// 进行默认的数据发送
	}
	headers := make([]byte, 17)
	headers[0] = MagicNumber // magic number
	headers[1] = Version     // version
	// full length
	headers[6] = byte(msgResponse)                                 // 消息类型
	headers[7] = byte(rsp.CompressType)                            // 压缩类型
	headers[8] = byte(rsp.SerializeType)                           // 序列化
	binary.BigEndian.PutUint64(headers[9:], uint64(rsp.RequestId)) // 请求id

	// 	先序列化
	se := loadSerializer(rsp.SerializeType)
	body, err := se.Serialize(rsp.Data)
	if err != nil {
		return err
	}
	com := loadCompression(rsp.CompressType)
	body, err = com.Compress(body)
	if err != nil {
		return err
	}
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
 * Run
 * @Author：Jack-Z
 * @Description: 运行服务
 * @receiver s
 */
func (s *G2TcpServer) Run() {
	for {
		conn, err := s.listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		g2Conn := &G2TcpConn{
			conn:    conn,
			rspChan: make(chan *G2RpcResponse, 1),
		}
		// 一直接收数据
		go s.readHandler(g2Conn)
		go s.writeHandler(g2Conn)
	}
}

/**
 * Stop
 * @Author：Jack-Z
 * @Description: 终止服务
 * @receiver s
 */
func (s *G2TcpServer) Stop() {
	err := s.listen.Close()
	if err != nil {
		log.Println(err)
	}
}

/**
 * readHandler
 * @Author：Jack-Z
 * @Description: 接收并读取数据
 * @receiver s
 * @param conn
 */
func (s *G2TcpServer) readHandler(conn *G2TcpConn) {
	// 解码
	msg, err := s.decodeFrame(conn)
	if err != nil {
		rsp := &G2RpcResponse{}
		rsp.Code = 500
		rsp.Msg = err.Error()
		conn.rspChan <- rsp
		return
	}

	if msg.Header.MessageType == msgRequest {
		req := msg.Data.(*G2RpcRequest)
		rsp := &G2RpcResponse{RequestId: req.RequestId}
		rsp.SerializeType = msg.Header.SerializeType
		rsp.CompressType = msg.Header.CompressType

		serviceName := req.ServiceName
		service, ok := s.serviceMap[serviceName]
		if !ok {
			rsp := &G2RpcResponse{}
			rsp.Code = 500
			rsp.Msg = errors.New("no service found").Error()
			conn.rspChan <- rsp
			return
		}

		methodName := req.MethodName
		method := reflect.ValueOf(service).MethodByName(methodName)
		if method.IsNil() {
			rsp := &G2RpcResponse{}
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

/**
 * writeHandler
 * @Author：Jack-Z
 * @Description: 写入数据发送
 * @receiver s
 * @param conn
 */
func (s *G2TcpServer) writeHandler(conn *G2TcpConn) {
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
 * decodeFrame
 * @Author：Jack-Z
 * @Description: 编码
 * @receiver s
 * @param conn
 * @return *G2RpcMessage
 * @return error
 */
func (s *G2TcpServer) decodeFrame(conn *G2TcpConn) (*G2RpcMessage, error) {
	header := make([]byte, 17)
	_, err := io.ReadFull(conn.conn, header)
	if err != nil {
		return nil, err
	}

	mn := header[0]
	if mn != MagicNumber {
		return nil, errors.New("magic number error")
	}

	vs := header[1]

	fullLength := int32(binary.BigEndian.Uint32(header[2:6]))
	msgType := header[6]                                    // 消息类型
	cprType := header[7]                                    // 压缩类型
	seType := header[8]                                     // 序列化类型
	requestId := int64(binary.BigEndian.Uint32(header[9:])) // 请求id

	msg := &G2RpcMessage{}
	msg.Header.MagicNumber = mn
	msg.Header.Version = vs
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(msgType)
	msg.Header.CompressType = CompressType(cprType)
	msg.Header.SerializeType = SerializeType(seType)
	msg.Header.RequestId = requestId

	// 请求体
	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(conn.conn, body)
	if err != nil {
		return nil, err
	}
	compress := loadCompression(CompressType(cprType))
	if compress == nil {
		return nil, errors.New("no compress")
	}

	body, err = compress.UnCompress(body)
	if err == nil {
		return nil, err
	}

	serialize := loadSerializer(SerializeType(seType))
	if serialize == nil {
		return nil, errors.New("no serializer")
	}

	if MessageType(msgType) == msgRequest {
		req := &G2RpcRequest{}
		err := serialize.Deserialize(body, req)
		if err != nil {
			return nil, err
		}
		msg.Data = req
		return msg, nil
	}

	if MessageType(msgType) == msgResponse {
		rsp := &G2RpcResponse{}
		err := serialize.Deserialize(body, rsp)
		if err != nil {
			return nil, err
		}
		msg.Data = rsp
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
func loadSerializer(serializeType SerializeType) Serializer {
	switch serializeType {
	case Gob:
		return GobSerializer{}
	case ProtoBuff:
		return GobSerializer{}
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
func loadCompression(compressType CompressType) CompressInterface {
	switch compressType {
	case Gzip:
		return GzipCompress{}
	}
	return nil
}
