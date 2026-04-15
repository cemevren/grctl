package machine

import (
	"context"
	"fmt"
	"log/slog"

	intr "grctl/server/types"
	ext "grctl/server/types/external/v1"

	"github.com/vmihailenco/msgpack/v5"
)

type TimerCanceler interface {
	CancelTimer(ctx context.Context, wfID ext.WFID, kind ext.TimerKind, timerID ext.TimerID) error
}

type InboxEventDeleter interface {
	DeleteInboxEvent(ctx context.Context, seqID uint64) error
}

type BgTaskHandler struct {
	timers        TimerCanceler
	inbox         InboxEventDeleter
	maxDeliveries uint64
}

func NewBgTaskHandler(timers TimerCanceler, inbox InboxEventDeleter, maxDeliveries uint64) *BgTaskHandler {
	return &BgTaskHandler{
		timers:        timers,
		inbox:         inbox,
		maxDeliveries: maxDeliveries,
	}
}

func (h *BgTaskHandler) Handle(ctx context.Context, task ext.BackgroundTask, numDelivered uint64) intr.HandleResult {
	if numDelivered > h.maxDeliveries {
		slog.Warn("background task exceeded max deliveries, discarding",
			"kind", task.Kind,
			"deduplicationID", task.DeduplicationID,
			"numDelivered", numDelivered,
			"maxDeliveries", h.maxDeliveries,
		)
		return intr.Processed()
	}

	switch task.Kind {
	case ext.BackgroundTaskKindDeleteTimer:
		return h.handleDeleteTimer(ctx, task)
	case ext.BackgroundTaskKindDeleteInboxEvent:
		return h.handleDeleteInboxEvent(ctx, task)
	default:
		slog.Warn("unknown background task kind, discarding", "kind", task.Kind)
		return intr.Processed()
	}
}

func (h *BgTaskHandler) handleDeleteTimer(ctx context.Context, task ext.BackgroundTask) intr.HandleResult {
	var payload ext.DeleteTimerPayload
	if err := msgpack.Unmarshal(task.Payload, &payload); err != nil {
		slog.Error("failed to unmarshal delete timer payload, discarding",
			"deduplicationID", task.DeduplicationID,
			"error", err,
		)
		return intr.Processed()
	}

	if err := h.timers.CancelTimer(ctx, payload.WFID, payload.Kind, payload.TimerID); err != nil {
		slog.Warn("failed to cancel timer, will retry",
			"wfID", payload.WFID,
			"kind", payload.Kind,
			"timerID", payload.TimerID,
			"error", err,
		)
		return intr.Retryable(NackDelay)
	}

	slog.Debug("timer deleted by background task",
		"wfID", payload.WFID,
		"kind", payload.Kind,
		"timerID", payload.TimerID,
	)
	return intr.Processed()
}

func (h *BgTaskHandler) handleDeleteInboxEvent(ctx context.Context, task ext.BackgroundTask) intr.HandleResult {
	var payload ext.DeleteInboxEventPayload
	if err := msgpack.Unmarshal(task.Payload, &payload); err != nil {
		slog.Error("failed to unmarshal delete inbox event payload, discarding",
			"deduplicationID", task.DeduplicationID,
			"error", err,
		)
		return intr.Processed()
	}

	if err := h.inbox.DeleteInboxEvent(ctx, payload.SeqID); err != nil {
		slog.Warn("failed to delete inbox event, will retry",
			"seqID", payload.SeqID,
			"error", fmt.Sprintf("%v", err),
		)
		return intr.Retryable(NackDelay)
	}

	slog.Debug("inbox event deleted by background task", "seqID", payload.SeqID)
	return intr.Processed()
}
