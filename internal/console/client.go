/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	kjson "k8s.io/apimachinery/pkg/util/json"
)

type ApiClient struct {
	Http    *http.Client
	Jwt     string
	BaseUrl string
	OrgId   string
}

type tokenResponse struct {
	AccessToken string `json:"accessToken" yaml:"accessToken"`
}

type loggingRoundTripper struct {
	Transport http.RoundTripper
}

func (t *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	logger := logging.LoggerFromContext(req.Context())

	logger.Debug("-->", "method", req.Method, "url", req.URL)

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	logger.Debug("<--", "status", resp.StatusCode, "url", resp.Request.URL, "t", time.Since(start))

	return resp, err
}

func NewApiClient(ctx context.Context, host string, apiKeyId string, apiKeySecret string, orgid string) (*ApiClient, error) {

	h := &http.Client{
		Transport: &loggingRoundTripper{
			Transport: http.DefaultTransport,
		},
	}

	baseUrl := fmt.Sprintf("%s/api/msc/v1/organizations/%s", host, orgid)

	url := fmt.Sprintf("%s/credentials/v3/token", baseUrl)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY-ID", apiKeyId)
	req.Header.Add("X-API-KEY", apiKeySecret)
	req.Header.Add("X-SNOWPLOW-CLI", util.VersionInfo)

	if fromMCP, ok := ctx.Value(util.MCPSourceContextKey{}).(bool); ok && fromMCP {
		req.Header.Add("X-SNOWPLOW-CLI-SOURCE", "mcp")
	}
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("bad token request")
	}
	body, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(ctx, resp.Body)
	if err != nil {
		return nil, err
	}

	var token tokenResponse
	err = kjson.Unmarshal(body, &token)
	if err != nil {
		return nil, err
	}

	return &ApiClient{Http: h, Jwt: token.AccessToken, BaseUrl: baseUrl, OrgId: orgid}, nil
}

func ConsoleRequest(method string, path string, client *ApiClient, cnx context.Context, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(cnx, method, path, body)
	if err != nil {
		return req, err
	}

	addStandardHeaders(req, cnx, client)

	return req, err
}

func DoConsoleRequest(method string, path string, client *ApiClient, cnx context.Context, body io.Reader) (*http.Response, error) {
	req, err := ConsoleRequest(method, path, client, cnx, body)
	if err != nil {
		return nil, err
	}
	return client.Http.Do(req)
}

func addStandardHeaders(req *http.Request, cnx context.Context, client *ApiClient) {
	addStandardHeadersWithJwt(req, cnx, client.Jwt)
}

func addStandardHeadersWithJwt(req *http.Request, cnx context.Context, jwt string) {
	auth := fmt.Sprintf("Bearer %s", jwt)
	req.Header.Add("authorization", auth)
	req.Header.Add("X-SNOWPLOW-CLI", util.VersionInfo)

	if fromMCP, ok := cnx.Value(util.MCPSourceContextKey{}).(bool); ok && fromMCP {
		req.Header.Add("X-SNOWPLOW-CLI-SOURCE", "mcp")
	}
}
