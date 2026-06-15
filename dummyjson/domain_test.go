package dummyjson

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "dummyjson" {
		t.Errorf("Scheme = %q, want dummyjson", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "dummyjson" {
		t.Errorf("Identity.Binary = %q, want dummyjson", info.Identity.Binary)
	}
}

func TestClassifyNumeric(t *testing.T) {
	typ, id, err := Domain{}.Classify("42")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "product" {
		t.Errorf("type = %q, want product", typ)
	}
	if id != "42" {
		t.Errorf("id = %q, want 42", id)
	}
}

func TestClassifyQuery(t *testing.T) {
	typ, id, err := Domain{}.Classify("iphone")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "query" {
		t.Errorf("type = %q, want query", typ)
	}
	if id != "iphone" {
		t.Errorf("id = %q, want iphone", id)
	}
}

func TestLocateProduct(t *testing.T) {
	got, err := Domain{}.Locate("product", "1")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	want := "https://dummyjson.com/products/1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocateQuery(t *testing.T) {
	got, err := Domain{}.Locate("query", "phone")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	want := "https://dummyjson.com/products/search?q=phone"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "1")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}
