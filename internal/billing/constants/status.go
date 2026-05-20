// Package constants defines shared constants for the billing domain module.
package constants

// BillingStatus represents the status of a billing record.
type BillingStatus string

// Billing status constants.
const (
	BillingStatusPending    BillingStatus = "pending"
	BillingStatusCalculated BillingStatus = "calculated"
	BillingStatusInvoiced   BillingStatus = "invoiced"
	BillingStatusPaid       BillingStatus = "paid"
)
