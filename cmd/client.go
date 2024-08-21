package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type ApiClient struct {
	Http *http.Client
	Jwt  string
	BaseUrl string
}

type tokenResponse struct {
	AccessToken string
}

func NewApiClient(ctx context.Context, host string, apikey string, orgid string) (*ApiClient, error) {

	h := &http.Client{}

	baseUrl := fmt.Sprintf("https://%s/api/msc/v1/organizations/%s", host, orgid)

	url := fmt.Sprintf("%s/credentials/v2/token", baseUrl)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY", apikey)
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("bad token request")
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var token tokenResponse
	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, err
	}

	return &ApiClient{Http: &http.Client{}, Jwt: token.AccessToken, BaseUrl: baseUrl}, nil
}
