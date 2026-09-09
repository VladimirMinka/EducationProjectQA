package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "awesomeProject/gen/store/api/order/v1"
	"awesomeProject/internal/auth"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
)

type OrderHandler struct {
	pb.UnimplementedOrderServiceServer
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func mapOrderToProto(order repository.Order) *pb.Order {
	var items []*pb.OrderItem
	for _, item := range order.Items {
		items = append(items, &pb.OrderItem{
			ProductId:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
		})
	}

	created := ""
	if !order.CreatedAt.IsZero() {
		created = order.CreatedAt.UTC().Format(time.RFC3339)
	}

	snap := order.DeliverySnapshot
	info := &pb.DeliveryInfo{
		RecipientName: snap.RecipientName,
		Phone:         snap.Phone,
		City:          snap.City,
		Street:        snap.Street,
		Building:      snap.Building,
		Apartment:     snap.Apartment,
		PostalCode:    snap.PostalCode,
		AddressLine:   snap.AddressLine,
		PickupCode:    snap.PickupCode,
		PickupAddress: snap.PickupAddress,
		WorkHours:     snap.WorkHours,
	}

	return &pb.Order{
		Id:               order.ID,
		UserId:           order.UserID,
		Items:            items,
		TotalAmountCents: order.TotalAmountCents,
		Status:           pb.OrderStatus(order.Status),
		CreatedAt:        created,
		DeliveryMethod:   pb.DeliveryMethod(order.DeliveryMethod),
		DeliveryFeeCents: order.DeliveryFeeCents,
		DeliveryInfo:     info,
	}
}

func callerIsAdmin(ctx context.Context) bool {
	return auth.RoleFromContext(ctx) == "admin"
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}
	if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	order, err := h.svc.CreateOrder(service.CreateOrderInput{
		UserID:        req.GetUserId(),
		Method:        int32(req.GetDeliveryMethod()),
		AddressID:     req.GetAddressId(),
		PickupPointID: req.GetPickupPointId(),
	})
	if err != nil {
		if mapped := mapUserNotFound(err); mapped != nil {
			return nil, mapped
		}
		if strings.Contains(err.Error(), "корзина пуста") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		if mapped := mapDeliveryErr(err); status.Code(mapped) != codes.Internal {
			return nil, mapped
		}
		return nil, status.Errorf(codes.Internal, "ошибка создания заказа: %v", err)
	}

	return &pb.OrderResponse{Order: mapOrderToProto(order)}, nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}
	if !callerIsAdmin(ctx) {
		if err := ensureCallerMatchesUser(ctx, req.GetUserId()); err != nil {
			return nil, err
		}
	}

	orders, err := h.svc.ListOrders(req.GetUserId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		if mapped := mapUserNotFound(err); mapped != nil {
			return nil, mapped
		}
		return nil, mapOrderErr(err)
	}

	out := make([]*pb.Order, 0, len(orders))
	for _, order := range orders {
		out = append(out, mapOrderToProto(order))
	}
	return &pb.ListOrdersResponse{Orders: out}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id обязателен")
	}

	order, err := h.svc.GetOrder(req.GetOrderId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		return nil, mapOrderErr(err)
	}

	return &pb.OrderResponse{Order: mapOrderToProto(order)}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id обязателен")
	}

	order, err := h.svc.CancelOrder(req.GetOrderId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx))
	if err != nil {
		return nil, mapOrderErr(err)
	}
	return &pb.OrderResponse{Order: mapOrderToProto(order)}, nil
}

func (h *OrderHandler) DeleteOrder(ctx context.Context, req *pb.DeleteOrderRequest) (*pb.DeleteOrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id обязателен")
	}

	if err := h.svc.DeleteOrder(req.GetOrderId(), auth.UserIDFromContext(ctx), callerIsAdmin(ctx)); err != nil {
		return nil, mapOrderErr(err)
	}
	return &pb.DeleteOrderResponse{}, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id обязателен")
	}
	if req.GetFromStatus() == pb.OrderStatus_ORDER_STATUS_UNSPECIFIED || req.GetToStatus() == pb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "from_status и to_status обязательны")
	}

	order, err := h.svc.UpdateOrderStatus(
		req.GetOrderId(),
		int32(req.GetFromStatus()),
		int32(req.GetToStatus()),
		auth.UserIDFromContext(ctx),
		callerIsAdmin(ctx),
	)
	if err != nil {
		return nil, mapOrderErr(err)
	}
	return &pb.OrderResponse{Order: mapOrderToProto(order)}, nil
}

func (h *OrderHandler) ListOrderJobs(ctx context.Context, req *pb.ListOrderJobsRequest) (*pb.ListOrderJobsResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id обязателен")
	}
	jobs, err := h.svc.ListOrderJobs(req.GetOrderId())
	if err != nil {
		return nil, mapOrderErr(err)
	}
	out := make([]*pb.OrderJob, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, mapOrderJob(j))
	}
	return &pb.ListOrderJobsResponse{Jobs: out}, nil
}

func mapOrderJob(j repository.OrderJob) *pb.OrderJob {
	runAt := ""
	if !j.RunAt.IsZero() {
		runAt = j.RunAt.UTC().Format(time.RFC3339)
	}
	created := ""
	if !j.CreatedAt.IsZero() {
		created = j.CreatedAt.UTC().Format(time.RFC3339)
	}
	processed := ""
	if j.ProcessedAt != nil {
		processed = j.ProcessedAt.UTC().Format(time.RFC3339)
	}
	statusVal := pb.OrderJobStatus_ORDER_JOB_STATUS_UNSPECIFIED
	switch j.Status {
	case repository.OrderJobPending:
		statusVal = pb.OrderJobStatus_ORDER_JOB_STATUS_PENDING
	case repository.OrderJobDone:
		statusVal = pb.OrderJobStatus_ORDER_JOB_STATUS_DONE
	case repository.OrderJobFailed:
		statusVal = pb.OrderJobStatus_ORDER_JOB_STATUS_FAILED
	case repository.OrderJobCancelled:
		statusVal = pb.OrderJobStatus_ORDER_JOB_STATUS_CANCELLED
	}
	return &pb.OrderJob{
		Id:          j.ID,
		OrderId:     j.OrderID,
		FromStatus:  pb.OrderStatus(j.FromStatus),
		ToStatus:    pb.OrderStatus(j.ToStatus),
		RunAt:       runAt,
		Status:      statusVal,
		Attempts:    j.Attempts,
		LastError:   j.LastError,
		CreatedAt:   created,
		ProcessedAt: processed,
	}
}

func mapOrderErr(err error) error {
	if errors.Is(err, repository.ErrOrderNotFound) || errors.Is(err, service.ErrOrderNotFound) {
		return status.Error(codes.NotFound, "заказ не найден")
	}
	if errors.Is(err, service.ErrPermissionDenied) {
		return status.Error(codes.PermissionDenied, "нет доступа к заказу")
	}
	if errors.Is(err, service.ErrInvalidTransition) {
		return status.Error(codes.FailedPrecondition, "недопустимый переход статуса")
	}
	if errors.Is(err, service.ErrStatusMismatch) {
		return status.Error(codes.FailedPrecondition, "текущий статус не совпадает с from_status")
	}
	if strings.Contains(err.Error(), "order_id не может быть пустым") {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Errorf(codes.Internal, "ошибка заказа: %v", err)
}
