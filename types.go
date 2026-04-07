package dian

import (
	"time"

	codes "github.com/SimpleX-Corp/go-dian-codes"
)

// DocumentType represents the type of electronic document.
// Alias to go-dian-codes.DocumentType for backwards compatibility.
type DocumentType = codes.DocumentType

// Document type constants - re-exported from go-dian-codes.
const (
	DocInvoice         = codes.DocInvoice         // "01" Factura electrónica de venta
	DocExportInvoice   = codes.DocExportInvoice   // "02" Factura de exportación
	DocContingency     = codes.DocContingency     // "03" Factura de contingencia
	DocContingencyDIAN = codes.DocContingencyDIAN // "04" Factura contingencia DIAN
	DocCreditNote      = codes.DocCreditNote      // "91" Nota crédito
	DocDebitNote       = codes.DocDebitNote       // "92" Nota débito
)

// Party represents a business entity (issuer or customer).
type Party struct {
	// NIT without verification digit
	NIT string `json:"nit"`

	// Verification digit (0-9)
	DV string `json:"dv,omitempty"`

	// Business name
	Name string `json:"name"`

	// Document type (31=NIT, 13=CC, 22=CE, etc.)
	DocType string `json:"doc_type,omitempty"`

	// Tax responsibilities (O-13, O-15, O-23, O-47, R-99-PN, etc.)
	TaxResponsibilities []string `json:"tax_responsibilities,omitempty"`

	// Address
	Address Address `json:"address,omitempty"`

	// Contact
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// Address represents a physical address.
type Address struct {
	Street     string `json:"street,omitempty"`
	City       string `json:"city,omitempty"`
	CityCode   string `json:"city_code,omitempty"`   // DANE code
	Department string `json:"department,omitempty"`
	DeptCode   string `json:"dept_code,omitempty"`   // DANE code
	Country    string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"` // ISO 3166-1
	PostalCode string `json:"postal_code,omitempty"`
}

// InvoiceLine represents a line item in an invoice.
type InvoiceLine struct {
	// Line number (1, 2, 3...)
	Number int `json:"number"`

	// Quantity
	Quantity float64 `json:"quantity"`

	// Unit of measure (EA, KGM, LTR, etc.)
	UnitCode string `json:"unit_code"`

	// Description
	Description string `json:"description"`

	// Unit price before taxes
	UnitPrice float64 `json:"unit_price"`

	// Product/service code (UNSPSC recommended)
	ProductCode string `json:"product_code,omitempty"`

	// Taxes for this line
	Taxes []Tax `json:"taxes,omitempty"`

	// Discounts for this line
	Discounts []Discount `json:"discounts,omitempty"`
}

// Tax represents a tax applied.
type Tax struct {
	// Tax type (01=IVA, 02=IC, 03=ICA, 04=INC, etc.)
	Type string `json:"type"`

	// Tax percentage (19, 5, 0, etc.)
	Percent float64 `json:"percent"`

	// Tax amount
	Amount float64 `json:"amount"`

	// Taxable base
	TaxableBase float64 `json:"taxable_base"`
}

// Discount represents a discount applied.
type Discount struct {
	// Discount percentage
	Percent float64 `json:"percent,omitempty"`

	// Discount amount
	Amount float64 `json:"amount"`

	// Reason
	Reason string `json:"reason,omitempty"`
}

// Payment represents payment information.
type Payment struct {
	// Payment method (1=Cash, 2=Credit, etc.)
	Method string `json:"method"`

	// Payment means (10=Cash, 20=Check, 31=Transfer, 42=Account deposit, etc.)
	Means string `json:"means"`

	// Due date for credit payments
	DueDate *time.Time `json:"due_date,omitempty"`
}

// InvoiceRequest is the main struct to create an invoice.
// This is the "clean JSON" API - just fill this and call SendInvoiceRequest().
type InvoiceRequest struct {
	// Document type
	Type DocumentType `json:"type"`

	// Invoice prefix (from DIAN resolution)
	Prefix string `json:"prefix"`

	// Invoice number
	Number string `json:"number"`

	// Issue date (defaults to now)
	IssueDate *time.Time `json:"issue_date,omitempty"`

	// Issue time (defaults to now)
	IssueTime *time.Time `json:"issue_time,omitempty"`

	// Currency (COP default)
	Currency string `json:"currency,omitempty"`

	// Issuer (seller)
	Issuer Party `json:"issuer"`

	// Customer (buyer)
	Customer Party `json:"customer"`

	// Line items
	Lines []InvoiceLine `json:"lines"`

	// Payment info
	Payment Payment `json:"payment,omitempty"`

	// Global discounts (applied to total)
	Discounts []Discount `json:"discounts,omitempty"`

	// Notes/comments
	Notes []string `json:"notes,omitempty"`

	// Order reference (if applicable)
	OrderReference string `json:"order_reference,omitempty"`

	// For credit/debit notes: reference to original invoice
	InvoiceReference *DocumentReference `json:"invoice_reference,omitempty"`

	// DIAN resolution info
	Resolution *Resolution `json:"resolution,omitempty"`

	// Software ID for CUFE calculation
	SoftwareID string `json:"software_id,omitempty"`

	// Software PIN for CUFE calculation
	SoftwarePIN string `json:"software_pin,omitempty"`

	// Technical key from resolution
	TechnicalKey string `json:"technical_key,omitempty"`
}

// DocumentReference references another document (for credit/debit notes).
type DocumentReference struct {
	// Original document number
	Number string `json:"number"`

	// Original document CUFE
	CUFE string `json:"cufe,omitempty"`

	// Issue date of original
	IssueDate time.Time `json:"issue_date"`
}

// Resolution contains DIAN resolution information.
type Resolution struct {
	// Resolution number
	Number string `json:"number"`

	// Resolution date
	Date time.Time `json:"date"`

	// Prefix authorized
	Prefix string `json:"prefix"`

	// Range start
	RangeFrom int64 `json:"range_from"`

	// Range end
	RangeTo int64 `json:"range_to"`

	// Validity start date
	ValidFrom time.Time `json:"valid_from"`

	// Validity end date
	ValidTo time.Time `json:"valid_to"`

	// Technical key
	TechnicalKey string `json:"technical_key"`
}

// CreditNoteRequest for credit notes.
type CreditNoteRequest struct {
	InvoiceRequest

	// Correction concept (1=Partial return, 2=Cancellation, 3=Discount, 4=Price adjust, 5=Other)
	CorrectionConcept string `json:"correction_concept"`
}

// DebitNoteRequest for debit notes.
type DebitNoteRequest struct {
	InvoiceRequest

	// Correction concept
	CorrectionConcept string `json:"correction_concept"`
}

// SupportDocRequest for support documents (documento soporte).
type SupportDocRequest struct {
	InvoiceRequest

	// Supplier is not required to invoice (non-responsible for IVA)
	SupplierNotRequired bool `json:"supplier_not_required"`
}
