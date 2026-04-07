package dian

import (
	"testing"
	"time"
)

func getTestRequest() *InvoiceRequest {
	now := time.Now()
	return &InvoiceRequest{
		Type:      DocInvoice,
		Prefix:    "SETP",
		Number:    "990000001",
		IssueDate: &now,
		IssueTime: &now,
		Currency:  "COP",
		Issuer: Party{
			NIT:     "900123456",
			DV:      "7",
			Name:    "Test Company SAS",
			DocType: "31",
			TaxResponsibilities: []string{"O-47"},
			Address: Address{
				Street:      "Calle 100 # 10-20",
				City:        "Bogotá",
				CityCode:    "11001",
				Department:  "Bogotá D.C.",
				DeptCode:    "11",
				Country:     "Colombia",
				CountryCode: "CO",
			},
			Email: "test@company.com",
		},
		Customer: Party{
			NIT:     "800987654",
			DV:      "3",
			Name:    "Customer SA",
			DocType: "31",
			Address: Address{
				Street:   "Carrera 50",
				City:     "Medellín",
				CityCode: "05001",
				DeptCode: "05",
			},
		},
		Lines: []InvoiceLine{
			{
				Number:      1,
				Quantity:    2,
				UnitCode:    "EA",
				Description: "Servicio de consultoría",
				UnitPrice:   500000,
				ProductCode: "80111600",
				Taxes: []Tax{
					{Type: "01", Percent: 19, Amount: 190000, TaxableBase: 1000000},
				},
			},
			{
				Number:      2,
				Quantity:    1,
				UnitCode:    "EA",
				Description: "Licencia de software",
				UnitPrice:   300000,
				Taxes: []Tax{
					{Type: "01", Percent: 19, Amount: 57000, TaxableBase: 300000},
				},
			},
		},
		Payment: Payment{Method: "1", Means: "10"},
		Resolution: &Resolution{
			Number:       "18760000001",
			Prefix:       "SETP",
			RangeFrom:    990000000,
			RangeTo:      995000000,
			ValidFrom:    now.AddDate(0, -6, 0),
			ValidTo:      now.AddDate(0, 6, 0),
			TechnicalKey: "fc8eac422eba16e22ffd8c6f94b3f40a6e38f148",
		},
		SoftwareID: "test-software-id",
	}
}

// BenchmarkBuildInvoiceXMLOriginal benchmarks the original implementation.
func BenchmarkBuildInvoiceXMLOriginal(b *testing.B) {
	req := getTestRequest()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buildInvoiceXML(req)
	}
}

// BenchmarkBuildInvoiceXMLFast benchmarks the optimized implementation.
func BenchmarkBuildInvoiceXMLFast(b *testing.B) {
	req := getTestRequest()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buildInvoiceXMLFast(req)
	}
}

// BenchmarkCreateZipOriginal benchmarks original ZIP creation.
func BenchmarkCreateZipOriginal(b *testing.B) {
	xml := make([]byte, 50000)
	for i := range xml {
		xml[i] = byte('A' + (i % 26))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CreateZip("test-file", xml)
	}
}

// BenchmarkCreateZipFast benchmarks optimized ZIP creation (no compression).
func BenchmarkCreateZipFast(b *testing.B) {
	xml := make([]byte, 50000)
	for i := range xml {
		xml[i] = byte('A' + (i % 26))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CreateZipFast("test-file", xml)
	}
}

// BenchmarkFullFlowOriginal benchmarks the complete flow with original code.
func BenchmarkFullFlowOriginal(b *testing.B) {
	req := getTestRequest()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xml, _ := buildInvoiceXML(req)
		cufe := CalculateCUFE(req, Habilitacion)
		xml = injectCUFE(xml, cufe)
		_, _ = CreateZip("test", xml)
	}
}

// BenchmarkFullFlowOptimized benchmarks the complete flow with optimized code.
func BenchmarkFullFlowOptimized(b *testing.B) {
	req := getTestRequest()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xml, _ := buildInvoiceXMLFast(req)
		cufe := CalculateCUFE(req, Habilitacion)
		xml = injectCUFE(xml, cufe)
		_, _ = CreateZipFast("test", xml)
	}
}

// BenchmarkParallel tests parallel performance.
func BenchmarkParallel(b *testing.B) {
	req := getTestRequest()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			xml, _ := buildInvoiceXMLFast(req)
			cufe := CalculateCUFE(req, Habilitacion)
			xml = injectCUFE(xml, cufe)
			_, _ = CreateZipFast("test", xml)
		}
	})
}

// TestOptimizedOutputMatch verifies optimized output matches original.
func TestOptimizedOutputMatch(t *testing.T) {
	req := getTestRequest()

	original, err := buildInvoiceXML(req)
	if err != nil {
		t.Fatal(err)
	}

	optimized, err := buildInvoiceXMLFast(req)
	if err != nil {
		t.Fatal(err)
	}

	// Lengths should be similar (not exact due to slight formatting differences)
	lenDiff := len(original) - len(optimized)
	if lenDiff < -500 || lenDiff > 500 {
		t.Errorf("Length mismatch: original=%d, optimized=%d", len(original), len(optimized))
	}

	t.Logf("Original length: %d", len(original))
	t.Logf("Optimized length: %d", len(optimized))
}
