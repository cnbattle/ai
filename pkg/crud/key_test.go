package crud

import (
	"testing"
)

func TestKeyBuilder_Primary(t *testing.T) {
	b := NewKeyBuilder()
	got := b.Primary("cache", "users", 1)
	want := "cache:users:id:1"
	if got != want {
		t.Errorf("Primary() = %q, want %q", got, want)
	}
}

func TestKeyBuilder_Field(t *testing.T) {
	b := NewKeyBuilder()
	got := b.Field("cache", "users", "name", "张三")
	want := "cache:users:name:张三"
	if got != want {
		t.Errorf("Field() = %q, want %q", got, want)
	}
}

func TestKeyBuilder_Composite(t *testing.T) {
	b := NewKeyBuilder()
	got := b.Composite("cache", "users", []string{"name", "email"}, []any{"张三", "a@b.com"})
	want := "cache:users:idx_name|email:张三|a@b.com"
	if got != want {
		t.Errorf("Composite() = %q, want %q", got, want)
	}
}

func TestKeyBuilder_Composite_LengthMismatch(t *testing.T) {
	b := NewKeyBuilder()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for fields/values length mismatch")
		}
		msg, ok := r.(string)
		if !ok || len(msg) == 0 {
			t.Fatalf("expected panic message, got %v", r)
		}
	}()
	b.Composite("cache", "users", []string{"name"}, []any{"张三", "extra"})
}

func TestKeyBuilder_WithSeparator(t *testing.T) {
	b := NewKeyBuilder().WithSeparator("#")
	got := b.Primary("cache", "users", 42)
	want := "cache#users#id#42"
	if got != want {
		t.Errorf("Primary() = %q, want %q", got, want)
	}
}

func TestKeyBuilder_MatchPrefix(t *testing.T) {
	b := NewKeyBuilder()
	got := b.MatchPrefix("cache", "users")
	want := "cache:users:"
	if got != want {
		t.Errorf("MatchPrefix() = %q, want %q", got, want)
	}
}
