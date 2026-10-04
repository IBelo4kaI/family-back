package receipt

import "github.com/google/uuid"

type CheckRequest struct {
	QRRaw string `json:"qrraw"`
}

type Item struct {
	Name     string  `json:"name"`
	Price    int64   `json:"price"`
	Quantity float64 `json:"quantity"`
	Sum      int64   `json:"sum"`
}

// Receipt — чек, приведённый к нашему виду. Суммы в копейках.
type Receipt struct {
	Key        string `json:"receiptKey"`
	Date       string `json:"date"`
	TotalSum   int64  `json:"totalSum"`
	SellerINN  string `json:"sellerInn"`
	SellerName string `json:"sellerName"`
	Items      []Item `json:"items"`
}

type SaveInput struct {
	Key        string `json:"receiptKey"`
	Date       string `json:"date"`
	TotalSum   int64  `json:"totalSum"`
	SellerINN  string `json:"sellerInn"`
	SellerName string `json:"sellerName"`
	CategoryID string `json:"categoryId"`
	Scope      string `json:"scope"`
	Items      []Item `json:"items"`
}

type Transaction struct {
	ID         uuid.UUID `json:"id"`
	Amount     int64     `json:"amount"`
	Date       string    `json:"date"`
	CategoryID string    `json:"categoryId"`
	Type       string    `json:"type"`
	Scope      string    `json:"scope"`
	AuthorID   uuid.UUID `json:"authorId"`
	Source     string    `json:"source"`
}

type StoredItem struct {
	ID         uuid.UUID `json:"id"`
	Position   int32     `json:"position"`
	Name       string    `json:"name"`
	Price      int64     `json:"price"`
	Quantity   float64   `json:"quantity"`
	Sum        int64     `json:"sum"`
	CategoryID *string   `json:"categoryId"`
}
