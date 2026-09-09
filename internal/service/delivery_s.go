package service

import (
	"errors"
	"fmt"
	"strings"

	"awesomeProject/internal/repository"
)

type DeliveryRepository interface {
	ListAddresses(userID string) ([]repository.Address, error)
	GetAddress(userID, addressID string) (repository.Address, error)
	CreateAddress(addr repository.Address) (repository.Address, error)
	UpdateAddress(addr repository.Address) (repository.Address, error)
	DeleteAddress(userID, addressID string) error
	ListActivePickupPoints() ([]repository.PickupPoint, error)
	GetPickupPoint(id string) (repository.PickupPoint, error)
}

type DeliveryService struct {
	repo  DeliveryRepository
	users UserProvider
}

func NewDeliveryService(repo DeliveryRepository, users UserProvider) *DeliveryService {
	return &DeliveryService{repo: repo, users: users}
}

var (
	ErrAddressNotFound     = repository.ErrAddressNotFound
	ErrPickupPointNotFound = repository.ErrPickupPointNotFound
	ErrDeliveryDenied      = errors.New("нет доступа")
	ErrInactivePickup      = errors.New("пункт выдачи неактивен")
)

func (s *DeliveryService) requireUser(userID string) error {
	if userID == "" {
		return errors.New("user_id не может быть пустым")
	}
	if _, err := s.users.GetUser(userID); err != nil {
		return fmt.Errorf("пользователь не найден: %w", err)
	}
	return nil
}

func (s *DeliveryService) ListAddresses(userID, callerID string, isAdmin bool) ([]repository.Address, error) {
	if err := s.requireUser(userID); err != nil {
		return nil, err
	}
	if !isAdmin && userID != callerID {
		return nil, ErrDeliveryDenied
	}
	return s.repo.ListAddresses(userID)
}

func (s *DeliveryService) GetAddress(userID, addressID, callerID string, isAdmin bool) (repository.Address, error) {
	if err := s.requireUser(userID); err != nil {
		return repository.Address{}, err
	}
	if !isAdmin && userID != callerID {
		return repository.Address{}, ErrDeliveryDenied
	}
	if addressID == "" {
		return repository.Address{}, errors.New("address_id обязателен")
	}
	return s.repo.GetAddress(userID, addressID)
}

func (s *DeliveryService) CreateAddress(addr repository.Address, callerID string, isAdmin bool) (repository.Address, error) {
	if err := s.requireUser(addr.UserID); err != nil {
		return repository.Address{}, err
	}
	if !isAdmin && addr.UserID != callerID {
		return repository.Address{}, ErrDeliveryDenied
	}
	if err := validateAddress(addr); err != nil {
		return repository.Address{}, err
	}
	return s.repo.CreateAddress(addr)
}

func (s *DeliveryService) UpdateAddress(addr repository.Address, callerID string, isAdmin bool) (repository.Address, error) {
	if err := s.requireUser(addr.UserID); err != nil {
		return repository.Address{}, err
	}
	if !isAdmin && addr.UserID != callerID {
		return repository.Address{}, ErrDeliveryDenied
	}
	if addr.ID == "" {
		return repository.Address{}, errors.New("address_id обязателен")
	}
	if err := validateAddress(addr); err != nil {
		return repository.Address{}, err
	}
	return s.repo.UpdateAddress(addr)
}

func (s *DeliveryService) DeleteAddress(userID, addressID, callerID string, isAdmin bool) error {
	if err := s.requireUser(userID); err != nil {
		return err
	}
	if !isAdmin && userID != callerID {
		return ErrDeliveryDenied
	}
	if addressID == "" {
		return errors.New("address_id обязателен")
	}
	return s.repo.DeleteAddress(userID, addressID)
}

func (s *DeliveryService) ListPickupPoints() ([]repository.PickupPoint, error) {
	return s.repo.ListActivePickupPoints()
}

func (s *DeliveryService) GetPickupPoint(id string) (repository.PickupPoint, error) {
	if id == "" {
		return repository.PickupPoint{}, errors.New("pickup_point_id обязателен")
	}
	return s.repo.GetPickupPoint(id)
}

// ResolveForOrder validates delivery choice and builds fee + snapshot.
func (s *DeliveryService) ResolveForOrder(userID string, method int32, addressID, pickupPointID string, merchandiseSubtotal int64) (int32, int64, repository.DeliverySnapshot, error) {
	switch method {
	case repository.DeliveryMethodCourier:
		if addressID == "" {
			return 0, 0, repository.DeliverySnapshot{}, errors.New("address_id обязателен для курьерской доставки")
		}
		if pickupPointID != "" {
			return 0, 0, repository.DeliverySnapshot{}, errors.New("pickup_point_id не используется для курьерской доставки")
		}
		addr, err := s.repo.GetAddress(userID, addressID)
		if err != nil {
			return 0, 0, repository.DeliverySnapshot{}, err
		}
		fee := repository.CourierDeliveryFeeCents
		if merchandiseSubtotal >= repository.FreeDeliverySubtotalCents {
			fee = 0
		}
		line := formatAddressLine(addr)
		return method, fee, repository.DeliverySnapshot{
			RecipientName: addr.RecipientName,
			Phone:         addr.Phone,
			City:          addr.City,
			Street:        addr.Street,
			Building:      addr.Building,
			Apartment:     addr.Apartment,
			PostalCode:    addr.PostalCode,
			AddressLine:   line,
		}, nil

	case repository.DeliveryMethodPickup:
		if pickupPointID == "" {
			return 0, 0, repository.DeliverySnapshot{}, errors.New("pickup_point_id обязателен для самовывоза")
		}
		if addressID != "" {
			return 0, 0, repository.DeliverySnapshot{}, errors.New("address_id не используется для самовывоза")
		}
		pp, err := s.repo.GetPickupPoint(pickupPointID)
		if err != nil {
			return 0, 0, repository.DeliverySnapshot{}, err
		}
		if !pp.Active {
			return 0, 0, repository.DeliverySnapshot{}, ErrInactivePickup
		}
		return method, 0, repository.DeliverySnapshot{
			PickupCode:    pp.Code,
			PickupAddress: pp.City + ", " + pp.Address,
			City:          pp.City,
			WorkHours:     pp.WorkHours,
			AddressLine:   pp.City + ", " + pp.Address,
		}, nil

	default:
		return 0, 0, repository.DeliverySnapshot{}, errors.New("delivery_method обязателен (COURIER или PICKUP)")
	}
}

func validateAddress(addr repository.Address) error {
	if strings.TrimSpace(addr.City) == "" {
		return errors.New("city обязателен")
	}
	if strings.TrimSpace(addr.Street) == "" {
		return errors.New("street обязателен")
	}
	if strings.TrimSpace(addr.Building) == "" {
		return errors.New("building обязателен")
	}
	if strings.TrimSpace(addr.RecipientName) == "" {
		return errors.New("recipient_name обязателен")
	}
	if strings.TrimSpace(addr.Phone) == "" {
		return errors.New("phone обязателен")
	}
	return nil
}

func formatAddressLine(addr repository.Address) string {
	parts := []string{addr.City, addr.Street + ", " + addr.Building}
	if addr.Apartment != "" {
		parts[1] += ", кв. " + addr.Apartment
	}
	if addr.PostalCode != "" {
		parts = append([]string{addr.PostalCode}, parts...)
	}
	return strings.Join(parts, ", ")
}
