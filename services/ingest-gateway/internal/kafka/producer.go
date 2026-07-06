// Package kafka — продюсер событий в topic-per-domain (ADR-0001, S1-4).
package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"google.golang.org/protobuf/proto"
)

// Producer публикует protobuf Envelope в Kafka с zstd-сжатием.
type Producer struct {
	brokers []string
	writers map[string]*kafka.Writer
	mu      sync.Mutex
}

// NewProducer создаёт продюсер. brokers — список bootstrap-серверов.
func NewProducer(brokers []string) *Producer {
	return &Producer{
		brokers: brokers,
		writers: make(map[string]*kafka.Writer),
	}
}

func (p *Producer) writer(topic string) *kafka.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(p.brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		Compression:            compress.Zstd,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
		Async:                  false,
	}
	p.writers[topic] = w
	return w
}

// Publish сериализует сообщение и пишет в указанный топик.
func (p *Producer) Publish(ctx context.Context, topic, key string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	w := p.writer(topic)
	return w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: data,
	})
}

// PublishBatch пишет несколько сообщений в один топик одним вызовом.
func (p *Producer) PublishBatch(ctx context.Context, topic string, msgs []kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	w := p.writer(topic)
	return w.WriteMessages(ctx, msgs...)
}

// Ping проверяет доступность брокера (для /readyz).
func (p *Producer) Ping(ctx context.Context) error {
	conn, err := kafka.DialContext(ctx, "tcp", p.brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Brokers()
	return err
}

// Close закрывает все writer'ы.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var first error
	for topic, w := range p.writers {
		if err := w.Close(); err != nil && first == nil {
			first = fmt.Errorf("close topic %s: %w", topic, err)
		}
		delete(p.writers, topic)
	}
	return first
}
