package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"awesomeProject/internal/logging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryLoggingInterceptor logs each unary call as JSON with method, status,
// duration, user_id, role, and redacted request/response bodies.
func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		ctx, callInfo := logging.WithCallInfo(ctx)

		resp, err := handler(ctx, req)

		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.String("code", status.Code(err).String()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.Any("request", logging.ProtoBody(req)),
			slog.Any("response", logging.ProtoBody(resp)),
		}
		if callInfo != nil {
			if callInfo.UserID != "" {
				attrs = append(attrs, slog.String("user_id", callInfo.UserID))
			}
			if callInfo.Role != "" {
				attrs = append(attrs, slog.String("role", callInfo.Role))
			}
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", status.Convert(err).Message()))
		}

		logging.Logger.LogAttrs(ctx, slog.LevelInfo, "grpc", attrs...)
		return resp, err
	}
}

type ctxKey int

const (
	userIDKey ctxKey = 1
	roleKey   ctxKey = 2
)

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func ContextWithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(roleKey).(string)
	return v
}

func UnaryInterceptor(jwt *Manager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		token, tokenErr := bearerFromMetadata(ctx)
		if tokenErr == nil {
			userID, _, role, parseErr := jwt.Parse(token)
			if parseErr == nil {
				ctx = ContextWithUserID(ctx, userID)
				ctx = ContextWithRole(ctx, role)
				if ci := logging.CallInfoFrom(ctx); ci != nil {
					ci.UserID = userID
					ci.Role = role
				}
			} else if requiresAuth(info.FullMethod) {
				return nil, status.Error(codes.Unauthenticated, "недействительный или просроченный токен")
			}
		} else if requiresAuth(info.FullMethod) {
			return nil, status.Error(codes.Unauthenticated, "требуется Authorization: Bearer <token>")
		}

		if requiresAdmin(info.FullMethod) && RoleFromContext(ctx) != "admin" {
			return nil, status.Error(codes.PermissionDenied, "требуется роль admin")
		}

		return handler(ctx, req)
	}
}

func requiresAuth(fullMethod string) bool {
	if strings.Contains(fullMethod, "DeliveryService") {
		return !strings.HasSuffix(fullMethod, "/ListPickupPoints") &&
			!strings.HasSuffix(fullMethod, "/GetPickupPoint")
	}
	return strings.Contains(fullMethod, "CartService") ||
		strings.Contains(fullMethod, "OrderService") ||
		strings.Contains(fullMethod, "PromoService") ||
		strings.HasSuffix(fullMethod, "/DeleteUser")
}

func requiresAdmin(fullMethod string) bool {
	return strings.Contains(fullMethod, "PromoService") ||
		strings.HasSuffix(fullMethod, "/ListOrderJobs")
}

func bearerFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	vals := md.Get("authorization")
	if len(vals) == 0 {
		vals = md.Get("Authorization")
	}
	if len(vals) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization")
	}

	parts := strings.SplitN(vals[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header")
	}
	return parts[1], nil
}
