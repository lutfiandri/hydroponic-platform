package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestPocketBaseLoginLookupAndSave(t *testing.T) {
	loginCalls, lookupCalls, saveCalls := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/collections/_superusers/auth-with-password":
			loginCalls++
			var body map[string]string
			if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&body) != nil || !reflect.DeepEqual(body, map[string]string{"identity": "admin@example.com", "password": "secret"}) {
				t.Errorf("unexpected login request: method=%s body=%v", r.Method, body)
			}
			w.Write([]byte(`{"token":"test-token","record":{"id":"admin-record","email":"admin@example.com"}}`))
		case "/api/collections/device/records":
			lookupCalls++
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "test-token" || r.URL.Query().Get("filter") != `device_code = "tank\"\\01"` || r.URL.Query().Get("perPage") != "1" {
				t.Errorf("unexpected lookup: method=%s header=%q query=%s", r.Method, r.Header.Get("Authorization"), r.URL.RawQuery)
			}
			w.Write([]byte(`{"page":1,"perPage":1,"totalItems":1,"totalPages":1,"items":[{"id":"device-record","device_code":"tank\"\\01"}]}`))
		case "/api/collections/device_reading/records":
			saveCalls++
			var body map[string]any
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "test-token" || r.Header.Get("Content-Type") != "application/json" || json.NewDecoder(r.Body).Decode(&body) != nil {
				t.Errorf("unexpected save request")
			}
			want := map[string]any{"device_id": "device-record", "ppm": float64(0), "has_ppm": true}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("saved = %#v, want %#v", body, want)
			}
			w.Write([]byte(`{"id":"reading-record","device_id":"device-record","ppm":0,"has_ppm":true}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	pb := pocketBase{baseURL: server.URL, client: server.Client()}
	if err := pb.login(context.Background(), "admin@example.com", "secret"); err != nil {
		t.Fatal(err)
	}
	id, err := pb.findDevice(context.Background(), "tank\"\\01")
	if err != nil || id != "device-record" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if err := pb.saveRecord(context.Background(), "device_reading", map[string]any{"device_id": id, "ppm": float64(0), "has_ppm": true}); err != nil {
		t.Fatal(err)
	}
	if loginCalls != 1 || lookupCalls != 1 || saveCalls != 1 {
		t.Fatalf("login=%d lookup=%d save=%d", loginCalls, lookupCalls, saveCalls)
	}
}

func TestPocketBaseRejectsInvalidResponsesWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name      string
		operation string
		status    int
		body      string
	}{
		{"login unauthorized", "login", 401, `{"message":"unauthorized"}`},
		{"login invalid JSON", "login", 200, `{`},
		{"login missing token", "login", 200, `{"record":{"id":"admin"}}`},
		{"lookup server failure", "lookup", 500, `{"message":"unavailable"}`},
		{"lookup invalid JSON", "lookup", 200, `{`},
		{"lookup unknown device", "lookup", 200, `{"page":1,"perPage":1,"totalItems":0,"totalPages":0,"items":[]}`},
		{"lookup missing id", "lookup", 200, `{"items":[{}]}`},
		{"save server failure", "save", 500, `{"message":"unavailable"}`},
		{"save unauthorized", "save", 401, `{"message":"unauthorized"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			pb := pocketBase{baseURL: server.URL, client: server.Client()}
			var err error
			switch tc.operation {
			case "login":
				err = pb.login(context.Background(), "admin@example.com", "secret")
			case "lookup":
				_, err = pb.findDevice(context.Background(), "tank-01")
			case "save":
				err = pb.saveRecord(context.Background(), "device_event", map[string]any{"device_id": "device"})
			}
			if err == nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestPocketBaseRequestHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pb := pocketBase{baseURL: "http://127.0.0.1:1", client: &http.Client{Timeout: time.Second}}
	if err := pb.saveRecord(ctx, "device_event", map[string]any{}); err == nil {
		t.Fatal("expected canceled request to fail")
	}
}
