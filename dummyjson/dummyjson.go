// Package dummyjson is the library behind the dummyjson command line:
// the HTTP client, request shaping, and typed data models for the DummyJSON API
// (https://dummyjson.com/).
//
// No API key is required. The Client paces requests, sets a real User-Agent,
// and retries transient failures (429 and 5xx).
package dummyjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to DummyJSON.
const DefaultUserAgent = "dummyjson-cli/0.1 (tamnd87@gmail.com)"

// Host is the API hostname.
const Host = "dummyjson.com"

// BaseURL is the root every API request is built from.
const BaseURL = "https://" + Host

// Product is a DummyJSON product record.
type Product struct {
	ID       int     `kit:"id" json:"id"`
	Title    string  `json:"title"`
	Category string  `json:"category"`
	Brand    string  `json:"brand"`
	Price    float64 `json:"price"`
	Rating   float64 `json:"rating"`
	Stock    int     `json:"stock"`
}

// Category is a DummyJSON product category.
type Category struct {
	Slug string `kit:"id" json:"slug"`
	Name string `json:"name"`
}

// User is a DummyJSON user record.
type User struct {
	ID        int    `kit:"id" json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
	Username  string `json:"username"`
	City      string `json:"city"`
	Country   string `json:"country"`
}

// Post is a DummyJSON post record.
type Post struct {
	ID     int    `kit:"id" json:"id"`
	Title  string `json:"title"`
	Tags   string `json:"tags"`
	Views  int    `json:"views"`
	Likes  int    `json:"likes"`
	UserID int    `json:"user_id"`
}

// Quote is a DummyJSON quote record.
type Quote struct {
	ID     int    `kit:"id" json:"id"`
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

// Recipe is a DummyJSON recipe record.
type Recipe struct {
	ID       int    `kit:"id" json:"id"`
	Name     string `json:"name"`
	Cuisine  string `json:"cuisine"`
	Difficulty string `json:"difficulty"`
	PrepTime int    `json:"prep_time"`
	CookTime int    `json:"cook_time"`
	Servings int    `json:"servings"`
	Calories int    `json:"calories"`
}

// --- wire types for JSON decoding ---

type wireProduct struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	Brand    string  `json:"brand"`
	Rating   float64 `json:"rating"`
	Stock    int     `json:"stock"`
}

func (w wireProduct) toProduct() Product {
	return Product{
		ID:       w.ID,
		Title:    w.Title,
		Price:    w.Price,
		Category: w.Category,
		Brand:    w.Brand,
		Rating:   w.Rating,
		Stock:    w.Stock,
	}
}

type wireAddress struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type wireUser struct {
	ID        int         `json:"id"`
	FirstName string      `json:"firstName"`
	LastName  string      `json:"lastName"`
	Email     string      `json:"email"`
	Age       int         `json:"age"`
	Gender    string      `json:"gender"`
	Username  string      `json:"username"`
	Address   wireAddress `json:"address"`
}

func (w wireUser) toUser() User {
	return User{
		ID:        w.ID,
		FirstName: w.FirstName,
		LastName:  w.LastName,
		Email:     w.Email,
		Age:       w.Age,
		Gender:    w.Gender,
		Username:  w.Username,
		City:      w.Address.City,
		Country:   w.Address.Country,
	}
}

type wirePost struct {
	ID    int      `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Reactions struct {
		Likes    int `json:"likes"`
		Dislikes int `json:"dislikes"`
	} `json:"reactions"`
	Views  int `json:"views"`
	UserID int `json:"userId"`
}

func (w wirePost) toPost() Post {
	return Post{
		ID:     w.ID,
		Title:  w.Title,
		Tags:   strings.Join(w.Tags, ", "),
		Views:  w.Views,
		Likes:  w.Reactions.Likes,
		UserID: w.UserID,
	}
}

type wireRecipe struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Cuisine         string `json:"cuisine"`
	Difficulty      string `json:"difficulty"`
	PrepTimeMinutes int    `json:"prepTimeMinutes"`
	CookTimeMinutes int    `json:"cookTimeMinutes"`
	Servings        int    `json:"servings"`
	CaloriesPerServing int `json:"caloriesPerServing"`
}

func (w wireRecipe) toRecipe() Recipe {
	return Recipe{
		ID:         w.ID,
		Name:       w.Name,
		Cuisine:    w.Cuisine,
		Difficulty: w.Difficulty,
		PrepTime:   w.PrepTimeMinutes,
		CookTime:   w.CookTimeMinutes,
		Servings:   w.Servings,
		Calories:   w.CaloriesPerServing,
	}
}

type wireCategory struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

// --- wire list types ---

type wireProductList struct {
	Products []wireProduct `json:"products"`
	Total    int           `json:"total"`
}

type wireUserList struct {
	Users []wireUser `json:"users"`
	Total int        `json:"total"`
}

type wirePostList struct {
	Posts []wirePost `json:"posts"`
	Total int        `json:"total"`
}

type wireQuoteList struct {
	Quotes []Quote `json:"quotes"`
	Total  int     `json:"total"`
}

type wireRecipeList struct {
	Recipes []wireRecipe `json:"recipes"`
	Total   int          `json:"total"`
}

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Timeout:   15 * time.Second,
		Retries:   3,
	}
}

// Client talks to the DummyJSON API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client with the given configuration.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// ListProducts fetches products, optionally filtered by category.
func (c *Client) ListProducts(ctx context.Context, category string, limit int) ([]Product, int, error) {
	var rawURL string
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	if category != "" {
		rawURL = c.cfg.BaseURL + "/products/category/" + url.PathEscape(category) + "?" + params.Encode()
	} else {
		rawURL = c.cfg.BaseURL + "/products?" + params.Encode()
	}
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireProductList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse products: %w", err)
	}
	out := make([]Product, len(resp.Products))
	for i, w := range resp.Products {
		out[i] = w.toProduct()
	}
	return out, resp.Total, nil
}

// SearchProducts searches products by query string.
func (c *Client) SearchProducts(ctx context.Context, q string, limit int) ([]Product, int, error) {
	params := url.Values{}
	params.Set("q", q)
	params.Set("limit", fmt.Sprintf("%d", limit))
	rawURL := c.cfg.BaseURL + "/products/search?" + params.Encode()
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireProductList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse products search: %w", err)
	}
	out := make([]Product, len(resp.Products))
	for i, w := range resp.Products {
		out[i] = w.toProduct()
	}
	return out, resp.Total, nil
}

// ListCategories fetches the list of product categories.
func (c *Client) ListCategories(ctx context.Context) ([]Category, error) {
	rawURL := c.cfg.BaseURL + "/products/categories"
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var wires []wireCategory
	if err := json.Unmarshal(body, &wires); err != nil {
		return nil, fmt.Errorf("parse categories: %w", err)
	}
	out := make([]Category, len(wires))
	for i, w := range wires {
		out[i] = Category{Name: w.Name, Slug: w.Slug}
	}
	return out, nil
}

// ListUsers fetches a page of users.
func (c *Client) ListUsers(ctx context.Context, limit int) ([]User, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	rawURL := c.cfg.BaseURL + "/users?" + params.Encode()
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireUserList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse users: %w", err)
	}
	out := make([]User, len(resp.Users))
	for i, w := range resp.Users {
		out[i] = w.toUser()
	}
	return out, resp.Total, nil
}

// ListPosts fetches a page of posts.
func (c *Client) ListPosts(ctx context.Context, limit int) ([]Post, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	rawURL := c.cfg.BaseURL + "/posts?" + params.Encode()
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wirePostList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse posts: %w", err)
	}
	out := make([]Post, len(resp.Posts))
	for i, w := range resp.Posts {
		out[i] = w.toPost()
	}
	return out, resp.Total, nil
}

// ListQuotes fetches a page of quotes.
func (c *Client) ListQuotes(ctx context.Context, limit int) ([]Quote, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	rawURL := c.cfg.BaseURL + "/quotes?" + params.Encode()
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireQuoteList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse quotes: %w", err)
	}
	return resp.Quotes, resp.Total, nil
}

// ListRecipes fetches a page of recipes.
func (c *Client) ListRecipes(ctx context.Context, limit int) ([]Recipe, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	rawURL := c.cfg.BaseURL + "/recipes?" + params.Encode()
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireRecipeList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse recipes: %w", err)
	}
	out := make([]Recipe, len(resp.Recipes))
	for i, w := range resp.Recipes {
		out[i] = w.toRecipe()
	}
	return out, resp.Total, nil
}

// --- internal helpers ---

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
