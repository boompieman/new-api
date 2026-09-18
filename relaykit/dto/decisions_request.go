package dto

import (
	"encoding/json"
	"net/http"

	"github.com/QuantumNous/new-api/relaykit/types"
)

// DecisionsRequest is a TypeSafe / OpenRouter System One (decisions) request.
// RawBody preserves the original JSON so unknown fields are forwarded intact.
type DecisionsRequest struct {
	Model     string          `json:"model"`
	State     json.RawMessage `json:"state"`
	Questions json.RawMessage `json:"questions"`
	RawBody   json.RawMessage `json:"-"`
}

func (r *DecisionsRequest) GetTokenCountMeta() *types.TokenCountMeta {
	combineText := ""
	if len(r.RawBody) > 0 {
		combineText = string(r.RawBody)
	}
	return &types.TokenCountMeta{
		CombineText: combineText,
		TokenType:   types.TokenTypeTokenizer,
	}
}

func (r *DecisionsRequest) IsStream(_ *http.Request) bool {
	return false
}

func (r *DecisionsRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
