package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	orderv1.UnimplementedOrderServiceServer
	service *service.OrderService
	logger  *slog.Logger
}

func NewGRPCHandler(service *service.OrderService, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (h *GRPCHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	order, err := h.service.CreateOrder(ctx, domain.Order{
		CustomerID:      req.GetCustomerId(),
		Items:           convertCreateOrderItems(req.GetItems()),
		ShippingAddress: convertAddress(req.GetShippingAddress()),
	})
	if err != nil {
		return nil, mapOrderError(err)
	}

	h.logger.InfoContext(ctx, "order created", slog.String("order_id", order.ID), slog.String("customer_id", order.CustomerID))

	return &orderv1.CreateOrderResponse{
		Order: convertOrderToProto(order),
	}, nil
}

func (h *GRPCHandler) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	order, err := h.service.GetOrderByID(ctx, req.GetOrderId())
	if err != nil {
		return nil, mapOrderError(err)
	}

	return &orderv1.GetOrderResponse{
		Order: convertOrderToProto(order),
	}, nil
}

func (h *GRPCHandler) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	order, err := h.service.CancelOrder(ctx, req.GetOrderId())
	if err != nil {
		return nil, mapOrderError(err)
	}

	h.logger.InfoContext(ctx, "order cancelled", slog.String("order_id", order.ID), slog.String("customer_id", order.CustomerID))

	return &orderv1.CancelOrderResponse{
		Order: convertOrderToProto(order),
	}, nil
}

func convertCreateOrderItems(items []*orderv1.CreateOrderItem) []domain.OrderItem {
	converted := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		converted = append(converted, domain.OrderItem{
			ProductID:   item.GetProductId(),
			SKU:         item.GetSku(),
			ProductName: item.GetProductName(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   convertMoney(item.GetUnitPrice()),
		})
	}
	return converted
}

func convertOrderItemsToProto(items []domain.OrderItem) []*orderv1.OrderItem {
	converted := make([]*orderv1.OrderItem, 0, len(items))
	for _, item := range items {
		converted = append(converted, &orderv1.OrderItem{
			ProductId:   item.ProductID,
			Sku:         item.SKU,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   convertMoneyToProto(item.UnitPrice),
			TotalPrice:  convertMoneyToProto(item.TotalPrice),
		})
	}
	return converted
}

func convertMoney(money *orderv1.Money) domain.Money {
	if money == nil {
		return domain.Money{}
	}

	return domain.Money{
		Currency:    money.GetCurrency(),
		AmountCents: money.GetAmountCents(),
	}
}

func convertMoneyToProto(money domain.Money) *orderv1.Money {
	return &orderv1.Money{
		Currency:    money.Currency,
		AmountCents: money.AmountCents,
	}
}

func convertAddress(address *orderv1.Address) domain.Address {
	if address == nil {
		return domain.Address{}
	}

	return domain.Address{
		Country:    address.GetCountry(),
		City:       address.GetCity(),
		Street:     address.GetStreet(),
		PostalCode: address.GetPostalCode(),
		House:      address.GetHouse(),
		Apartment:  address.GetApartment(),
	}
}

func convertAddressToProto(address domain.Address) *orderv1.Address {
	return &orderv1.Address{
		Country:    address.Country,
		City:       address.City,
		Street:     address.Street,
		PostalCode: address.PostalCode,
		House:      address.House,
		Apartment:  address.Apartment,
	}
}

func convertOrderToProto(order domain.Order) *orderv1.Order {
	return &orderv1.Order{
		OrderId:         order.ID,
		CustomerId:      order.CustomerID,
		Items:           convertOrderItemsToProto(order.Items),
		TotalAmount:     convertMoneyToProto(order.TotalAmount),
		Status:          order.Status,
		ShippingAddress: convertAddressToProto(order.ShippingAddress),
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       order.UpdatedAt.Format(time.RFC3339),
	}
}

func mapOrderError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidOrder), errors.Is(err, domain.ErrInvalidOrderID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrOrderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrOrderAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
