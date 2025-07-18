package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL   string
	Password  string
	SID       string
	CSRFToken string
	client    *http.Client
}

type PiholeData struct {
	Adlists    []string `json:"adlists"`
	Blacklist  []string `json:"blacklist"`
	Whitelist  []string `json:"whitelist"`
	Groups     []string `json:"groups"`
	DNSRecords []string `json:"dns_records"`
	DHCP       []string `json:"dhcp"`
}

func NewClient(baseURL, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetData() (*PiholeData, error) {
	data := &PiholeData{}

	adlists, err := c.getAdlists()
	if err != nil {
		return nil, fmt.Errorf("failed to get adlists: %w", err)
	}
	data.Adlists = adlists

	blacklist, err := c.getBlacklist()
	if err != nil {
		return nil, fmt.Errorf("failed to get blacklist: %w", err)
	}
	data.Blacklist = blacklist

	whitelist, err := c.getWhitelist()
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelist: %w", err)
	}
	data.Whitelist = whitelist

	groups, err := c.getGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}
	data.Groups = groups

	dnsRecords, err := c.getDNSRecords()
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS records: %w", err)
	}
	data.DNSRecords = dnsRecords

	dhcp, err := c.getDHCP()
	if err != nil {
		return nil, fmt.Errorf("failed to get DHCP: %w", err)
	}
	data.DHCP = dhcp

	return data, nil
}

func (c *Client) authenticate() error {
	authURL := fmt.Sprintf("%s/api/auth", c.BaseURL)

	// JSONペイロードを作成
	payload := map[string]string{
		"password": c.Password,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp map[string]interface{}
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	// レスポンス構造の修正: session.sid と session.csrf
	if session, ok := authResp["session"].(map[string]interface{}); ok {
		if sid, ok := session["sid"].(string); ok {
			c.SID = sid
		} else {
			return fmt.Errorf("session.sid not found in auth response")
		}

		if csrf, ok := session["csrf"].(string); ok {
			c.CSRFToken = csrf
		} else {
			return fmt.Errorf("session.csrf not found in auth response")
		}
	} else {
		return fmt.Errorf("session object not found in auth response")
	}

	return nil
}

func (c *Client) UpdateData(data *PiholeData) error {
	if err := c.updateAdlists(data.Adlists); err != nil {
		return fmt.Errorf("failed to update adlists: %w", err)
	}

	if err := c.updateBlacklist(data.Blacklist); err != nil {
		return fmt.Errorf("failed to update blacklist: %w", err)
	}

	if err := c.updateWhitelist(data.Whitelist); err != nil {
		return fmt.Errorf("failed to update whitelist: %w", err)
	}

	if err := c.updateGroups(data.Groups); err != nil {
		return fmt.Errorf("failed to update groups: %w", err)
	}

	if err := c.updateDNSRecords(data.DNSRecords); err != nil {
		return fmt.Errorf("failed to update DNS records: %w", err)
	}

	if err := c.updateDHCP(data.DHCP); err != nil {
		return fmt.Errorf("failed to update DHCP: %w", err)
	}

	return nil
}

func (c *Client) makeRequest(method, endpoint string, params url.Values) ([]byte, error) {
	if c.SID == "" {
		if err := c.authenticate(); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	reqURL := fmt.Sprintf("%s/api/%s", c.BaseURL, endpoint)

	if params == nil {
		params = url.Values{}
	}

	var req *http.Request
	var err error

	if method == "GET" {
		params.Set("sid", c.SID)
		if len(params) > 0 {
			reqURL += "?" + params.Encode()
		}
		req, err = http.NewRequest(method, reqURL, nil)
	} else {
		// POSTリクエストの場合はJSONで送信
		var body []byte
		if len(params) > 0 {
			data := make(map[string]string)
			data["sid"] = c.SID
			for key, values := range params {
				if len(values) > 0 {
					data[key] = values[0]
				}
			}
			body, err = json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request data: %w", err)
			}
		} else {
			data := map[string]string{"sid": c.SID}
			body, err = json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request data: %w", err)
			}
		}

		req, err = http.NewRequest(method, reqURL, bytes.NewBuffer(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-FTL-CSRF", c.CSRFToken)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) getAdlists() ([]string, error) {
	body, err := c.makeRequest("GET", "lists", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var adlists []string
	if lists, ok := result["lists"].([]interface{}); ok {
		for _, item := range lists {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if listType, ok := itemMap["type"].(string); ok && listType == "block" {
					if address, ok := itemMap["address"].(string); ok {
						adlists = append(adlists, address)
					}
				}
			}
		}
	}

	return adlists, nil
}

func (c *Client) getBlacklist() ([]string, error) {
	body, err := c.makeRequest("GET", "domains", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var blacklist []string
	if domains, ok := result["domains"].([]interface{}); ok {
		for _, item := range domains {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if domainType, ok := itemMap["type"].(string); ok && domainType == "block" {
					if domain, ok := itemMap["domain"].(string); ok {
						blacklist = append(blacklist, domain)
					}
				}
			}
		}
	}

	return blacklist, nil
}

func (c *Client) getWhitelist() ([]string, error) {
	body, err := c.makeRequest("GET", "domains", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var whitelist []string
	if domains, ok := result["domains"].([]interface{}); ok {
		for _, item := range domains {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if domainType, ok := itemMap["type"].(string); ok && domainType == "allow" {
					if domain, ok := itemMap["domain"].(string); ok {
						whitelist = append(whitelist, domain)
					}
				}
			}
		}
	}

	return whitelist, nil
}

func (c *Client) getGroups() ([]string, error) {
	body, err := c.makeRequest("GET", "groups", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var groups []string
	if groupsData, ok := result["groups"].([]interface{}); ok {
		for _, item := range groupsData {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if name, ok := itemMap["name"].(string); ok {
					groups = append(groups, name)
				}
			}
		}
	}

	return groups, nil
}

func (c *Client) getDNSRecords() ([]string, error) {
	// Teleporter APIを使用するため、空のリストを返す
	// 実際のデータはGetBackup/RestoreBackupメソッドで処理
	return []string{}, nil
}

func (c *Client) getDHCP() ([]string, error) {
	return []string{}, nil
}

func (c *Client) updateAdlists(adlists []string) error {
	for _, adlist := range adlists {
		params := url.Values{}
		params.Set("address", adlist)

		_, err := c.makeRequest("POST", "lists", params)
		if err != nil {
			return fmt.Errorf("failed to add adlist %s: %w", adlist, err)
		}
	}
	return nil
}

func (c *Client) updateBlacklist(blacklist []string) error {
	for _, domain := range blacklist {
		params := url.Values{}
		params.Set("domain", domain)
		params.Set("type", "block")

		_, err := c.makeRequest("POST", "domains", params)
		if err != nil {
			return fmt.Errorf("failed to add blacklist domain %s: %w", domain, err)
		}
	}
	return nil
}

func (c *Client) updateWhitelist(whitelist []string) error {
	for _, domain := range whitelist {
		params := url.Values{}
		params.Set("domain", domain)
		params.Set("type", "allow")

		_, err := c.makeRequest("POST", "domains", params)
		if err != nil {
			return fmt.Errorf("failed to add whitelist domain %s: %w", domain, err)
		}
	}
	return nil
}

func (c *Client) updateGroups(groups []string) error {
	for _, group := range groups {
		params := url.Values{}
		params.Set("name", group)

		_, err := c.makeRequest("POST", "groups", params)
		if err != nil {
			return fmt.Errorf("failed to add group %s: %w", group, err)
		}
	}
	return nil
}

func (c *Client) updateDNSRecords(dnsRecords []string) error {
	for _, record := range dnsRecords {
		parts := strings.Split(record, "=")
		if len(parts) != 2 {
			continue
		}

		params := url.Values{}
		params.Set("domain", parts[0])
		params.Set("ip", parts[1])

		_, err := c.makeRequest("POST", "dns", params)
		if err != nil {
			return fmt.Errorf("failed to add DNS record %s: %w", record, err)
		}
	}
	return nil
}

func (c *Client) updateDHCP(dhcp []string) error {
	return nil
}

// GetBackup downloads a backup from Pi-hole using the Teleporter API
func (c *Client) GetBackup() ([]byte, error) {
	if c.SID == "" {
		if err := c.authenticate(); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	reqURL := fmt.Sprintf("%s/api/teleporter", c.BaseURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup request: %w", err)
	}

	req.Header.Set("sid", c.SID)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download backup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backup download failed with status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// RestoreBackup uploads a backup to Pi-hole using the Teleporter API
func (c *Client) RestoreBackup(backupData []byte) error {
	return c.RestoreBackupWithOptions(backupData, nil)
}

// RestoreBackupWithOptions uploads a backup to Pi-hole using the Teleporter API with specific import options
func (c *Client) RestoreBackupWithOptions(backupData []byte, importOptions map[string]bool) error {
	if c.SID == "" {
		if err := c.authenticate(); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	reqURL := fmt.Sprintf("%s/api/teleporter", c.BaseURL)

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file part
	part, err := writer.CreateFormFile("file", "pihole_backup.zip")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(backupData); err != nil {
		return fmt.Errorf("failed to write backup data: %w", err)
	}

	// Add resourceName field
	if err := writer.WriteField("resourceName", "pihole_backup.zip"); err != nil {
		return fmt.Errorf("failed to write resourceName field: %w", err)
	}

	// Add import options if provided
	if importOptions != nil {
		importJSON, err := json.Marshal(importOptions)
		if err != nil {
			return fmt.Errorf("failed to marshal import options: %w", err)
		}

		if err := writer.WriteField("import", string(importJSON)); err != nil {
			return fmt.Errorf("failed to write import options field: %w", err)
		}
	}

	writer.Close()

	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create restore request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("sid", c.SID)
	req.Header.Set("X-FTL-CSRF", c.CSRFToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload backup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backup restore failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
