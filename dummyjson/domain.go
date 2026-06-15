package dummyjson

import (
	"context"
	"fmt"
	"strconv"
	"unicode"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the dummyjson driver.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "dummyjson",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "dummyjson",
			Short:  "A command line for DummyJSON fake data API.",
			Long: `A command line for the DummyJSON fake data API.

dummyjson reads products, users, posts, quotes, and recipes from
dummyjson.com over HTTPS, shapes them into clean records, and prints output
that pipes into the rest of your tools. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/dummyjson-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "products", Group: "read", List: true,
		Summary: "List products (--category, --limit)"}, listProducts)

	kit.Handle(app, kit.OpMeta{Name: "search", Group: "read", List: true,
		Summary:  "Search products by keyword",
		Args:     []kit.Arg{{Name: "query", Help: "search query"}}}, searchProducts)

	kit.Handle(app, kit.OpMeta{Name: "categories", Group: "read", List: true,
		Summary: "List product categories"}, listCategories)

	kit.Handle(app, kit.OpMeta{Name: "users", Group: "read", List: true,
		Summary: "List users (--limit)"}, listUsers)

	kit.Handle(app, kit.OpMeta{Name: "posts", Group: "read", List: true,
		Summary: "List posts (--limit)"}, listPosts)

	kit.Handle(app, kit.OpMeta{Name: "quotes", Group: "read", List: true,
		Summary: "List quotes (--limit)"}, listQuotes)

	kit.Handle(app, kit.OpMeta{Name: "recipes", Group: "read", List: true,
		Summary: "List recipes (--limit)"}, listRecipes)
}

// newClient builds the Client from kit config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- input structs ---

type productsInput struct {
	Category string  `kit:"flag" help:"filter by category slug"`
	Limit    int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client   *Client `kit:"inject"`
}

type searchInput struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type categoriesInput struct {
	Client *Client `kit:"inject"`
}

type usersInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type postsInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type quotesInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type recipesInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listProducts(ctx context.Context, in productsInput, emit func(*Product) error) error {
	items, _, err := in.Client.ListProducts(ctx, in.Category, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func searchProducts(ctx context.Context, in searchInput, emit func(*Product) error) error {
	items, _, err := in.Client.SearchProducts(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listCategories(ctx context.Context, in categoriesInput, emit func(*Category) error) error {
	items, err := in.Client.ListCategories(ctx)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listUsers(ctx context.Context, in usersInput, emit func(*User) error) error {
	items, _, err := in.Client.ListUsers(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listPosts(ctx context.Context, in postsInput, emit func(*Post) error) error {
	items, _, err := in.Client.ListPosts(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listQuotes(ctx context.Context, in quotesInput, emit func(*Quote) error) error {
	items, _, err := in.Client.ListQuotes(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listRecipes(ctx context.Context, in recipesInput, emit func(*Recipe) error) error {
	items, _, err := in.Client.ListRecipes(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

// Classify turns an identifier into (type, id).
// Numeric input -> ("product", input); otherwise -> ("query", input).
func (Domain) Classify(input string) (string, string, error) {
	allDigits := len(input) > 0
	for _, r := range input {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits {
		if _, err := strconv.Atoi(input); err == nil {
			return "product", input, nil
		}
	}
	return "query", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "product":
		return fmt.Sprintf("%s/products/%s", BaseURL, id), nil
	case "query":
		return fmt.Sprintf("%s/products/search?q=%s", BaseURL, id), nil
	default:
		return "", errs.Usage("dummyjson has no resource type %q", t)
	}
}
