package logging

import (
	"testing"
)

func TestRedact(t *testing.T) {
	in := map[string]any{
		"email":    "a@b.c",
		"password": "secret",
		"user": map[string]any{
			"access_token": "jwt",
			"name":         "Ann",
		},
		"items": []any{
			map[string]any{"token": "x", "qty": 1.0},
		},
	}
	out := Redact(in).(map[string]any)
	if out["password"] != "[REDACTED]" {
		t.Fatalf("password: %v", out["password"])
	}
	if out["email"] != "a@b.c" {
		t.Fatalf("email: %v", out["email"])
	}
	user := out["user"].(map[string]any)
	if user["access_token"] != "[REDACTED]" || user["name"] != "Ann" {
		t.Fatalf("user: %v", user)
	}
	item := out["items"].([]any)[0].(map[string]any)
	if item["token"] != "[REDACTED]" || item["qty"] != 1.0 {
		t.Fatalf("item: %v", item)
	}
}

func TestTruncateArrays(t *testing.T) {
	products := make([]any, 50)
	for i := range products {
		products[i] = map[string]any{"id": i, "name": "p"}
	}
	out := Truncate(map[string]any{"products": products}).(map[string]any)
	sum := out["products"].(map[string]any)
	if sum["_total"] != 50 {
		t.Fatalf("total: %v", sum["_total"])
	}
	if sum["_omitted"] != 47 {
		t.Fatalf("omitted: %v", sum["_omitted"])
	}
	sample := sum["sample"].([]any)
	if len(sample) != 3 {
		t.Fatalf("sample len: %d", len(sample))
	}
}

func TestTruncateShortArrayUnchanged(t *testing.T) {
	in := []any{"a", "b"}
	out := Truncate(in).([]any)
	if len(out) != 2 || out[0] != "a" {
		t.Fatalf("got %v", out)
	}
}
