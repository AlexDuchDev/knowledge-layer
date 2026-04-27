package jobqueue

import "github.com/knowledgelayer/api/internal/platform/queue"

// Shim: prefer importing github.com/knowledgelayer/api/internal/platform/queue in new code.

type (
	Publisher              = queue.Publisher
	KnowledgeJobRunPayload = queue.KnowledgeJobRunPayload
)

const TaskKnowledgeJobRun = queue.TaskKnowledgeJobRun

func NewPublisher(redisURL string) (*Publisher, error) {
	return queue.NewPublisher(redisURL)
}
