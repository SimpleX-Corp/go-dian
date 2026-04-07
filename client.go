package dian

import (
	"context"
	"time"

	client "github.com/SimpleX-Corp/go-dian-client"
	soap "github.com/SimpleX-Corp/go-soap-builder"
	wssec "github.com/SimpleX-Corp/go-wssecurity"
	xades "github.com/SimpleX-Corp/go-xades-dian"
)

// Environment selects the DIAN endpoint.
type Environment = soap.Environment

// Available environments.
const (
	Habilitacion = soap.Habilitacion
	Produccion   = soap.Produccion
)

// Client is a unified DIAN client that handles the complete workflow.
type Client struct {
	env          Environment
	soapBuilder  *soap.Builder
	wsSigner     *wssec.Signer
	xadesSigner  *xades.Signer
	httpClient   *client.Client
	xadesCert    *xades.Certificate
	wssCert      *wssec.Certificate
}

// Options configures the Client.
type Options struct {
	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration

	// MaxRetries for failed requests (default: 0)
	MaxRetries int

	// RetryDelay between retries (default: 1s)
	RetryDelay time.Duration
}

// NewClient creates a new unified DIAN client.
// The certificate is used for both XAdES and WS-Security signing.
func NewClient(certPath, password string, env Environment, opts ...Options) (*Client, error) {
	// Load certificate for XAdES signing
	xadesCert, err := xades.LoadP12(certPath, password)
	if err != nil {
		return nil, err
	}

	// Load certificate for WS-Security signing
	wssCert, err := wssec.LoadP12(certPath, password)
	if err != nil {
		return nil, err
	}

	// Apply options
	opt := Options{
		Timeout:    30 * time.Second,
		MaxRetries: 0,
		RetryDelay: time.Second,
	}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Create HTTP client
	httpClient := client.NewClient(
		client.WithTimeout(opt.Timeout),
		client.WithRetry(opt.MaxRetries, opt.RetryDelay),
	)

	return &Client{
		env:         env,
		soapBuilder: soap.NewBuilder(env),
		wsSigner:    wssec.NewSigner(wssCert),
		xadesSigner: xades.NewSigner(xadesCert),
		httpClient:  httpClient,
		xadesCert:   xadesCert,
		wssCert:     wssCert,
	}, nil
}

// NewClientFromBytes creates a client from certificate bytes.
func NewClientFromBytes(certData []byte, password string, env Environment, opts ...Options) (*Client, error) {
	xadesCert, err := xades.ParseP12(certData, password)
	if err != nil {
		return nil, err
	}

	wssCert, err := wssec.LoadP12Bytes(certData, password)
	if err != nil {
		return nil, err
	}

	opt := Options{
		Timeout:    30 * time.Second,
		MaxRetries: 0,
		RetryDelay: time.Second,
	}
	if len(opts) > 0 {
		opt = opts[0]
	}

	httpClient := client.NewClient(
		client.WithTimeout(opt.Timeout),
		client.WithRetry(opt.MaxRetries, opt.RetryDelay),
	)

	return &Client{
		env:         env,
		soapBuilder: soap.NewBuilder(env),
		wsSigner:    wssec.NewSigner(wssCert),
		xadesSigner: xades.NewSigner(xadesCert),
		httpClient:  httpClient,
		xadesCert:   xadesCert,
		wssCert:     wssCert,
	}, nil
}

// Close releases resources.
func (c *Client) Close() {
	c.httpClient.Close()
}

// Response wraps the DIAN response with convenience methods.
type Response struct {
	// IsValid indicates if the document was accepted
	IsValid bool

	// StatusCode is the DIAN status code (00=success, 66=pending, 99=error)
	StatusCode string

	// StatusDescription is the human-readable status
	StatusDescription string

	// StatusMessage contains additional information
	StatusMessage string

	// DocumentKey is the unique document identifier (CUFE/CUDE)
	DocumentKey string

	// ErrorMessages contains validation errors if any
	ErrorMessages []string

	// RawXML is the complete raw response
	RawXML []byte
}

// IsPending returns true if the document is still being processed.
func (r *Response) IsPending() bool {
	return r.StatusCode == "66"
}

// IsSuccess returns true if the document was processed successfully.
func (r *Response) IsSuccess() bool {
	return r.StatusCode == "00" && r.IsValid
}

// IsError returns true if there was a validation error.
func (r *Response) IsError() bool {
	return r.StatusCode == "99" || !r.IsValid
}

// convertResponse converts client.Response to our Response type.
func convertResponse(r *client.Response) *Response {
	return &Response{
		IsValid:           r.IsValid,
		StatusCode:        r.StatusCode,
		StatusDescription: r.StatusDescription,
		StatusMessage:     r.StatusMessage,
		DocumentKey:       r.XmlDocumentKey,
		ErrorMessages:     r.ErrorMessages,
		RawXML:            r.RawXML,
	}
}

// send is the internal method that handles SOAP building, signing, and sending.
func (c *Client) send(ctx context.Context, method soap.Method, body []byte) (*Response, error) {
	// 1. Build SOAP envelope
	envelope, err := c.soapBuilder.Build(method, body)
	if err != nil {
		return nil, err
	}

	// 2. Sign with WS-Security
	signedEnvelope, err := c.wsSigner.Sign(envelope.XML, envelope.BodyID, envelope.ToID)
	if err != nil {
		return nil, err
	}

	// 3. Send to DIAN
	resp, err := c.httpClient.SendContext(ctx, signedEnvelope, envelope.EndpointURL, envelope.SOAPAction)
	if err != nil {
		return nil, err
	}

	return convertResponse(resp), nil
}

// SendBillSync sends an invoice synchronously.
func (c *Client) SendBillSync(fileName string, zipContent []byte) (*Response, error) {
	return c.SendBillSyncContext(context.Background(), fileName, zipContent)
}

// SendBillSyncContext sends an invoice synchronously with context.
func (c *Client) SendBillSyncContext(ctx context.Context, fileName string, zipContent []byte) (*Response, error) {
	body := soap.BodySendBillSync(fileName, zipContent)
	return c.send(ctx, soap.SendBillSync, body)
}

// SendBillAsync sends an invoice asynchronously.
func (c *Client) SendBillAsync(fileName string, zipContent []byte) (*Response, error) {
	return c.SendBillAsyncContext(context.Background(), fileName, zipContent)
}

// SendBillAsyncContext sends an invoice asynchronously with context.
func (c *Client) SendBillAsyncContext(ctx context.Context, fileName string, zipContent []byte) (*Response, error) {
	body := soap.BodySendBillAsync(fileName, zipContent)
	return c.send(ctx, soap.SendBillAsync, body)
}

// SendTestSetAsync sends a test set for habilitacion.
func (c *Client) SendTestSetAsync(fileName string, zipContent []byte, testSetID string) (*Response, error) {
	return c.SendTestSetAsyncContext(context.Background(), fileName, zipContent, testSetID)
}

// SendTestSetAsyncContext sends a test set with context.
func (c *Client) SendTestSetAsyncContext(ctx context.Context, fileName string, zipContent []byte, testSetID string) (*Response, error) {
	body := soap.BodySendTestSetAsync(fileName, zipContent, testSetID)
	return c.send(ctx, soap.SendTestSetAsync, body)
}

// GetStatus checks the status of a submitted document.
func (c *Client) GetStatus(trackID string) (*Response, error) {
	return c.GetStatusContext(context.Background(), trackID)
}

// GetStatusContext checks status with context.
func (c *Client) GetStatusContext(ctx context.Context, trackID string) (*Response, error) {
	body := soap.BodyGetStatus(trackID)
	return c.send(ctx, soap.GetStatus, body)
}

// GetStatusZip gets the response as a ZIP file.
func (c *Client) GetStatusZip(trackID string) (*Response, error) {
	return c.GetStatusZipContext(context.Background(), trackID)
}

// GetStatusZipContext gets status ZIP with context.
func (c *Client) GetStatusZipContext(ctx context.Context, trackID string) (*Response, error) {
	body := soap.BodyGetStatusZip(trackID)
	return c.send(ctx, soap.GetStatusZip, body)
}

// GetNumberingRange queries authorized numbering ranges.
func (c *Client) GetNumberingRange(accountCode, accountCodeT, softwareCode string) (*Response, error) {
	return c.GetNumberingRangeContext(context.Background(), accountCode, accountCodeT, softwareCode)
}

// GetNumberingRangeContext queries numbering ranges with context.
func (c *Client) GetNumberingRangeContext(ctx context.Context, accountCode, accountCodeT, softwareCode string) (*Response, error) {
	body := soap.BodyGetNumberingRange(accountCode, accountCodeT, softwareCode)
	return c.send(ctx, soap.GetNumberingRange, body)
}

// SignInvoice signs a UBL invoice XML with XAdES-EPES.
// Returns the signed XML ready to be zipped and sent.
func (c *Client) SignInvoice(invoiceXML []byte) ([]byte, error) {
	return c.xadesSigner.Sign(invoiceXML)
}

// SignInvoiceWithOptions signs with custom options.
func (c *Client) SignInvoiceWithOptions(invoiceXML []byte, opts xades.Options) ([]byte, error) {
	return c.xadesSigner.WithOptions(opts).Sign(invoiceXML)
}

// CertificateInfo returns information about the loaded certificate.
func (c *Client) CertificateInfo() CertInfo {
	return CertInfo{
		Subject:      c.xadesCert.Subject(),
		Issuer:       c.xadesCert.Issuer(),
		SerialNumber: c.xadesCert.SerialNumber(),
		IsValid:      c.xadesCert.IsValid(),
		IsExpired:    c.xadesCert.IsExpired(),
	}
}

// CertInfo contains certificate information.
type CertInfo struct {
	Subject      string
	Issuer       string
	SerialNumber string
	IsValid      bool
	IsExpired    bool
}
