package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type pocketBase struct {
	baseURL string
	client  *http.Client
	token   string
}

func (pb *pocketBase) login(ctx context.Context, email, password string) error {
	var response struct {
		Token string `json:"token"`
	}
	if err := pb.request(ctx, http.MethodPost, "/api/collections/_superusers/auth-with-password", map[string]string{
		"identity": email, "password": password,
	}, &response); err != nil {
		return err
	}
	if response.Token == "" {
		return errors.New("PocketBase login returned no token")
	}
	pb.token = response.Token
	return nil
}

func (pb *pocketBase) findDevice(ctx context.Context, deviceCode string) (string, error) {
	query := url.Values{
		"filter":  {"device_code = " + strconv.Quote(deviceCode)},
		"perPage": {"1"},
		"fields":  {"id"},
	}
	var response struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := pb.request(ctx, http.MethodGet, "/api/collections/device/records?"+query.Encode(), nil, &response); err != nil {
		return "", err
	}
	if len(response.Items) == 0 || response.Items[0].ID == "" {
		return "", errors.New("device is not registered")
	}
	return response.Items[0].ID, nil
}

func (pb *pocketBase) saveRecord(ctx context.Context, collection string, record map[string]any) error {
	return pb.request(ctx, http.MethodPost, "/api/collections/"+collection+"/records", record, nil)
}

func (pb *pocketBase) request(ctx context.Context, method, path string, body, result any) error {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return errors.New("cannot encode PocketBase request")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, pb.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("invalid PocketBase request URL")
	}
	req.Header.Set("Content-Type", "application/json")
	if pb.token != "" {
		req.Header.Set("Authorization", pb.token)
	}
	response, err := pb.client.Do(req)
	if err != nil {
		return fmt.Errorf("PocketBase request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("PocketBase returned HTTP %d", response.StatusCode)
	}
	if result != nil {
		if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(result); err != nil {
			return errors.New("invalid PocketBase response JSON")
		}
	} else {
		_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		if err != nil {
			return errors.New("cannot read PocketBase response")
		}
	}
	return nil
}
