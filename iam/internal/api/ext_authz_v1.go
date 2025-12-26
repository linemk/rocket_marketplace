package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	statusv3 "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	"github.com/linemk/rocket-shop/iam/internal/service/auth"
)

const (
	SessionCookieName = "X-Session-Uuid"

	HeaderUserUUID    = "X-User-Uuid"
	HeaderUserLogin   = "X-User-Login"
	HeaderContentType = "content-type"
	HeaderAuthStatus  = "X-Auth-Status"

	HeaderCookie        = "cookie"
	HeaderAuthorization = "authorization"

	ContentTypeJSON = "application/json"

	AuthStatusDenied = "denied"
)

type extAuthzV1Handler struct {
	authService auth.Service
	authv3.UnimplementedAuthorizationServer
}

func NewExtAuthzV1Handler(authService auth.Service) authv3.AuthorizationServer {
	return &extAuthzV1Handler{
		authService: authService,
	}
}

func (h *extAuthzV1Handler) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	log.Printf("External Authorization Check called")

	sessionUUID, err := h.extractSessionUUID(req)
	if err != nil {
		log.Printf("Session extraction failed: %v", err)
		return h.denyRequest("Missing or invalid session", 403), nil
	}

	log.Printf("Extracted session_uuid: %s", sessionUUID)

	user, err := h.authService.Whoami(ctx, sessionUUID)
	if err != nil {
		log.Printf("Whoami failed: %v", err)
		return h.denyRequest("Invalid session", 403), nil
	}

	return h.allowRequest(user.UserUUID, user.Login), nil
}

func (h *extAuthzV1Handler) extractSessionUUID(req *authv3.CheckRequest) (string, error) {
	if req.Attributes == nil || req.Attributes.Request == nil {
		return "", fmt.Errorf("no HTTP request found")
	}

	headers := req.Attributes.Request.Http.Headers

	if cookieHeader, ok := headers[HeaderCookie]; ok && cookieHeader != "" {
		sessionUUID := h.extractSessionFromCookies(cookieHeader)
		if sessionUUID != "" {
			return sessionUUID, nil
		}
	}

	return "", fmt.Errorf("session uuid not found in cookies")
}

func (h *extAuthzV1Handler) extractSessionFromCookies(cookieHeader string) string {
	req := &http.Request{Header: make(http.Header)}
	req.Header.Add(HeaderCookie, cookieHeader)

	if cookie, err := req.Cookie(SessionCookieName); err == nil {
		var sessionUUID string
		sessionUUID, err = url.QueryUnescape(cookie.Value)
		if err != nil {
			return cookie.Value
		}

		return sessionUUID
	}

	return ""
}

func (h *extAuthzV1Handler) allowRequest(userUUID, userLogin string) *authv3.CheckResponse {
	headers := []*corev3.HeaderValueOption{
		{
			Header: &corev3.HeaderValue{
				Key:   HeaderUserUUID,
				Value: userUUID,
			},
		},
		{
			Header: &corev3.HeaderValue{
				Key:   HeaderUserLogin,
				Value: userLogin,
			},
		},
	}

	return &authv3.CheckResponse{
		Status: &statusv3.Status{Code: 0},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers:         headers,
				HeadersToRemove: []string{HeaderCookie, HeaderAuthorization},
			},
		},
	}
}

func (h *extAuthzV1Handler) denyRequest(message string, statusCode int32) *authv3.CheckResponse {
	return &authv3.CheckResponse{
		Status: &statusv3.Status{Code: int32(codes.Unauthenticated)},
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{
					Code: typev3.StatusCode(statusCode),
				},
				Body: fmt.Sprintf(`{"error": "%s"}`, message),
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   HeaderContentType,
							Value: ContentTypeJSON,
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   HeaderAuthStatus,
							Value: AuthStatusDenied,
						},
					},
				},
			},
		},
	}
}
