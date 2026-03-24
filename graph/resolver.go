package graph

import "github.com/riyadennis/pbac/internal/service"

type Resolver struct {
	svc *service.PolicyService
}

func NewResolver(svc *service.PolicyService) *Resolver {
	return &Resolver{svc: svc}
}