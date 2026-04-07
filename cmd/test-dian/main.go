// Command test-dian is a manual testing tool for DIAN sandbox.
//
// Usage:
//
//	go run main.go -cert cert.p12 -pass password -action test
//
// Actions:
//
//	test      - Test certificate and connection
//	ranges    - Query numbering ranges
//	invoice   - Send test invoice
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/SimpleX-Corp/go-dian"
)

func main() {
	certPath := flag.String("cert", "", "Path to .p12 certificate")
	certPass := flag.String("pass", "", "Certificate password")
	action := flag.String("action", "test", "Action: test, ranges, invoice")
	env := flag.String("env", "hab", "Environment: hab or prod")
	nit := flag.String("nit", "", "NIT for ranges query")
	softwareID := flag.String("software-id", "", "Software ID")
	testSetID := flag.String("testset-id", "", "Test Set ID for habilitación")
	flag.Parse()

	if *certPath == "" || *certPass == "" {
		fmt.Println("Usage: test-dian -cert <path> -pass <password> -action <action>")
		fmt.Println("\nActions:")
		fmt.Println("  test     - Test certificate and connection")
		fmt.Println("  ranges   - Query numbering ranges (requires -nit)")
		fmt.Println("  invoice  - Send test invoice")
		os.Exit(1)
	}

	// Determine environment
	environment := dian.Habilitacion
	if *env == "prod" {
		environment = dian.Produccion
	}

	fmt.Printf("Environment: %s\n", *env)
	fmt.Printf("Certificate: %s\n", *certPath)
	fmt.Println()

	// Create client
	client, err := dian.NewClient(*certPath, *certPass, environment)
	if err != nil {
		fmt.Printf("ERROR: Failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Show certificate info
	info := client.CertificateInfo()
	fmt.Println("=== Certificate Info ===")
	fmt.Printf("Subject:      %s\n", info.Subject)
	fmt.Printf("Issuer:       %s\n", info.Issuer)
	fmt.Printf("Serial:       %s\n", info.SerialNumber)
	fmt.Printf("Valid:        %v\n", info.IsValid)
	fmt.Printf("Expired:      %v\n", info.IsExpired)
	fmt.Println()

	if !info.IsValid {
		fmt.Println("WARNING: Certificate is not valid!")
	}

	switch *action {
	case "test":
		fmt.Println("Certificate loaded successfully!")
		fmt.Println("Ready to connect to DIAN.")

	case "ranges":
		if *nit == "" {
			fmt.Println("ERROR: -nit required for ranges query")
			os.Exit(1)
		}
		queryRanges(client, *nit, *softwareID)

	case "invoice":
		sendTestInvoice(client, *nit, *softwareID, *testSetID)

	default:
		fmt.Printf("Unknown action: %s\n", *action)
	}
}

func queryRanges(client *dian.Client, nit, softwareID string) {
	fmt.Println("=== Querying Numbering Ranges ===")
	fmt.Printf("NIT: %s\n", nit)
	fmt.Printf("Software ID: %s\n", softwareID)
	fmt.Println()

	resp, err := client.GetNumberingRange(nit, nit, softwareID)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	printResponse(resp)
}

func sendTestInvoice(client *dian.Client, nit, softwareID, testSetID string) {
	fmt.Println("=== Sending Test Invoice ===")

	if nit == "" {
		nit = "900123456" // Placeholder
	}

	now := time.Now()

	// Create test invoice
	req := dian.InvoiceRequest{
		Type:      dian.DocInvoice,
		Prefix:    "SETP",
		Number:    fmt.Sprintf("%d", now.Unix()%1000000),
		IssueDate: &now,
		IssueTime: &now,
		Currency:  dian.CurrencyCOP,
		Issuer: dian.Party{
			NIT:     nit,
			DV:      "7",
			Name:    "EMPRESA DE PRUEBA SAS",
			DocType: dian.IDNIT,
			TaxResponsibilities: []string{"O-47"},
			Address: dian.Address{
				Street:      "Calle 100 # 10-20",
				City:        "Bogotá",
				CityCode:    "11001",
				Department:  "Bogotá D.C.",
				DeptCode:    "11",
				Country:     "Colombia",
				CountryCode: "CO",
			},
			Email: "test@example.com",
		},
		Customer: dian.Party{
			NIT:     "222222222",
			DV:      "2",
			Name:    "CLIENTE DE PRUEBA",
			DocType: dian.IDNIT,
			Address: dian.Address{
				City:     "Bogotá",
				CityCode: "11001",
				DeptCode: "11",
			},
		},
		Lines: []dian.InvoiceLine{
			{
				Number:      1,
				Quantity:    1,
				UnitCode:    "EA",
				Description: "Servicio de prueba para habilitación DIAN",
				UnitPrice:   100000,
				ProductCode: "80111600",
				Taxes: []dian.Tax{
					{
						Type:        dian.TaxIVA, // IVA
						Percent:     19,
						Amount:      19000,
						TaxableBase: 100000,
					},
				},
			},
		},
		Payment: dian.Payment{
			Method: dian.PaymentCash, // Contado
			Means:  dian.MeansCash, // Efectivo
		},
		SoftwareID: softwareID,
		Notes:      []string{"Factura de prueba - Proceso de habilitación"},
	}

	fmt.Printf("Invoice: %s%s\n", req.Prefix, req.Number)
	fmt.Printf("Issuer: %s (%s)\n", req.Issuer.Name, req.Issuer.NIT)
	fmt.Printf("Customer: %s (%s)\n", req.Customer.Name, req.Customer.NIT)
	fmt.Printf("Total: $%.2f COP\n", 119000.0)
	fmt.Println()

	var resp *dian.Response
	var err error

	if testSetID != "" {
		fmt.Printf("Sending to Test Set: %s\n", testSetID)
		resp, err = client.SendTestSet(req, testSetID)
	} else {
		fmt.Println("Sending synchronously...")
		resp, err = client.Send(req)
	}

	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	printResponse(resp)
}

func printResponse(resp *dian.Response) {
	fmt.Println()
	fmt.Println("=== DIAN Response ===")
	fmt.Printf("Valid:       %v\n", resp.IsValid)
	fmt.Printf("Status Code: %s\n", resp.StatusCode)
	fmt.Printf("Status:      %s\n", resp.StatusDescription)

	if resp.StatusMessage != "" {
		fmt.Printf("Message:     %s\n", resp.StatusMessage)
	}

	if resp.DocumentKey != "" {
		fmt.Printf("CUFE:        %s\n", resp.DocumentKey)
	}

	if len(resp.ErrorMessages) > 0 {
		fmt.Println("\nErrors:")
		for _, msg := range resp.ErrorMessages {
			fmt.Printf("  - %s\n", msg)
		}
	}

	// Pretty print raw response (first 500 chars)
	if len(resp.RawXML) > 0 {
		fmt.Println("\nRaw Response (truncated):")
		raw := string(resp.RawXML)
		if len(raw) > 500 {
			raw = raw[:500] + "..."
		}
		fmt.Println(raw)
	}

	// JSON output
	fmt.Println("\nJSON Response:")
	jsonResp, _ := json.MarshalIndent(map[string]interface{}{
		"success":      resp.IsSuccess(),
		"pending":      resp.IsPending(),
		"status_code":  resp.StatusCode,
		"status":       resp.StatusDescription,
		"document_key": resp.DocumentKey,
		"errors":       resp.ErrorMessages,
	}, "", "  ")
	fmt.Println(string(jsonResp))
}
