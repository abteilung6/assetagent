package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/chat"
)

type ChatService interface {
	Chat(ctx context.Context, provider, model string, messages []chat.Message) (chat.Result, error)
	StreamChat(ctx context.Context, provider, model string, messages []chat.Message, write chat.StreamWriter) error
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

	pageCtx, err := pageContextFromRequest(req.Context)
	if err != nil {
		writeValidationError(w, err.Error())
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

	ctx, endSpan := chat.StartHTTPChatSpan(r.Context(), provider, model, len(messages), false, chat.LastUserMessage(messages))
	defer endSpan()
	ctx = chat.WithPageContext(ctx, pageCtx)

	result, err := h.chat.Chat(ctx, provider, model, messages)
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

func pageContextFromRequest(raw *gen.ChatPageContext) (chat.PageContext, error) {
	if raw == nil {
		return chat.PageContext{}, nil
	}
	pageCtx := chat.PageContext{}
	if raw.Route != nil {
		pageCtx.Route = *raw.Route
	}
	if raw.BaselineId != nil {
		pageCtx.BaselineID = raw.BaselineId.String()
	}
	if raw.ReviewId != nil {
		pageCtx.ReviewID = raw.ReviewId.String()
	}
	if raw.ForecastId != nil {
		pageCtx.ForecastID = raw.ForecastId.String()
	}
	if raw.YyyyMm != nil {
		pageCtx.YYYYMM = *raw.YyyyMm
	}
	if raw.From != nil {
		pageCtx.From = raw.From.Time.Format("2006-01-02")
	}
	if raw.To != nil {
		pageCtx.To = raw.To.Time.Format("2006-01-02")
	}
	if raw.Tab != nil {
		pageCtx.Tab = *raw.Tab
	}
	if raw.Q != nil {
		pageCtx.Q = *raw.Q
	}
	if err := chat.ValidatePageContext(pageCtx); err != nil {
		return chat.PageContext{}, err
	}
	return pageCtx, nil
}
