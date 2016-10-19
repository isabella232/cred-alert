package queue_test

import (
	"context"
	"cred-alert/pubsubrunner"
	"cred-alert/queue"
	"cred-alert/queue/queuefakes"
	"os"

	"cloud.google.com/go/pubsub"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Processor", func() {
	var (
		logger        *lagertest.TestLogger
		firstMessage  *pubsub.Message
		secondMessage *pubsub.Message
		handler       *queuefakes.FakeHandler
		subscription  *pubsub.Subscription
		topic         *pubsub.Topic
		client        *pubsub.Client

		psRunner *pubsubrunner.Runner
		runner   ifrit.Runner
		process  ifrit.Process
	)

	BeforeEach(func() {
		psRunner = &pubsubrunner.Runner{}
		psRunner.Setup()

		logger = lagertest.NewTestLogger("processor")

		firstMessage = &pubsub.Message{
			Attributes: map[string]string{
				"id": "some-id",
			},
		}

		secondMessage = &pubsub.Message{
			Attributes: map[string]string{
				"id": "some-other-id",
			},
		}

		var err error
		ctx := context.Background()
		client, err = pubsub.NewClient(ctx, "a-project-id")
		Expect(err).NotTo(HaveOccurred())

		topic, err = client.CreateTopic(ctx, "a-topic-id")
		Expect(err).NotTo(HaveOccurred())

		subscription, err = client.CreateSubscription(ctx, "a-subscription-id", topic, 0, nil)
		Expect(err).NotTo(HaveOccurred())

		_, err = topic.Publish(ctx, firstMessage, secondMessage)
		Expect(err).NotTo(HaveOccurred())

		handler = &queuefakes.FakeHandler{}
	})

	AfterEach(func() {
		ginkgomon.Interrupt(process)
		client.Close()
		psRunner.Teardown()
	})

	JustBeforeEach(func() {
		runner = queue.NewProcessor(logger, subscription, handler)
		process = ginkgomon.Invoke(runner)
	})

	Context("when the runner is signaled", func() {
		It("exits gracefully", func() {
			process.Signal(os.Interrupt)
			Eventually(process.Wait()).Should(Receive())
		})

		It("does not process any more messages", func() {
			Eventually(handler.ProcessMessageCallCount).Should(Equal(2))
			process.Signal(os.Interrupt)

			_, err := topic.Publish(context.Background(), firstMessage)
			Expect(err).NotTo(HaveOccurred())

			Consistently(handler.ProcessMessageCallCount).Should(Equal(2))
		})
	})

	It("tries to process the messages", func() {
		Eventually(handler.ProcessMessageCallCount).Should(Equal(2))
		message := handler.ProcessMessageArgsForCall(0)
		Expect(message.Attributes).To(Equal(firstMessage.Attributes))

		message = handler.ProcessMessageArgsForCall(1)
		Expect(message.Attributes).To(Equal(secondMessage.Attributes))
	})
})