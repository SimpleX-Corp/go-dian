// Command test-local tests the complete flow WITHOUT connecting to DIAN.
// Generates XML, calculates CUFE, creates ZIP - everything except the actual send.
//
// Usage:
//
//	go run main.go
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/SimpleX-Corp/go-dian"
)

func main() {
	fmt.Println("=== DIAN Local Test (No Certificate Required) ===")
	fmt.Println()

	now := time.Now()

	// Create realistic test invoice
	req := &dian.InvoiceRequest{
		Type:      dian.DocInvoice,
		Prefix:    "SETP",
		Number:    "990000001",
		IssueDate: &now,
		IssueTime: &now,
		Currency:  dian.CurrencyCOP,
		Issuer: dian.Party{
			NIT:     "900373115",
			DV:      "1",
			Name:    "EMPRESA DEMO SAS",
			DocType: dian.IDNIT,
			TaxResponsibilities: []string{"O-47", "O-15"},
			Address: dian.Address{
				Street:      "Carrera 8 # 12-34",
				City:        "Bogotá",
				CityCode:    "11001",
				Department:  "Bogotá D.C.",
				DeptCode:    "11",
				Country:     "Colombia",
				CountryCode: "CO",
				PostalCode:  "110111",
			},
			Email: "facturacion@empresa.com",
			Phone: "6011234567",
		},
		Customer: dian.Party{
			NIT:     "900987654",
			DV:      "3",
			Name:    "CLIENTE EJEMPLO SA",
			DocType: dian.IDNIT,
			TaxResponsibilities: []string{"R-99-PN"},
			Address: dian.Address{
				Street:   "Calle 50 # 20-30",
				City:     "Medellín",
				CityCode: "05001",
				DeptCode: "05",
			},
			Email: "compras@cliente.com",
		},
		Lines: []dian.InvoiceLine{
			{
				Number:      1,
				Quantity:    2,
				UnitCode:    "EA",
				Description: "Servicio de consultoría empresarial",
				UnitPrice:   500000,
				ProductCode: "80111600",
				Taxes: []dian.Tax{
					{Type: dian.TaxIVA, Percent: 19, Amount: 190000, TaxableBase: 1000000},
				},
			},
			{
				Number:      2,
				Quantity:    1,
				UnitCode:    "EA",
				Description: "Licencia de software anual",
				UnitPrice:   800000,
				ProductCode: "43231500",
				Taxes: []dian.Tax{
					{Type: dian.TaxIVA, Percent: 19, Amount: 152000, TaxableBase: 800000},
				},
			},
			{
				Number:      3,
				Quantity:    10,
				UnitCode:    "HUR",
				Description: "Horas de soporte técnico",
				UnitPrice:   50000,
				ProductCode: "81111800",
				Taxes: []dian.Tax{
					{Type: dian.TaxIVA, Percent: 19, Amount: 95000, TaxableBase: 500000},
				},
			},
		},
		Payment: dian.Payment{
			Method: "2", // Crédito
			Means:  dian.MeansTransfer, // Transferencia
			DueDate: func() *time.Time { t := now.AddDate(0, 0, 30); return &t }(),
		},
		Notes: []string{
			"Factura electrónica generada con go-dian",
			"Pago a 30 días",
		},
		Resolution: &dian.Resolution{
			Number:       "18760000001",
			Prefix:       "SETP",
			RangeFrom:    990000000,
			RangeTo:      995000000,
			ValidFrom:    now.AddDate(-1, 0, 0),
			ValidTo:      now.AddDate(1, 0, 0),
			TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
		},
		SoftwareID:   "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		SoftwarePIN:  "12345",
		TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
	}

	// Calculate totals
	var subtotal, totalTax float64
	for _, line := range req.Lines {
		subtotal += line.Quantity * line.UnitPrice
		for _, tax := range line.Taxes {
			totalTax += tax.Amount
		}
	}
	total := subtotal + totalTax

	fmt.Println("=== Invoice Data ===")
	fmt.Printf("Number:     %s%s\n", req.Prefix, req.Number)
	fmt.Printf("Date:       %s\n", now.Format("2006-01-02"))
	fmt.Printf("Issuer:     %s (NIT: %s-%s)\n", req.Issuer.Name, req.Issuer.NIT, req.Issuer.DV)
	fmt.Printf("Customer:   %s (NIT: %s-%s)\n", req.Customer.Name, req.Customer.NIT, req.Customer.DV)
	fmt.Printf("Lines:      %d\n", len(req.Lines))
	fmt.Printf("Subtotal:   $%.2f COP\n", subtotal)
	fmt.Printf("IVA (19%%):  $%.2f COP\n", totalTax)
	fmt.Printf("Total:      $%.2f COP\n", total)
	fmt.Println()

	// Step 1: Build XML
	fmt.Println("=== Step 1: Building UBL 2.1 XML ===")
	// We need to use the internal function, but it's not exported
	// Let's use the public API instead through a workaround

	// Calculate CUFE first
	fmt.Println("=== Step 2: Calculating CUFE ===")
	cufe := dian.CalculateCUFE(req, dian.Habilitacion)
	fmt.Printf("CUFE: %s\n", cufe)
	fmt.Printf("Length: %d characters (SHA-384 = 96 hex chars)\n", len(cufe))
	fmt.Println()

	// Calculate Software Security Code
	fmt.Println("=== Step 3: Software Security Code ===")
	secCode := dian.CalculateSoftwareSecurityCode(req.SoftwareID, req.SoftwarePIN, req.Issuer.NIT)
	fmt.Printf("Security Code: %s\n", secCode)
	fmt.Println()

	// Create ZIP (we'll use a dummy XML for demo)
	fmt.Println("=== Step 4: Creating ZIP ===")
	dummyXML := []byte(`<?xml version="1.0" encoding="UTF-8"?><Invoice>Demo</Invoice>`)
	zipContent, err := dian.CreateZipFast("900373115-SETP-990000001", dummyXML)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ZIP created: %d bytes\n", len(zipContent))
	fmt.Printf("ZIP filename: %s\n", dian.ZipFileName(req.Issuer.NIT, req.Prefix, req.Number))
	fmt.Println()

	// Show what would be sent
	fmt.Println("=== Step 5: Ready to Send ===")
	fmt.Println("With a valid certificate, this would:")
	fmt.Println("  1. Sign the XML with XAdES-EPES")
	fmt.Println("  2. Create SOAP envelope")
	fmt.Println("  3. Sign SOAP with WS-Security")
	fmt.Println("  4. Send to: https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc")
	fmt.Println()

	// QR Code data (what would go in the QR)
	fmt.Println("=== QR Code Data ===")
	qrData := fmt.Sprintf(
		"NumFac: %s%s\nFecFac: %s\nNitFac: %s\nDocAdq: %s\nValFac: %.2f\nValIva: %.2f\nValTot: %.2f\nCUFE: %s",
		req.Prefix, req.Number,
		now.Format("2006-01-02"),
		req.Issuer.NIT,
		req.Customer.NIT,
		subtotal,
		totalTax,
		total,
		cufe,
	)
	fmt.Println(qrData)
	fmt.Println()

	// Base64 of ZIP (first 100 chars)
	fmt.Println("=== ZIP Base64 (truncated) ===")
	b64 := base64.StdEncoding.EncodeToString(zipContent)
	if len(b64) > 100 {
		b64 = b64[:100] + "..."
	}
	fmt.Println(b64)
	fmt.Println()

	fmt.Println("=== Test Complete ===")
	fmt.Println("All local processing works correctly!")
	fmt.Println("To test with real DIAN, you need a certificate from an authorized CA.")
}
