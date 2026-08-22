package opa

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
)

// Engine evaluates RBAC policies using OPA/Rego.
type Engine struct {
	query rego.PreparedEvalQuery
}

const defaultPolicy = `
package berth.authz

import future.keywords.if
import future.keywords.in

default allow := false

# Role hierarchy
role_hierarchy = {
	"OWNER": ["OWNER", "ADMIN", "COLLABORATOR", "VIEWER"],
	"ADMIN": ["ADMIN", "COLLABORATOR", "VIEWER"],
	"COLLABORATOR": ["COLLABORATOR", "VIEWER"],
	"VIEWER": ["VIEWER"],
	"MEMBER": ["MEMBER"]
}

# Action permissions per role
permissions = {
	"OWNER": ["create", "read", "update", "delete", "invite", "manage"],
	"ADMIN": ["create", "read", "update", "delete", "invite"],
	"COLLABORATOR": ["create", "read", "update"],
	"VIEWER": ["read"],
	"MEMBER": ["read"]
}

allow if {
	role := input.user_role
	action := input.action
	action in permissions[role]
}

allow if {
	input.user_id == input.resource_owner_id
}
`

// NewEngine compiles the default policy.
func NewEngine() (*Engine, error) {
	r := rego.New(
		rego.Query("data.berth.authz.allow"),
		rego.Module("policy.rego", defaultPolicy),
	)

	query, err := r.PrepareForEval(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to compile OPA policy: %w", err)
	}

	return &Engine{query: query}, nil
}

// Eval evaluates a policy decision.
func (e *Engine) Eval(ctx context.Context, input map[string]interface{}) (bool, error) {
	results, err := e.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("policy evaluation failed: %w", err)
	}
	if len(results) == 0 {
		return false, nil
	}
	if len(results[0].Expressions) == 0 {
		return false, nil
	}
	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return false, nil
	}
	return allowed, nil
}
