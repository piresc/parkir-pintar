package asynq

import (
	"github.com/hibiken/asynq"
)

type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

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

func (s *Server) RegisterHandlers(reservationExpireHandler, paymentTimeoutHandler asynq.Handler) {
	s.mux.Handle(TypeReservationExpire, reservationExpireHandler)
	s.mux.Handle(TypePaymentHoldTimeout, paymentTimeoutHandler)
}

func (s *Server) Start() error {
	return s.server.Start(s.mux)
}

func (s *Server) Shutdown() {
	s.server.Shutdown()
}
