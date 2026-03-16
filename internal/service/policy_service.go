package service

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
	"github.com/riyadennis/pbac/internal/model"
	"github.com/riyadennis/pbac/internal/repository"
)

type PolicyService struct {
	repo *repository.PolicyRepository
}

func NewPolicyService(repo *repository.PolicyRepository) *PolicyService {
	return &PolicyService{repo: repo}
}

func (s *PolicyService) Create(ctx context.Context, req *model.CreatePolicyRequest) (*model.Policy, error) {
	if err := validateRegoContent(req.Module, req.Content); err != nil {
		return nil, fmt.Errorf("invalid rego content: %w", err)
	}
	return s.repo.Create(ctx, req)
}

func (s *PolicyService) GetByID(ctx context.Context, id string) (*model.Policy, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PolicyService) List(ctx context.Context) ([]*model.Policy, error) {
	return s.repo.List(ctx)
}

func (s *PolicyService) Update(ctx context.Context, id string, req *model.UpdatePolicyRequest) (*model.Policy, error) {
	if err := validateRegoContent(req.Module, req.Content); err != nil {
		return nil, fmt.Errorf("invalid rego content: %w", err)
	}
	return s.repo.Update(ctx, id, req)
}

func (s *PolicyService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *PolicyService) Evaluate(ctx context.Context, id string, req *model.EvaluateRequest) (*model.EvaluateResponse, error) {
	policy, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	query := req.Query
	if query == "" {
		query = fmt.Sprintf("data.%s.allow", policy.Module)
	}

	r := rego.New(
		rego.Query(query),
		rego.Module(policy.Module+".rego", policy.Content),
		rego.Input(req.Input),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("evaluate policy: %w", err)
	}

	resp := &model.EvaluateResponse{}
	if len(rs) > 0 && len(rs[0].Expressions) > 0 {
		resp.Result = rs[0].Expressions[0].Value
		if allow, ok := rs[0].Expressions[0].Value.(bool); ok {
			resp.Allow = allow
		}
	}

	return resp, nil
}

func validateRegoContent(module, content string) error {
	r := rego.New(
		rego.Query("data."+module+".allow"),
		rego.Module(module+".rego", content),
	)

	if _, err := r.PrepareForEval(context.Background()); err != nil {
		return err
	}

	return nil
}