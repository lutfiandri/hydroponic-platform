package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type DeviceReadingHandler struct {
	FindDevice func(context.Context, string) (string, error)
	SaveRecord func(context.Context, string, map[string]any) error
}

func (h DeviceReadingHandler) Handle(ctx context.Context, deviceCode string, payload []byte) error {
	var reading struct {
		RecordedAt        string   `json:"recorded_at"`
		WaterLevel        *float64 `json:"water_level"`
		PPM               *float64 `json:"ppm"`
		WaterTemperature  *float64 `json:"water_temperature"`
		BatteryPercentage *float64 `json:"battery_percentage"`
	}
	if err := json.Unmarshal(payload, &reading); err != nil {
		return errors.New("invalid reading JSON")
	}
	recordedAt, err := time.Parse(time.RFC3339Nano, reading.RecordedAt)
	if err != nil {
		return errors.New("recorded_at must be RFC3339")
	}
	if reading.WaterLevel == nil && reading.PPM == nil && reading.WaterTemperature == nil && reading.BatteryPercentage == nil {
		return errors.New("at least one sensor measurement is required")
	}
	if reading.WaterLevel != nil && (*reading.WaterLevel < 0 || *reading.WaterLevel > 100) {
		return errors.New("water_level must be between 0 and 100")
	}
	if reading.PPM != nil && *reading.PPM < 0 {
		return errors.New("ppm must be nonnegative")
	}
	if reading.BatteryPercentage != nil && (*reading.BatteryPercentage < 0 || *reading.BatteryPercentage > 100) {
		return errors.New("battery_percentage must be between 0 and 100")
	}
	deviceID, err := h.FindDevice(ctx, deviceCode)
	if err != nil {
		return fmt.Errorf("find device: %w", err)
	}
	record := map[string]any{
		"device_id":   deviceID,
		"recorded_at": recordedAt.UTC().Format("2006-01-02 15:04:05.000Z"),
	}
	for field, value := range map[string]*float64{
		"water_level":        reading.WaterLevel,
		"ppm":                reading.PPM,
		"water_temperature":  reading.WaterTemperature,
		"battery_percentage": reading.BatteryPercentage,
	} {
		record[field] = float64(0)
		record["has_"+field] = value != nil
		if value != nil {
			record[field] = *value
		}
	}
	if err := h.SaveRecord(ctx, "device_reading", record); err != nil {
		return fmt.Errorf("save reading: %w", err)
	}
	return nil
}
