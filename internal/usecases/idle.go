package usecases

import (
    "context"
    "fmt"
    "github.com/acidsailor/ts3afkmover/configs"
    "github.com/acidsailor/ts3afkmover/internal/controllers"
    "log/slog"
    "strconv"
    "time"
)

type Ts3IdleUsecase struct {
    Ts3Client *controllers.Ts3Client
    Config    *configs.Config
    Logger    *slog.Logger
}

func NewTs3IdleUsecase(config *configs.Config, logger *slog.Logger,
    ts3client *controllers.Ts3Client) *Ts3IdleUsecase {
    return &Ts3IdleUsecase{
        Ts3Client: ts3client,
        Config:    config,
        Logger:    logger,
    }
}

func (t Ts3IdleUsecase) MoveIdleClients(ctx context.Context) {
    clients := t.Ts3Client.ClientListWithTimes(ctx)

    stats := struct {
        MovedClients   int `json:"moved_clients"`
        NotIdleClients int `json:"not_idle_clients"`
        SkippedClients int `json:"skipped_clients"`
        ErroredClients int `json:"errored_clients"`
    }{}

    for _, v := range clients.Body {
        clientType, err := strconv.Atoi(v.ClientType)
        if err != nil {
            t.Logger.ErrorContext(
                ctx,
                "error converting client type to int",
                slog.String("err", err.Error()),
            )
            stats.ErroredClients++
            continue
        }

        // regular client = 0, query client = 1
        // we don't touch query clients
        if clientType != 0 {
            stats.SkippedClients++
            continue
        }

        clientId, err := strconv.Atoi(v.Clid)
        if err != nil {
            t.Logger.ErrorContext(
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
            t.Logger.ErrorContext(
                ctx,
                "error converting channel id to int",
                slog.String("cid", v.Cid),
                slog.String("err", err.Error()),
            )
            stats.ErroredClients++
            continue
        }

        // we don't touch clients that are already the in idle channel
        if channelId == t.Config.IdleChannelId {
            stats.SkippedClients++
            continue
        }

        clientIdleTime, err := strconv.Atoi(v.ClientIdleTime)
        if err != nil {
            t.Logger.ErrorContext(
                ctx,
                "error converting client idle time to int",
                slog.String("client_idle_time", v.ClientIdleTime),
                slog.String("err", err.Error()),
            )
            stats.ErroredClients++
            continue
        }

        if time.Millisecond*time.Duration(clientIdleTime) >
            time.Minute*time.Duration(t.Config.IdleTime) {
            if ok := t.Ts3Client.MoveClient(ctx, clientId, t.Config.IdleChannelId); !ok {
                t.Logger.ErrorContext(
                    ctx,
                    "error moving client to Idle Channel",
                    slog.Int("clid", clientId),
                    slog.Int("cid", t.Config.IdleChannelId),
                )
                stats.ErroredClients++
                continue
            }

            stats.MovedClients++
            t.Logger.InfoContext(
                ctx,
                fmt.Sprintf("User %s has been moved to Idle Channel", v.ClientNickname),
                slog.Int("clid", clientId),
                slog.Int("cid", t.Config.IdleChannelId),
            )

            if ok := t.Ts3Client.SendGM(ctx,
                fmt.Sprintf(t.Config.MessageTemplate, v.ClientNickname),
            ); !ok {
                t.Logger.ErrorContext(
                    ctx,
                    "error sending move message",
                    slog.Int("clid", clientId),
                    slog.Int("cid", t.Config.IdleChannelId),
                )
                continue
            }
        } else {
            stats.NotIdleClients++
        }
    }
    t.Logger.InfoContext(ctx, "stats for the last run", slog.Any("stats", stats))
}
