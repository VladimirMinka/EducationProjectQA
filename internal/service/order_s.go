package service

import (
	"errors"
	"fmt"
	"os"
	"time"

	"awesomeProject/internal/repository"
)

const (
	OrderStatusCreated   int32 = 1
	OrderStatusPaid      int32 = 2
	OrderStatusShipped   int32 = 3
	OrderStatusCancelled int32 = 4
	OrderStatusCompleted int32 = 5
)

var (
	ErrOrderNotFound     = repository.ErrOrderNotFound
	ErrInvalidTransition = errors.New("недопустимый переход статуса")
	ErrStatusMismatch    = errors.New("текущий статус не совпадает с from_status")
	ErrPermissionDenied  = errors.New("нет доступа к заказу")
)

type OrderRepository interface {
	CreateOrder(order repository.Order) (repository.Order, error)
	GetOrder(orderID string) (repository.Order, error)
	ListOrders(userID string) ([]repository.Order, error)
	UpdateOrderStatus(orderID string, fromStatus, toStatus int32) (repository.Order, error)
	DeleteOrder(orderID string) error
}

type OrderJobRepository interface {
	Enqueue(job repository.OrderJob) (repository.OrderJob, error)
	ListByOrder(orderID string) ([]repository.OrderJob, error)
}

type CartProvider interface {
	GetCartForOrder(userID string) (CartTotals, error)
	ClearCart(userID string) error
}

type DeliveryResolver interface {
	ResolveForOrder(userID string, method int32, addressID, pickupPointID string, merchandiseSubtotal int64) (int32, int64, repository.DeliverySnapshot, error)
}

type OrderService struct {
	repo     OrderRepository
	jobs     OrderJobRepository
	cart     CartProvider
	catalog  CatalogProvider
	users    UserProvider
	delivery DeliveryResolver
	delay    time.Duration
}

func NewOrderService(
	repo OrderRepository,
	jobs OrderJobRepository,
	cart CartProvider,
	catalog CatalogProvider,
	users UserProvider,
	delivery DeliveryResolver,
) *OrderService {
	return &OrderService{
		repo:     repo,
		jobs:     jobs,
		cart:     cart,
		catalog:  catalog,
		users:    users,
		delivery: delivery,
		delay:    orderStatusDelayFromEnv(),
	}
}

func orderStatusDelayFromEnv() time.Duration {
	raw := os.Getenv("ORDER_STATUS_DELAY")
	if raw == "" {
		return 10 * time.Minute
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 10 * time.Minute
	}
	return d
}

func (s *OrderService) StatusDelay() time.Duration {
	return s.delay
}

type CreateOrderInput struct {
	UserID        string
	Method        int32
	AddressID     string
	PickupPointID string
}

func (s *OrderService) CreateOrder(in CreateOrderInput) (repository.Order, error) {
	if in.UserID == "" {
		return repository.Order{}, errors.New("user_id не может быть пустым")
	}
	if _, err := s.users.GetUser(in.UserID); err != nil {
		return repository.Order{}, fmt.Errorf("пользователь не найден: %w", err)
	}

	totals, err := s.cart.GetCartForOrder(in.UserID)
	if err != nil {
		return repository.Order{}, fmt.Errorf("не удалось получить корзину: %v", err)
	}

	var orderItems []repository.OrderItem
	var actualTotal int64

	for _, cartItem := range totals.Cart.Items {
		product, err := s.catalog.GetProduct(cartItem.ProductID)
		if err != nil {
			return repository.Order{}, fmt.Errorf("товар %s недоступен: %v", cartItem.ProductID, err)
		}

		orderItems = append(orderItems, repository.OrderItem{
			ProductID:  cartItem.ProductID,
			Quantity:   cartItem.Quantity,
			PriceCents: product.PriceCents,
		})
		actualTotal += product.PriceCents * int64(cartItem.Quantity)
	}

	// Drop promocode at checkout; keep combo-only path via totals.Combo if we recompute.
	// Intentional divergence: order total ignores applied promocode from cart.
	if totals.ComboDiscountApplied {
		actualTotal = actualTotal - actualTotal/10
	}

	method, fee, snap, err := s.delivery.ResolveForOrder(in.UserID, in.Method, in.AddressID, in.PickupPointID, totals.SubtotalCents)
	if err != nil {
		return repository.Order{}, err
	}

	newOrder := repository.Order{
		UserID:           in.UserID,
		Items:            orderItems,
		TotalAmountCents: actualTotal + fee,
		DeliveryMethod:   method,
		DeliveryFeeCents: fee,
		DeliverySnapshot: snap,
	}

	savedOrder, err := s.repo.CreateOrder(newOrder)
	if err != nil {
		return repository.Order{}, err
	}

	_ = s.cart.ClearCart(in.UserID)

	return savedOrder, nil
}

func (s *OrderService) ListOrders(userID string, callerID string, isAdmin bool) ([]repository.Order, error) {
	if userID == "" {
		return nil, errors.New("user_id не может быть пустым")
	}
	if !isAdmin && userID != callerID {
		return nil, ErrPermissionDenied
	}
	if _, err := s.users.GetUser(userID); err != nil {
		return nil, fmt.Errorf("пользователь не найден: %w", err)
	}
	return s.repo.ListOrders(userID)
}

func (s *OrderService) GetOrder(orderID string, callerID string, isAdmin bool) (repository.Order, error) {
	if orderID == "" {
		return repository.Order{}, errors.New("order_id не может быть пустым")
	}
	order, err := s.repo.GetOrder(orderID)
	if err != nil {
		return repository.Order{}, err
	}
	if !isAdmin && order.UserID != callerID {
		return repository.Order{}, ErrPermissionDenied
	}
	return order, nil
}

func (s *OrderService) CancelOrder(orderID string, callerID string, isAdmin bool) (repository.Order, error) {
	order, err := s.repo.GetOrder(orderID)
	if err != nil {
		return repository.Order{}, err
	}
	if !isAdmin && order.UserID != callerID {
		return repository.Order{}, ErrPermissionDenied
	}
	if order.Status != OrderStatusCreated {
		return repository.Order{}, ErrInvalidTransition
	}
	return s.repo.UpdateOrderStatus(orderID, OrderStatusCreated, OrderStatusCancelled)
}

func (s *OrderService) DeleteOrder(orderID string, callerID string, isAdmin bool) error {
	if orderID == "" {
		return errors.New("order_id не может быть пустым")
	}
	order, err := s.repo.GetOrder(orderID)
	if err != nil {
		return err
	}
	if !isAdmin && order.UserID != callerID {
		return ErrPermissionDenied
	}
	return s.repo.DeleteOrder(orderID)
}

func (s *OrderService) UpdateOrderStatus(orderID string, fromStatus, toStatus int32, callerID string, isAdmin bool) (repository.Order, error) {
	order, err := s.repo.GetOrder(orderID)
	if err != nil {
		return repository.Order{}, err
	}
	if !isAdmin && order.UserID != callerID {
		return repository.Order{}, ErrPermissionDenied
	}

	if order.Status != fromStatus {
		return repository.Order{}, ErrStatusMismatch
	}
	if !manualTransitionAllowed(fromStatus, toStatus) {
		return repository.Order{}, ErrInvalidTransition
	}
	updated, err := s.repo.UpdateOrderStatus(orderID, fromStatus, toStatus)
	if err != nil {
		return repository.Order{}, err
	}

	if fromStatus == OrderStatusCreated && toStatus == OrderStatusPaid && s.jobs != nil {
		_, _ = s.jobs.Enqueue(repository.OrderJob{
			OrderID:    orderID,
			FromStatus: OrderStatusPaid,
			ToStatus:   OrderStatusShipped,
			RunAt:      time.Now().Add(s.delay),
		})
	}
	return updated, nil
}

func (s *OrderService) ListOrderJobs(orderID string) ([]repository.OrderJob, error) {
	if orderID == "" {
		return nil, errors.New("order_id не может быть пустым")
	}
	if _, err := s.repo.GetOrder(orderID); err != nil {
		return nil, err
	}
	if s.jobs == nil {
		return []repository.OrderJob{}, nil
	}
	return s.jobs.ListByOrder(orderID)
}

func manualTransitionAllowed(from, to int32) bool {
	switch {
	case from == OrderStatusCreated && to == OrderStatusPaid:
		return true
	case from == OrderStatusCreated && to == OrderStatusCancelled:
		return true
	default:
		return false
	}
}
