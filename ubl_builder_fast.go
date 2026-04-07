package dian

import (
	"bytes"
	"strconv"
	"sync"
	"time"
)

// Buffer pool for XML building
var xmlBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 32768))
	},
}

// Pre-computed byte slices for common XML elements
var (
	xmlHeader      = []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	invoiceOpen    = []byte(`<Invoice ` + nsInvoice + `>` + "\n")
	invoiceClose   = []byte(`</Invoice>`)
	ublExtOpen     = []byte(`<ext:UBLExtensions>`)
	ublExtClose    = []byte(`</ext:UBLExtensions>` + "\n")
	extOpen        = []byte(`<ext:UBLExtension><ext:ExtensionContent>`)
	extClose       = []byte(`</ext:ExtensionContent></ext:UBLExtension>`)
	dianExtOpen    = []byte(`<sts:DianExtensions>`)
	dianExtClose   = []byte(`</sts:DianExtensions>`)
	newline        = []byte{'\n'}

	// Common elements
	cbcID          = []byte(`<cbc:ID>`)
	cbcIDClose     = []byte(`</cbc:ID>`)
	cbcName        = []byte(`<cbc:Name>`)
	cbcNameClose   = []byte(`</cbc:Name>`)
)

// buildInvoiceXMLFast builds UBL XML with optimized allocations.
func buildInvoiceXMLFast(req *InvoiceRequest) ([]byte, error) {
	buf := xmlBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer xmlBufferPool.Put(buf)

	// Pre-grow buffer
	buf.Grow(32768)

	// Set defaults
	now := time.Now()
	issueDate := now
	issueTime := now
	if req.IssueDate != nil {
		issueDate = *req.IssueDate
	}
	if req.IssueTime != nil {
		issueTime = *req.IssueTime
	}
	currency := req.Currency
	if currency == "" {
		currency = "COP"
	}

	invoiceTypeCode := string(req.Type)
	if invoiceTypeCode == "" {
		invoiceTypeCode = "01"
	}

	// XML header
	buf.Write(xmlHeader)
	buf.Write(invoiceOpen)

	// UBL Extensions
	buf.Write(ublExtOpen)
	buf.Write(extOpen)
	buf.Write(extClose)

	// DIAN Extensions
	buf.Write(extOpen)
	buf.Write(dianExtOpen)
	writeInvoiceControlFast(buf, req)
	writeInvoiceSourceFast(buf)
	writeSoftwareProviderFast(buf, req)
	writeSoftwareInfoFast(buf, req)
	buf.Write(dianExtClose)
	buf.Write(extClose)
	buf.Write(ublExtClose)

	// UBL Version
	buf.WriteString(`<cbc:UBLVersionID>UBL 2.1</cbc:UBLVersionID>` + "\n")
	buf.WriteString(`<cbc:CustomizationID>10</cbc:CustomizationID>` + "\n")
	buf.WriteString(`<cbc:ProfileID>DIAN 2.1</cbc:ProfileID>` + "\n")
	buf.WriteString(`<cbc:ProfileExecutionID>2</cbc:ProfileExecutionID>` + "\n")

	// Invoice ID - avoid fmt.Sprintf
	buf.Write(cbcID)
	buf.WriteString(req.Prefix)
	buf.WriteString(req.Number)
	buf.Write(cbcIDClose)
	buf.Write(newline)

	// UUID placeholder
	buf.WriteString(`<cbc:UUID schemeID="2" schemeName="CUFE-SHA384"></cbc:UUID>` + "\n")

	// Issue Date/Time - avoid time.Format allocation
	buf.WriteString(`<cbc:IssueDate>`)
	writeDate(buf, issueDate)
	buf.WriteString(`</cbc:IssueDate>` + "\n")
	buf.WriteString(`<cbc:IssueTime>`)
	writeTime(buf, issueTime)
	buf.WriteString(`</cbc:IssueTime>` + "\n")

	// Invoice Type Code
	buf.WriteString(`<cbc:InvoiceTypeCode>`)
	buf.WriteString(invoiceTypeCode)
	buf.WriteString(`</cbc:InvoiceTypeCode>` + "\n")

	// Notes
	for _, note := range req.Notes {
		buf.WriteString(`<cbc:Note>`)
		escapeXMLFast(buf, note)
		buf.WriteString(`</cbc:Note>` + "\n")
	}

	// Currency
	buf.WriteString(`<cbc:DocumentCurrencyCode>`)
	buf.WriteString(currency)
	buf.WriteString(`</cbc:DocumentCurrencyCode>` + "\n")

	// Line count
	buf.WriteString(`<cbc:LineCountNumeric>`)
	buf.WriteString(strconv.Itoa(len(req.Lines)))
	buf.WriteString(`</cbc:LineCountNumeric>` + "\n")

	// Order Reference
	if req.OrderReference != "" {
		buf.WriteString(`<cac:OrderReference><cbc:ID>`)
		buf.WriteString(req.OrderReference)
		buf.WriteString(`</cbc:ID></cac:OrderReference>` + "\n")
	}

	// Invoice Reference
	if req.InvoiceReference != nil {
		writeDocumentReferenceFast(buf, req.InvoiceReference)
	}

	// Parties
	writeSupplierPartyFast(buf, &req.Issuer)
	writeCustomerPartyFast(buf, &req.Customer)

	// Payment
	writePaymentMeansFast(buf, &req.Payment, issueDate)

	// Tax Totals
	writeTaxTotalsFast(buf, req.Lines, currency)

	// Monetary Totals
	lineExtension, taxExclusive, taxInclusive, payable := calculateTotals(req)
	writeLegalMonetaryTotalFast(buf, lineExtension, taxExclusive, taxInclusive, payable, currency)

	// Invoice Lines
	for i, line := range req.Lines {
		writeInvoiceLineFast(buf, &line, i+1, currency)
	}

	// Close
	buf.Write(invoiceClose)

	// Return copy
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// writeDate writes date in YYYY-MM-DD format without allocation
func writeDate(buf *bytes.Buffer, t time.Time) {
	year, month, day := t.Date()
	buf.WriteString(strconv.Itoa(year))
	buf.WriteByte('-')
	if month < 10 {
		buf.WriteByte('0')
	}
	buf.WriteString(strconv.Itoa(int(month)))
	buf.WriteByte('-')
	if day < 10 {
		buf.WriteByte('0')
	}
	buf.WriteString(strconv.Itoa(day))
}

// writeTime writes time in HH:MM:SS-05:00 format
func writeTime(buf *bytes.Buffer, t time.Time) {
	hour, min, sec := t.Clock()
	if hour < 10 {
		buf.WriteByte('0')
	}
	buf.WriteString(strconv.Itoa(hour))
	buf.WriteByte(':')
	if min < 10 {
		buf.WriteByte('0')
	}
	buf.WriteString(strconv.Itoa(min))
	buf.WriteByte(':')
	if sec < 10 {
		buf.WriteByte('0')
	}
	buf.WriteString(strconv.Itoa(sec))
	buf.WriteString("-05:00")
}

// writeAmount writes a float with 2 decimal places
func writeAmount(buf *bytes.Buffer, amount float64) {
	// Fast path for common amounts
	intPart := int64(amount)
	fracPart := int64((amount - float64(intPart)) * 100)
	if fracPart < 0 {
		fracPart = -fracPart
	}

	buf.WriteString(strconv.FormatInt(intPart, 10))
	buf.WriteByte('.')
	if fracPart < 10 {
		buf.WriteByte('0')
	}
	buf.WriteString(strconv.FormatInt(fracPart, 10))
}

// escapeXMLFast escapes XML characters directly into buffer
func escapeXMLFast(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteByte(s[i])
		}
	}
}

func calculateTotals(req *InvoiceRequest) (lineExt, taxExcl, taxIncl, payable float64) {
	var totalTax float64
	for _, line := range req.Lines {
		lineTotal := line.Quantity * line.UnitPrice
		lineExt += lineTotal
		for _, tax := range line.Taxes {
			totalTax += tax.Amount
		}
	}
	taxExcl = lineExt
	taxIncl = lineExt + totalTax
	payable = taxIncl
	for _, d := range req.Discounts {
		payable -= d.Amount
	}
	return
}

// Stub implementations - use same logic as original but with buffer writes
func writeInvoiceControlFast(buf *bytes.Buffer, req *InvoiceRequest) {
	buf.WriteString(`<sts:InvoiceControl>`)
	if req.Resolution != nil {
		buf.WriteString(`<sts:InvoiceAuthorization>`)
		buf.WriteString(req.Resolution.Number)
		buf.WriteString(`</sts:InvoiceAuthorization>`)
		buf.WriteString(`<sts:AuthorizationPeriod><cbc:StartDate>`)
		writeDate(buf, req.Resolution.ValidFrom)
		buf.WriteString(`</cbc:StartDate><cbc:EndDate>`)
		writeDate(buf, req.Resolution.ValidTo)
		buf.WriteString(`</cbc:EndDate></sts:AuthorizationPeriod>`)
		buf.WriteString(`<sts:AuthorizedInvoices><sts:Prefix>`)
		buf.WriteString(req.Resolution.Prefix)
		buf.WriteString(`</sts:Prefix><sts:From>`)
		buf.WriteString(strconv.FormatInt(req.Resolution.RangeFrom, 10))
		buf.WriteString(`</sts:From><sts:To>`)
		buf.WriteString(strconv.FormatInt(req.Resolution.RangeTo, 10))
		buf.WriteString(`</sts:To></sts:AuthorizedInvoices>`)
	}
	buf.WriteString(`</sts:InvoiceControl>`)
}

func writeInvoiceSourceFast(buf *bytes.Buffer) {
	buf.WriteString(`<sts:InvoiceSource><cbc:IdentificationCode listAgencyID="6" listAgencyName="United Nations Economic Commission for Europe" listSchemeURI="urn:oasis:names:specification:ubl:codelist:gc:CountryIdentificationCode-2.1">CO</cbc:IdentificationCode></sts:InvoiceSource>`)
}

func writeSoftwareProviderFast(buf *bytes.Buffer, req *InvoiceRequest) {
	buf.WriteString(`<sts:SoftwareProvider><sts:ProviderID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="`)
	buf.WriteString(req.Issuer.DV)
	buf.WriteString(`" schemeName="31">`)
	buf.WriteString(req.Issuer.NIT)
	buf.WriteString(`</sts:ProviderID><sts:SoftwareID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)">`)
	buf.WriteString(req.SoftwareID)
	buf.WriteString(`</sts:SoftwareID></sts:SoftwareProvider>`)
}

func writeSoftwareInfoFast(buf *bytes.Buffer, req *InvoiceRequest) {
	buf.WriteString(`<sts:SoftwareSecurityCode schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)"></sts:SoftwareSecurityCode>`)
	buf.WriteString(`<sts:AuthorizationProvider><sts:AuthorizationProviderID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="4" schemeName="31">800197268</sts:AuthorizationProviderID></sts:AuthorizationProvider>`)
	buf.WriteString(`<sts:QRCode></sts:QRCode>`)
}

func writeSupplierPartyFast(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:AccountingSupplierParty><cbc:AdditionalAccountID>1</cbc:AdditionalAccountID><cac:Party>`)
	buf.WriteString(`<cac:PartyName><cbc:Name>`)
	escapeXMLFast(buf, party.Name)
	buf.WriteString(`</cbc:Name></cac:PartyName>`)
	writeAddressFast(buf, &party.Address)
	writePartyTaxSchemeFast(buf, party)
	writePartyLegalEntityFast(buf, party)
	if party.Email != "" || party.Phone != "" {
		buf.WriteString(`<cac:Contact>`)
		if party.Phone != "" {
			buf.WriteString(`<cbc:Telephone>`)
			buf.WriteString(party.Phone)
			buf.WriteString(`</cbc:Telephone>`)
		}
		if party.Email != "" {
			buf.WriteString(`<cbc:ElectronicMail>`)
			buf.WriteString(party.Email)
			buf.WriteString(`</cbc:ElectronicMail>`)
		}
		buf.WriteString(`</cac:Contact>`)
	}
	buf.WriteString(`</cac:Party></cac:AccountingSupplierParty>` + "\n")
}

func writeCustomerPartyFast(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:AccountingCustomerParty><cbc:AdditionalAccountID>2</cbc:AdditionalAccountID><cac:Party>`)
	buf.WriteString(`<cac:PartyName><cbc:Name>`)
	escapeXMLFast(buf, party.Name)
	buf.WriteString(`</cbc:Name></cac:PartyName>`)
	writeAddressFast(buf, &party.Address)
	writePartyTaxSchemeFast(buf, party)
	writePartyLegalEntityFast(buf, party)
	if party.Email != "" || party.Phone != "" {
		buf.WriteString(`<cac:Contact>`)
		if party.Phone != "" {
			buf.WriteString(`<cbc:Telephone>`)
			buf.WriteString(party.Phone)
			buf.WriteString(`</cbc:Telephone>`)
		}
		if party.Email != "" {
			buf.WriteString(`<cbc:ElectronicMail>`)
			buf.WriteString(party.Email)
			buf.WriteString(`</cbc:ElectronicMail>`)
		}
		buf.WriteString(`</cac:Contact>`)
	}
	buf.WriteString(`</cac:Party></cac:AccountingCustomerParty>` + "\n")
}

func writeAddressFast(buf *bytes.Buffer, addr *Address) {
	buf.WriteString(`<cac:PhysicalLocation><cac:Address>`)
	if addr.CityCode != "" {
		buf.WriteString(`<cbc:ID>`)
		buf.WriteString(addr.CityCode)
		buf.WriteString(`</cbc:ID>`)
	}
	if addr.City != "" {
		buf.WriteString(`<cbc:CityName>`)
		escapeXMLFast(buf, addr.City)
		buf.WriteString(`</cbc:CityName>`)
	}
	if addr.PostalCode != "" {
		buf.WriteString(`<cbc:PostalZone>`)
		buf.WriteString(addr.PostalCode)
		buf.WriteString(`</cbc:PostalZone>`)
	}
	if addr.DeptCode != "" {
		buf.WriteString(`<cbc:CountrySubentity>`)
		escapeXMLFast(buf, addr.Department)
		buf.WriteString(`</cbc:CountrySubentity>`)
		buf.WriteString(`<cbc:CountrySubentityCode>`)
		buf.WriteString(addr.DeptCode)
		buf.WriteString(`</cbc:CountrySubentityCode>`)
	}
	if addr.Street != "" {
		buf.WriteString(`<cac:AddressLine><cbc:Line>`)
		escapeXMLFast(buf, addr.Street)
		buf.WriteString(`</cbc:Line></cac:AddressLine>`)
	}
	countryCode := addr.CountryCode
	if countryCode == "" {
		countryCode = "CO"
	}
	countryName := addr.Country
	if countryName == "" {
		countryName = "Colombia"
	}
	buf.WriteString(`<cac:Country><cbc:IdentificationCode>`)
	buf.WriteString(countryCode)
	buf.WriteString(`</cbc:IdentificationCode><cbc:Name languageID="es">`)
	buf.WriteString(countryName)
	buf.WriteString(`</cbc:Name></cac:Country>`)
	buf.WriteString(`</cac:Address></cac:PhysicalLocation>`)
}

func writePartyTaxSchemeFast(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:PartyTaxScheme><cbc:RegistrationName>`)
	escapeXMLFast(buf, party.Name)
	buf.WriteString(`</cbc:RegistrationName>`)
	docType := party.DocType
	if docType == "" {
		docType = "31"
	}
	buf.WriteString(`<cbc:CompanyID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="`)
	buf.WriteString(party.DV)
	buf.WriteString(`" schemeName="`)
	buf.WriteString(docType)
	buf.WriteString(`">`)
	buf.WriteString(party.NIT)
	buf.WriteString(`</cbc:CompanyID>`)
	if len(party.TaxResponsibilities) > 0 {
		for _, resp := range party.TaxResponsibilities {
			buf.WriteString(`<cbc:TaxLevelCode listName="48">`)
			buf.WriteString(resp)
			buf.WriteString(`</cbc:TaxLevelCode>`)
		}
	} else {
		buf.WriteString(`<cbc:TaxLevelCode listName="48">R-99-PN</cbc:TaxLevelCode>`)
	}
	buf.WriteString(`<cac:TaxScheme><cbc:ID>01</cbc:ID><cbc:Name>IVA</cbc:Name></cac:TaxScheme></cac:PartyTaxScheme>`)
}

func writePartyLegalEntityFast(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:PartyLegalEntity><cbc:RegistrationName>`)
	escapeXMLFast(buf, party.Name)
	buf.WriteString(`</cbc:RegistrationName>`)
	docType := party.DocType
	if docType == "" {
		docType = "31"
	}
	buf.WriteString(`<cbc:CompanyID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="`)
	buf.WriteString(party.DV)
	buf.WriteString(`" schemeName="`)
	buf.WriteString(docType)
	buf.WriteString(`">`)
	buf.WriteString(party.NIT)
	buf.WriteString(`</cbc:CompanyID></cac:PartyLegalEntity>`)
}

func writePaymentMeansFast(buf *bytes.Buffer, payment *Payment, issueDate time.Time) {
	method := payment.Method
	if method == "" {
		method = "1"
	}
	means := payment.Means
	if means == "" {
		means = "10"
	}
	dueDate := issueDate
	if payment.DueDate != nil {
		dueDate = *payment.DueDate
	}
	buf.WriteString(`<cac:PaymentMeans><cbc:ID>`)
	buf.WriteString(method)
	buf.WriteString(`</cbc:ID><cbc:PaymentMeansCode>`)
	buf.WriteString(means)
	buf.WriteString(`</cbc:PaymentMeansCode><cbc:PaymentDueDate>`)
	writeDate(buf, dueDate)
	buf.WriteString(`</cbc:PaymentDueDate></cac:PaymentMeans>` + "\n")
}

func writeTaxTotalsFast(buf *bytes.Buffer, lines []InvoiceLine, currency string) {
	taxByType := make(map[string]struct{ amount, taxableBase, percent float64 })
	for _, line := range lines {
		for _, tax := range line.Taxes {
			t := taxByType[tax.Type]
			t.amount += tax.Amount
			t.taxableBase += tax.TaxableBase
			t.percent = tax.Percent
			taxByType[tax.Type] = t
		}
	}
	for taxType, t := range taxByType {
		buf.WriteString(`<cac:TaxTotal><cbc:TaxAmount currencyID="`)
		buf.WriteString(currency)
		buf.WriteString(`">`)
		writeAmount(buf, t.amount)
		buf.WriteString(`</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount currencyID="`)
		buf.WriteString(currency)
		buf.WriteString(`">`)
		writeAmount(buf, t.taxableBase)
		buf.WriteString(`</cbc:TaxableAmount><cbc:TaxAmount currencyID="`)
		buf.WriteString(currency)
		buf.WriteString(`">`)
		writeAmount(buf, t.amount)
		buf.WriteString(`</cbc:TaxAmount><cac:TaxCategory><cbc:Percent>`)
		writeAmount(buf, t.percent)
		buf.WriteString(`</cbc:Percent><cac:TaxScheme><cbc:ID>`)
		buf.WriteString(taxType)
		buf.WriteString(`</cbc:ID><cbc:Name>`)
		buf.WriteString(getTaxName(taxType))
		buf.WriteString(`</cbc:Name></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>` + "\n")
	}
}

func writeLegalMonetaryTotalFast(buf *bytes.Buffer, lineExt, taxExcl, taxIncl, payable float64, currency string) {
	buf.WriteString(`<cac:LegalMonetaryTotal><cbc:LineExtensionAmount currencyID="`)
	buf.WriteString(currency)
	buf.WriteString(`">`)
	writeAmount(buf, lineExt)
	buf.WriteString(`</cbc:LineExtensionAmount><cbc:TaxExclusiveAmount currencyID="`)
	buf.WriteString(currency)
	buf.WriteString(`">`)
	writeAmount(buf, taxExcl)
	buf.WriteString(`</cbc:TaxExclusiveAmount><cbc:TaxInclusiveAmount currencyID="`)
	buf.WriteString(currency)
	buf.WriteString(`">`)
	writeAmount(buf, taxIncl)
	buf.WriteString(`</cbc:TaxInclusiveAmount><cbc:PayableAmount currencyID="`)
	buf.WriteString(currency)
	buf.WriteString(`">`)
	writeAmount(buf, payable)
	buf.WriteString(`</cbc:PayableAmount></cac:LegalMonetaryTotal>` + "\n")
}

func writeInvoiceLineFast(buf *bytes.Buffer, line *InvoiceLine, number int, currency string) {
	lineTotal := line.Quantity * line.UnitPrice
	buf.WriteString(`<cac:InvoiceLine><cbc:ID>`)
	buf.WriteString(strconv.Itoa(number))
	buf.WriteString(`</cbc:ID><cbc:InvoicedQuantity unitCode="`)
	buf.WriteString(line.UnitCode)
	buf.WriteString(`">`)
	writeAmount(buf, line.Quantity)
	buf.WriteString(`</cbc:InvoicedQuantity><cbc:LineExtensionAmount currencyID="`)
	buf.WriteString(currency)
	buf.WriteString(`">`)
	writeAmount(buf, lineTotal)
	buf.WriteString(`</cbc:LineExtensionAmount>`)
	for _, tax := range line.Taxes {
		buf.WriteString(`<cac:TaxTotal><cbc:TaxAmount currencyID="`)
		buf.WriteString(currency)
		buf.WriteString(`">`)
		writeAmount(buf, tax.Amount)
		buf.WriteString(`</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount currencyID="`)
		buf.WriteString(currency)
		buf.WriteString(`">`)
		writeAmount(buf, tax.TaxableBase)
		buf.WriteString(`</cbc:TaxableAmount><cbc:TaxAmount currencyID="`)
		buf.WriteString(currency)
		buf.WriteString(`">`)
		writeAmount(buf, tax.Amount)
		buf.WriteString(`</cbc:TaxAmount><cac:TaxCategory><cbc:Percent>`)
		writeAmount(buf, tax.Percent)
		buf.WriteString(`</cbc:Percent><cac:TaxScheme><cbc:ID>`)
		buf.WriteString(tax.Type)
		buf.WriteString(`</cbc:ID><cbc:Name>`)
		buf.WriteString(getTaxName(tax.Type))
		buf.WriteString(`</cbc:Name></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`)
	}
	buf.WriteString(`<cac:Item><cbc:Description>`)
	escapeXMLFast(buf, line.Description)
	buf.WriteString(`</cbc:Description>`)
	if line.ProductCode != "" {
		buf.WriteString(`<cac:StandardItemIdentification><cbc:ID schemeID="999">`)
		buf.WriteString(line.ProductCode)
		buf.WriteString(`</cbc:ID></cac:StandardItemIdentification>`)
	}
	buf.WriteString(`</cac:Item><cac:Price><cbc:PriceAmount currencyID="`)
	buf.WriteString(currency)
	buf.WriteString(`">`)
	writeAmount(buf, line.UnitPrice)
	buf.WriteString(`</cbc:PriceAmount><cbc:BaseQuantity unitCode="`)
	buf.WriteString(line.UnitCode)
	buf.WriteString(`">1.00</cbc:BaseQuantity></cac:Price></cac:InvoiceLine>` + "\n")
}

func writeDocumentReferenceFast(buf *bytes.Buffer, ref *DocumentReference) {
	buf.WriteString(`<cac:BillingReference><cac:InvoiceDocumentReference><cbc:ID>`)
	buf.WriteString(ref.Number)
	buf.WriteString(`</cbc:ID>`)
	if ref.CUFE != "" {
		buf.WriteString(`<cbc:UUID schemeName="CUFE-SHA384">`)
		buf.WriteString(ref.CUFE)
		buf.WriteString(`</cbc:UUID>`)
	}
	buf.WriteString(`<cbc:IssueDate>`)
	writeDate(buf, ref.IssueDate)
	buf.WriteString(`</cbc:IssueDate></cac:InvoiceDocumentReference></cac:BillingReference>` + "\n")
}
