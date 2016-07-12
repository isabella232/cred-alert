package queue_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"

	"cred-alert/github/githubfakes"
	"cred-alert/metrics/metricsfakes"
	"cred-alert/notifications/notificationsfakes"
	"cred-alert/queue"
	"cred-alert/queue/queuefakes"
	"cred-alert/sniff"
)

var _ = Describe("Foreman", func() {
	var (
		foreman queue.Foreman
	)

	BeforeEach(func() {
		foreman = queue.NewForeman(
			&githubfakes.FakeClient{},
			func(lager.Logger, sniff.Scanner, func(sniff.Line)) {},
			&metricsfakes.FakeEmitter{},
			&notificationsfakes.FakeNotifier{},
		)
	})

	Describe("building runnable jobs from tasks", func() {
		Describe("DiffScan Task", func() {
			Context("when the foreman knows how to build the task", func() {
				It("builds the task", func() {
					task := &queuefakes.FakeAckTask{}
					task.TypeReturns("diff-scan")
					task.PayloadReturns(`{
					"owner":      "pivotal-cf",
					"repository": "cred-alert",
					"ref":        "refs/heads/my-branch",
					"from":       "abc123",
					"to":         "def456"
				}`)

					job, err := foreman.BuildJob(task)
					Expect(err).NotTo(HaveOccurred())

					diffScan, ok := job.(*queue.DiffScanJob)
					Expect(ok).To(BeTrue())

					Expect(diffScan.Owner).To(Equal("pivotal-cf"))
					Expect(diffScan.Repository).To(Equal("cred-alert"))
					Expect(diffScan.Ref).To(Equal("refs/heads/my-branch"))
					Expect(diffScan.From).To(Equal("abc123"))
					Expect(diffScan.To).To(Equal("def456"))
				})
			})

			Context("payload is not valid json", func() {
				It("returns an error", func() {
					task := &queuefakes.FakeAckTask{}
					task.TypeReturns("diff-scan")
					task.PayloadReturns(`{broken-json":'seriously"}`)

					_, err := foreman.BuildJob(task)
					_, isJsonError := err.(*json.SyntaxError)
					Expect(isJsonError).To(BeTrue())
				})
			})
		})

		Context("when the foreman doesn't know how to build the task", func() {
			It("returns an error", func() {
				task := &queuefakes.FakeAckTask{}
				task.TypeReturns("unknown-task-type")

				_, err := foreman.BuildJob(task)
				Expect(err).To(MatchError("unknown task type: unknown-task-type"))
			})
		})
	})
})
