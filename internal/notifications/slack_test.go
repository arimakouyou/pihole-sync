package notifications

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSlackNotifier(t *testing.T) {
	notifier := NewSlackNotifier("http://test.com", true)

	assert.Equal(t, "http://test.com", notifier.webhookURL)
	assert.True(t, notifier.enabled)
}

func TestNotifyErrorDisabled(t *testing.T) {
	notifier := NewSlackNotifier("", false)

	err := notifier.NotifyError("test", "test details")
	assert.NoError(t, err)
}

func TestNotifyErrorSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, true)

	err := notifier.NotifyError("test error", "test details")
	assert.NoError(t, err)
}

func TestNotifyErrorFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, true)

	err := notifier.NotifyError("test error", "test details")
	assert.Error(t, err)
}
