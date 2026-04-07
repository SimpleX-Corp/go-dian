package dian

import (
	"crypto/sha512"
	"encoding/hex"
	"strings"
	"time"

	"github.com/SimpleX-Corp/go-cufe"
)

// CalculateCUFE calculates the CUFE for an invoice using go-cufe library.
func CalculateCUFE(req *InvoiceRequest, env Environment) string {
	// Calculate totals from lines
	var totalBeforeTax float64
	var taxes []cufe.Tax

	// Aggregate taxes by type
	taxAmounts := map[cufe.TaxCode]float64{
		cufe.TaxIVA: 0,
		cufe.TaxINC: 0,
		cufe.TaxICA: 0,
	}

	for _, line := range req.Lines {
		totalBeforeTax += line.Quantity * line.UnitPrice
		for _, tax := range line.Taxes {
			switch tax.Type {
			case TaxIVA:
				taxAmounts[cufe.TaxIVA] += tax.Amount
			case TaxINC:
				taxAmounts[cufe.TaxINC] += tax.Amount
			case TaxICA:
				taxAmounts[cufe.TaxICA] += tax.Amount
			}
		}
	}

	// Build taxes slice
	for code, amount := range taxAmounts {
		taxes = append(taxes, cufe.Tax{Code: code, Amount: amount})
	}

	// Calculate total payable
	totalPayable := totalBeforeTax
	for _, t := range taxes {
		totalPayable += t.Amount
	}

	// Get issue date/time
	now := time.Now()
	issueDate := now
	if req.IssueDate != nil {
		issueDate = *req.IssueDate
	}
	if req.IssueTime != nil {
		// Combine date from IssueDate with time from IssueTime
		t := *req.IssueTime
		issueDate = time.Date(
			issueDate.Year(), issueDate.Month(), issueDate.Day(),
			t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
			issueDate.Location(),
		)
	}

	// Get technical key
	techKey := req.TechnicalKey
	if techKey == "" && req.Resolution != nil {
		techKey = req.Resolution.TechnicalKey
	}

	// Map environment
	cufeEnv := cufe.EnvTesting
	if env == Produccion {
		cufeEnv = cufe.EnvProduction
	}

	// Build invoice for go-cufe
	inv := cufe.Invoice{
		Number:         req.Prefix + req.Number,
		IssueDate:      issueDate,
		TotalBeforeTax: totalBeforeTax,
		Taxes:          taxes,
		TotalPayable:   totalPayable,
		IssuerNIT:      req.Issuer.NIT,
		CustomerNIT:    req.Customer.NIT,
		TechnicalKey:   techKey,
		Environment:    cufeEnv,
	}

	// Calculate using go-cufe
	result, err := cufe.Calculate(inv)
	if err != nil {
		// Fallback: return empty string on error (shouldn't happen with valid data)
		return ""
	}

	return result.Code
}

// CalculateCUDE calculates the CUDE for credit/debit notes.
func CalculateCUDE(req *InvoiceRequest, env Environment) string {
	// For credit/debit notes, CUDE uses SoftwarePIN instead of TechnicalKey
	// and document type in the hash. Using go-cufe's CUDE calculation.

	var totalBeforeTax float64
	var taxes []cufe.Tax

	taxAmounts := map[cufe.TaxCode]float64{
		cufe.TaxIVA: 0,
		cufe.TaxINC: 0,
		cufe.TaxICA: 0,
	}

	for _, line := range req.Lines {
		totalBeforeTax += line.Quantity * line.UnitPrice
		for _, tax := range line.Taxes {
			switch tax.Type {
			case TaxIVA:
				taxAmounts[cufe.TaxIVA] += tax.Amount
			case TaxINC:
				taxAmounts[cufe.TaxINC] += tax.Amount
			case TaxICA:
				taxAmounts[cufe.TaxICA] += tax.Amount
			}
		}
	}

	for code, amount := range taxAmounts {
		taxes = append(taxes, cufe.Tax{Code: code, Amount: amount})
	}

	totalPayable := totalBeforeTax
	for _, t := range taxes {
		totalPayable += t.Amount
	}

	now := time.Now()
	issueDate := now
	if req.IssueDate != nil {
		issueDate = *req.IssueDate
	}
	if req.IssueTime != nil {
		t := *req.IssueTime
		issueDate = time.Date(
			issueDate.Year(), issueDate.Month(), issueDate.Day(),
			t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
			issueDate.Location(),
		)
	}

	cufeEnv := cufe.EnvTesting
	if env == Produccion {
		cufeEnv = cufe.EnvProduction
	}

	// Map document type
	var docType cufe.DocumentType
	switch req.Type {
	case DocCreditNote:
		docType = cufe.DocCreditNote
	case DocDebitNote:
		docType = cufe.DocDebitNote
	default:
		docType = cufe.DocCreditNote
	}

	doc := cufe.Document{
		Type:           docType,
		Number:         req.Prefix + req.Number,
		IssueDate:      issueDate,
		TotalBeforeTax: totalBeforeTax,
		Taxes:          taxes,
		TotalPayable:   totalPayable,
		IssuerNIT:      req.Issuer.NIT,
		CustomerNIT:    req.Customer.NIT,
		SoftwarePIN:    req.SoftwarePIN,
		Environment:    cufeEnv,
	}

	result, err := cufe.CalculateCUDE(doc)
	if err != nil {
		return ""
	}

	return result.Code
}

// CalculateSoftwareSecurityCode calculates the software security code.
// SHA384(SoftwareID + PIN + NIT)
func CalculateSoftwareSecurityCode(softwareID, pin, nit string) string {
	data := softwareID + pin + nit
	hash := sha512.Sum384([]byte(data))
	return strings.ToLower(hex.EncodeToString(hash[:]))
}
