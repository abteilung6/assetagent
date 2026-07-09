package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/chat"
)

type ChatService interface {
	Chat(ctx context.Context, messages []chat.Message) (chat.Result, error)
}

func (h *Handler) PostChat(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.chat.Chat(r.Context(), messages)
	if err != nil {
		writeChatError(w, err)
		return
	}

	toolCalls := make([]gen.ChatToolCall, len(result.ToolCalls))
	for i, call := range result.ToolCalls {
		input, err := rawJSONToMap(call.Input)
		if err != nil {
			writeInternalError(w, "failed to encode tool input")
			return
		}
		toolResult, err := rawJSONToMap(call.Result)
		if err != nil {
			writeInternalError(w, "failed to encode tool result")
			return
		}

		toolCalls[i] = gen.ChatToolCall{
			Name:   call.Name,
			Input:  input,
			Result: toolResult,
		}
	}

	writeJSON(w, http.StatusOK, gen.ChatResponse{
		Answer:    result.Answer,
		ToolCalls: toolCalls,
	})
}

func rawJSONToMap(raw json.RawMessage) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}

	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]interface{}{}, nil
	}

	return out, nil
}
