// - Handle errors explicitly; never ignore errors
package usecase

import (
	"context"

	"parkir-pintar/internal/search"
	"parkir-pintar/internal/search/repository"

	"golang.org/x/sync/singleflight"
)

// ReadModelRepository is the local interface for the read model persistence layer.
type ReadModelRepository interface {
	UpsertSpot(ctx context.Context, spot search.SpotData) error
	DeleteSpot(ctx context.Context, spotID string) error
}

type searchUsecase struct {
	repo          repository.Repository
	readModelRepo ReadModelRepository
	redis         RedisClient
	sf            singleflight.Group
}

func NewUsecase(repo repository.Repository, readModelRepo ReadModelRepository, redis RedisClient) search.Usecase {
	return &searchUsecase{
		repo:          repo,
		readModelRepo: readModelRepo,
		redis:         redis,
	}
}
