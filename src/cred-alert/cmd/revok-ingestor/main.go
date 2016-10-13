package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	flags "github.com/jessevdk/go-flags"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"

	"cred-alert/ingestor"
	"cred-alert/metrics"
	"cred-alert/queue"
	"cred-alert/revok"
)

type Opts struct {
	Port     uint16 `short:"p" long:"port" description:"the port to listen on" default:"8080" env:"PORT" value-name:"PORT"`
	Endpoint string `long:"endpoint" description:"the endpoint to forward tasks to" env:"ENDPOINT" value-name:"URL" required:"true"`

	GitHub struct {
		WebhookSecretToken string `short:"w" long:"github-webhook-secret-token" description:"github webhook secret token" env:"GITHUB_WEBHOOK_SECRET_TOKEN" value-name:"TOKEN" required:"true"`
	} `group:"GitHub Options"`

	Metrics struct {
		SentryDSN     string `long:"sentry-dsn" description:"DSN to emit to Sentry with" env:"SENTRY_DSN" value-name:"DSN"`
		DatadogAPIKey string `long:"datadog-api-key" description:"key to emit to datadog" env:"DATADOG_API_KEY" value-name:"KEY"`
		Environment   string `long:"environment" description:"environment tag for metrics" env:"ENVIRONMENT" value-name:"NAME" default:"development"`
	} `group:"Metrics Options"`
}

func main() {
	var opts Opts

	logger := lager.NewLogger("revok-ingestor")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	logger.Debug("starting")

	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		logger.Fatal("failed", err)
		os.Exit(1)
	}

	if opts.Metrics.SentryDSN != "" {
		logger.RegisterSink(revok.NewSentrySink(opts.Metrics.SentryDSN, opts.Metrics.Environment))
	}

	emitter := metrics.BuildEmitter(opts.Metrics.DatadogAPIKey, opts.Metrics.Environment)
	generator := queue.NewGenerator()

	enqueuer := queue.NewHTTPEnqueuer(logger, opts.Endpoint)
	in := ingestor.NewIngestor(enqueuer, emitter, "revok", generator)

	router := http.NewServeMux()
	router.Handle("/webhook", ingestor.Handler(logger, in, opts.GitHub.WebhookSecretToken))
	router.Handle("/healthcheck", revok.ObliviousHealthCheck())

	members := []grouper.Member{
		{"api", http_server.New(fmt.Sprintf(":%d", opts.Port), router)},
	}

	runner := sigmon.New(grouper.NewParallel(os.Interrupt, members))

	serverLogger := logger.Session("server", lager.Data{
		"port": opts.Port,
	})
	serverLogger.Info("starting")
	err = <-ifrit.Invoke(runner).Wait()
	if err != nil {
		serverLogger.Error("failed", err)
	}
}