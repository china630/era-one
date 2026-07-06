// Package grpcserver — gRPC IngestService (ADR-0008, S1-3).
package grpcserver

import (
	"context"
	"log"
	"time"

	erav1 "era/contracts/gen/era/v1"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"era/services/ingest-gateway/internal/ingest"
	kafkapkg "era/services/ingest-gateway/internal/kafka"
)

// Server реализует IngestService.
type Server struct {
	erav1.UnimplementedIngestServiceServer
	producer *kafkapkg.Producer
}

func New(producer *kafkapkg.Producer) *Server {
	return &Server{producer: producer}
}

func (s *Server) PushEvents(ctx context.Context, batch *erav1.EventBatch) (*erav1.BatchAck, error) {
	if batch == nil {
		return &erav1.BatchAck{Status: erav1.Status_STATUS_REJECTED, Message: "empty batch"}, nil
	}

	now := time.Now().UTC()
	var rejected [][]byte
	byTopic := make(map[string][]kafkago.Message)

	for _, env := range batch.GetEvents() {
		res, err := ingest.ValidateAndEnrich(env, now)
		if err != nil {
			if env != nil {
				rejected = append(rejected, env.GetEventId())
			}
			log.Printf("reject event: %v", err)
			continue
		}
		data, err := proto.Marshal(res.Envelope)
		if err != nil {
			rejected = append(rejected, env.GetEventId())
			continue
		}
		byTopic[res.Topic] = append(byTopic[res.Topic], kafkago.Message{
			Key:   []byte(res.Key),
			Value: data,
		})
	}

	accepted := 0
	for topic, msgs := range byTopic {
		if err := s.producer.PublishBatch(context.Background(), topic, msgs); err != nil {
			log.Printf("kafka batch publish failed topic=%s: %v", topic, err)
			return &erav1.BatchAck{
				BatchId:      batch.GetBatchId(),
				Status:       erav1.Status_STATUS_RETRY,
				RetryAfterMs: 1000,
				Message:      err.Error(),
			}, nil
		}
		accepted += len(msgs)
	}

	status := erav1.Status_STATUS_ACCEPTED
	if len(rejected) > 0 && accepted > 0 {
		status = erav1.Status_STATUS_PARTIAL
	} else if accepted == 0 {
		status = erav1.Status_STATUS_REJECTED
	}

	log.Printf("PushEvents batch=%x agent=%s tenant=%s accepted=%d rejected=%d",
		batch.GetBatchId(), batch.GetAgentId(), batch.GetTenantId(), accepted, len(rejected))

	return &erav1.BatchAck{
		BatchId:          batch.GetBatchId(),
		Status:           status,
		RejectedEventIds: flatten(rejected),
		Message:          "ACCEPTED",
	}, nil
}

func (s *Server) Register(_ context.Context, req *erav1.RegisterRequest) (*erav1.RegisterResponse, error) {
	log.Printf("Register agent=%s tenant=%s hostname=%s", req.GetAgentId(), req.GetTenantId(), req.GetHostname())
	return &erav1.RegisterResponse{
		PolicyVersion:        "1.0.0-dev",
		PolicyBundleRef:      "local://policy/dev",
		HeartbeatIntervalSec: 60,
	}, nil
}

func flatten(ids [][]byte) [][]byte {
	out := make([][]byte, 0, len(ids))
	for _, id := range ids {
		if len(id) > 0 {
			out = append(out, id)
		}
	}
	return out
}
