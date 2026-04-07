package dian

import (
	"bytes"
	"context"
	"fmt"
)

// Send sends an invoice/document to DIAN.
// This is the main method - just pass your data, we handle everything:
// 1. Build UBL XML
// 2. Calculate CUFE/CUDE
// 3. Sign with XAdES-EPES
// 4. Create ZIP
// 5. Build SOAP envelope
// 6. Sign with WS-Security
// 7. Send to DIAN
//
// Example:
//
//	resp, err := client.Send(dian.InvoiceRequest{
//	    Type:   dian.DocInvoice,
//	    Prefix: "SETP",
//	    Number: "990000001",
//	    Issuer: dian.Party{
//	        NIT:  "900123456",
//	        DV:   "7",
//	        Name: "Mi Empresa SAS",
//	    },
//	    Customer: dian.Party{
//	        NIT:  "800987654",
//	        Name: "Cliente SA",
//	    },
//	    Lines: []dian.InvoiceLine{{
//	        Quantity:    1,
//	        UnitCode:    "EA",
//	        Description: "Servicio de consultoría",
//	        UnitPrice:   1000000,
//	        Taxes: []dian.Tax{{
//	            Type:        "01",
//	            Percent:     19,
//	            Amount:      190000,
//	            TaxableBase: 1000000,
//	        }},
//	    }},
//	})
func (c *Client) Send(req InvoiceRequest) (*Response, error) {
	return c.SendContext(context.Background(), req)
}

// SendContext sends with context support for cancellation/timeout.
func (c *Client) SendContext(ctx context.Context, req InvoiceRequest) (*Response, error) {
	// 1. Build UBL XML from request
	xml, err := buildInvoiceXML(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to build XML: %w", err)
	}

	// 2. Calculate CUFE and inject into XML
	cufe := CalculateCUFE(&req, c.env)
	xml = injectCUFE(xml, cufe)

	// 3. Sign with XAdES-EPES
	signedXML, err := c.xadesSigner.Sign(xml)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// 4. Create ZIP
	xmlFileName := fmt.Sprintf("%s-%s-%s", req.Issuer.NIT, req.Prefix, req.Number)
	zipContent, err := CreateZip(xmlFileName, signedXML)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	// 5-7. Send (SOAP build + WS-Security + HTTP)
	zipFileName := xmlFileName + ".zip"
	resp, err := c.SendBillSyncContext(ctx, zipFileName, zipContent)
	if err != nil {
		return nil, err
	}

	// Add CUFE to response
	if resp.DocumentKey == "" {
		resp.DocumentKey = cufe
	}

	return resp, nil
}

// SendAsync sends asynchronously (for large documents).
func (c *Client) SendAsync(req InvoiceRequest) (*Response, error) {
	return c.SendAsyncContext(context.Background(), req)
}

// SendAsyncContext sends asynchronously with context.
func (c *Client) SendAsyncContext(ctx context.Context, req InvoiceRequest) (*Response, error) {
	xml, err := buildInvoiceXML(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to build XML: %w", err)
	}

	cufe := CalculateCUFE(&req, c.env)
	xml = injectCUFE(xml, cufe)

	signedXML, err := c.xadesSigner.Sign(xml)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	xmlFileName := fmt.Sprintf("%s-%s-%s", req.Issuer.NIT, req.Prefix, req.Number)
	zipContent, err := CreateZip(xmlFileName, signedXML)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	zipFileName := xmlFileName + ".zip"
	return c.SendBillAsyncContext(ctx, zipFileName, zipContent)
}

// SendTestSet sends to test set for habilitacion process.
func (c *Client) SendTestSet(req InvoiceRequest, testSetID string) (*Response, error) {
	return c.SendTestSetContext(context.Background(), req, testSetID)
}

// SendTestSetContext sends test set with context.
func (c *Client) SendTestSetContext(ctx context.Context, req InvoiceRequest, testSetID string) (*Response, error) {
	xml, err := buildInvoiceXML(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to build XML: %w", err)
	}

	cufe := CalculateCUFE(&req, c.env)
	xml = injectCUFE(xml, cufe)

	signedXML, err := c.xadesSigner.Sign(xml)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	xmlFileName := fmt.Sprintf("%s-%s-%s", req.Issuer.NIT, req.Prefix, req.Number)
	zipContent, err := CreateZip(xmlFileName, signedXML)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	zipFileName := xmlFileName + ".zip"
	return c.SendTestSetAsyncContext(ctx, zipFileName, zipContent, testSetID)
}

// SendCreditNote sends a credit note.
func (c *Client) SendCreditNote(req CreditNoteRequest) (*Response, error) {
	req.Type = DocCreditNote
	return c.Send(req.InvoiceRequest)
}

// SendDebitNote sends a debit note.
func (c *Client) SendDebitNote(req DebitNoteRequest) (*Response, error) {
	req.Type = DocDebitNote
	return c.Send(req.InvoiceRequest)
}

// injectCUFE replaces the empty UUID with the calculated CUFE.
func injectCUFE(xml []byte, cufe string) []byte {
	// Find and replace empty UUID
	placeholder := []byte(`<cbc:UUID schemeID="2" schemeName="CUFE-SHA384"></cbc:UUID>`)
	replacement := []byte(fmt.Sprintf(`<cbc:UUID schemeID="2" schemeName="CUFE-SHA384">%s</cbc:UUID>`, cufe))
	return bytes.Replace(xml, placeholder, replacement, 1)
}

// Legacy types for backward compatibility (use InvoiceRequest instead)

// Invoice represents raw XML invoice (legacy - use InvoiceRequest).
type Invoice struct {
	NIT    string
	Prefix string
	Number string
	XML    []byte
}

// SendInvoice sends raw XML invoice (legacy - use Send instead).
func (c *Client) SendInvoice(inv Invoice) (*Response, error) {
	return c.SendInvoiceContext(context.Background(), inv)
}

// SendInvoiceContext sends raw XML with context (legacy).
func (c *Client) SendInvoiceContext(ctx context.Context, inv Invoice) (*Response, error) {
	signedXML, err := c.xadesSigner.Sign(inv.XML)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	xmlFileName := fmt.Sprintf("%s-%s-%s", inv.NIT, inv.Prefix, inv.Number)
	zipContent, err := CreateZip(xmlFileName, signedXML)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	return c.SendBillSyncContext(ctx, xmlFileName+".zip", zipContent)
}
