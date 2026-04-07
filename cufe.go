package dian

import (
	"crypto/sha512"
	"encoding/hex"
	"strings"
	"time"

	"github.com/SimpleX-Corp/go-cufe"
	"github.com/SimpleX-Corp/go-dian-codes"
)

// CalculateCUFE calculates the CUFE for an invoice using go-cufe library.
// go-cufe is a pure algorithm library - we provide all codes from go-dian-codes.
func CalculateCUFE(req *InvoiceRequest, env Environment) string {
	// Calculate totals from lines
	var totalBeforeTax float64

	// Aggregate taxes by type (using codes from go-dian-codes)
	taxAmounts := map[string]float64{
		string(codes.TaxIVA): 0,
		string(codes.TaxINC): 0,
		string(codes.TaxICA): 0,
	}

	for _, line := range req.Lines {
		totalBeforeTax += line.Quantity * line.UnitPrice
		for _, tax := range line.Taxes {
			switch tax.Type {
			case TaxIVA:
				taxAmounts[string(codes.TaxIVA)] += tax.Amount
			case TaxINC:
				taxAmounts[string(codes.TaxINC)] += tax.Amount
			case TaxICA:
				taxAmounts[string(codes.TaxICA)] += tax.Amount
			}
		}
	}

	// Calculate total payable
	totalPayable := totalBeforeTax
	for _, amount := range taxAmounts {
		totalPayable += amount
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

	// Map environment to string code
	envCode := "2" // Test/Habilitación
	if env == Produccion {
		envCode = "1"
	}

	// Build input for go-cufe (pure algorithm - all values are raw strings)
	input := cufe.CUFEInput{
		Number:         req.Prefix + req.Number,
		IssueDate:      issueDate,
		TotalBeforeTax: totalBeforeTax,
		Tax1Code:       string(codes.TaxIVA), // "01"
		Tax1Amount:     taxAmounts[string(codes.TaxIVA)],
		Tax2Code:       string(codes.TaxINC), // "04"
		Tax2Amount:     taxAmounts[string(codes.TaxINC)],
		Tax3Code:       string(codes.TaxICA), // "03"
		Tax3Amount:     taxAmounts[string(codes.TaxICA)],
		TotalPayable:   totalPayable,
		IssuerNIT:      req.Issuer.NIT,
		CustomerNIT:    req.Customer.NIT,
		TechnicalKey:   techKey,
		Environment:    envCode,
	}

	result := cufe.CalculateCUFE(input)
	return result.Hash
}

// CalculateCUDE calculates the CUDE for credit/debit notes.
// Uses SoftwarePIN instead of TechnicalKey.
func CalculateCUDE(req *InvoiceRequest, env Environment) string {
	var totalBeforeTax float64

	taxAmounts := map[string]float64{
		string(codes.TaxIVA): 0,
		string(codes.TaxINC): 0,
		string(codes.TaxICA): 0,
	}

	for _, line := range req.Lines {
		totalBeforeTax += line.Quantity * line.UnitPrice
		for _, tax := range line.Taxes {
			switch tax.Type {
			case TaxIVA:
				taxAmounts[string(codes.TaxIVA)] += tax.Amount
			case TaxINC:
				taxAmounts[string(codes.TaxINC)] += tax.Amount
			case TaxICA:
				taxAmounts[string(codes.TaxICA)] += tax.Amount
			}
		}
	}

	totalPayable := totalBeforeTax
	for _, amount := range taxAmounts {
		totalPayable += amount
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

	envCode := "2"
	if env == Produccion {
		envCode = "1"
	}

	input := cufe.CUDEInput{
		Number:         req.Prefix + req.Number,
		IssueDate:      issueDate,
		TotalBeforeTax: totalBeforeTax,
		Tax1Code:       string(codes.TaxIVA),
		Tax1Amount:     taxAmounts[string(codes.TaxIVA)],
		Tax2Code:       string(codes.TaxINC),
		Tax2Amount:     taxAmounts[string(codes.TaxINC)],
		Tax3Code:       string(codes.TaxICA),
		Tax3Amount:     taxAmounts[string(codes.TaxICA)],
		TotalPayable:   totalPayable,
		IssuerNIT:      req.Issuer.NIT,
		CustomerNIT:    req.Customer.NIT,
		SoftwarePIN:    req.SoftwarePIN,
		Environment:    envCode,
	}

	result := cufe.CalculateCUDE(input)
	return result.Hash
}

// CalculateSoftwareSecurityCode calculates the software security code.
// SHA384(SoftwareID + PIN + NIT)
func CalculateSoftwareSecurityCode(softwareID, pin, nit string) string {
	data := softwareID + pin + nit
	hash := sha512.Sum384([]byte(data))
	return strings.ToLower(hex.EncodeToString(hash[:]))
}
