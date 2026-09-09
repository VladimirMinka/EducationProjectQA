package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "awesomeProject/gen/store/api/catalog/v1"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
)

type CatalogHandler struct {
	pb.UnimplementedCatalogServiceServer
	svc *service.CatalogService
}

func NewCatalogHandler(svc *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func mapProduct(p repository.Product) *pb.Product {
	return &pb.Product{
		Id:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		PriceCents:    p.PriceCents,
		StockQuantity: p.StockQuantity,
		Brand:         p.Brand,
		CategoryId:    p.CategoryID,
	}
}

func (h *CatalogHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	product, err := h.svc.GetProduct(req.GetProductId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Товар с ID '%s' не найден", req.GetProductId())
		}
		if err.Error() == "product_id cannot be empty" {
			return nil, status.Error(codes.InvalidArgument, "ID товара не может быть пустым")
		}
		return nil, status.Errorf(codes.Internal, "Внутренняя ошибка сервера: %v", err)
	}
	return &pb.ProductResponse{Product: mapProduct(product)}, nil
}

func (h *CatalogHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	params := service.ListProductsParams{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
		Query:     req.GetQ(),
		Brand:     req.GetBrand(),
		CategoryID: req.GetCategoryId(),
		Sort:      req.GetSort(),
	}
	if req.GetMinPriceCents() > 0 {
		params.HasMinPrice = true
		params.MinPriceCents = req.GetMinPriceCents()
	}
	if req.GetMaxPriceCents() > 0 {
		params.HasMaxPrice = true
		params.MaxPriceCents = req.GetMaxPriceCents()
	}
	if req.InStock != nil {
		v := req.GetInStock()
		params.InStock = &v
	}

	result, err := h.svc.ListProducts(params)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPageSize) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrInvalidPageToken) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrInvalidPriceRange) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrInvalidSort) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Не удалось получить список товаров: %v", err)
	}

	pbProducts := make([]*pb.Product, 0, len(result.Products))
	for _, p := range result.Products {
		pbProducts = append(pbProducts, mapProduct(p))
	}
	return &pb.ListProductsResponse{
		Products:      pbProducts,
		NextPageToken: result.NextPageToken,
		TotalCount:    result.TotalCount,
	}, nil
}

func (h *CatalogHandler) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	cats, err := h.svc.ListCategories()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Не удалось получить категории: %v", err)
	}
	out := make([]*pb.Category, 0, len(cats))
	for _, c := range cats {
		out = append(out, &pb.Category{Id: c.ID, Slug: c.Slug, Name: c.Name})
	}
	return &pb.ListCategoriesResponse{Categories: out}, nil
}

func (h *CatalogHandler) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.CategoryResponse, error) {
	c, err := h.svc.GetCategory(req.GetCategoryId())
	if err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, "категория не найдена")
		}
		if err.Error() == "category_id cannot be empty" {
			return nil, status.Error(codes.InvalidArgument, "category_id обязателен")
		}
		return nil, status.Errorf(codes.Internal, "ошибка категории: %v", err)
	}
	return &pb.CategoryResponse{Category: &pb.Category{Id: c.ID, Slug: c.Slug, Name: c.Name}}, nil
}
