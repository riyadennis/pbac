package model

import "time"

type Policy struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Module      string    `json:"module" db:"module"`    // Rego module name e.g. "authz.policy"
	Content     string    `json:"content" db:"content"` // Rego policy content
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type EvaluateRequest struct {
	Input map[string]interface{} `json:"input"`
	Query string                 `json:"query"` // e.g. "data.authz.allow"
}

type EvaluateResponse struct {
	Result interface{} `json:"result"`
	Allow  bool        `json:"allow"`
}

type CreatePolicyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Module      string `json:"module"`
	Content     string `json:"content"`
}

type UpdatePolicyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Module      string `json:"module"`
	Content     string `json:"content"`
}