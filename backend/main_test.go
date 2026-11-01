package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setConfigEnv(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for name, value := range map[string]string{
		"MQTT_BROKER": "", "MQTT_CLIENT_ID": "", "MQTT_USERNAME": "", "MQTT_PASSWORD": "",
		"POCKETBASE_URL": "", "POCKETBASE_ADMIN_EMAIL": "admin@example.com", "POCKETBASE_ADMIN_PASSWORD": "secret",
	} {
		t.Setenv(name, value)
	}
}

func TestLoadConfigReadsParentEnvFromBackend(t *testing.T) {
	setConfigEnv(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "backend"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("POCKETBASE_ADMIN_EMAIL=file@example.com\nPOCKETBASE_ADMIN_PASSWORD='secret # with spaces'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(root, "backend"))
	for _, name := range []string{"POCKETBASE_ADMIN_EMAIL", "POCKETBASE_ADMIN_PASSWORD"} {
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.adminEmail != "file@example.com" || cfg.adminPassword != "secret # with spaces" {
		t.Fatal("parent .env credentials were not loaded correctly")
	}
}

func TestLoadConfigEnvPrecedence(t *testing.T) {
	setConfigEnv(t)
	root := t.TempDir()
	backend := filepath.Join(root, "backend")
	if err := os.Mkdir(backend, 0o700); err != nil {
		t.Fatal(err)
	}
	for filename, content := range map[string]string{
		filepath.Join(root, ".env"):    "POCKETBASE_ADMIN_EMAIL=root@example.com\nPOCKETBASE_ADMIN_PASSWORD=root-secret\nMQTT_BROKER=tcp://root-broker:1883\n",
		filepath.Join(backend, ".env"): "POCKETBASE_ADMIN_EMAIL=local@example.com\nPOCKETBASE_ADMIN_PASSWORD=local-secret\n",
	} {
		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(backend)
	for _, name := range []string{"POCKETBASE_ADMIN_PASSWORD", "MQTT_BROKER"} {
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.adminEmail != "admin@example.com" || cfg.adminPassword != "local-secret" || cfg.broker != "tcp://root-broker:1883" {
		t.Fatal("expected process env > local .env > parent .env")
	}
}

func TestLoadConfigRejectsMalformedEnvWithoutLeakingSecrets(t *testing.T) {
	setConfigEnv(t)
	if err := os.WriteFile(".env", []byte("POCKETBASE_ADMIN_PASSWORD='do-not-leak-this-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected malformed .env to fail")
	}
	if strings.Contains(err.Error(), "do-not-leak-this-secret") {
		t.Fatal("dotenv error leaked a secret")
	}
}

func TestLoadConfigDefaultsAndOverrides(t *testing.T) {
	setConfigEnv(t)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.broker != "tcp://localhost:1883" || cfg.clientID != "hydroponic-backend" || cfg.pocketBaseURL != "http://localhost:8090" || cfg.adminEmail != "admin@example.com" || cfg.adminPassword != "secret" {
		t.Fatal("incorrect default configuration")
	}
	t.Setenv("MQTT_BROKER", "ssl://broker.example.com:8883")
	t.Setenv("MQTT_CLIENT_ID", "backend-02")
	t.Setenv("MQTT_USERNAME", "backend")
	t.Setenv("MQTT_PASSWORD", "mqtt-secret")
	t.Setenv("POCKETBASE_URL", "https://pb.example.com/")
	cfg, err = loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.broker != "ssl://broker.example.com:8883" || cfg.clientID != "backend-02" || cfg.username != "backend" || cfg.password != "mqtt-secret" || cfg.pocketBaseURL != "https://pb.example.com" {
		t.Fatal("configuration overrides were not applied")
	}
}

func TestLoadConfigRejectsInvalidSettings(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"POCKETBASE_ADMIN_EMAIL", ""},
		{"POCKETBASE_ADMIN_PASSWORD", ""},
		{"MQTT_BROKER", "localhost:1883"},
		{"MQTT_BROKER", "http://localhost:1883"},
		{"MQTT_BROKER", "tcp://localhost"},
		{"MQTT_BROKER", "tcp://localhost:0"},
		{"MQTT_BROKER", "tcp://localhost:99999"},
		{"MQTT_BROKER", "tcp://user:secret@localhost:1883"},
		{"MQTT_BROKER", "tcp://localhost:1883/path"},
		{"MQTT_PASSWORD", "secret-without-username"},
		{"MQTT_CLIENT_ID", " "},
		{"POCKETBASE_URL", "/"},
		{"POCKETBASE_URL", "not-a-url"},
		{"POCKETBASE_URL", "ftp://localhost:8090"},
		{"POCKETBASE_URL", "http://user:secret@localhost:8090"},
		{"POCKETBASE_URL", "http://localhost:8090?token=secret"},
	} {
		t.Run(tc.name+"="+tc.value, func(t *testing.T) {
			setConfigEnv(t)
			t.Setenv(tc.name, tc.value)
			if _, err := loadConfig(); err == nil {
				t.Fatal("expected invalid configuration to fail")
			}
		})
	}
}

func TestHandleMessageRoutesAndRejectsTopics(t *testing.T) {
	for _, tc := range []struct {
		topic     string
		retained  bool
		wantRoute string
		wantError bool
	}{
		{"devices/tank-01/reading", false, "reading", false},
		{"devices/table-01/event", false, "event", false},
		{"devices/tank-01/reading", true, "", false},
		{"devices/tank-01/event", true, "", false},
		{"devices//reading", false, "", true},
		{"other/tank-01/reading", false, "", true},
		{"devices/tank-01/unknown", false, "", true},
		{"devices/tank-01/reading/extra", false, "", true},
		{"devices/tank-01", false, "", true},
	} {
		t.Run(tc.topic+map[bool]string{true: " retained", false: ""}[tc.retained], func(t *testing.T) {
			route, calls := "", 0
			callback := func(name string) func(context.Context, string, []byte) error {
				return func(_ context.Context, code string, payload []byte) error {
					calls++
					route = name
					if (code != "tank-01" && code != "table-01") || string(payload) != "payload" {
						t.Fatal("routing changed device identity or payload")
					}
					return nil
				}
			}
			err := handleMessage(context.Background(), tc.topic, []byte("payload"), tc.retained, callback("reading"), callback("event"))
			if (err != nil) != tc.wantError || route != tc.wantRoute || calls > 1 || (tc.wantRoute == "" && calls != 0) {
				t.Fatalf("err=%v route=%q calls=%d", err, route, calls)
			}
		})
	}
}

func TestHandleMessageDoesNotRetryErrors(t *testing.T) {
	failure := errors.New("storage unavailable")
	calls := 0
	callback := func(context.Context, string, []byte) error {
		calls++
		return failure
	}
	err := handleMessage(context.Background(), "devices/tank-01/reading", []byte(`{}`), false, callback, callback)
	if !errors.Is(err, failure) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
