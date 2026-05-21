package asynq

import (
	"fmt"

	"github.com/hibiken/asynq"
)

type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewServer(redisAddr, redisPassword string, concurrency int) *Server {
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword}
	if concurrency <= 0 {
		concurrency = 10
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: concurrency,
	})
	mux := asynq.NewServeMux()
	return &Server{server: srv, mux: mux}
}

// Register registers a handler for a task type.
func (s *Server) Register(taskType string, handler asynq.Handler) {
	s.mux.Handle(taskType, handler)
}

func (s *Server) Start() error {
	if err := s.server.Start(s.mux); err != nil {
		return fmt.Errorf("start asynq server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown() {
	s.server.Shutdown()
}
