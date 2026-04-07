# CLAUDE.md - Development Guidelines for go-dian

## Project Overview

**go-dian** is a unified Go client for DIAN Colombia electronic invoicing.

### Architecture

This package integrates the complete DIAN stack:

```
go-dian (THIS - unified API)
├── go-xades-dian     -> XAdES-EPES invoice signatures
├── go-soap-builder   -> SOAP 1.2 envelope construction
├── go-wssecurity     -> WS-Security signing
└── go-dian-client    -> HTTP transport
```

### Key Features

- Single API for all DIAN operations
- Certificate loaded once, used for both XAdES and WS-Security
- Context support for cancellation/timeouts
- Configurable retries with exponential backoff
- Thread-safe concurrent usage

## Usage

```go
// Create client
client, err := dian.NewClient("cert.p12", "password", dian.Habilitacion)
defer client.Close()

// Sign invoice (XAdES-EPES)
signedXML, err := client.SignInvoice(invoiceXML)

// Send to DIAN
response, err := client.SendBillSync("invoice.zip", zipContent)

// Check result
if response.IsSuccess() {
    fmt.Println("CUFE:", response.DocumentKey)
}
```

## DIAN Methods

| Method | Description |
|--------|-------------|
| SendBillSync | Synchronous invoice submission |
| SendBillAsync | Asynchronous invoice submission |
| SendTestSetAsync | Test set for habilitacion |
| GetStatus | Check submission status |
| GetStatusZip | Get response as ZIP |
| GetNumberingRange | Query authorized ranges |

## Dependencies

This package depends on:
- github.com/SimpleX-Corp/go-xades-dian
- github.com/SimpleX-Corp/go-soap-builder
- github.com/SimpleX-Corp/go-wssecurity
- github.com/SimpleX-Corp/go-dian-client
