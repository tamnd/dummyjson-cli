package dummyjson_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/dummyjson-cli/dummyjson"
)

func newTestClient(ts *httptest.Server) *dummyjson.Client {
	cfg := dummyjson.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 1
	return dummyjson.NewClient(cfg)
}

// TestUserAgent checks that every request carries dummy-cli in User-Agent.
func TestUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		resp := map[string]any{"products": []any{}, "total": 0}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, _, _ = c.ListProducts(context.Background(), "", 10)

	if !strings.Contains(gotUA, "dummy-cli") {
		t.Errorf("User-Agent = %q, want it to contain dummy-cli", gotUA)
	}
}

// TestListProducts checks that a products list response is parsed correctly.
func TestListProducts(t *testing.T) {
	fixture := map[string]any{
		"products": []any{
			map[string]any{
				"id":       1,
				"title":    "Essence Mascara",
				"price":    9.99,
				"rating":   4.94,
				"brand":    "Essence",
				"category": "beauty",
			},
		},
		"total": 194,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") == "" {
			t.Error("limit param missing")
		}
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, total, err := c.ListProducts(context.Background(), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 194 {
		t.Errorf("total = %d, want 194", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != 1 {
		t.Errorf("ID = %d, want 1", items[0].ID)
	}
	if items[0].Title != "Essence Mascara" {
		t.Errorf("Title = %q", items[0].Title)
	}
	if items[0].Price != 9.99 {
		t.Errorf("Price = %f", items[0].Price)
	}
	if items[0].Brand != "Essence" {
		t.Errorf("Brand = %q", items[0].Brand)
	}
	if items[0].Rating != 4.94 {
		t.Errorf("Rating = %f", items[0].Rating)
	}
}

// TestListProductsByCategory checks the category path is used.
func TestListProductsByCategory(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		resp := map[string]any{"products": []any{}, "total": 0}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, _, err := c.ListProducts(context.Background(), "beauty", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "/products/category/beauty") {
		t.Errorf("path = %q, want to contain /products/category/beauty", gotPath)
	}
}

// TestSearchProducts checks that the search query param is forwarded.
func TestSearchProducts(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		resp := map[string]any{"products": []any{}, "total": 0}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, _, err := c.SearchProducts(context.Background(), "phone", 10)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "phone" {
		t.Errorf("q = %q, want phone", gotQuery)
	}
}

// TestListUsers checks that a users list response is parsed and field names remapped.
func TestListUsers(t *testing.T) {
	fixture := map[string]any{
		"users": []any{
			map[string]any{
				"id":        1,
				"firstName": "Emily",
				"lastName":  "Johnson",
				"email":     "emily.johnson@x.dummyjson.com",
				"phone":     "+1 555-123-4567",
				"age":       29,
				"gender":    "female",
			},
		},
		"total": 208,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, total, err := c.ListUsers(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 208 {
		t.Errorf("total = %d, want 208", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].FirstName != "Emily" {
		t.Errorf("FirstName = %q", items[0].FirstName)
	}
	if items[0].LastName != "Johnson" {
		t.Errorf("LastName = %q", items[0].LastName)
	}
	if items[0].Email != "emily.johnson@x.dummyjson.com" {
		t.Errorf("Email = %q", items[0].Email)
	}
}

// TestListPosts checks that reactions.likes is mapped to Likes and userId to UserID.
func TestListPosts(t *testing.T) {
	fixture := map[string]any{
		"posts": []any{
			map[string]any{
				"id":    1,
				"title": "His mother had always taught him",
				"body":  "His mother had always taught him not to ever think of himself as better than others.",
				"tags":  []string{"history", "american"},
				"reactions": map[string]any{
					"likes":    192,
					"dislikes": 25,
				},
				"userId": 121,
			},
		},
		"total": 251,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, total, err := c.ListPosts(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 251 {
		t.Errorf("total = %d, want 251", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Likes != 192 {
		t.Errorf("Likes = %d, want 192", items[0].Likes)
	}
	if items[0].UserID != 121 {
		t.Errorf("UserID = %d, want 121", items[0].UserID)
	}
	if len(items[0].Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(items[0].Tags))
	}
}

// TestListTodos checks that userId is mapped to UserID.
func TestListTodos(t *testing.T) {
	fixture := map[string]any{
		"todos": []any{
			map[string]any{
				"id":        1,
				"todo":      "Do something nice for someone you care about",
				"completed": false,
				"userId":    152,
			},
		},
		"total": 254,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, total, err := c.ListTodos(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 254 {
		t.Errorf("total = %d, want 254", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Todo != "Do something nice for someone you care about" {
		t.Errorf("Todo = %q", items[0].Todo)
	}
	if items[0].UserID != 152 {
		t.Errorf("UserID = %d, want 152", items[0].UserID)
	}
	if items[0].Completed != false {
		t.Errorf("Completed = true, want false")
	}
}

// TestListQuotes checks that quotes are parsed correctly.
func TestListQuotes(t *testing.T) {
	fixture := map[string]any{
		"quotes": []any{
			map[string]any{
				"id":     1,
				"quote":  "Your heart is the size of an ocean.",
				"author": "Rumi",
			},
		},
		"total": 1450,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, total, err := c.ListQuotes(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1450 {
		t.Errorf("total = %d, want 1450", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Author != "Rumi" {
		t.Errorf("Author = %q", items[0].Author)
	}
}

// TestListRecipes checks that prepTimeMinutes/cookTimeMinutes are remapped.
func TestListRecipes(t *testing.T) {
	fixture := map[string]any{
		"recipes": []any{
			map[string]any{
				"id":              1,
				"name":            "Classic Margherita Pizza",
				"cuisine":         "Italian",
				"prepTimeMinutes": 20,
				"cookTimeMinutes": 15,
				"servings":        4,
				"ingredients":     []string{"Pizza dough", "Tomato sauce", "Mozzarella"},
			},
		},
		"total": 50,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, total, err := c.ListRecipes(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if total != 50 {
		t.Errorf("total = %d, want 50", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].PrepTime != 20 {
		t.Errorf("PrepTime = %d, want 20", items[0].PrepTime)
	}
	if items[0].CookTime != 15 {
		t.Errorf("CookTime = %d, want 15", items[0].CookTime)
	}
	if items[0].Cuisine != "Italian" {
		t.Errorf("Cuisine = %q", items[0].Cuisine)
	}
	if len(items[0].Ingredients) != 3 {
		t.Errorf("len(Ingredients) = %d, want 3", len(items[0].Ingredients))
	}
}

// TestListCategories checks that the categories array is parsed.
func TestListCategories(t *testing.T) {
	fixture := []any{
		map[string]any{"name": "Beauty", "slug": "beauty", "url": "https://dummyjson.com/products/category/beauty"},
		map[string]any{"name": "Fragrances", "slug": "fragrances", "url": "https://dummyjson.com/products/category/fragrances"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Name != "Beauty" {
		t.Errorf("Name = %q, want Beauty", items[0].Name)
	}
	if items[0].Slug != "beauty" {
		t.Errorf("Slug = %q, want beauty", items[0].Slug)
	}
}

// TestRetry checks that 429 triggers a retry.
func TestRetry(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		resp := map[string]any{"products": []any{}, "total": 0}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	cfg := dummyjson.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := dummyjson.NewClient(cfg)

	_, _, err := c.ListProducts(context.Background(), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if hits < 2 {
		t.Errorf("hits = %d, want at least 2", hits)
	}
}
