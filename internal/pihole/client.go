package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	
	payload := url.Values{}
	payload.Set("password", c.Password)
	
	req, err := http.NewRequest("POST", authURL, bytes.NewBufferString(payload.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
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
	
	if sid, ok := authResp["session_id"].(string); ok {
		c.SID = sid
	} else {
		return fmt.Errorf("session_id not found in auth response")
	}
	
	if csrf, ok := authResp["csrf_token"].(string); ok {
		c.CSRFToken = csrf
	} else {
		return fmt.Errorf("csrf_token not found in auth response")
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
	
	reqURL := fmt.Sprintf("%s/admin/api.php", c.BaseURL)
	
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
		params.Set("sid", c.SID)
		req, err = http.NewRequest(method, reqURL, bytes.NewBufferString(params.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	params := url.Values{}
	params.Set("list", "adlist")
	
	body, err := c.makeRequest("GET", "", params)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var adlists []string
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if address, ok := itemMap["address"].(string); ok {
					adlists = append(adlists, address)
				}
			}
		}
	}
	
	return adlists, nil
}

func (c *Client) getBlacklist() ([]string, error) {
	params := url.Values{}
	params.Set("list", "black")
	
	body, err := c.makeRequest("GET", "", params)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var blacklist []string
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if domain, ok := itemMap["domain"].(string); ok {
					blacklist = append(blacklist, domain)
				}
			}
		}
	}
	
	return blacklist, nil
}

func (c *Client) getWhitelist() ([]string, error) {
	params := url.Values{}
	params.Set("list", "white")
	
	body, err := c.makeRequest("GET", "", params)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var whitelist []string
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if domain, ok := itemMap["domain"].(string); ok {
					whitelist = append(whitelist, domain)
				}
			}
		}
	}
	
	return whitelist, nil
}

func (c *Client) getGroups() ([]string, error) {
	params := url.Values{}
	params.Set("action", "get_groups")
	
	body, err := c.makeRequest("GET", "", params)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var groups []string
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
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
	params := url.Values{}
	params.Set("action", "get_custom_dns")
	
	body, err := c.makeRequest("GET", "", params)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var dnsRecords []string
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if domain, ok := itemMap["domain"].(string); ok {
					if ip, ok := itemMap["ip"].(string); ok {
						dnsRecords = append(dnsRecords, fmt.Sprintf("%s=%s", domain, ip))
					}
				}
			}
		}
	}
	
	return dnsRecords, nil
}

func (c *Client) getDHCP() ([]string, error) {
	return []string{}, nil
}

func (c *Client) updateAdlists(adlists []string) error {
	for _, adlist := range adlists {
		params := url.Values{}
		params.Set("action", "add_adlist")
		params.Set("address", adlist)
		
		_, err := c.makeRequest("POST", "", params)
		if err != nil {
			return fmt.Errorf("failed to add adlist %s: %w", adlist, err)
		}
	}
	return nil
}

func (c *Client) updateBlacklist(blacklist []string) error {
	for _, domain := range blacklist {
		params := url.Values{}
		params.Set("action", "add")
		params.Set("domain", domain)
		params.Set("list", "black")
		
		_, err := c.makeRequest("POST", "", params)
		if err != nil {
			return fmt.Errorf("failed to add blacklist domain %s: %w", domain, err)
		}
	}
	return nil
}

func (c *Client) updateWhitelist(whitelist []string) error {
	for _, domain := range whitelist {
		params := url.Values{}
		params.Set("action", "add")
		params.Set("domain", domain)
		params.Set("list", "white")
		
		_, err := c.makeRequest("POST", "", params)
		if err != nil {
			return fmt.Errorf("failed to add whitelist domain %s: %w", domain, err)
		}
	}
	return nil
}

func (c *Client) updateGroups(groups []string) error {
	for _, group := range groups {
		params := url.Values{}
		params.Set("action", "add_group")
		params.Set("name", group)
		
		_, err := c.makeRequest("POST", "", params)
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
		params.Set("action", "add_custom_dns")
		params.Set("domain", parts[0])
		params.Set("ip", parts[1])
		
		_, err := c.makeRequest("POST", "", params)
		if err != nil {
			return fmt.Errorf("failed to add DNS record %s: %w", record, err)
		}
	}
	return nil
}

func (c *Client) updateDHCP(dhcp []string) error {
	return nil
}
