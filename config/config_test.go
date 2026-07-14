package config

import (
	"os"
	"testing"
	"time"
)

// clearEnv unsets every TS3_* var for the duration of the test and restores the
// originals afterwards (t.Setenv registers the restore, then we unset).
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TS3_URL", "TS3_API_KEY", "TS3_VSERVER_ID", "TS3_IDLE_TIME",
		"TS3_IDLE_CHANNEL_ID", "TS3_IDLE_CHECK_INTERVAL", "TS3_REQUEST_TIMEOUT",
		"TS3_MESSAGE_TEMPLATE",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
}

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("TS3_URL", "http://ts3.example:10080")
	t.Setenv("TS3_API_KEY", "secret")
	t.Setenv("TS3_IDLE_CHANNEL_ID", "42")
}

func TestNewConfig_Defaults(t *testing.T) {
	clearEnv(t)
	setRequired(t)

	config, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if config.VServerId != 1 {
		t.Errorf("VServerId = %d, want 1", config.VServerId)
	}
	if config.IdleTime != 60 {
		t.Errorf("IdleTime = %d, want 60", config.IdleTime)
	}
	if config.IdleCheckInterval != 5 {
		t.Errorf("IdleCheckInterval = %d, want 5", config.IdleCheckInterval)
	}
	if config.RequestTimeout != 15 {
		t.Errorf("RequestTimeout = %d, want 15", config.RequestTimeout)
	}
	if config.MessageTemplate == "" {
		t.Error("MessageTemplate is empty, want default")
	}
}

func TestNewConfig_MissingRequired(t *testing.T) {
	cases := map[string]func(*testing.T){
		"missing url": func(t *testing.T) {
			t.Setenv("TS3_API_KEY", "secret")
			t.Setenv("TS3_IDLE_CHANNEL_ID", "42")
		},
		"missing api key": func(t *testing.T) {
			t.Setenv("TS3_URL", "http://ts3.example:10080")
			t.Setenv("TS3_IDLE_CHANNEL_ID", "42")
		},
		"missing idle channel": func(t *testing.T) {
			t.Setenv("TS3_URL", "http://ts3.example:10080")
			t.Setenv("TS3_API_KEY", "secret")
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			setup(t)
			if _, err := New(); err == nil {
				t.Fatal(
					"New() = nil error, want error for missing required var",
				)
			}
		})
	}
}

func TestNewConfig_TrailingSlashTrimmed(t *testing.T) {
	cases := map[string]string{
		"one trailing slash": "http://ts3.example:10080/",
		"no trailing slash":  "http://ts3.example:10080",
	}
	const want = "http://ts3.example:10080"

	for name, url := range cases {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			setRequired(t)
			t.Setenv("TS3_URL", url)

			config, err := New()
			if err != nil {
				t.Fatalf("New() returned error: %v", err)
			}
			if config.Url != want {
				t.Errorf("Url = %q, want %q", config.Url, want)
			}
		})
	}
}

func TestNewConfig_Validation(t *testing.T) {
	cases := map[string]struct {
		key, val string
	}{
		"zero interval panics ticker": {"TS3_IDLE_CHECK_INTERVAL", "0"},
		"negative interval":           {"TS3_IDLE_CHECK_INTERVAL", "-1"},
		"negative idle time":          {"TS3_IDLE_TIME", "-1"},
		"negative request timeout":    {"TS3_REQUEST_TIMEOUT", "-1"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			setRequired(t)
			t.Setenv(c.key, c.val)

			if _, err := New(); err == nil {
				t.Fatalf(
					"New() = nil error, want error for %s=%s",
					c.key,
					c.val,
				)
			}
		})
	}
}

func TestConfig_DurationHelpers(t *testing.T) {
	config := &Config{IdleTime: 60, IdleCheckInterval: 5, RequestTimeout: 15}

	if got := config.IdleThreshold(); got != 60*time.Minute {
		t.Errorf("IdleThreshold() = %v, want %v", got, 60*time.Minute)
	}
	if got := config.TickInterval(); got != 5*time.Minute {
		t.Errorf("TickInterval() = %v, want %v", got, 5*time.Minute)
	}
	if got := config.RequestTimeoutDuration(); got != 15*time.Second {
		t.Errorf("RequestTimeoutDuration() = %v, want %v", got, 15*time.Second)
	}
}
