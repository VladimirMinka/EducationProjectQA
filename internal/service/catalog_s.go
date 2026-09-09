package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"awesomeProject/internal/repository"
)

var (
	ErrInvalidPageSize  = errors.New("недопустимый page_size")
	ErrInvalidPageToken = errors.New("недопустимый page_token")
	ErrInvalidPriceRange = errors.New("min_price_cents больше max_price_cents")
	ErrInvalidSort      = errors.New("недопустимый sort")
)

type CatalogRepository interface {
	GetProduct(id string) (repository.Product, error)
	ListProducts(filter repository.ProductFilter) ([]repository.Product, int, error)
	ListCategories() ([]repository.Category, error)
	GetCategory(id string) (repository.Category, error)
}

type CatalogService struct {
	repo CatalogRepository
}

func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

type ListProductsParams struct {
	PageSize      int32
	PageToken     string
	Query         string
	Brand         string
	CategoryID    string
	MinPriceCents int64
	MaxPriceCents int64
	HasMinPrice   bool
	HasMaxPrice   bool
	InStock       *bool
	Sort          string
}

type ListProductsResult struct {
	Products      []repository.Product
	NextPageToken string
	TotalCount    int32
}

func (s *CatalogService) GetProduct(id string) (repository.Product, error) {
	if id == "" {
		return repository.Product{}, errors.New("product_id cannot be empty")
	}
	return s.repo.GetProduct(id)
}

func (s *CatalogService) ListProducts(p ListProductsParams) (ListProductsResult, error) {
	pageSize := int(p.PageSize)
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize < 0 || pageSize > 50 {
		return ListProductsResult{}, ErrInvalidPageSize
	}

	offset := 0
	if p.PageToken != "" {
		var err error
		offset, err = decodePageToken(p.PageToken)
		if err != nil {
			return ListProductsResult{}, ErrInvalidPageToken
		}
	}

	if p.HasMinPrice && p.HasMaxPrice && p.MinPriceCents > p.MaxPriceCents {
		return ListProductsResult{}, ErrInvalidPriceRange
	}

	sort := strings.TrimSpace(p.Sort)
	switch sort {
	case "", "price_asc", "price_desc", "name_asc", "name_desc":
	default:
		return ListProductsResult{}, ErrInvalidSort
	}

	filter := repository.ProductFilter{
		Query:      strings.TrimSpace(p.Query),
		Brand:      strings.TrimSpace(p.Brand),
		CategoryID: strings.TrimSpace(p.CategoryID),
		InStock:    p.InStock,
		Sort:       sort,
		Limit:      pageSize,
		Offset:     offset,
	}
	if p.HasMinPrice {
		v := p.MinPriceCents
		filter.MinPriceCents = &v
	}
	if p.HasMaxPrice {
		v := p.MaxPriceCents
		filter.MaxPriceCents = &v
	}

	products, total, err := s.repo.ListProducts(filter)
	if err != nil {
		return ListProductsResult{}, err
	}

	next := ""
	if offset+len(products) < total {
		next = encodePageToken(offset + len(products))
	}
	return ListProductsResult{
		Products:      products,
		NextPageToken: next,
		TotalCount:    int32(total),
	}, nil
}

func (s *CatalogService) ListCategories() ([]repository.Category, error) {
	return s.repo.ListCategories()
}

func (s *CatalogService) GetCategory(id string) (repository.Category, error) {
	if id == "" {
		return repository.Category{}, errors.New("category_id cannot be empty")
	}
	return s.repo.GetCategory(id)
}

func encodePageToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("o:%d", offset)))
}

func decodePageToken(token string) (int, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 2 || parts[0] != "o" {
		return 0, fmt.Errorf("bad token")
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("bad offset")
	}
	return n, nil
}
