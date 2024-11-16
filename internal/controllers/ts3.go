package controllers

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/acidsailor/ts3afkmover/configs"
    "io"
    "log/slog"
    "net/http"
    "strings"
    "time"
)

type Ts3Client struct {
    Config *configs.Config
    Logger *slog.Logger
}

type Ts3ResponseStatus struct {
    Status struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    } `json:"status"`
}

type Ts3ClientListResponseWithTimes struct {
    Body []struct {
        Cid                 string `json:"cid"`
        Clid                string `json:"clid"`
        ClientCreated       string `json:"client_created"`
        ClientDatabaseId    string `json:"client_database_id"`
        ClientIdleTime      string `json:"client_idle_time"`
        ClientLastconnected string `json:"client_lastconnected"`
        ClientNickname      string `json:"client_nickname"`
        ClientType          string `json:"client_type"`
    } `json:"body"`
    Ts3ResponseStatus
}

func NewTs3Client(config *configs.Config, logger *slog.Logger) *Ts3Client {
    return &Ts3Client{
        Config: config,
        Logger: logger,
    }
}

func (t Ts3Client) SendGM(ctx context.Context, message string) bool {
    httpClient := &http.Client{
        Timeout: time.Second * time.Duration(t.Config.RequestTimeout),
    }

    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/gm", t.Config.Url),
        strings.NewReader(fmt.Sprintf("{\"msg\": \"%s\"}", message)),
    )

    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error creating http request",
            slog.String("err", err.Error()),
        )
        return false
    }

    req.Header.Add("X-Api-Key", fmt.Sprintf("%s", t.Config.ApiKey))

    resp, err := httpClient.Do(req)

    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error sending http request",
            slog.String("err", err.Error()),
        )
        return false
    }

    defer func() {
        io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
    }()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error reading response body",
            slog.String("err", err.Error()),
        )
        return false
    }

    ts3ResponseStatus := Ts3ResponseStatus{}

    if err = json.Unmarshal(body, &ts3ResponseStatus); err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error unmarshalling response body",
            slog.String("err", err.Error()),
        )
        return false
    }

    return resp.StatusCode == 200 && ts3ResponseStatus.Status.Code == 0
}

func (t Ts3Client) ClientListWithTimes(ctx context.Context) Ts3ClientListResponseWithTimes {

    httpClient := &http.Client{
        Timeout: time.Second * time.Duration(t.Config.RequestTimeout),
    }

    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/%d/clientlist", t.Config.Url, t.Config.VServerId),
        strings.NewReader("{\"-times\": \"\"}"),
    )

    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error creating http request",
            slog.String("err", err.Error()),
        )
        return Ts3ClientListResponseWithTimes{}
    }

    req.Header.Add("X-Api-Key", fmt.Sprintf("%s", t.Config.ApiKey))

    resp, err := httpClient.Do(req)

    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error sending http request",
            slog.String("err", err.Error()),
        )
        return Ts3ClientListResponseWithTimes{}
    }

    defer func() {
        io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
    }()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error reading response body",
            slog.String("err", err.Error()),
        )
        return Ts3ClientListResponseWithTimes{}
    }

    clientListResp := Ts3ClientListResponseWithTimes{}

    if err = json.Unmarshal(body, &clientListResp); err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error unmarshalling response body",
            slog.String("err", err.Error()),
        )
        return Ts3ClientListResponseWithTimes{}
    }

    return clientListResp
}

func (t Ts3Client) MoveClient(ctx context.Context, clid int, cid int) bool {
    httpClient := &http.Client{
        Timeout: time.Second * time.Duration(t.Config.RequestTimeout),
    }

    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/%d/clientmove", t.Config.Url, t.Config.VServerId),
        strings.NewReader(fmt.Sprintf("{\"clid\": \"%d\", \"cid\": \"%d\"}",
            clid, cid)),
    )

    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error creating http request",
            slog.String("err", err.Error()),
        )
        return false
    }

    req.Header.Add("X-Api-Key", fmt.Sprintf("%s", t.Config.ApiKey))

    resp, err := httpClient.Do(req)

    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error sending http request",
            slog.String("err", err.Error()),
        )
        return false
    }

    defer func() {
        io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
    }()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error reading response body",
            slog.String("err", err.Error()),
        )
        return false
    }

    ts3ResponseStatus := Ts3ResponseStatus{}

    if err = json.Unmarshal(body, &ts3ResponseStatus); err != nil {
        t.Logger.ErrorContext(
            ctx,
            "error unmarshalling response body",
            slog.String("err", err.Error()),
        )
        return false
    }

    return resp.StatusCode == 200 && ts3ResponseStatus.Status.Code == 0
}
