package ts3

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/acidsailor/restkit"

	"github.com/acidsailor/ts3afkmover/config"
)

type Client struct {
	client    *restkit.Client
	vserverID int
}

type ResponseStatus struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
}

// OK reports whether the TS3 WebQuery status envelope indicates success.
func (s ResponseStatus) OK() bool {
	return s.Status.Code == 0
}

type ClientEntry struct {
	Cid                 string `json:"cid"`
	Clid                string `json:"clid"`
	ClientCreated       string `json:"client_created"`
	ClientDatabaseId    string `json:"client_database_id"`
	ClientIdleTime      string `json:"client_idle_time"`
	ClientLastconnected string `json:"client_lastconnected"`
	ClientNickname      string `json:"client_nickname"`
	ClientType          string `json:"client_type"`
}

type ClientList struct {
	Body []ClientEntry `json:"body"`
	ResponseStatus
}

func New(cfg *config.Config) (*Client, error) {
	client, err := restkit.New(
		cfg.Url,
		restkit.WithName("ts3"),
		restkit.WithHTTPClient(&http.Client{
			Timeout: cfg.RequestTimeoutDuration(),
		}),
		restkit.WithHook(apiKeyHook(cfg.ApiKey)),
	)
	if err != nil {
		return nil, err
	}

	return &Client{client: client, vserverID: cfg.VServerId}, nil
}

// apiKeyHook pins the WebQuery X-Api-Key header on every request.
func apiKeyHook(key string) restkit.RequestHook {
	return func(r *http.Request) error {
		r.Header.Set("X-Api-Key", key)
		return nil
	}
}

// logUnsuccessful records a 2xx response whose TS3 status envelope still
// reported failure (transport-level failures are already carried in err).
func logUnsuccessful(ctx context.Context, op string, s ResponseStatus) {
	slog.ErrorContext(
		ctx,
		op,
		slog.Int("ts3_code", s.Status.Code),
		slog.String("ts3_message", s.Status.Message),
	)
}

func (t *Client) SendGM(ctx context.Context, message string) bool {
	resp, err := restkit.Do[ResponseStatus](
		ctx, t.client, http.MethodPost, "/gm",
		map[string]string{"msg": message},
	)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"sending global message failed",
			slog.String("err", err.Error()),
		)
		return false
	}

	if !resp.OK() {
		logUnsuccessful(ctx, "sending global message failed", resp)
		return false
	}

	return true
}

func (t *Client) ClientListWithTimes(
	ctx context.Context,
) (ClientList, bool) {
	resp, err := restkit.Do[ClientList](
		ctx, t.client, http.MethodPost,
		fmt.Sprintf("/%d/clientlist", t.vserverID),
		map[string]string{"-times": ""},
	)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"fetching client list failed",
			slog.String("err", err.Error()),
		)
		return ClientList{}, false
	}

	if !resp.OK() {
		logUnsuccessful(ctx, "fetching client list failed", resp.ResponseStatus)
		return ClientList{}, false
	}

	return resp, true
}

func (t *Client) MoveClient(ctx context.Context, clid, cid int) bool {
	resp, err := restkit.Do[ResponseStatus](
		ctx, t.client, http.MethodPost,
		fmt.Sprintf("/%d/clientmove", t.vserverID),
		map[string]string{
			"clid": strconv.Itoa(clid),
			"cid":  strconv.Itoa(cid),
		},
	)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"moving client failed",
			slog.String("err", err.Error()),
		)
		return false
	}

	if !resp.OK() {
		logUnsuccessful(ctx, "moving client failed", resp)
		return false
	}

	return true
}
