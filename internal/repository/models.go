package repository

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("product not found")
var ErrUserNotFound = errors.New("user not found")
var ErrUserHasActiveOrders = errors.New("user has active orders")
var ErrPromocodeNotFound = errors.New("promocode not found")
var ErrOrderNotFound = errors.New("order not found")
var ErrAddressNotFound = errors.New("address not found")
var ErrPickupPointNotFound = errors.New("pickup point not found")
var ErrCategoryNotFound = errors.New("category not found")

// Terminal order statuses: user may be deleted once only these remain.
const (
	OrderStatusCancelled int32 = 4
	OrderStatusCompleted int32 = 5
)

const (
	DeliveryMethodCourier int32 = 1
	DeliveryMethodPickup  int32 = 2
)

const (
	CourierDeliveryFeeCents   int64 = 29900
	FreeDeliverySubtotalCents int64 = 500000
)

const (
	OrderJobPending   = "pending"
	OrderJobDone      = "done"
	OrderJobFailed    = "failed"
	OrderJobCancelled = "cancelled"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

type Category struct {
	ID   string
	Slug string
	Name string
}

type Product struct {
	ID            string
	Name          string
	Description   string
	PriceCents    int64
	StockQuantity int32
	Brand         string
	CategoryID    string
}

type ProductFilter struct {
	Query         string
	Brand         string
	CategoryID    string
	MinPriceCents *int64
	MaxPriceCents *int64
	InStock       *bool
	Sort          string
	Limit         int
	Offset        int
}

type CartItem struct {
	ProductID string
	Quantity  int32
}

type Cart struct {
	UserID    string
	Items     map[string]CartItem
	Promocode string
	UpdatedAt time.Time
}

type OrderItem struct {
	ProductID  string
	Quantity   int32
	PriceCents int64
}

type DeliverySnapshot struct {
	RecipientName string `json:"recipient_name,omitempty"`
	Phone         string `json:"phone,omitempty"`
	City          string `json:"city,omitempty"`
	Street        string `json:"street,omitempty"`
	Building      string `json:"building,omitempty"`
	Apartment     string `json:"apartment,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	AddressLine   string `json:"address_line,omitempty"`
	PickupCode    string `json:"pickup_code,omitempty"`
	PickupAddress string `json:"pickup_address,omitempty"`
	WorkHours     string `json:"work_hours,omitempty"`
}

type Order struct {
	ID               string
	UserID           string
	Items            []OrderItem
	TotalAmountCents int64
	Status           int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeliveryMethod   int32
	DeliveryFeeCents int64
	DeliverySnapshot DeliverySnapshot
}

type Address struct {
	ID            string
	UserID        string
	Title         string
	City          string
	Street        string
	Building      string
	Apartment     string
	PostalCode    string
	RecipientName string
	Phone         string
	IsDefault     bool
	CreatedAt     time.Time
}

type PickupPoint struct {
	ID        string
	Code      string
	City      string
	Address   string
	WorkHours string
	Active    bool
}

type OrderJob struct {
	ID          string
	OrderID     string
	FromStatus  int32
	ToStatus    int32
	RunAt       time.Time
	Status      string
	Attempts    int32
	LastError   string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

const (
	DiscountPercent    = "percent"
	DiscountFixedCents = "fixed_cents"
)

type Promocode struct {
	Code          string
	DiscountType  string
	DiscountValue int64
	Active        bool
	ExpiresAt     *time.Time
}
