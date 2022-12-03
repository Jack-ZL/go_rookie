package tracer

// jaeger：链路追踪

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"net/http"
)

/**
 * CreateTracer
 * @Author：Jack-Z
 * @Description: 创建一个追踪器
 * @param serviceName 服务名称
 * @param samplerConfig
 * @param reporter
 * @param options
 * @return opentracing.Tracer
 * @return io.Closer
 * @return error
 */
func CreateTracer(serviceName string, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) (opentracing.Tracer, io.Closer, error) {
	var cfg = config.Configuration{
		ServiceName: serviceName,
		Sampler:     samplerConfig, //采样器配置
		Reporter:    reporter,      //配置客户端如何上报追踪信息
	}
	tracer, closer, err := cfg.NewTracer(options...)
	return tracer, closer, err
}

/**
 * CreateTracerHeader
 * @Author：Jack-Z
 * @Description: 带有上下文解析的追踪器
 * @param serviceName
 * @param header
 * @param samplerConfig
 * @param reporter
 * @param options
 * @return opentracing.Tracer
 * @return io.Closer
 * @return opentracing.SpanContext
 * @return error
 */
func CreateTracerHeader(serviceName string, header http.Header, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) (opentracing.Tracer, io.Closer, opentracing.SpanContext, error) {
	var cfg = config.Configuration{
		ServiceName: serviceName,
		Sampler:     samplerConfig,
		Reporter:    reporter,
	}
	tracer, closer, err := cfg.NewTracer(options...)
	// 继承别的进程传递过来的上下文
	spanContext, _ := tracer.Extract(opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(header))

	return tracer, closer, spanContext, err
}
