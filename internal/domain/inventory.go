package domain

import (
	"fmt"
	"time"
)

// SparePart is a replaceable component kept in inventory.
type SparePart struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Stock      int       `json:"stock"`
	SafetyLine int       `json:"safety_line"`
	Unit       string    `json:"unit"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewSparePart creates a new spare part entry.
func NewSparePart(id, name, unit string, stock, safetyLine int) *SparePart {
	return &SparePart{
		ID:         id,
		Name:       name,
		Stock:      stock,
		SafetyLine: safetyLine,
		Unit:       unit,
		UpdatedAt:  time.Now(),
	}
}

// Consume reduces stock by the given quantity.
func (p *SparePart) Consume(qty int) error {
	if qty <= 0 {
		return fmt.Errorf("consume quantity must be positive")
	}
	if p.Stock < qty {
		return fmt.Errorf("insufficient stock for part %s: have %d, need %d", p.ID, p.Stock, qty)
	}
	p.Stock -= qty
	p.UpdatedAt = time.Now()
	return nil
}

// BelowSafetyLine reports whether replenishment is needed.
func (p *SparePart) BelowSafetyLine() bool {
	return p.Stock < p.SafetyLine
}

// ReplenishmentStatus tracks the state of a restock request.
type ReplenishmentStatus string

const (
	ReplenishmentPending   ReplenishmentStatus = "pending"
	ReplenishmentFulfilled ReplenishmentStatus = "fulfilled"
)

// ReplenishmentRequest is an auto-generated restock order.
type ReplenishmentRequest struct {
	ID        string              `json:"id"`
	PartID    string              `json:"part_id"`
	Quantity  int                 `json:"quantity"`
	Status    ReplenishmentStatus `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
}

// NewReplenishmentRequest creates a pending replenishment request.
func NewReplenishmentRequest(id, partID string, quantity int) *ReplenishmentRequest {
	return &ReplenishmentRequest{
		ID:        id,
		PartID:    partID,
		Quantity:  quantity,
		Status:    ReplenishmentPending,
		CreatedAt: time.Now(),
	}
}
