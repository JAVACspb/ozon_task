package graph

import "github.com/ozosanek/ozon_task/internal/service"

type Resolver struct {
	service *service.Service
}

func NewResolver(service *service.Service) *Resolver {
	return &Resolver{service: service}
}
