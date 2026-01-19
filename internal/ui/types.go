package ui

// SelectableStock represents a stock in a selection list
type SelectableStock struct {
	Code      string
	Name      string
	Quantity  int     // For portfolio stocks
	CostPrice float64 // For portfolio stocks
}
