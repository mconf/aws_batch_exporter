package awsBatchExporter

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/batch/batchiface"
	"github.com/prometheus/client_golang/prometheus"
)

var DescribeJobQueuesWithContext = func(c *Collector, ctx context.Context, input *batch.DescribeJobQueuesInput) (*batch.DescribeJobQueuesOutput, error) {
	return c.client.DescribeJobQueuesWithContext(ctx, input)
}

var ListJobsWithContext = func(c *Collector, ctx context.Context, input *batch.ListJobsInput) (*batch.ListJobsOutput, error) {
	return c.client.ListJobsWithContext(ctx, input)
}

type Collector struct {
	client  batchiface.BatchAPI
	region  string
	timeout time.Duration
}

const (
	namespace = "aws_batch"
	timeout   = 10 * time.Second
)

var (
	jobStatus = []string{
		batch.JobStatusSubmitted,
		batch.JobStatusPending,
		batch.JobStatusRunnable,
		batch.JobStatusStarting,
		batch.JobStatusRunning,
		batch.JobStatusFailed,
		batch.JobStatusSucceeded,
	}

	jobSubmitted = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "submitted_jobs"),
		"Jobs in the queue that are in the SUBMITTED state",
		[]string{"region", "queue"}, nil,
	)

	jobPending = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "pending_jobs"),
		"Jobs in the queue that are in the PENDING state",
		[]string{"region", "queue"}, nil,
	)

	jobRunnable = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "runnable_jobs"),
		"Jobs in the queue that are in the RUNNABLE state",
		[]string{"region", "queue"}, nil,
	)

	jobStarting = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "starting_jobs"),
		"Jobs in the queue that are in the STARTING state",
		[]string{"region", "queue"}, nil,
	)

	jobRunning = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "running_jobs"),
		"Jobs in the queue that are in the RUNNING state",
		[]string{"region", "queue"}, nil,
	)

	jobFailed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "failed_jobs"),
		"Jobs in the queue that are in the FAILED state",
		[]string{"region", "queue"}, nil,
	)

	jobSucceeded = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "succeeded_jobs"),
		"Jobs in the queue that are in the SUCCEEDED state",
		[]string{"region", "queue"}, nil,
	)

	jobDescMap = map[string]*prometheus.Desc{
		batch.JobStatusSubmitted: jobSubmitted,
		batch.JobStatusPending:   jobPending,
		batch.JobStatusRunnable:  jobRunnable,
		batch.JobStatusStarting:  jobStarting,
		batch.JobStatusRunning:   jobRunning,
		batch.JobStatusFailed:    jobFailed,
		batch.JobStatusSucceeded: jobSucceeded,
	}
)

type JobResult struct {
	id     string
	queue  string
	name   string
	status string
}

func New(region string) (*Collector, error) {
	s, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, err
	}

	return &Collector{
		client:  batch.New(s),
		region:  region,
		timeout: timeout,
	}, nil
}

func (*Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- jobSubmitted
	ch <- jobPending
	ch <- jobRunnable
	ch <- jobStarting
	ch <- jobRunning
	ch <- jobFailed
	ch <- jobSucceeded
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := DescribeJobQueuesWithContext(c, ctx, &batch.DescribeJobQueuesInput{})
	if err != nil {
		log.Printf("Error collecting metrics: %v\n", err)
		return
	}
	var wg sync.WaitGroup
	for _, d := range r.JobQueues {
		wg.Add(1)
		go func(d batch.JobQueueDetail) {
			defer wg.Done()
			var results []JobResult
			for _, s := range jobStatus {
				r, err := ListJobsWithContext(c, ctx, &batch.ListJobsInput{JobQueue: d.JobQueueName, JobStatus: &s})
				if err != nil {
					log.Printf("Error collecting job status metrics: %v\n", err)
					continue
				}
				for _, j := range r.JobSummaryList {
					results = append(results, JobResult{id: *j.JobId, queue: *d.JobQueueName, name: *j.JobName, status: *j.Status})
				}
			}
			c.collectJobDetailStatus(ch, results)
		}(*d)
	}
	wg.Wait()
}

func (c *Collector) collectJobDetailStatus(ch chan<- prometheus.Metric, results []JobResult) {
	statusMap := make(map[string](map[string]int))
	for _, r := range results {
		if statusMap[r.status] == nil {
			statusMap[r.status] = make(map[string]int)
		}
		statusMap[r.status][r.queue] += 1
	}
	for status, queueMap := range statusMap {
		for queue, value := range queueMap {
			ch <- prometheus.MustNewConstMetric(jobDescMap[status], prometheus.GaugeValue, float64(value), c.region, queue)
		}
	}
}
