package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riyadennis/pbac/internal/model"
)

var ErrNotFound = errors.New("policy not found")

type PolicyRepository struct {
	db *pgxpool.Pool
}

func NewPolicyRepository(db *pgxpool.Pool) *PolicyRepository {
	return &PolicyRepository{db: db}
}

func (r *PolicyRepository) Create(ctx context.Context, req *model.CreatePolicyRequest) (*model.Policy, error) {
	p := &model.Policy{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Module:      req.Module,
		Content:     req.Content,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	q := `INSERT INTO policies (id, name, description, module, content, created_at, updated_at)
          VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(ctx, q, p.ID, p.Name, p.Description, p.Module, p.Content, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create policy: %w", err)
	}

	return p, nil
}

func (r *PolicyRepository) GetByID(ctx context.Context, id string) (*model.Policy, error) {
	q := `SELECT id, name, description, module, content, created_at, updated_at
          FROM policies WHERE id = $1`

	row := r.db.QueryRow(ctx, q, id)
	p := &model.Policy{}

	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Module, &p.Content, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get policy by id: %w", err)
	}

	return p, nil
}

func (r *PolicyRepository) List(ctx context.Context) ([]*model.Policy, error) {
	q := `SELECT id, name, description, module, content, created_at, updated_at
          FROM policies ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []*model.Policy
	for rows.Next() {
		p := &model.Policy{}
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Module, &p.Content, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, p)
	}

	return policies, nil
}

func (r *PolicyRepository) Update(ctx context.Context, id string, req *model.UpdatePolicyRequest) (*model.Policy, error) {
	q := `UPDATE policies SET name=$1, description=$2, module=$3, content=$4, updated_at=$5
          WHERE id=$6
          RETURNING id, name, description, module, content, created_at, updated_at`

	row := r.db.QueryRow(ctx, q, req.Name, req.Description, req.Module, req.Content, time.Now().UTC(), id)
	p := &model.Policy{}

	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Module, &p.Content, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update policy: %w", err)
	}

	return p, nil
}

func (r *PolicyRepository) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM policies WHERE id=$1`
	result, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}