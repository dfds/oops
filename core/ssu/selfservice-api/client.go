package selfservice_api

import (
	"encoding/json"
	"fmt"
	"go.dfds.cloud/oops/core/util"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"k8s.io/utils/env"
)

type Client struct {
	httpClient  *http.Client
	tokenClient *util.TokenClient
	config      Config
}

type Config struct {
	Host         string `json:"host"`
	TenantId     string `json:"tenantId"`
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Scope        string `json:"scope"`
}

func (c *Client) prepareHttpRequest(h *http.Request) error {
	err := c.RefreshAuth()
	if err != nil {
		return err
	}

	h.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.tokenClient.Token.GetToken()))
	h.Header.Set("User-Agent", "oops - github.com/dfds/oops")

	return nil
}

func (c *Client) GetCapabilities() ([]*GetCapabilitiesResponseContextCapability, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/system/legacy/aad-aws-sync", c.config.Host), nil)
	if err != nil {
		return nil, err
	}
	err = c.prepareHttpRequest(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response returned unexpected status code: %d", resp.StatusCode)
	}

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload []*GetCapabilitiesResponseContextCapability

	err = json.Unmarshal(rawData, &payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (c *Client) GetCapabilityMetadata(id string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/capabilities/%s/metadata", c.config.Host, id), nil)
	if err != nil {
		return nil, err
	}

	err = c.prepareHttpRequest(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	preDeserialise, err := strconv.Unquote(string(buf))
	if err != nil {
		return nil, err
	}

	var payload map[string]interface{}

	err = json.Unmarshal([]byte(preDeserialise), &payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (c *Client) RefreshAuth() error {
	envToken := env.GetString("SSU_K8S_SELFSERVICEAPI_TOKEN", "")
	if envToken != "" {
		c.tokenClient.Token = util.NewBearerToken(envToken)
		return nil
	}

	err := c.tokenClient.RefreshAuth()
	return err
}

func (c *Client) getNewToken() (*util.RefreshAuthResponse, error) {
	reqPayload := url.Values{}
	reqPayload.Set("client_id", c.config.ClientId)
	reqPayload.Set("grant_type", "client_credentials")
	reqPayload.Set("scope", c.config.Scope)
	reqPayload.Set("client_secret", c.config.ClientSecret)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.config.TenantId), strings.NewReader(reqPayload.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response returned unexpected status code: %d", resp.StatusCode)
	}

	var tokenResponse *util.RefreshAuthResponse

	err = json.Unmarshal(rawData, &tokenResponse)
	if err != nil {
		return nil, err
	}

	return tokenResponse, nil
}

func NewClient(conf Config) *Client {
	payload := &Client{
		httpClient: http.DefaultClient,
		config:     conf,
	}
	payload.tokenClient = util.NewTokenClient(payload.getNewToken)
	return payload
}
