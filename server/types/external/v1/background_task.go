package external

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

type BackgroundTaskKind string

const (
	BackgroundTaskKindDeleteTimer      BackgroundTaskKind = "delete_timer"
	BackgroundTaskKindDeleteInboxEvent BackgroundTaskKind = "delete_inbox_event"
)

type BackgroundTask struct {
	Kind            BackgroundTaskKind `msgpack:"kind"`
	DeduplicationID DirectiveID        `msgpack:"deduplication_id"`
	Payload         []byte             `msgpack:"payload"`
}

// DeleteTimerPayload is the msgpack-encoded Payload for BackgroundTaskKindDeleteTimer.
type DeleteTimerPayload struct {
	WFID    WFID      `msgpack:"wf_id"`
	Kind    TimerKind `msgpack:"kind"`
	TimerID TimerID   `msgpack:"timer_id"`
}

// DeleteInboxEventPayload is the msgpack-encoded Payload for BackgroundTaskKindDeleteInboxEvent.
type DeleteInboxEventPayload struct {
	SeqID uint64 `msgpack:"seq_id"`
}

// DeriveBgTaskID produces a deterministic DirectiveID for a background task derived from
// the originating directive. This ensures dedup across redeliveries.
func DeriveBgTaskID(parentID DirectiveID, taskKind BackgroundTaskKind) DirectiveID {
	return DirectiveID(fmt.Sprintf("bg.%s.%s", taskKind, parentID))
}

// NewDeleteTimerTask constructs a BackgroundTask that deletes a specific timer by WFID, kind, and timer ID.
func NewDeleteTimerTask(parentID DirectiveID, wfID WFID, kind TimerKind, timerID TimerID) (BackgroundTask, error) {
	payload, err := msgpack.Marshal(&DeleteTimerPayload{WFID: wfID, Kind: kind, TimerID: timerID})
	if err != nil {
		return BackgroundTask{}, fmt.Errorf("failed to marshal delete timer payload: %w", err)
	}
	return BackgroundTask{
		Kind:            BackgroundTaskKindDeleteTimer,
		DeduplicationID: DeriveBgTaskID(parentID, BackgroundTaskKindDeleteTimer),
		Payload:         payload,
	}, nil
}

// NewDeleteInboxEventTask constructs a BackgroundTask that deletes an inbox event by stream sequence.
func NewDeleteInboxEventTask(parentID DirectiveID, seqID uint64) (BackgroundTask, error) {
	payload, err := msgpack.Marshal(&DeleteInboxEventPayload{SeqID: seqID})
	if err != nil {
		return BackgroundTask{}, fmt.Errorf("failed to marshal delete inbox event payload: %w", err)
	}
	return BackgroundTask{
		Kind:            BackgroundTaskKindDeleteInboxEvent,
		DeduplicationID: DeriveBgTaskID(parentID, BackgroundTaskKindDeleteInboxEvent),
		Payload:         payload,
	}, nil
}
