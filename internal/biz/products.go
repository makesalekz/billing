package biz

import "gitlab.calendaria.team/services/finance/invoices/internal/data"

type ProductUseCase struct {
	productRepo data.ProductRepo
}

func NewProductUseCase(productRepo data.ProductRepo) *ProductUseCase {
	return &ProductUseCase{
		productRepo: productRepo,
	}
}
