package tracing

import "google.golang.org/grpc/metadata"

// MetadataCarrier адаптирует gRPC metadata для реализации propagation.TextMapCarrier
type MetadataCarrier struct {
	metadata.MD
}

// Get возвращает значение по ключу из metadata
func (mc MetadataCarrier) Get(key string) string {
	values := mc.MD.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set устанавливает значение по ключу в metadata
func (mc MetadataCarrier) Set(key, value string) {
	mc.MD.Set(key, value)
}

// Keys возвращает все ключи из metadata
func (mc MetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.MD))
	for k := range mc.MD {
		keys = append(keys, k)
	}
	return keys
}
