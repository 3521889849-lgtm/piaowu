package trace

import (
	"fmt"
	"io"
	"log"
	"strings"

	"example_shop/common/config"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func InitJaeger(serviceName string) (io.Closer, error) {
	if strings.TrimSpace(serviceName) == "" {
		return nil, fmt.Errorf("serviceName为空")
	}
	if config.Cfg == nil || !config.Cfg.Tracing.Enabled {
		return nil, nil
	}

	jcfg := config.Cfg.Tracing.Jaeger
	collectorEndpoint := strings.TrimSpace(strings.ReplaceAll(jcfg.CollectorEndpoint, "`", ""))
	agentHost := strings.TrimSpace(strings.ReplaceAll(jcfg.AgentHost, "`", ""))
	if collectorEndpoint == "" && agentHost == "" {
		log.Printf("tracing enabled but no jaeger endpoint set, service=%s", serviceName)
	} else {
		log.Printf("tracing enabled, service=%s collector=%s agent=%s:%d", serviceName, collectorEndpoint, agentHost, jcfg.AgentPort)
	}
	cfg := jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  strings.TrimSpace(jcfg.SamplerType),
			Param: jcfg.SamplerParam,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           jcfg.LogSpans,
			CollectorEndpoint:  collectorEndpoint,
			LocalAgentHostPort: strings.TrimSpace(fmt.Sprintf("%s:%d", agentHost, jcfg.AgentPort)),
		},
	}
	if agentHost == "" {
		cfg.Reporter.LocalAgentHostPort = ""
	}

	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}
