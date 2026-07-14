package idle

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/acidsailor/ts3afkmover/config"
	"github.com/acidsailor/ts3afkmover/internal/ts3"
)

// ts3Client is the slice of the ts3 client the mover depends on. Depending on
// this interface (rather than *ts3.Client directly) lets the sweep logic be
// unit-tested against a fake. *ts3.Client satisfies it.
type ts3Client interface {
	ClientListWithTimes(ctx context.Context) (ts3.ClientList, bool)
	MoveClient(ctx context.Context, clid, cid int) bool
	SendGM(ctx context.Context, message string) bool
}

type Mover struct {
	client ts3Client
	config *config.Config
}

func New(cfg *config.Config, client ts3Client) *Mover {
	return &Mover{
		client: client,
		config: cfg,
	}
}

func (t *Mover) MoveIdleClients(ctx context.Context) {
	clients, ok := t.client.ClientListWithTimes(ctx)
	if !ok {
		// The fetch failed (already logged with cause). Skip this sweep rather
		// than treating an empty result as "nobody is online".
		slog.ErrorContext(ctx, "skipping run: could not fetch client list")
		return
	}

	stats := struct {
		MovedClients   int `json:"moved_clients"`
		NotIdleClients int `json:"not_idle_clients"`
		SkippedClients int `json:"skipped_clients"`
		ErroredClients int `json:"errored_clients"`
	}{}

	for _, v := range clients.Body {
		clientType, err := strconv.Atoi(v.ClientType)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"error converting client type to int",
				slog.String("err", err.Error()),
			)
			stats.ErroredClients++
			continue
		}

		// client_type: 0 = regular voice client, 1 = ServerQuery client.
		// Skip anything that isn't a regular client (i.e. query clients).
		if clientType != 0 {
			stats.SkippedClients++
			continue
		}

		clientId, err := strconv.Atoi(v.Clid)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"error converting client id to int",
				slog.String("clid", v.Clid),
				slog.String("err", err.Error()),
			)
			stats.ErroredClients++
			continue
		}

		channelId, err := strconv.Atoi(v.Cid)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"error converting channel id to int",
				slog.String("cid", v.Cid),
				slog.String("err", err.Error()),
			)
			stats.ErroredClients++
			continue
		}

		// we don't touch clients that are already in the idle channel
		if channelId == t.config.IdleChannelId {
			stats.SkippedClients++
			continue
		}

		clientIdleTime, err := strconv.Atoi(v.ClientIdleTime)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"error converting client idle time to int",
				slog.String("client_idle_time", v.ClientIdleTime),
				slog.String("err", err.Error()),
			)
			stats.ErroredClients++
			continue
		}

		if time.Duration(
			clientIdleTime,
		)*time.Millisecond <= t.config.IdleThreshold() {
			stats.NotIdleClients++
			continue
		}

		// All three logger calls below share the same clid/cid pair.
		logger := slog.With(
			slog.Int("clid", clientId),
			slog.Int("cid", t.config.IdleChannelId),
		)

		if ok := t.client.MoveClient(
			ctx,
			clientId,
			t.config.IdleChannelId,
		); !ok {
			logger.ErrorContext(ctx, "error moving client to Idle Channel")
			stats.ErroredClients++
			continue
		}

		stats.MovedClients++
		logger.InfoContext(
			ctx,
			fmt.Sprintf(
				"User %s has been moved to Idle Channel",
				v.ClientNickname,
			),
		)

		if ok := t.client.SendGM(ctx,
			fmt.Sprintf(t.config.MessageTemplate, v.ClientNickname),
		); !ok {
			logger.ErrorContext(ctx, "error sending move message")
		}
	}

	slog.InfoContext(
		ctx,
		"stats for the last run",
		slog.Any("stats", stats),
	)
}
