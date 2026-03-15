// Package stepmarkhttp provides [net/http] middleware for [stepmark].
//
// The middleware conditionally enables tracing based on a [TriggerFunc]
// (e.g. a request header or query parameter) and optionally writes the
// collected trace to a response header or passes it to a callback.
//
// Works out of the box with the standard library, Chi, and any router
// that accepts the func(http.Handler) http.Handler middleware signature.
//
// # Quick Start
//
//	mux.Handle("/api/", stepmarkhttp.Middleware(
//	    stepmarkhttp.HeaderTrigger("X-Stepmark"),
//	    stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
//	)(apiHandler))
//
// Send X-Stepmark: true with your request, and the response includes
// an X-Stepmark-Trace header containing the full JSON trace.
package stepmarkhttp
