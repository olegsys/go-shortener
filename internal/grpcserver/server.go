// Package grpcserver реализует gRPC-адаптер к бизнес-логике сервиса сокращения URL
package grpcserver

import (
	"context"
	"net/url"
	"path"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	shortenerpb "github.com/olegsys/go-shortener/api/shortener/v1"
	"github.com/olegsys/go-shortener/internal/audit"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/model"
)

// shortenerService описывает часть бизнес-логики gRPC-сервера
type shortenerService interface {
	Shorten(ctx context.Context, longURL string) (string, bool, error)
	Resolve(ctx context.Context, shortURL string) (string, bool, bool, error)
	GetUserURLs(ctx context.Context) ([]model.URLPair, error)
}

// Server реализует gRPC ShortenerServiceServer
type Server struct {
	shortenerpb.UnimplementedShortenerServiceServer

	svc     shortenerService
	auditor audit.Auditor
}

// New создаёт gRPC-сервер
func New(svc shortenerService, auditor audit.Auditor) *Server {
	if auditor == nil {
		auditor = audit.NoopAuditor{}
	}

	return &Server{
		svc:     svc,
		auditor: auditor,
	}
}

// ShortenURL сокращает URL
func (s *Server) ShortenURL(
	ctx context.Context,
	req *shortenerpb.URLShortenRequest,
) (*shortenerpb.URLShortenResponse, error) {
	if strings.TrimSpace(req.GetUrl()) == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	shortURL, conflict, err := s.svc.Shorten(ctx, req.GetUrl())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to shorten url")
	}

	s.auditor.Publish(ctx, audit.Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userIDFromContext(ctx),
		URL:    req.GetUrl(),
	})

	if conflict {
		st := status.New(codes.AlreadyExists, "original url already shortened")

		detailed, err := st.WithDetails(&shortenerpb.URLShortenResponse{
			Result: shortURL,
		})
		if err == nil {
			return nil, detailed.Err()
		}

		return nil, st.Err()
	}

	return &shortenerpb.URLShortenResponse{
		Result: shortURL,
	}, nil
}

// ExpandURL возвращает оригинальный URL по короткому идентификатору
func (s *Server) ExpandURL(
	ctx context.Context,
	req *shortenerpb.URLExpandRequest,
) (*shortenerpb.URLExpandResponse, error) {
	id := extractShortID(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	longURL, exists, isDeleted, err := s.svc.Resolve(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to resolve url")
	}

	if !exists {
		return nil, status.Error(codes.NotFound, "url not found")
	}

	if isDeleted {
		return nil, status.Error(codes.NotFound, "url deleted")
	}

	s.auditor.Publish(ctx, audit.Event{
		TS:     time.Now().Unix(),
		Action: "follow",
		UserID: userIDFromContext(ctx),
		URL:    longURL,
	})

	return &shortenerpb.URLExpandResponse{
		Result: longURL,
	}, nil
}

// ListUserURLs возвращает URL текущего пользователя
func (s *Server) ListUserURLs(
	ctx context.Context,
	_ *emptypb.Empty,
) (*shortenerpb.UserURLsResponse, error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "authorization required")
	}

	urls, err := s.svc.GetUserURLs(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list user urls")
	}

	resp := &shortenerpb.UserURLsResponse{}

	for _, pair := range urls {
		resp.Url = append(resp.Url, &shortenerpb.URLData{
			ShortUrl:    pair.ShortURL,
			OriginalUrl: pair.LongURL,
		})
	}

	return resp, nil
}

func userIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)
	return userID
}

func extractShortID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		if base := path.Base(u.Path); base != "" && base != "/" && base != "." {
			return base
		}
	}

	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		return raw[idx+1:]
	}

	return raw
}
