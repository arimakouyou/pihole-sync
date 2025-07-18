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
	client := NewClient("http://test.local", "test-password")

	assert.Equal(t, "http://test.local", client.BaseURL)
	assert.Equal(t, "test-password", client.Password)
	assert.NotNil(t, client.client)
}

func TestMakeRequest(t *testing.T) {
	authCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			authCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}
		assert.Contains(t, r.URL.RawQuery, "sid=test-sid")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "enabled"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	body, err := client.makeRequest("GET", "", nil)
	require.NoError(t, err)

	assert.True(t, authCalled)
	assert.Contains(t, string(body), "status")
	assert.Equal(t, "test-sid", client.SID)
	assert.Equal(t, "test-csrf", client.CSRFToken)
}

func TestMakeRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	_, err := client.makeRequest("GET", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestGetData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}

		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/api/lists":
			w.Write([]byte(`{"lists": [{"address": "example.com", "type": "block"}]}`))
		case "/api/domains":
			w.Write([]byte(`{"domains": [{"domain": "test.com", "type": "block"}, {"domain": "test.com", "type": "allow"}]}`))
		case "/api/groups":
			w.Write([]byte(`{"groups": []}`))
		default:
			w.Write([]byte(`{"data": []}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

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
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	data := &PiholeData{
		Adlists:   []string{"example.com"},
		Blacklist: []string{"bad.com"},
	}

	err := client.UpdateData(data)
	assert.NoError(t, err)
}

func TestUpdateDataAlwaysErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	data := &PiholeData{
		Adlists:   []string{"example.com"},
		Blacklist: []string{"bad.com"},
	}

	err := client.UpdateData(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestGetDataInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session_id": "test-sid", "csrf_token": "test-csrf"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	_, err := client.GetData()
	assert.Error(t, err)
}

func TestGetDataEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}

		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/api/lists":
			w.Write([]byte(`{"lists": []}`))
		case "/api/domains":
			w.Write([]byte(`{"domains": []}`))
		case "/api/groups":
			w.Write([]byte(`{"groups": []}`))
		default:
			w.Write([]byte(`{"data": []}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

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

	client := NewClient(server.URL, "test-password")
	client.client.Timeout = 10 * time.Millisecond

	_, err := client.makeRequest("GET", "", nil)
	assert.Error(t, err)
}

func TestUpdateDataNetworkError(t *testing.T) {
	client := NewClient("http://invalid-host-that-does-not-exist.local", "test-password")

	data := &PiholeData{
		Adlists: []string{"example.com"},
	}

	err := client.UpdateData(data)
	assert.Error(t, err)
}

func TestGetDataNetworkError(t *testing.T) {
	client := NewClient("http://invalid-host-that-does-not-exist.local", "test-password")

	_, err := client.GetData()
	assert.Error(t, err)
}

func TestMakeRequestInvalidURL(t *testing.T) {
	client := NewClient("invalid-url", "test-password")

	_, err := client.makeRequest("GET", "", nil)
	assert.Error(t, err)
}

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	err := client.authenticate()
	assert.NoError(t, err)
	assert.Equal(t, "test-sid", client.SID)
	assert.Equal(t, "test-csrf", client.CSRFToken)
}

func TestAuthenticateFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "wrong-password")

	err := client.authenticate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestAuthenticateInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"invalid": "response"}`))
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	err := client.authenticate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session object not found")
}

func TestAuthenticateInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	err := client.authenticate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse auth response")
}

func TestAuthenticateMissingCSRFToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid"}}`))
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	err := client.authenticate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session.csrf not found")
}

func TestMakeRequestWithExistingSession(t *testing.T) {
	authCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			authCallCount++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}
		assert.Contains(t, r.URL.RawQuery, "sid=test-sid")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "enabled"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")
	client.SID = "test-sid"
	client.CSRFToken = "test-csrf"

	_, err := client.makeRequest("GET", "", nil)
	require.NoError(t, err)

	assert.Equal(t, 0, authCallCount, "Should not authenticate when session already exists")
}

func TestMakeRequestPOSTWithCSRF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}

		if r.Method == "POST" {
			assert.Equal(t, "test-csrf", r.Header.Get("X-FTL-CSRF"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	_, err := client.makeRequest("POST", "", nil)
	require.NoError(t, err)
}

func TestClientWithEmptyPassword(t *testing.T) {
	client := NewClient("http://test.local", "")

	assert.Equal(t, "", client.Password)
	assert.Equal(t, "http://test.local", client.BaseURL)
}

func TestUpdateDataWithEmptyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session": {"sid": "test-sid", "csrf": "test-csrf"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-password")

	data := &PiholeData{}

	err := client.UpdateData(data)
	assert.NoError(t, err)
}
