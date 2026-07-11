package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/chat"
)

func (h *Handler) PostChatStream(w http.ResponseWriter, r *http.Request) {
	var req gen.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}

	if h.chat == nil {
		writeInternalError(w, "chat is not configured")
		return
	}

	if len(req.Messages) == 0 {
		writeValidationError(w, chat.ErrEmptyMessages.Error())
		return
	}

	messages := make([]chat.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = chat.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	provider := ""
	if req.Provider != nil {
		provider = string(*req.Provider)
	}
	model := ""
	if req.Model != nil {
		model = *req.Model
	}

	initSSE(w)

	_ = h.chat.StreamChat(r.Context(), provider, model, messages, func(event string, data any) error {
		return writeSSE(w, event, data)
	})
}
