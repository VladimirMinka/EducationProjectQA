package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"awesomeProject/internal/repository"
)

type CatalogRepository struct {
	db *sql.DB
}

func NewCatalogRepository(db *sql.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) GetProduct(id string) (repository.Product, error) {
	const q = `
		SELECT id, name, description, price_cents, stock_quantity, brand, COALESCE(category_id::text, '')
		FROM products
		WHERE id = $1
	`

	var p repository.Product
	err := r.db.QueryRowContext(context.Background(), q, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.PriceCents,
		&p.StockQuantity,
		&p.Brand,
		&p.CategoryID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Product{}, repository.ErrNotFound
	}
	if err != nil {
		return repository.Product{}, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *CatalogRepository) ListProducts(filter repository.ProductFilter) ([]repository.Product, int, error) {
	where := make([]string, 0)
	args := make([]any, 0)
	next := 1

	if filter.Query != "" {
		where = append(where, fmt.Sprintf("(name ILIKE '%%' || $%d || '%%' OR description ILIKE '%%' || $%d || '%%')", next, next))
		args = append(args, filter.Query)
		next++
	}
	if filter.Brand != "" {
		where = append(where, fmt.Sprintf("brand = $%d", next))
		args = append(args, filter.Brand)
		next++
	}
	if filter.CategoryID != "" {
		where = append(where, fmt.Sprintf("category_id = $%d", next))
		args = append(args, filter.CategoryID)
		next++
	}
	if filter.MinPriceCents != nil {
		where = append(where, fmt.Sprintf("price_cents >= $%d", next))
		args = append(args, *filter.MinPriceCents)
		next++
	}
	if filter.MaxPriceCents != nil {
		where = append(where, fmt.Sprintf("price_cents <= $%d", next))
		args = append(args, *filter.MaxPriceCents)
		next++
	}
	if filter.InStock != nil && *filter.InStock {
		where = append(where, "stock_quantity > 0")
	} else if filter.InStock != nil && !*filter.InStock {
		where = append(where, "stock_quantity <= 0")
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := "id ASC"
	switch filter.Sort {
	case "price_asc":
		orderBy = "price_cents ASC, id ASC"
	case "price_desc":
		orderBy = "price_cents DESC, id ASC"
	case "name_asc":
		orderBy = "name ASC, id ASC"
	case "name_desc":
		orderBy = "name DESC, id ASC"
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM products %s`, whereSQL)
	var total int
	if err := r.db.QueryRowContext(context.Background(), countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	listQ := fmt.Sprintf(`
		SELECT id, name, description, price_cents, stock_quantity, brand, COALESCE(category_id::text, '')
		FROM products
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBy, next, next+1)

	rows, err := r.db.QueryContext(context.Background(), listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]repository.Product, 0)
	for rows.Next() {
		var p repository.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceCents, &p.StockQuantity, &p.Brand, &p.CategoryID); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list products rows: %w", err)
	}
	return products, total, nil
}

func (r *CatalogRepository) ListCategories() ([]repository.Category, error) {
	const q = `SELECT id, slug, name FROM categories ORDER BY name`
	rows, err := r.db.QueryContext(context.Background(), q)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	out := make([]repository.Category, 0)
	for rows.Next() {
		var c repository.Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) GetCategory(id string) (repository.Category, error) {
	const q = `SELECT id, slug, name FROM categories WHERE id = $1`
	var c repository.Category
	err := r.db.QueryRowContext(context.Background(), q, id).Scan(&c.ID, &c.Slug, &c.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Category{}, repository.ErrCategoryNotFound
	}
	if err != nil {
		return repository.Category{}, fmt.Errorf("get category: %w", err)
	}
	return c, nil
}
