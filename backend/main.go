package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"hidroponic-platform-backend/handler"
)

type config struct {
	broker        string
	clientID      string
	username      string
	password      string
	pocketBaseURL string
	adminEmail    string
	adminPassword string
}

func loadConfig() (config, error) {
	for _, path := range []string{".env", "../.env"} {
		if err := godotenv.Load(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return config{}, fmt.Errorf("cannot load %s: check file permissions and dotenv syntax", path)
		}
	}
	cfg := config{
		broker:        os.Getenv("MQTT_BROKER"),
		clientID:      os.Getenv("MQTT_CLIENT_ID"),
		username:      os.Getenv("MQTT_USERNAME"),
		password:      os.Getenv("MQTT_PASSWORD"),
		pocketBaseURL: os.Getenv("POCKETBASE_URL"),
		adminEmail:    os.Getenv("POCKETBASE_ADMIN_EMAIL"),
		adminPassword: os.Getenv("POCKETBASE_ADMIN_PASSWORD"),
	}
	if cfg.broker == "" {
		cfg.broker = "tcp://localhost:1883"
	}
	if cfg.clientID == "" {
		cfg.clientID = "hydroponic-backend"
	}
	if cfg.pocketBaseURL == "" {
		cfg.pocketBaseURL = "http://localhost:8090"
	}
	if strings.TrimSpace(cfg.adminEmail) == "" || cfg.adminPassword == "" {
		return config{}, errors.New("POCKETBASE_ADMIN_EMAIL and POCKETBASE_ADMIN_PASSWORD are required")
	}
	if strings.TrimSpace(cfg.clientID) == "" {
		return config{}, errors.New("MQTT_CLIENT_ID must not be blank")
	}
	if cfg.password != "" && cfg.username == "" {
		return config{}, errors.New("MQTT_PASSWORD requires MQTT_USERNAME")
	}
	broker, err := url.Parse(cfg.broker)
	if err != nil || broker.Hostname() == "" || broker.User != nil || broker.RawQuery != "" || broker.Fragment != "" || broker.Path != "" || (broker.Scheme != "tcp" && broker.Scheme != "ssl" && broker.Scheme != "tls") {
		return config{}, errors.New("MQTT_BROKER must be a tcp://, ssl://, or tls:// host:port URL without credentials or a path")
	}
	port, err := strconv.Atoi(broker.Port())
	if err != nil || port < 1 || port > 65535 {
		return config{}, errors.New("MQTT_BROKER requires a valid port")
	}
	cfg.pocketBaseURL = strings.TrimRight(cfg.pocketBaseURL, "/")
	pbURL, err := url.Parse(cfg.pocketBaseURL)
	if err != nil || pbURL.Hostname() == "" || pbURL.User != nil || pbURL.RawQuery != "" || pbURL.Fragment != "" || pbURL.Path != "" || (pbURL.Scheme != "http" && pbURL.Scheme != "https") {
		return config{}, errors.New("POCKETBASE_URL must be an http(s) origin without credentials, path, or query")
	}
	return cfg, nil
}

func handleMessage(ctx context.Context, topic string, payload []byte, retained bool, reading, event func(context.Context, string, []byte) error) error {
	if retained {
		return nil
	}
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" || strings.TrimSpace(parts[1]) == "" {
		return errors.New("invalid MQTT device topic")
	}
	switch parts[2] {
	case "reading":
		return reading(ctx, parts[1], payload)
	case "event":
		return event(ctx, parts[1], payload)
	default:
		return errors.New("unknown MQTT device topic")
	}
}

func run(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	pb := &pocketBase{
		baseURL: cfg.pocketBaseURL,
		client: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	if err := pb.login(ctx, cfg.adminEmail, cfg.adminPassword); err != nil {
		return fmt.Errorf("PocketBase login: %w", err)
	}
	reading := handler.DeviceReadingHandler{FindDevice: pb.findDevice, SaveRecord: pb.saveRecord}
	event := handler.DeviceEventHandler{FindDevice: pb.findDevice, SaveRecord: pb.saveRecord}
	failures := make(chan error, 1)
	opts := mqtt.NewClientOptions().AddBroker(cfg.broker).SetClientID(cfg.clientID).
		SetUsername(cfg.username).SetPassword(cfg.password).
		SetAutoReconnect(true).SetConnectRetry(false).
		SetConnectTimeout(10 * time.Second).SetKeepAlive(60 * time.Second)
	opts.SetDefaultPublishHandler(func(_ mqtt.Client, message mqtt.Message) {
		slog.Info("MQTT message received", "topic", message.Topic(), "payload", string(message.Payload()))
		if err := handleMessage(ctx, message.Topic(), message.Payload(), message.Retained(), reading.Handle, event.Handle); err != nil {
			attrs := []any{"topic", message.Topic(), "error", err}
			if parts := strings.Split(message.Topic(), "/"); len(parts) == 3 {
				attrs = append(attrs, "device_code", parts[1])
			}
			slog.Error("MQTT message dropped", attrs...)
		}
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		token := client.SubscribeMultiple(map[string]byte{"devices/+/reading": 1, "devices/+/event": 1}, nil)
		err := mqtt.WaitTokenTimeout(token, 10*time.Second)
		if err == nil {
			for topic, qos := range token.(*mqtt.SubscribeToken).Result() {
				if qos == 0x80 {
					err = fmt.Errorf("broker rejected subscription to %s", topic)
					break
				}
			}
		}
		if err != nil {
			select {
			case failures <- fmt.Errorf("MQTT subscription: %w", err):
			default:
			}
			return
		}
		slog.Info("MQTT subscriptions ready")
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		slog.Warn("MQTT connection lost; reconnecting", "error", err)
	})
	client := mqtt.NewClient(opts)
	defer client.Disconnect(1000)
	if err := mqtt.WaitTokenTimeout(client.Connect(), 15*time.Second); err != nil {
		return fmt.Errorf("MQTT connect: %w", err)
	}
	select {
	case <-ctx.Done():
		slog.Info("Backend stopping")
		return nil
	case err := <-failures:
		return err
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil && ctx.Err() == nil {
		slog.Error("Backend stopped", "error", err)
		os.Exit(1)
	}
}
