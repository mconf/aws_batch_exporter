package awsBatchExporter

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

func TestCollector(t *testing.T) {
	region := "us-east-1"

	type Job struct {
		ID     string
		Name   string
		Status string
	}

	// Queue name must be an integer
	queues := []string{"0", "1"}
	jobs := [][]Job{
		// Queue1
		{
			{
				ID:     "1",
				Name:   "job1",
				Status: batch.JobStatusSubmitted,
			},
			{
				ID:     "2",
				Name:   "job2",
				Status: batch.JobStatusSubmitted,
			},
			{
				ID:     "3",
				Name:   "job3",
				Status: batch.JobStatusPending,
			},
			{
				ID:     "4",
				Name:   "job4",
				Status: batch.JobStatusFailed,
			},
		},
		// Queue2
		{
			{
				ID:     "5",
				Name:   "job5",
				Status: batch.JobStatusSubmitted,
			},
			{
				ID:     "6",
				Name:   "job6",
				Status: batch.JobStatusSubmitted,
			},
			{
				ID:     "7",
				Name:   "job7",
				Status: batch.JobStatusPending,
			},
			{
				ID:     "8",
				Name:   "job8",
				Status: batch.JobStatusFailed,
			},
		},
	}

	DescribeJobQueuesWithContext = func(c *Collector, ctx context.Context, input *batch.DescribeJobQueuesInput) (*batch.DescribeJobQueuesOutput, error) {
		var output batch.DescribeJobQueuesOutput

		for i := range queues {
			output.JobQueues = append(output.JobQueues, &batch.JobQueueDetail{JobQueueName: &queues[i]})
		}

		return &output, nil
	}

	ListJobsWithContext = func(c *Collector, ctx context.Context, input *batch.ListJobsInput) (*batch.ListJobsOutput, error) {
		var output batch.ListJobsOutput

		queue, _ := strconv.Atoi(*input.JobQueue)
		status := *input.JobStatus

		for i, job := range jobs[queue] {
			if job.Status == status {
				output.JobSummaryList = append(output.JobSummaryList, &batch.JobSummary{JobId: &jobs[queue][i].ID, JobName: &jobs[queue][i].Name, Status: &jobs[queue][i].Status})
			}
		}

		return &output, nil
	}

	collector, err := New(region)
	assert.Nil(t, err)

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		assert.Nil(t, http.ListenAndServe(":8080", nil))
	}()

	response, err := http.Get("http://localhost:8080/metrics")

	assert.Nil(t, err)
	assert.EqualValues(t, 200, response.StatusCode)
}
