package constants

import "errors"

var (
	ErrNoData         = errors.New("no analytics data available")
	ErrInvalidHorizon = errors.New("invalid prediction horizon")
	ErrRecordFailed   = errors.New("failed to record analytics event")
)
