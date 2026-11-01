package handler

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestDeviceReadingMapsMeasurements(t *testing.T) {
	var saved map[string]any
	h := DeviceReadingHandler{
		FindDevice: func(ctx context.Context, code string) (string, error) {
			if code != "tank-01" {
				t.Fatalf("device code = %q", code)
			}
			return "device-record", nil
		},
		SaveRecord: func(ctx context.Context, collection string, record map[string]any) error {
			if collection != "device_reading" {
				t.Fatalf("collection = %q", collection)
			}
			saved = record
			return nil
		},
	}
	err := h.Handle(context.Background(), "tank-01", []byte(`{"recorded_at":"2026-11-01T17:00:00+07:00","water_level":75,"ppm":0,"water_temperature":null,"device_id":"forged"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"device_id": "device-record", "recorded_at": "2026-11-01 10:00:00.000Z",
		"water_level": float64(75), "has_water_level": true,
		"ppm": float64(0), "has_ppm": true,
		"water_temperature": float64(0), "has_water_temperature": false,
		"battery_percentage": float64(0), "has_battery_percentage": false,
	}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("saved = %#v, want %#v", saved, want)
	}
}

func TestDeviceReadingRejectsInvalidPayload(t *testing.T) {
	for _, payload := range []string{
		`{`, `null`, `[]`,
		`{"recorded_at":"bad","ppm":1}`,
		`{"ppm":1}`,
		`{"recorded_at":"2026-11-01T10:00:00Z"}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","ppm":null}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","ppm":-1}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","ppm":"100"}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","water_level":-1}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","water_level":101}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","battery_percentage":-1}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","battery_percentage":101}`,
		`{"recorded_at":"2026-11-01T10:00:00Z","ppm":1} {}`,
	} {
		t.Run(payload, func(t *testing.T) {
			h := DeviceReadingHandler{
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

func TestDeviceReadingAcceptsBoundaries(t *testing.T) {
	h := DeviceReadingHandler{
		FindDevice: func(context.Context, string) (string, error) { return "device", nil },
		SaveRecord: func(_ context.Context, _ string, record map[string]any) error {
			if record["battery_percentage"] != float64(100) || record["water_temperature"] != float64(-5) {
				t.Fatalf("measurements = %#v", record)
			}
			return nil
		},
	}
	if err := h.Handle(context.Background(), "tank-01", []byte(`{"recorded_at":"2026-11-01T10:00:00Z","water_level":0,"ppm":0,"water_temperature":-5,"battery_percentage":100}`)); err != nil {
		t.Fatal(err)
	}
}

func TestDeviceReadingReturnsStorageErrorsWithoutRetry(t *testing.T) {
	for _, lookupFails := range []bool{true, false} {
		t.Run(map[bool]string{true: "unknown device", false: "save failure"}[lookupFails], func(t *testing.T) {
			failure := errors.New("unavailable")
			lookups, saves := 0, 0
			h := DeviceReadingHandler{
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
			err := h.Handle(context.Background(), "tank-01", []byte(`{"recorded_at":"2026-11-01T10:00:00Z","ppm":1}`))
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
