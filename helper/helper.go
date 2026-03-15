package helper

import (
	"context"

	"github.com/temp/breadcrumb"
)

// RecordGlobalBreadcrumb is a convenience wrapper that records a global trace event
// only when breadcrumb tracing is enabled on the context.
func RecordGlobalBreadcrumb(ctx context.Context, stage string, action string, meta map[string]interface{}) {
	if breadcrumb.IsBreadcrumbTracingEnabled(ctx) {
		breadcrumb.RecordGlobal(ctx, stage, action, meta)
	}
}

// RecordExternalRequestBreadcrumb records a global trace event representing an
// outbound request to an external service.
func RecordExternalRequestBreadcrumb(ctx context.Context, serviceName string, endpoint string, request any) {
	if breadcrumb.IsBreadcrumbTracingEnabled(ctx) {
		meta := make(map[string]interface{})
		meta["request"] = request
		breadcrumb.RecordGlobal(ctx, "EXTERNAL_REQUEST"+"-"+serviceName, endpoint, meta)
	}
}
