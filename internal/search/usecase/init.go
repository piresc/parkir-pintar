// - Handle errors explicitly; never ignore errors
package usecase

import (
	"parkir-pintar/internal/search/repository"

	"golang.org/x/sync/singleflight"
)

type searchUsecase struct {
	repo  repository.Repository
	redis RedisClient
	sf    singleflight.Group
}

func NewUsecase(repo repository.Repository, redis RedisClient) Usecase {
	return &searchUsecase{
		repo:  repo,
		redis: redis,
	}
}
