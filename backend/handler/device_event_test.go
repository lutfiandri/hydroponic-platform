package handler

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestDeviceEventMapsFields(t *testing.T) {
	var saved map[string]any
	h := DeviceEventHandler{
		FindDevice: func(_ context.Context, code string) (string, error) {
			if code != "tank-01" {
				t.Fatalf("device code = %q", code)
			}
			return "device-record", nil
		},
		SaveRecord: func(_ context.Context, collection string, record map[string]any) error {
			if collection != "device_event" {
				t.Fatalf("collection = %q", collection)
			}
			saved = record
			return nil
		},
	}
	err := h.Handle(context.Background(), "tank-01", []byte(`{"occurred_at":"2026-11-01T17:01:00+07:00","event_type":"custom_event","source":"automatic","status":"success","message":"Pompa dinyalakan","device_id":"forged"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"device_id": "device-record", "occurred_at": "2026-11-01 10:01:00.000Z",
		"event_type": "custom_event", "source": "automatic", "status": "success", "message": "Pompa dinyalakan",
	}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("saved = %#v, want %#v", saved, want)
	}
}

func TestDeviceEventRejectsInvalidPayload(t *testing.T) {
	for _, payload := range []string{
		`{`, `null`, `[]`,
		`{"occurred_at":"bad","event_type":"pump_on","source":"manual","status":"success"}`,
		`{"event_type":"pump_on","source":"manual","status":"success"}`,
		`{"occurred_at":"2026-11-01T10:00:00Z","event_type":"  ","source":"manual","status":"success"}`,
		`{"occurred_at":"2026-11-01T10:00:00Z","source":"manual","status":"success"}`,
		`{"occurred_at":"2026-11-01T10:00:00Z","event_type":"pump_on","source":"invalid","status":"success"}`,
		`{"occurred_at":"2026-11-01T10:00:00Z","event_type":"pump_on","status":"success"}`,
		`{"occurred_at":"2026-11-01T10:00:00Z","event_type":"pump_on","source":"manual","status":"invalid"}`,
		`{"occurred_at":"2026-11-01T10:00:00Z","event_type":"pump_on","source":"manual"}`,
	} {
		t.Run(payload, func(t *testing.T) {
			h := DeviceEventHandler{
				FindDevice: func(context.Context, string) (string, error) {
					t.Fatal("invalid payload reached device lookup")
					return "", nil
				},
				SaveRecord: func(context.Context, string, map[string]any) error {
					t.Fatal("invalid payload reached save")
					return nil
				},
			}
			if err := h.Handle(context.Background(), "tank-01", []byte(payload)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestDeviceEventAcceptsAllEnums(t *testing.T) {
	for _, source := range []string{"automatic", "manual", "system"} {
		for _, status := range []string{"pending", "success", "failed"} {
			h := DeviceEventHandler{
				FindDevice: func(context.Context, string) (string, error) { return "device", nil },
				SaveRecord: func(_ context.Context, _ string, record map[string]any) error {
					if record["message"] != "" || record["source"] != source || record["status"] != status {
						t.Fatalf("event = %#v", record)
					}
					return nil
				},
			}
			payload := `{"occurred_at":"2026-11-01T10:00:00Z","event_type":"pump_on","source":"` + source + `","status":"` + status + `"}`
			if err := h.Handle(context.Background(), "tank-01", []byte(payload)); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestDeviceEventReturnsStorageErrorsWithoutRetry(t *testing.T) {
	for _, lookupFails := range []bool{true, false} {
		t.Run(map[bool]string{true: "unknown device", false: "save failure"}[lookupFails], func(t *testing.T) {
			failure := errors.New("unavailable")
			lookups, saves := 0, 0
			h := DeviceEventHandler{
				FindDevice: func(context.Context, string) (string, error) {
					lookups++
					if lookupFails {
						return "", failure
					}
					return "device", nil
				},
				SaveRecord: func(context.Context, string, map[string]any) error {
					saves++
					return failure
				},
			}
			err := h.Handle(context.Background(), "tank-01", []byte(`{"occurred_at":"2026-11-01T10:00:00Z","event_type":"pump_on","source":"system","status":"failed"}`))
			wantSaves := 1
			if lookupFails {
				wantSaves = 0
			}
			if !errors.Is(err, failure) || lookups != 1 || saves != wantSaves {
				t.Fatalf("err=%v lookups=%d saves=%d", err, lookups, saves)
			}
		})
	}
}
