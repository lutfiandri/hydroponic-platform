package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type DeviceEventHandler struct {
	FindDevice func(context.Context, string) (string, error)
	SaveRecord func(context.Context, string, map[string]any) error
}

func (h DeviceEventHandler) Handle(ctx context.Context, deviceCode string, payload []byte) error {
	var event struct {
		OccurredAt string `json:"occurred_at"`
		EventType  string `json:"event_type"`
		Source     string `json:"source"`
		Status     string `json:"status"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.New("invalid event JSON")
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, event.OccurredAt)
	if err != nil {
		return errors.New("occurred_at must be RFC3339")
	}
	if strings.TrimSpace(event.EventType) == "" {
		return errors.New("event_type is required")
	}
	if event.Source != "automatic" && event.Source != "manual" && event.Source != "system" {
		return errors.New("source must be automatic, manual, or system")
	}
	if event.Status != "pending" && event.Status != "success" && event.Status != "failed" {
		return errors.New("status must be pending, success, or failed")
	}
	deviceID, err := h.FindDevice(ctx, deviceCode)
	if err != nil {
		return fmt.Errorf("find device: %w", err)
	}
	record := map[string]any{
		"device_id":   deviceID,
		"occurred_at": occurredAt.UTC().Format("2006-01-02 15:04:05.000Z"),
		"event_type":  event.EventType,
		"source":      event.Source,
		"status":      event.Status,
		"message":     event.Message,
	}
	if err := h.SaveRecord(ctx, "device_event", record); err != nil {
		return fmt.Errorf("save event: %w", err)
	}
	return nil
}
