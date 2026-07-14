package ts3

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/acidsailor/ts3afkmover/config"
)

func TestMain(m *testing.M) {
	// Silence the global logger; the client logs via slog.Default().
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func newTestClient(
	t *testing.T,
	handler http.HandlerFunc,
) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := &config.Config{Url: server.URL, ApiKey: "secret", VServerId: 1}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	return client, server
}

// okStatus writes a successful WebQuery status envelope.
func okStatus(w http.ResponseWriter) {
	_, _ = w.Write([]byte(`{"status":{"code":0,"message":"ok"}}`))
}

func TestSendGM_EscapesNicknameProducingValidJSON(t *testing.T) {
	// A nickname with a quote must not break the JSON body.
	message := `User Bob" was moved`
	var gotBody map[string]string

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret" {
			t.Errorf(
				"X-Api-Key = %q, want %q",
				r.Header.Get("X-Api-Key"),
				"secret",
			)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("request body is not valid JSON: %v (%s)", err, body)
		}
		okStatus(w)
	})

	if ok := client.SendGM(context.Background(), message); !ok {
		t.Fatal("SendGM() = false, want true")
	}
	if gotBody["msg"] != message {
		t.Errorf("decoded msg = %q, want %q", gotBody["msg"], message)
	}
}

func TestSendGM_SuccessAndFailureContract(t *testing.T) {
	cases := map[string]struct {
		httpStatus int
		body       string
		want       bool
	}{
		"http 200 + code 0": {http.StatusOK, `{"status":{"code":0}}`, true},
		"http 200 + code != 0": {
			http.StatusOK,
			`{"status":{"code":1281,"message":"nope"}}`,
			false,
		},
		"http 500": {
			http.StatusInternalServerError,
			`{"status":{"code":0}}`,
			false,
		},
		"unauthorized http 403": {
			http.StatusForbidden,
			`{"status":{"code":0}}`,
			false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(
				t,
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(c.httpStatus)
					_, _ = w.Write([]byte(c.body))
				},
			)

			if got := client.SendGM(context.Background(), "hi"); got != c.want {
				t.Errorf("SendGM() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestClientListWithTimes_SignalsFailure(t *testing.T) {
	cases := map[string]struct {
		httpStatus int
		body       string
		wantOK     bool
		wantLen    int
	}{
		"success with clients": {
			http.StatusOK,
			`{"body":[{"clid":"5","client_type":"0"}],"status":{"code":0}}`,
			true, 1,
		},
		"success empty list": {
			http.StatusOK, `{"body":[],"status":{"code":0}}`, true, 0,
		},
		"ts3 error envelope": {
			http.StatusOK,
			`{"body":[],"status":{"code":1281,"message":"db error"}}`,
			false,
			0,
		},
		"http error": {
			http.StatusForbidden, `{"body":[],"status":{"code":0}}`, false, 0,
		},
		"malformed body": {
			http.StatusOK, `not json`, false, 0,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(
				t,
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(c.httpStatus)
					_, _ = w.Write([]byte(c.body))
				},
			)

			resp, ok := client.ClientListWithTimes(context.Background())
			if ok != c.wantOK {
				t.Errorf("ok = %v, want %v", ok, c.wantOK)
			}
			if len(resp.Body) != c.wantLen {
				t.Errorf("len(Body) = %d, want %d", len(resp.Body), c.wantLen)
			}
		})
	}
}

func TestMoveClient_SendsIntegerIDsAsJSON(t *testing.T) {
	var gotBody map[string]string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("request body is not valid JSON: %v (%s)", err, body)
		}
		okStatus(w)
	})

	if ok := client.MoveClient(context.Background(), 5, 9); !ok {
		t.Fatal("MoveClient() = false, want true")
	}
	if gotBody["clid"] != "5" || gotBody["cid"] != "9" {
		t.Errorf("body = %+v, want clid=5 cid=9", gotBody)
	}
}
