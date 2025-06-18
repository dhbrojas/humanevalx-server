package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, w io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	logger := createLogger(w, zapcore.InfoLevel)
	defer logger.Sync()

	config, err := parseConfig(args)
	must(err)

	srv := NewServer(
		logger,
		config,
	)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.Host, config.Port),
		Handler: srv,
	}

	go func() {
		logger.Info("starting http server", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("error listening and serving", zap.Error(err))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("shutting down http server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down http server", zap.Error(err))
		}
	}()
	wg.Wait()

	return nil
}

func createLogger(w io.Writer, level zapcore.Level) *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	writerSyncer := zapcore.AddSync(w)
	core := zapcore.NewCore(encoder, writerSyncer, level)
	return zap.New(core, zap.WithCaller(false))
}

type Config struct {
	Host                     string
	Port                     string
	MaxConcurrentEvaluations int
	MaxTimeoutSecs           float64
}

func parseConfig(args []string) (*Config, error) {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	config := &Config{}
	fs.StringVar(&config.Host, "host", "0.0.0.0", "host to listen on")
	fs.StringVar(&config.Port, "port", "8080", "port to listen on")
	fs.IntVar(&config.MaxConcurrentEvaluations, "max-concurrent-evaluations", 16, "maximum number of concurrent evaluations")
	fs.Float64Var(&config.MaxTimeoutSecs, "max-timeout-secs", 60, "maximum timeout in seconds")
	must(fs.Parse(args[1:]))
	return config, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
