package main

import (
	"net/http"
	"net/http/pprof"
	"os"

	"golang.org/x/oauth2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/jessevdk/go-flags"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"

	"cred-alert/github"
	"cred-alert/metrics"
	"cred-alert/notifications"
	"cred-alert/queue"
	"cred-alert/sniff"
	"cred-alert/worker"
)

type Opts struct {
	GitHub struct {
		AccessToken string `short:"a" long:"access-token" description:"github api access token" env:"GITHUB_ACCESS_TOKEN" value-name:"TOKEN" required:"true"`
	} `group:"GitHub Options"`

	Datadog struct {
		APIKey      string `long:"datadog-api-key" description:"key to emit to datadog" env:"DATA_DOG_API_KEY" value-name:"KEY"`
		Environment string `long:"datadog-environment" description:"environment tag for datadog" env:"DATA_DOG_ENVIRONMENT_TAG" value-name:"NAME" default:"development"`
	} `group:"Datadog Options"`

	Slack struct {
		WebhookUrl string `long:"slack-webhook-url" description:"Slack webhook URL" env:"SLACK_WEBHOOK_URL" value-name:"WEBHOOK"`
	} `group:"Slack Options"`

	AWS struct {
		AwsAccessKey       string `long:"aws-access-key" description:"access key for aws SQS service" env:"AWS_ACCESS_KEY" value-name:"ACCESS_KEY"`
		AwsSecretAccessKey string `long:"aws-secret-key" description:"secret access key for aws SQS service" env:"AWS_SECRET_ACCESS_KEY" value-name:"SECRET_KEY"`
		AwsRegion          string `long:"aws-region" description:"aws region for SQS service" env:"AWS_REGION" value-name:"REGION"`
		SqsQueueName       string `long:"sqs-queue-name" description:"queue name to use with SQS" env:"SQS_QUEUE_NAME" value-name:"QUEUE_NAME"`
	} `group:"AWS Options"`
}

func main() {
	var opts Opts

	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		os.Exit(1)
	}

	logger := lager.NewLogger("cred-alert")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: opts.GitHub.AccessToken},
	)
	httpClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	emitter := metrics.BuildEmitter(opts.Datadog.APIKey, opts.Datadog.Environment)
	ghClient := github.NewClient(github.DefaultGitHubURL, httpClient, emitter)
	notifier := notifications.NewSlackNotifier(opts.Slack.WebhookUrl)
	foreman := queue.NewForeman(ghClient, sniff.Sniff, emitter, notifier)

	taskQueue, err := createQueue(opts, logger)
	if err != nil {
		logger.Error("Could not create queue", err)
		os.Exit(1)
	}

	backgroundWorker := worker.New(logger, foreman, taskQueue, emitter)

	members := []grouper.Member{
		{"worker", backgroundWorker},
		{"debug", http_server.New(
			"127.0.0.1:6060",
			debugHandler(),
		)},
	}

	runner := sigmon.New(grouper.NewParallel(os.Interrupt, members))

	err = <-ifrit.Invoke(runner).Wait()
	if err != nil {
		logger.Error("running-server-failed", err)
	}
}

func createQueue(opts Opts, logger lager.Logger) (queue.Queue, error) {
	if sqsValuesExist(opts) {
		return createSqsQueue(opts)
	}

	return queue.NewNullQueue(logger), nil
}

func sqsValuesExist(opts Opts) bool {
	if opts.AWS.AwsAccessKey != "" &&
		opts.AWS.AwsSecretAccessKey != "" &&
		opts.AWS.AwsRegion != "" &&
		opts.AWS.SqsQueueName != "" {

		return true
	}

	return false
}

func createSqsQueue(opts Opts) (queue.Queue, error) {
	creds := credentials.NewStaticCredentials(
		opts.AWS.AwsAccessKey,
		opts.AWS.AwsSecretAccessKey,
		"")
	config := aws.NewConfig().WithCredentials(creds).WithRegion(opts.AWS.AwsRegion)
	service := sqs.New(session.New(config))
	queue, err := queue.BuildSQSQueue(service, opts.AWS.SqsQueueName)
	if err != nil {
		return nil, err
	}
	return queue, nil
}

func debugHandler() http.Handler {
	debugRouter := http.NewServeMux()
	debugRouter.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	debugRouter.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	debugRouter.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	debugRouter.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	debugRouter.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	return debugRouter
}
