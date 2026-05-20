// Package errors defines domain-specific sentinel errors for the billing service.
package constants

import "errors"

// Billing domain errors.
var (
	ErrNotFound               = errors.New("billing record not found")
	ErrConflict               = errors.New("billing record conflict: duplicate")
	ErrConcurrentModification = errors.New("billing record concurrent modification")
	ErrInvalidStatus          = errors.New("billing record in invalid status for operation")
	ErrCannotCalculate        = errors.New("cannot calculate fee: billing not in pending status")
	ErrCannotInvoice          = errors.New("cannot invoice: billing not in calculated status")
	ErrAlreadyInvoiced        = errors.New("billing record already invoiced")
)
