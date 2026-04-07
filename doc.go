// Package dian provides a unified client for DIAN Colombia electronic invoicing.
//
// This package integrates all components of the DIAN stack:
//   - go-xades-dian: XAdES-EPES signatures for invoices
//   - go-soap-builder: SOAP 1.2 envelope construction
//   - go-wssecurity: WS-Security signing
//   - go-dian-client: HTTP transport
//
// # Quick Start
//
//	// Create client (loads certificate once)
//	client, err := dian.NewClient("cert.p12", "password", dian.Habilitacion)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Send invoice synchronously
//	response, err := client.SendBillSync("invoice.zip", zipContent)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if response.IsSuccess() {
//	    fmt.Println("CUFE:", response.DocumentKey)
//	}
//
// # Architecture
//
// The client handles the complete workflow:
//
//	1. Build SOAP envelope with WS-Addressing
//	2. Sign envelope with WS-Security (RSA-SHA256)
//	3. Send to DIAN endpoint
//	4. Parse and return response
//
// For invoice signing (XAdES-EPES), use SignInvoice() before sending.
package dian
