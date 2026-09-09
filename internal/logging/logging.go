package logging

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	// Logger writes JSON lines to stdout (ELK / docker logs friendly).
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	protoMarshal = protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
)

type ctxKey int

const callInfoKey ctxKey = 1

// CallInfo is filled by the auth interceptor so the outer logging interceptor
// can include user_id / role after the handler returns.
type CallInfo struct {
	UserID string
	Role   string
}

// WithCallInfo attaches a mutable CallInfo bag to ctx.
func WithCallInfo(ctx context.Context) (context.Context, *CallInfo) {
	info := &CallInfo{}
	return context.WithValue(ctx, callInfoKey, info), info
}

// CallInfoFrom returns the bag previously attached with WithCallInfo, if any.
func CallInfoFrom(ctx context.Context) *CallInfo {
	v, _ := ctx.Value(callInfoKey).(*CallInfo)
	return v
}

var sensitiveKeys = map[string]struct{}{
	"password":      {},
	"access_token":  {},
	"refresh_token": {},
	"authorization": {},
	"token":         {},
}

const (
	maxArrayItems = 3
	maxStringLen  = 200
	maxBodyBytes  = 8 << 10 // hard cap after marshal (safety net)
)

// ProtoBody converts a protobuf message to a redacted, size-limited JSON value.
// Large repeated fields (e.g. catalog products) keep a short sample + total count.
func ProtoBody(msg any) any {
	pm, ok := msg.(proto.Message)
	if !ok || pm == nil {
		return nil
	}
	b, err := protoMarshal.Marshal(pm)
	if err != nil {
		return map[string]string{"_error": err.Error()}
	}
	if len(b) > maxBodyBytes {
		return map[string]any{
			"_truncated": true,
			"_bytes":    len(b),
		}
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return map[string]string{"_raw": string(b)}
	}
	return Truncate(Redact(v))
}

// Redact replaces sensitive object fields with "[REDACTED]".
func Redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if _, sens := sensitiveKeys[strings.ToLower(k)]; sens {
				out[k] = "[REDACTED]"
				continue
			}
			out[k] = Redact(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = Redact(val)
		}
		return out
	default:
		return v
	}
}

// Truncate shortens large arrays/strings for log volume (catalog lists, etc.).
func Truncate(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = Truncate(val)
		}
		return out
	case []any:
		if len(t) <= maxArrayItems {
			out := make([]any, len(t))
			for i, val := range t {
				out[i] = Truncate(val)
			}
			return out
		}
		sample := make([]any, maxArrayItems)
		for i := 0; i < maxArrayItems; i++ {
			sample[i] = Truncate(t[i])
		}
		return map[string]any{
			"_total":   len(t),
			"_omitted": len(t) - maxArrayItems,
			"sample":   sample,
		}
	case string:
		if len(t) > maxStringLen {
			return t[:maxStringLen] + "…"
		}
		return t
	default:
		return v
	}
}
