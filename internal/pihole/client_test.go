package pihole

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://test.local", "test-key")

	assert.Equal(t, "http://test.local", client.BaseURL)
	assert.Equal(t, "test-key", client.APIKey)
	assert.NotNil(t, client.client)
}

func TestMakeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "auth=test-key")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "enabled"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	body, err := client.makeRequest("GET", "", nil)
	require.NoError(t, err)

	assert.Contains(t, string(body), "status")
}

func TestMakeRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	_, err := client.makeRequest("GET", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		list := query.Get("list")
		action := query.Get("action")
		
		w.WriteHeader(http.StatusOK)
		
		if list == "adlist" {
			w.Write([]byte(`{"data": [{"address": "example.com"}]}`))
		} else if list == "black" {
			w.Write([]byte(`{"data": [{"domain": "test.com"}]}`))
		} else if list == "white" {
			w.Write([]byte(`{"data": [{"domain": "test.com"}]}`))
		} else if action == "get_groups" {
			w.Write([]byte(`{"data": []}`))
		} else if action == "get_custom_dns" {
			w.Write([]byte(`{"data": []}`))
		} else {
			w.Write([]byte(`{"data": []}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	data, err := client.GetData()
	require.NoError(t, err)

	assert.NotNil(t, data)
	assert.Equal(t, []string{"example.com"}, data.Adlists)
	assert.Equal(t, []string{"test.com"}, data.Blacklist)
	assert.Equal(t, []string{"test.com"}, data.Whitelist)
	
	if data.Groups == nil {
		data.Groups = []string{}
	}
	if data.DNSRecords == nil {
		data.DNSRecords = []string{}
	}
	if data.DHCP == nil {
		data.DHCP = []string{}
	}
	
	assert.Equal(t, []string{}, data.Groups)
	assert.Equal(t, []string{}, data.DNSRecords)
	assert.Equal(t, []string{}, data.DHCP)
}

func TestUpdateData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	data := &PiholeData{
		Adlists:   []string{"example.com"},
		Blacklist: []string{"bad.com"},
	}

	err := client.UpdateData(data)
	assert.NoError(t, err)
}

func TestUpdateDataAlwaysErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	data := &PiholeData{
		Adlists:   []string{"example.com"},
		Blacklist: []string{"bad.com"},
	}

	err := client.UpdateData(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetDataInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	_, err := client.GetData()
	assert.Error(t, err)
}

func TestGetDataEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	data, err := client.GetData()
	require.NoError(t, err)

	assert.Empty(t, data.Adlists)
	assert.Empty(t, data.Blacklist)
	assert.Empty(t, data.Whitelist)
	assert.Empty(t, data.Groups)
	assert.Empty(t, data.DNSRecords)
	assert.Empty(t, data.DHCP)
}

func TestMakeRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "enabled"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	client.client.Timeout = 10 * time.Millisecond

	_, err := client.makeRequest("GET", "", nil)
	assert.Error(t, err)
}

func TestUpdateDataNetworkError(t *testing.T) {
	client := NewClient("http://invalid-host-that-does-not-exist.local", "test-key")

	data := &PiholeData{
		Adlists: []string{"example.com"},
	}

	err := client.UpdateData(data)
	assert.Error(t, err)
}

func TestGetDataNetworkError(t *testing.T) {
	client := NewClient("http://invalid-host-that-does-not-exist.local", "test-key")

	_, err := client.GetData()
	assert.Error(t, err)
}

func TestMakeRequestInvalidURL(t *testing.T) {
	client := NewClient("invalid-url", "test-key")

	_, err := client.makeRequest("GET", "", nil)
	assert.Error(t, err)
}
