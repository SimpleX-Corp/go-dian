package dian

import (
	"strings"
	"testing"
	"time"
)

// TestBuildInvoiceXML verifies the UBL XML builder produces valid XML.
func TestBuildInvoiceXML(t *testing.T) {
	now := time.Now()
	req := &InvoiceRequest{
		Type:   DocInvoice,
		Prefix: "SETP",
		Number: "990000001",
		IssueDate: &now,
		IssueTime: &now,
		Currency: "COP",
		Issuer: Party{
			NIT:  "900123456",
			DV:   "7",
			Name: "Test Company SAS",
			DocType: "31",
			TaxResponsibilities: []string{"O-47", "O-15"},
			Address: Address{
				Street:     "Calle 100 # 10-20",
				City:       "Bogotá",
				CityCode:   "11001",
				Department: "Bogotá D.C.",
				DeptCode:   "11",
				Country:    "Colombia",
				CountryCode: "CO",
			},
			Email: "test@company.com",
			Phone: "6011234567",
		},
		Customer: Party{
			NIT:  "800987654",
			DV:   "3",
			Name: "Customer SA",
			DocType: "31",
			Address: Address{
				Street:   "Carrera 50 # 20-30",
				City:     "Medellín",
				CityCode: "05001",
				Department: "Antioquia",
				DeptCode: "05",
			},
			Email: "customer@example.com",
		},
		Lines: []InvoiceLine{
			{
				Number:      1,
				Quantity:    2,
				UnitCode:    "EA",
				Description: "Servicio de consultoría técnica",
				UnitPrice:   500000,
				ProductCode: "80111600",
				Taxes: []Tax{
					{
						Type:        "01",
						Percent:     19,
						Amount:      190000,
						TaxableBase: 1000000,
					},
				},
			},
			{
				Number:      2,
				Quantity:    1,
				UnitCode:    "EA",
				Description: "Licencia de software",
				UnitPrice:   300000,
				Taxes: []Tax{
					{
						Type:        "01",
						Percent:     19,
						Amount:      57000,
						TaxableBase: 300000,
					},
				},
			},
		},
		Payment: Payment{
			Method: "1",
			Means:  "10",
		},
		Notes: []string{"Factura generada electrónicamente"},
		Resolution: &Resolution{
			Number:       "18760000001",
			Prefix:       "SETP",
			RangeFrom:    990000000,
			RangeTo:      995000000,
			ValidFrom:    now.AddDate(0, -6, 0),
			ValidTo:      now.AddDate(0, 6, 0),
			TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
		},
		SoftwareID:  "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		SoftwarePIN: "12345",
	}

	xml, err := buildInvoiceXML(req)
	if err != nil {
		t.Fatalf("buildInvoiceXML failed: %v", err)
	}

	xmlStr := string(xml)

	// Verify XML structure
	checks := []struct {
		name    string
		contains string
	}{
		{"XML declaration", `<?xml version="1.0" encoding="UTF-8"?>`},
		{"Invoice root", "<Invoice"},
		{"UBL namespace", `urn:oasis:names:specification:ubl:schema:xsd:Invoice-2`},
		{"UBL Extensions", "<ext:UBLExtensions>"},
		{"DIAN Extensions", "<sts:DianExtensions>"},
		{"Invoice ID", "<cbc:ID>SETP990000001</cbc:ID>"},
		{"UUID placeholder", `<cbc:UUID schemeID="2" schemeName="CUFE-SHA384"></cbc:UUID>`},
		{"Invoice Type", "<cbc:InvoiceTypeCode>01</cbc:InvoiceTypeCode>"},
		{"Currency", "<cbc:DocumentCurrencyCode>COP</cbc:DocumentCurrencyCode>"},
		{"Supplier NIT", ">900123456</"},
		{"Customer NIT", ">800987654</"},
		{"Line 1 description", "Servicio de consultoría técnica"},
		{"Line 2 description", "Licencia de software"},
		{"Tax amount", "190000.00"},
		{"Resolution", "18760000001"},
	}

	for _, check := range checks {
		if !strings.Contains(xmlStr, check.contains) {
			t.Errorf("%s: expected XML to contain %q", check.name, check.contains)
		}
	}

	// Verify line count
	if !strings.Contains(xmlStr, "<cbc:LineCountNumeric>2</cbc:LineCountNumeric>") {
		t.Error("Expected line count of 2")
	}

	t.Logf("Generated XML length: %d bytes", len(xml))
}

// TestCalculateCUFE verifies CUFE calculation.
func TestCalculateCUFE(t *testing.T) {
	issueDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.FixedZone("COT", -5*3600))

	req := &InvoiceRequest{
		Prefix: "SETP",
		Number: "990000001",
		IssueDate: &issueDate,
		IssueTime: &issueDate,
		Issuer: Party{
			NIT: "900123456",
		},
		Customer: Party{
			NIT: "800987654",
		},
		Lines: []InvoiceLine{
			{
				Quantity:  1,
				UnitPrice: 1000000,
				Taxes: []Tax{
					{Type: "01", Amount: 190000},
					{Type: "04", Amount: 0},
					{Type: "03", Amount: 0},
				},
			},
		},
		TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
	}

	cufe := CalculateCUFE(req, Habilitacion)

	// CUFE should be 96 hex characters (SHA-384 = 384 bits = 48 bytes = 96 hex)
	if len(cufe) != 96 {
		t.Errorf("CUFE length should be 96, got %d", len(cufe))
	}

	// Should be lowercase hex
	for _, c := range cufe {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("CUFE should be lowercase hex, found: %c", c)
		}
	}

	t.Logf("CUFE: %s", cufe)
}

// TestInjectCUFE verifies CUFE injection into XML.
func TestInjectCUFE(t *testing.T) {
	xml := []byte(`<Invoice><cbc:UUID schemeID="2" schemeName="CUFE-SHA384"></cbc:UUID></Invoice>`)
	cufe := "abc123def456"

	result := injectCUFE(xml, cufe)

	expected := `<Invoice><cbc:UUID schemeID="2" schemeName="CUFE-SHA384">abc123def456</cbc:UUID></Invoice>`
	if string(result) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(result))
	}
}

// TestCreateZip verifies ZIP creation.
func TestCreateZip(t *testing.T) {
	xmlContent := []byte(`<?xml version="1.0"?><Invoice><ID>TEST001</ID></Invoice>`)

	zipContent, err := CreateZip("900123456-SETP-990000001", xmlContent)
	if err != nil {
		t.Fatalf("CreateZip failed: %v", err)
	}

	// ZIP signature: PK\x03\x04
	if len(zipContent) < 4 || zipContent[0] != 'P' || zipContent[1] != 'K' {
		t.Error("Invalid ZIP signature")
	}

	t.Logf("ZIP size: %d bytes", len(zipContent))
}

// TestZipFileName verifies ZIP filename generation.
func TestZipFileName(t *testing.T) {
	tests := []struct {
		nit, prefix, number, expected string
	}{
		{"900123456", "SETP", "990000001", "900123456-SETP-990000001.zip"},
		{"800987654", "FV01", "1", "800987654-FV01-1.zip"},
	}

	for _, tt := range tests {
		result := ZipFileName(tt.nit, tt.prefix, tt.number)
		if result != tt.expected {
			t.Errorf("ZipFileName(%s, %s, %s) = %s, want %s",
				tt.nit, tt.prefix, tt.number, result, tt.expected)
		}
	}
}

// TestSoftwareSecurityCode verifies software security code calculation.
func TestSoftwareSecurityCode(t *testing.T) {
	code := CalculateSoftwareSecurityCode(
		"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		"12345",
		"900123456",
	)

	if len(code) != 96 {
		t.Errorf("Security code length should be 96, got %d", len(code))
	}

	t.Logf("Software Security Code: %s", code)
}

// TestEscapeXML verifies XML escaping.
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"Hello World", "Hello World"},
		{"<script>", "&lt;script&gt;"},
		{"A & B", "A &amp; B"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&apos;s"},
		{"<>&\"'", "&lt;&gt;&amp;&quot;&apos;"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestDocumentTypes verifies document type constants.
func TestDocumentTypes(t *testing.T) {
	tests := []struct {
		docType DocumentType
		code    string
	}{
		{DocInvoice, "01"},
		{DocExportInvoice, "02"},
		{DocContingency, "03"},
		{DocCreditNote, "91"},
		{DocDebitNote, "92"},
		{DocSupportDoc, "05"},
		{DocPayroll, "102"},
	}

	for _, tt := range tests {
		if string(tt.docType) != tt.code {
			t.Errorf("DocumentType %v should be %s", tt.docType, tt.code)
		}
	}
}

// TestFullFlow tests the complete flow without actually connecting to DIAN.
// This test builds XML, calculates CUFE, and prepares everything for signing.
func TestFullFlow(t *testing.T) {
	now := time.Now()

	req := InvoiceRequest{
		Type:      DocInvoice,
		Prefix:    "SETP",
		Number:    "990000001",
		IssueDate: &now,
		IssueTime: &now,
		Issuer: Party{
			NIT:  "900123456",
			DV:   "7",
			Name: "Emisor Test SAS",
		},
		Customer: Party{
			NIT:  "800987654",
			DV:   "3",
			Name: "Cliente Test SA",
		},
		Lines: []InvoiceLine{
			{
				Quantity:    1,
				UnitCode:    "EA",
				Description: "Producto de prueba",
				UnitPrice:   100000,
				Taxes: []Tax{
					{Type: "01", Percent: 19, Amount: 19000, TaxableBase: 100000},
				},
			},
		},
		Payment: Payment{Method: "1", Means: "10"},
		Resolution: &Resolution{
			Number:       "18760000001",
			Prefix:       "SETP",
			RangeFrom:    990000000,
			RangeTo:      995000000,
			ValidFrom:    now.AddDate(-1, 0, 0),
			ValidTo:      now.AddDate(1, 0, 0),
			TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
		},
	}

	// Step 1: Build XML
	xml, err := buildInvoiceXML(&req)
	if err != nil {
		t.Fatalf("Failed to build XML: %v", err)
	}
	t.Logf("Step 1 - XML built: %d bytes", len(xml))

	// Step 2: Calculate CUFE
	cufe := CalculateCUFE(&req, Habilitacion)
	if len(cufe) != 96 {
		t.Fatalf("Invalid CUFE length: %d", len(cufe))
	}
	t.Logf("Step 2 - CUFE calculated: %s...", cufe[:32])

	// Step 3: Inject CUFE
	xml = injectCUFE(xml, cufe)
	if !strings.Contains(string(xml), cufe) {
		t.Fatal("CUFE not injected into XML")
	}
	t.Log("Step 3 - CUFE injected")

	// Step 4: Create ZIP (without signing - would need certificate)
	zipContent, err := CreateZip("900123456-SETP-990000001", xml)
	if err != nil {
		t.Fatalf("Failed to create ZIP: %v", err)
	}
	t.Logf("Step 4 - ZIP created: %d bytes", len(zipContent))

	// Verify ZIP structure
	if zipContent[0] != 'P' || zipContent[1] != 'K' {
		t.Fatal("Invalid ZIP structure")
	}

	t.Log("Full flow completed successfully (without DIAN connection)")
}

// BenchmarkBuildInvoiceXML benchmarks XML building performance.
func BenchmarkBuildInvoiceXML(b *testing.B) {
	now := time.Now()
	req := &InvoiceRequest{
		Type:      DocInvoice,
		Prefix:    "SETP",
		Number:    "990000001",
		IssueDate: &now,
		Issuer:    Party{NIT: "900123456", DV: "7", Name: "Test"},
		Customer:  Party{NIT: "800987654", Name: "Customer"},
		Lines: []InvoiceLine{
			{Quantity: 1, UnitCode: "EA", Description: "Test", UnitPrice: 100000,
				Taxes: []Tax{{Type: "01", Percent: 19, Amount: 19000, TaxableBase: 100000}}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buildInvoiceXML(req)
	}
}

// BenchmarkCalculateCUFE benchmarks CUFE calculation performance.
func BenchmarkCalculateCUFE(b *testing.B) {
	now := time.Now()
	req := &InvoiceRequest{
		Prefix:    "SETP",
		Number:    "990000001",
		IssueDate: &now,
		Issuer:    Party{NIT: "900123456"},
		Customer:  Party{NIT: "800987654"},
		Lines: []InvoiceLine{
			{Quantity: 1, UnitPrice: 100000, Taxes: []Tax{{Type: "01", Amount: 19000}}},
		},
		TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateCUFE(req, Habilitacion)
	}
}

// BenchmarkCreateZip benchmarks ZIP creation performance.
func BenchmarkCreateZip(b *testing.B) {
	xml := make([]byte, 50000) // 50KB typical invoice
	for i := range xml {
		xml[i] = byte('A' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CreateZip("test-file", xml)
	}
}
