package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"awesomeProject/internal/repository"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(order repository.Order) (repository.Order, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Order{}, fmt.Errorf("begin create order: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	order.ID = uuid.NewString()
	order.Status = 1 // ORDER_STATUS_CREATED
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	snap, err := json.Marshal(order.DeliverySnapshot)
	if err != nil {
		return repository.Order{}, fmt.Errorf("marshal delivery snapshot: %w", err)
	}

	const insertOrder = `
		INSERT INTO orders (
			id, user_id, total_amount_cents, status, created_at, updated_at,
			delivery_method, delivery_fee_cents, delivery_snapshot
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	if _, err := tx.ExecContext(ctx, insertOrder,
		order.ID, order.UserID, order.TotalAmountCents, order.Status, order.CreatedAt, order.UpdatedAt,
		order.DeliveryMethod, order.DeliveryFeeCents, snap,
	); err != nil {
		return repository.Order{}, fmt.Errorf("insert order: %w", err)
	}

	const insertItem = `
		INSERT INTO order_items (order_id, product_id, quantity, price_cents)
		VALUES ($1, $2, $3, $4)
	`
	for _, item := range order.Items {
		if _, err := tx.ExecContext(ctx, insertItem, order.ID, item.ProductID, item.Quantity, item.PriceCents); err != nil {
			return repository.Order{}, fmt.Errorf("insert order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return repository.Order{}, fmt.Errorf("commit create order: %w", err)
	}
	return order, nil
}

func (r *OrderRepository) GetOrder(orderID string) (repository.Order, error) {
	ctx := context.Background()

	const orderQ = `
		SELECT id, user_id, total_amount_cents, status, created_at, updated_at,
		       delivery_method, delivery_fee_cents, delivery_snapshot
		FROM orders
		WHERE id = $1
	`

	var order repository.Order
	var snap []byte
	err := r.db.QueryRowContext(ctx, orderQ, orderID).Scan(
		&order.ID,
		&order.UserID,
		&order.TotalAmountCents,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.DeliveryMethod,
		&order.DeliveryFeeCents,
		&snap,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Order{}, repository.ErrOrderNotFound
	}
	if err != nil {
		return repository.Order{}, fmt.Errorf("get order: %w", err)
	}
	if err := json.Unmarshal(snap, &order.DeliverySnapshot); err != nil {
		return repository.Order{}, fmt.Errorf("unmarshal delivery snapshot: %w", err)
	}

	items, err := r.loadItems(ctx, orderID)
	if err != nil {
		return repository.Order{}, err
	}
	order.Items = items
	return order, nil
}

func (r *OrderRepository) loadItems(ctx context.Context, orderID string) ([]repository.OrderItem, error) {
	const itemsQ = `
		SELECT product_id, quantity, price_cents
		FROM order_items
		WHERE order_id = $1
		ORDER BY product_id
	`
	rows, err := r.db.QueryContext(ctx, itemsQ, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()

	items := make([]repository.OrderItem, 0)
	for rows.Next() {
		var item repository.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.PriceCents); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get order items rows: %w", err)
	}
	return items, nil
}

func (r *OrderRepository) ListOrders(userID string) ([]repository.Order, error) {
	ctx := context.Background()
	const q = `
		SELECT id, user_id, total_amount_cents, status, created_at, updated_at,
		       delivery_method, delivery_fee_cents, delivery_snapshot
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]repository.Order, 0)
	for rows.Next() {
		var order repository.Order
		var snap []byte
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.TotalAmountCents,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.DeliveryMethod,
			&order.DeliveryFeeCents,
			&snap,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if err := json.Unmarshal(snap, &order.DeliverySnapshot); err != nil {
			return nil, fmt.Errorf("unmarshal delivery snapshot: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list orders rows: %w", err)
	}

	if err := r.loadItemsForOrders(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *OrderRepository) loadItemsForOrders(ctx context.Context, orders []repository.Order) error {
	if len(orders) == 0 {
		return nil
	}

	ids := make([]any, 0, len(orders))
	placeholders := make([]string, 0, len(orders))
	indexByID := make(map[string]int, len(orders))
	for i := range orders {
		orders[i].Items = make([]repository.OrderItem, 0)
		indexByID[orders[i].ID] = i
		ids = append(ids, orders[i].ID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	q := fmt.Sprintf(`
		SELECT order_id, product_id, quantity, price_cents
		FROM order_items
		WHERE order_id IN (%s)
		ORDER BY order_id, product_id
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, q, ids...)
	if err != nil {
		return fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID string
		var item repository.OrderItem
		if err := rows.Scan(&orderID, &item.ProductID, &item.Quantity, &item.PriceCents); err != nil {
			return fmt.Errorf("scan order item: %w", err)
		}
		idx, ok := indexByID[orderID]
		if !ok {
			continue
		}
		orders[idx].Items = append(orders[idx].Items, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list order items rows: %w", err)
	}
	return nil
}

func (r *OrderRepository) UpdateOrderStatus(orderID string, fromStatus, toStatus int32) (repository.Order, error) {
	ctx := context.Background()
	const q = `
		UPDATE orders
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND status = $2
		RETURNING id, user_id, total_amount_cents, status, created_at, updated_at,
		          delivery_method, delivery_fee_cents, delivery_snapshot
	`
	var order repository.Order
	var snap []byte
	err := r.db.QueryRowContext(ctx, q, orderID, fromStatus, toStatus).Scan(
		&order.ID,
		&order.UserID,
		&order.TotalAmountCents,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.DeliveryMethod,
		&order.DeliveryFeeCents,
		&snap,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Order{}, fmt.Errorf("status transition rejected")
	}
	if err != nil {
		return repository.Order{}, fmt.Errorf("update order status: %w", err)
	}
	if err := json.Unmarshal(snap, &order.DeliverySnapshot); err != nil {
		return repository.Order{}, fmt.Errorf("unmarshal delivery snapshot: %w", err)
	}
	items, err := r.loadItems(ctx, orderID)
	if err != nil {
		return repository.Order{}, err
	}
	order.Items = items
	return order, nil
}

func (r *OrderRepository) DeleteOrder(orderID string) error {
	res, err := r.db.ExecContext(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	if err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrOrderNotFound
	}
	return nil
}
