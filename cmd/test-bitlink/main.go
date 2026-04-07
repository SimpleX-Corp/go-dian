// Command test-bitlink tests with BITLINK S.A.S configuration for DIAN sandbox.
//
// Usage:
//
//	go run main.go                           # Test local (no cert needed)
//	go run main.go -cert cert.p12 -pass xxx  # Test with DIAN sandbox
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/SimpleX-Corp/go-dian"
)

// BITLINK S.A.S Configuration
const (
	CompanyName = "BITLINK S.A.S"
	SoftwareID  = "f3b19b5d-cc78-4c77-82ea-466037d9459c"
)

func main() {
	certPath := flag.String("cert", "", "Path to .p12 certificate")
	certPass := flag.String("pass", "", "Certificate password")
	nit := flag.String("nit", "", "NIT of BITLINK (without DV)")
	dv := flag.String("dv", "", "Verification digit")
	softwarePIN := flag.String("pin", "", "Software PIN from DIAN")
	testSetID := flag.String("testset", "", "Test Set ID for habilitación")
	prefix := flag.String("prefix", "SETP", "Invoice prefix")
	techKey := flag.String("techkey", "", "Technical key from resolution")
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           BITLINK S.A.S - DIAN Sandbox Test                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Company:     %s\n", CompanyName)
	fmt.Printf("Software ID: %s\n", SoftwareID)
	fmt.Printf("Mode:        SP (Software Propio)\n")
	fmt.Printf("Environment: Habilitación (Sandbox)\n")
	fmt.Println()

	if *certPath == "" {
		fmt.Println("⚠️  No certificate provided - running LOCAL test only")
		fmt.Println()
		runLocalTest(*nit, *dv, *softwarePIN, *prefix, *techKey)
		return
	}

	if *nit == "" || *dv == "" {
		fmt.Println("ERROR: -nit and -dv required for DIAN test")
		fmt.Println("Example: -nit 900123456 -dv 7")
		os.Exit(1)
	}

	runDIANTest(*certPath, *certPass, *nit, *dv, *softwarePIN, *prefix, *techKey, *testSetID)
}

func runLocalTest(nit, dv, pin, prefix, techKey string) {
	if nit == "" {
		nit = "900000000" // Placeholder
		dv = "0"
	}
	if pin == "" {
		pin = "12345"
	}
	if techKey == "" {
		techKey = "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148"
	}

	now := time.Now()
	invoiceNumber := fmt.Sprintf("%d", now.Unix()%1000000)

	req := createTestInvoice(nit, dv, pin, prefix, invoiceNumber, techKey, now)

	fmt.Println("=== Invoice Data ===")
	printInvoiceInfo(req)

	// Calculate CUFE
	fmt.Println("\n=== CUFE Calculation ===")
	cufe := dian.CalculateCUFE(req, dian.Habilitacion)
	fmt.Printf("CUFE: %s\n", cufe)

	// Software Security Code
	fmt.Println("\n=== Software Security Code ===")
	secCode := dian.CalculateSoftwareSecurityCode(SoftwareID, pin, nit)
	fmt.Printf("Code: %s\n", secCode)

	// Create ZIP
	fmt.Println("\n=== ZIP Creation ===")
	dummyXML := []byte(`<?xml version="1.0"?><Invoice/>`)
	zipContent, _ := dian.CreateZipFast(nit+"-"+prefix+"-"+invoiceNumber, dummyXML)
	fmt.Printf("ZIP Size: %d bytes\n", len(zipContent))
	fmt.Printf("Filename: %s\n", dian.ZipFileName(nit, prefix, invoiceNumber))

	fmt.Println("\n=== QR Data ===")
	printQRData(req, cufe)

	fmt.Println("\n✅ Local test completed!")
	fmt.Println("\nTo test with DIAN sandbox, run:")
	fmt.Printf("  go run main.go -cert your-cert.p12 -pass password -nit %s -dv %s -pin YOUR_PIN\n", nit, dv)
}

func runDIANTest(certPath, certPass, nit, dv, pin, prefix, techKey, testSetID string) {
	fmt.Println("=== Connecting to DIAN Sandbox ===")
	fmt.Printf("Certificate: %s\n", certPath)
	fmt.Println()

	// Create client
	client, err := dian.NewClient(certPath, certPass, dian.Habilitacion)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to load certificate: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Show certificate info
	info := client.CertificateInfo()
	fmt.Println("=== Certificate ===")
	fmt.Printf("Subject: %s\n", info.Subject)
	fmt.Printf("Issuer:  %s\n", info.Issuer)
	fmt.Printf("Valid:   %v\n", info.IsValid)
	if !info.IsValid {
		fmt.Println("❌ WARNING: Certificate is not valid!")
	}
	fmt.Println()

	// Create invoice
	now := time.Now()
	invoiceNumber := fmt.Sprintf("%d", now.Unix()%1000000)
	req := createTestInvoice(nit, dv, pin, prefix, invoiceNumber, techKey, now)

	fmt.Println("=== Sending Invoice ===")
	printInvoiceInfo(req)
	fmt.Println()

	var resp *dian.Response

	if testSetID != "" {
		fmt.Printf("Sending to Test Set: %s\n", testSetID)
		resp, err = client.SendTestSet(*req, testSetID)
	} else {
		fmt.Println("Sending synchronously...")
		resp, err = client.Send(*req)
	}

	if err != nil {
		fmt.Printf("❌ ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== DIAN Response ===")
	printResponse(resp)
}

func createTestInvoice(nit, dv, pin, prefix, number, techKey string, issueTime time.Time) *dian.InvoiceRequest {
	return &dian.InvoiceRequest{
		Type:      dian.DocInvoice,
		Prefix:    prefix,
		Number:    number,
		IssueDate: &issueTime,
		IssueTime: &issueTime,
		Currency:  dian.CurrencyCOP,
		Issuer: dian.Party{
			NIT:     nit,
			DV:      dv,
			Name:    CompanyName,
			DocType: dian.IDNIT,
			TaxResponsibilities: []string{"O-47", "O-15"},
			Address: dian.Address{
				Street:      "Dirección de prueba",
				City:        "Bogotá",
				CityCode:    "11001",
				Department:  "Bogotá D.C.",
				DeptCode:    "11",
				Country:     "Colombia",
				CountryCode: "CO",
			},
			Email: "facturacion@bitlink.com.co",
			Phone: "6011234567",
		},
		Customer: dian.Party{
			NIT:     "222222222222",
			DV:      "2",
			Name:    "CONSUMIDOR FINAL",
			DocType: dian.IDCedula, // Cédula de ciudadanía
			TaxResponsibilities: []string{"R-99-PN"},
			Address: dian.Address{
				City:     "Bogotá",
				CityCode: "11001",
				DeptCode: "11",
			},
			Email: "cliente@example.com",
		},
		Lines: []dian.InvoiceLine{
			{
				Number:      1,
				Quantity:    1,
				UnitCode:    "EA",
				Description: "Servicio de desarrollo de software - Prueba habilitación DIAN",
				UnitPrice:   100000,
				ProductCode: "43231500",
				Taxes: []dian.Tax{
					{
						Type:        dian.TaxIVA,
						Percent:     19,
						Amount:      19000,
						TaxableBase: 100000,
					},
				},
			},
		},
		Payment: dian.Payment{
			Method: dian.PaymentCash,
			Means:  dian.MeansCash,
		},
		Notes:        []string{"Factura de prueba para habilitación DIAN - " + CompanyName},
		SoftwareID:   SoftwareID,
		SoftwarePIN:  pin,
		TechnicalKey: techKey,
	}
}

func printInvoiceInfo(req *dian.InvoiceRequest) {
	subtotal := 0.0
	tax := 0.0
	for _, line := range req.Lines {
		subtotal += line.Quantity * line.UnitPrice
		for _, t := range line.Taxes {
			tax += t.Amount
		}
	}

	fmt.Printf("Number:   %s%s\n", req.Prefix, req.Number)
	fmt.Printf("Date:     %s\n", req.IssueDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Issuer:   %s (NIT: %s-%s)\n", req.Issuer.Name, req.Issuer.NIT, req.Issuer.DV)
	fmt.Printf("Customer: %s\n", req.Customer.Name)
	fmt.Printf("Subtotal: $%.2f COP\n", subtotal)
	fmt.Printf("IVA 19%%:  $%.2f COP\n", tax)
	fmt.Printf("Total:    $%.2f COP\n", subtotal+tax)
}

func printQRData(req *dian.InvoiceRequest, cufe string) {
	subtotal := 0.0
	tax := 0.0
	for _, line := range req.Lines {
		subtotal += line.Quantity * line.UnitPrice
		for _, t := range line.Taxes {
			tax += t.Amount
		}
	}

	fmt.Printf("NumFac: %s%s\n", req.Prefix, req.Number)
	fmt.Printf("FecFac: %s\n", req.IssueDate.Format("2006-01-02"))
	fmt.Printf("NitFac: %s\n", req.Issuer.NIT)
	fmt.Printf("DocAdq: %s\n", req.Customer.NIT)
	fmt.Printf("ValTot: %.2f\n", subtotal+tax)
	fmt.Printf("CUFE:   %s\n", cufe)
}

func printResponse(resp *dian.Response) {
	status := "❌"
	if resp.IsSuccess() {
		status = "✅"
	} else if resp.IsPending() {
		status = "⏳"
	}

	fmt.Printf("Status:  %s %s (%s)\n", status, resp.StatusDescription, resp.StatusCode)

	if resp.DocumentKey != "" {
		fmt.Printf("CUFE:    %s\n", resp.DocumentKey)
	}

	if resp.StatusMessage != "" {
		fmt.Printf("Message: %s\n", resp.StatusMessage)
	}

	if len(resp.ErrorMessages) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range resp.ErrorMessages {
			fmt.Printf("  • %s\n", e)
		}
	}

	// JSON
	fmt.Println("\nJSON Response:")
	out, _ := json.MarshalIndent(map[string]interface{}{
		"success":      resp.IsSuccess(),
		"pending":      resp.IsPending(),
		"status_code":  resp.StatusCode,
		"document_key": resp.DocumentKey,
		"errors":       resp.ErrorMessages,
	}, "", "  ")
	fmt.Println(string(out))
}
