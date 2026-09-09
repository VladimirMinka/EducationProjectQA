package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "awesomeProject/gen/store/api/delivery/v1"
	"awesomeProject/internal/auth"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
)

type DeliveryHandler struct {
	pb.UnimplementedDeliveryServiceServer
	svc *service.DeliveryService
}

func NewDeliveryHandler(svc *service.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{svc: svc}
}

func mapAddress(a repository.Address) *pb.Address {
	created := ""
	if !a.CreatedAt.IsZero() {
		created = a.CreatedAt.UTC().Format(time.RFC3339)
	}
	return &pb.Address{
		Id:            a.ID,
		UserId:        a.UserID,
		Title:         a.Title,
		City:          a.City,
		Street:        a.Street,
		Building:      a.Building,
		Apartment:     a.Apartment,
		PostalCode:    a.PostalCode,
		RecipientName: a.RecipientName,
		Phone:         a.Phone,
		IsDefault:     a.IsDefault,
		CreatedAt:     created,
	}
}

func mapPickup(p repository.PickupPoint) *pb.PickupPoint {
	return &pb.PickupPoint{
		Id:        p.ID,
		Code:      p.Code,
		City:      p.City,
		Address:   p.Address,
		WorkHours: p.WorkHours,
		Active:    p.Active,
	}
}

func (h *DeliveryHandler) ListAddresses(ctx context.Context, req *pb.ListAddressesRequest) (*pb.ListAddressesResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}
	if !callerIsAdmin(ctx) {
		if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
	}
	addrs, err := h.svc.ListAddresses(req.GetUserId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		return nil, mapDeliveryErr(err)
	}
	out := make([]*pb.Address, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, mapAddress(a))
	}
	return &pb.ListAddressesResponse{Addresses: out}, nil
}

func (h *DeliveryHandler) CreateAddress(ctx context.Context, req *pb.CreateAddressRequest) (*pb.AddressResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}
	if !callerIsAdmin(ctx) {
		if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
	}
	addr, err := h.svc.CreateAddress(repository.Address{
		UserID:        req.GetUserId(),
		Title:         req.GetTitle(),
		City:          req.GetCity(),
		Street:        req.GetStreet(),
		Building:      req.GetBuilding(),
		Apartment:     req.GetApartment(),
		PostalCode:    req.GetPostalCode(),
		RecipientName: req.GetRecipientName(),
		Phone:         req.GetPhone(),
		IsDefault:     req.GetIsDefault(),
	}, auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		return nil, mapDeliveryErr(err)
	}
	return &pb.AddressResponse{Address: mapAddress(addr)}, nil
}

func (h *DeliveryHandler) GetAddress(ctx context.Context, req *pb.GetAddressRequest) (*pb.AddressResponse, error) {
	if req.GetUserId() == "" || req.GetAddressId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id и address_id обязательны")
	}
	if !callerIsAdmin(ctx) {
		if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
	}
	addr, err := h.svc.GetAddress(req.GetUserId(), req.GetAddressId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		return nil, mapDeliveryErr(err)
	}
	return &pb.AddressResponse{Address: mapAddress(addr)}, nil
}

func (h *DeliveryHandler) UpdateAddress(ctx context.Context, req *pb.UpdateAddressRequest) (*pb.AddressResponse, error) {
	if req.GetUserId() == "" || req.GetAddressId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id и address_id обязательны")
	}
	if !callerIsAdmin(ctx) {
		if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
	}
	addr, err := h.svc.UpdateAddress(repository.Address{
		ID:            req.GetAddressId(),
		UserID:        req.GetUserId(),
		Title:         req.GetTitle(),
		City:          req.GetCity(),
		Street:        req.GetStreet(),
		Building:      req.GetBuilding(),
		Apartment:     req.GetApartment(),
		PostalCode:    req.GetPostalCode(),
		RecipientName: req.GetRecipientName(),
		Phone:         req.GetPhone(),
		IsDefault:     req.GetIsDefault(),
	}, auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		return nil, mapDeliveryErr(err)
	}
	return &pb.AddressResponse{Address: mapAddress(addr)}, nil
}

func (h *DeliveryHandler) DeleteAddress(ctx context.Context, req *pb.DeleteAddressRequest) (*pb.DeleteAddressResponse, error) {
	if req.GetUserId() == "" || req.GetAddressId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id и address_id обязательны")
	}
	if !callerIsAdmin(ctx) {
		if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
	}
	if err := h.svc.DeleteAddress(req.GetUserId(), req.GetAddressId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx)); err != nil {
		return nil, mapDeliveryErr(err)
	}
	return &pb.DeleteAddressResponse{}, nil
}

func (h *DeliveryHandler) ListPickupPoints(ctx context.Context, req *pb.ListPickupPointsRequest) (*pb.ListPickupPointsResponse, error) {
	points, err := h.svc.ListPickupPoints()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка списка ПВЗ: %v", err)
	}
	out := make([]*pb.PickupPoint, 0, len(points))
	for _, p := range points {
		out = append(out, mapPickup(p))
	}
	return &pb.ListPickupPointsResponse{PickupPoints: out}, nil
}

func (h *DeliveryHandler) GetPickupPoint(ctx context.Context, req *pb.GetPickupPointRequest) (*pb.PickupPointResponse, error) {
	p, err := h.svc.GetPickupPoint(req.GetPickupPointId())
	if err != nil {
		return nil, mapDeliveryErr(err)
	}
	return &pb.PickupPointResponse{PickupPoint: mapPickup(p)}, nil
}

func mapDeliveryErr(err error) error {
	if mapped := mapUserNotFound(err); mapped != nil {
		return mapped
	}
	if errors.Is(err, repository.ErrAddressNotFound) || errors.Is(err, service.ErrAddressNotFound) {
		return status.Error(codes.NotFound, "адрес не найден")
	}
	if errors.Is(err, repository.ErrPickupPointNotFound) || errors.Is(err, service.ErrPickupPointNotFound) {
		return status.Error(codes.NotFound, "пункт выдачи не найден")
	}
	if errors.Is(err, service.ErrDeliveryDenied) {
		return status.Error(codes.PermissionDenied, "нет доступа")
	}
	if errors.Is(err, service.ErrInactivePickup) {
		return status.Error(codes.FailedPrecondition, "пункт выдачи неактивен")
	}
	msg := err.Error()
	if strings.Contains(msg, "обязателен") || strings.Contains(msg, "не используется") || strings.Contains(msg, "delivery_method") {
		return status.Error(codes.InvalidArgument, msg)
	}
	return status.Errorf(codes.Internal, "ошибка доставки: %v", err)
}
