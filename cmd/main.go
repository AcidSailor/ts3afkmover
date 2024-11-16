package main

import (
    "context"
    "fmt"
    "github.com/acidsailor/ts3afkmover/configs"
    "github.com/acidsailor/ts3afkmover/internal/controllers"
    "github.com/acidsailor/ts3afkmover/internal/usecases"
    "log/slog"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        AddSource: true,
    }))

    config, err := configs.NewConfig()

    if err != nil {
        logger.Error(
            "error initializing config",
            slog.String("err", err.Error()),
        )
        os.Exit(1)
    }

    idleUsecase := usecases.NewTs3IdleUsecase(
        config, logger, controllers.NewTs3Client(
            config,
            logger,
        ),
    )

    fmt.Println()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    var wg sync.WaitGroup
    ticker := time.NewTicker(time.Duration(config.IdleCheckInterval) * time.Minute)
    sigs := make(chan os.Signal)

    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

    logger.InfoContext(ctx, "starting application")

    wg.Add(1)
    go func() {
        defer wg.Done()
        defer ticker.Stop()
        for {
            select {
            case <-sigs:
                return
            case <-ticker.C:
                idleUsecase.MoveIdleClients(ctx)
            }
        }
    }()
    wg.Wait()
}
