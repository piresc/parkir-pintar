package asynq

import (
	"github.com/hibiken/asynq"
)

// Server wraps the asynq server for processing background tasks.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewServer creates a new Asynq server connected to the given Redis address.
func NewServer(redisAddr, redisPassword string, concurrency int) *Server {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword},
		asynq.Config{
			Concurrency: concurrency,
		},
	)
	return &Server{
		server: srv,
		mux:    asynq.NewServeMux(),
	}
}

// RegisterHandlers registers task handlers for reservation expiry and payment timeout.
func (s *Server) RegisterHandlers(reservationExpireHandler, paymentTimeoutHandler asynq.Handler) {
	s.mux.Handle(TypeReservationExpire, reservationExpireHandler)
	s.mux.Handle(TypePaymentHoldTimeout, paymentTimeoutHandler)
}

// Start begins processing tasks. This blocks until Shutdown is called.
func (s *Server) Start() error {
	return s.server.Start(s.mux)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() {
	s.server.Shutdown()
}
