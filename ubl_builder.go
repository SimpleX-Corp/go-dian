package dian

import (
	"bytes"
	"fmt"
	"time"

	codes "github.com/SimpleX-Corp/go-dian-codes"
)

// UBL 2.1 namespaces
const (
	nsInvoice = `xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2" xmlns:sts="dian:gov:co:facturaelectronica:Structures-2-1" xmlns:xades="http://uri.etsi.org/01903/v1.3.2#" xmlns:xades141="http://uri.etsi.org/01903/v1.4.1#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`
	nsCreditNote = `xmlns="urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2" xmlns:sts="dian:gov:co:facturaelectronica:Structures-2-1" xmlns:xades="http://uri.etsi.org/01903/v1.3.2#" xmlns:xades141="http://uri.etsi.org/01903/v1.4.1#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`
	nsDebitNote  = `xmlns="urn:oasis:names:specification:ubl:schema:xsd:DebitNote-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2" xmlns:sts="dian:gov:co:facturaelectronica:Structures-2-1" xmlns:xades="http://uri.etsi.org/01903/v1.3.2#" xmlns:xades141="http://uri.etsi.org/01903/v1.4.1#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`
)

// buildInvoiceXML builds UBL 2.1 Invoice XML from InvoiceRequest.
func buildInvoiceXML(req *InvoiceRequest) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(32768) // Pre-allocate 32KB

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
		currency = codes.CurrencyCOP
	}

	// Calculate totals
	var lineExtensionAmount, taxExclusiveAmount, taxInclusiveAmount, payableAmount float64
	var totalTaxAmount float64

	for _, line := range req.Lines {
		lineTotal := line.Quantity * line.UnitPrice
		lineExtensionAmount += lineTotal
		for _, tax := range line.Taxes {
			totalTaxAmount += tax.Amount
		}
	}
	taxExclusiveAmount = lineExtensionAmount
	taxInclusiveAmount = lineExtensionAmount + totalTaxAmount
	payableAmount = taxInclusiveAmount

	// Apply global discounts
	var totalDiscounts float64
	for _, d := range req.Discounts {
		totalDiscounts += d.Amount
	}
	payableAmount -= totalDiscounts

	// Determine document type and namespace
	rootElement := "Invoice"
	ns := nsInvoice
	invoiceTypeCode := string(req.Type)
	if invoiceTypeCode == "" {
		invoiceTypeCode = string(codes.DocInvoice) // "01"
	}

	// XML declaration
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteByte('\n')

	// Root element
	fmt.Fprintf(&buf, `<%s %s>`, rootElement, ns)
	buf.WriteByte('\n')

	// UBL Extensions (for signature placeholder)
	buf.WriteString(`<ext:UBLExtensions>`)
	buf.WriteString(`<ext:UBLExtension><ext:ExtensionContent></ext:ExtensionContent></ext:UBLExtension>`)

	// DIAN Extensions
	buf.WriteString(`<ext:UBLExtension><ext:ExtensionContent><sts:DianExtensions>`)
	writeInvoiceControl(&buf, req)
	writeInvoiceSource(&buf)
	writeSoftwareProvider(&buf, req)
	writeSoftwareInfo(&buf, req)
	buf.WriteString(`</sts:DianExtensions></ext:ExtensionContent></ext:UBLExtension>`)
	buf.WriteString(`</ext:UBLExtensions>`)
	buf.WriteByte('\n')

	// UBL Version
	buf.WriteString(`<cbc:UBLVersionID>UBL 2.1</cbc:UBLVersionID>`)
	buf.WriteByte('\n')
	buf.WriteString(`<cbc:CustomizationID>10</cbc:CustomizationID>`)
	buf.WriteByte('\n')
	buf.WriteString(`<cbc:ProfileID>DIAN 2.1</cbc:ProfileID>`)
	buf.WriteByte('\n')
	buf.WriteString(`<cbc:ProfileExecutionID>2</cbc:ProfileExecutionID>`)
	buf.WriteByte('\n')

	// Invoice ID
	fmt.Fprintf(&buf, `<cbc:ID>%s%s</cbc:ID>`, req.Prefix, req.Number)
	buf.WriteByte('\n')

	// UUID placeholder (CUFE will be calculated later)
	buf.WriteString(`<cbc:UUID schemeID="2" schemeName="CUFE-SHA384"></cbc:UUID>`)
	buf.WriteByte('\n')

	// Issue Date/Time
	fmt.Fprintf(&buf, `<cbc:IssueDate>%s</cbc:IssueDate>`, issueDate.Format("2006-01-02"))
	buf.WriteByte('\n')
	fmt.Fprintf(&buf, `<cbc:IssueTime>%s</cbc:IssueTime>`, issueTime.Format("15:04:05-07:00"))
	buf.WriteByte('\n')

	// Invoice Type Code
	fmt.Fprintf(&buf, `<cbc:InvoiceTypeCode>%s</cbc:InvoiceTypeCode>`, invoiceTypeCode)
	buf.WriteByte('\n')

	// Notes
	for _, note := range req.Notes {
		fmt.Fprintf(&buf, `<cbc:Note>%s</cbc:Note>`, escapeXML(note))
		buf.WriteByte('\n')
	}

	// Currency
	fmt.Fprintf(&buf, `<cbc:DocumentCurrencyCode>%s</cbc:DocumentCurrencyCode>`, currency)
	buf.WriteByte('\n')

	// Line count
	fmt.Fprintf(&buf, `<cbc:LineCountNumeric>%d</cbc:LineCountNumeric>`, len(req.Lines))
	buf.WriteByte('\n')

	// Order Reference (if any)
	if req.OrderReference != "" {
		fmt.Fprintf(&buf, `<cac:OrderReference><cbc:ID>%s</cbc:ID></cac:OrderReference>`, req.OrderReference)
		buf.WriteByte('\n')
	}

	// Invoice Reference (for credit/debit notes)
	if req.InvoiceReference != nil {
		writeDocumentReference(&buf, req.InvoiceReference)
	}

	// Supplier (Issuer)
	writeSupplierParty(&buf, &req.Issuer)

	// Customer
	writeCustomerParty(&buf, &req.Customer)

	// Payment Means
	writePaymentMeans(&buf, &req.Payment, issueDate)

	// Tax Totals
	writeTaxTotals(&buf, req.Lines, currency)

	// Monetary Totals
	writeLegalMonetaryTotal(&buf, lineExtensionAmount, taxExclusiveAmount, taxInclusiveAmount, payableAmount, currency)

	// Invoice Lines
	for i, line := range req.Lines {
		writeInvoiceLine(&buf, &line, i+1, currency)
	}

	// Close root
	fmt.Fprintf(&buf, `</%s>`, rootElement)

	return buf.Bytes(), nil
}

func writeInvoiceControl(buf *bytes.Buffer, req *InvoiceRequest) {
	buf.WriteString(`<sts:InvoiceControl>`)
	if req.Resolution != nil {
		fmt.Fprintf(buf, `<sts:InvoiceAuthorization>%s</sts:InvoiceAuthorization>`, req.Resolution.Number)
		fmt.Fprintf(buf, `<sts:AuthorizationPeriod><cbc:StartDate>%s</cbc:StartDate><cbc:EndDate>%s</cbc:EndDate></sts:AuthorizationPeriod>`,
			req.Resolution.ValidFrom.Format("2006-01-02"),
			req.Resolution.ValidTo.Format("2006-01-02"))
		fmt.Fprintf(buf, `<sts:AuthorizedInvoices><sts:Prefix>%s</sts:Prefix><sts:From>%d</sts:From><sts:To>%d</sts:To></sts:AuthorizedInvoices>`,
			req.Resolution.Prefix, req.Resolution.RangeFrom, req.Resolution.RangeTo)
	}
	buf.WriteString(`</sts:InvoiceControl>`)
}

func writeInvoiceSource(buf *bytes.Buffer) {
	buf.WriteString(`<sts:InvoiceSource><cbc:IdentificationCode listAgencyID="6" listAgencyName="United Nations Economic Commission for Europe" listSchemeURI="urn:oasis:names:specification:ubl:codelist:gc:CountryIdentificationCode-2.1">CO</cbc:IdentificationCode></sts:InvoiceSource>`)
}

func writeSoftwareProvider(buf *bytes.Buffer, req *InvoiceRequest) {
	nit := req.Issuer.NIT
	dv := req.Issuer.DV
	buf.WriteString(`<sts:SoftwareProvider>`)
	fmt.Fprintf(buf, `<sts:ProviderID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="%s" schemeName="31">%s</sts:ProviderID>`, dv, nit)
	fmt.Fprintf(buf, `<sts:SoftwareID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)">%s</sts:SoftwareID>`, req.SoftwareID)
	buf.WriteString(`</sts:SoftwareProvider>`)
}

func writeSoftwareInfo(buf *bytes.Buffer, req *InvoiceRequest) {
	buf.WriteString(`<sts:SoftwareSecurityCode schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)"></sts:SoftwareSecurityCode>`)
	buf.WriteString(`<sts:AuthorizationProvider><sts:AuthorizationProviderID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="4" schemeName="31">800197268</sts:AuthorizationProviderID></sts:AuthorizationProvider>`)
	buf.WriteString(`<sts:QRCode></sts:QRCode>`)
}

func writeSupplierParty(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:AccountingSupplierParty>`)
	buf.WriteString(`<cbc:AdditionalAccountID>1</cbc:AdditionalAccountID>`)
	buf.WriteString(`<cac:Party>`)

	// Party Name
	fmt.Fprintf(buf, `<cac:PartyName><cbc:Name>%s</cbc:Name></cac:PartyName>`, escapeXML(party.Name))

	// Address
	writeAddress(buf, &party.Address)

	// Tax Scheme
	writePartyTaxScheme(buf, party)

	// Legal Entity
	writePartyLegalEntity(buf, party)

	// Contact
	if party.Email != "" || party.Phone != "" {
		buf.WriteString(`<cac:Contact>`)
		if party.Phone != "" {
			fmt.Fprintf(buf, `<cbc:Telephone>%s</cbc:Telephone>`, party.Phone)
		}
		if party.Email != "" {
			fmt.Fprintf(buf, `<cbc:ElectronicMail>%s</cbc:ElectronicMail>`, party.Email)
		}
		buf.WriteString(`</cac:Contact>`)
	}

	buf.WriteString(`</cac:Party>`)
	buf.WriteString(`</cac:AccountingSupplierParty>`)
	buf.WriteByte('\n')
}

func writeCustomerParty(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:AccountingCustomerParty>`)
	buf.WriteString(`<cbc:AdditionalAccountID>2</cbc:AdditionalAccountID>`)
	buf.WriteString(`<cac:Party>`)

	// Party Name
	fmt.Fprintf(buf, `<cac:PartyName><cbc:Name>%s</cbc:Name></cac:PartyName>`, escapeXML(party.Name))

	// Address
	writeAddress(buf, &party.Address)

	// Tax Scheme
	writePartyTaxScheme(buf, party)

	// Legal Entity
	writePartyLegalEntity(buf, party)

	// Contact
	if party.Email != "" || party.Phone != "" {
		buf.WriteString(`<cac:Contact>`)
		if party.Phone != "" {
			fmt.Fprintf(buf, `<cbc:Telephone>%s</cbc:Telephone>`, party.Phone)
		}
		if party.Email != "" {
			fmt.Fprintf(buf, `<cbc:ElectronicMail>%s</cbc:ElectronicMail>`, party.Email)
		}
		buf.WriteString(`</cac:Contact>`)
	}

	buf.WriteString(`</cac:Party>`)
	buf.WriteString(`</cac:AccountingCustomerParty>`)
	buf.WriteByte('\n')
}

func writeAddress(buf *bytes.Buffer, addr *Address) {
	buf.WriteString(`<cac:PhysicalLocation><cac:Address>`)

	if addr.CityCode != "" {
		fmt.Fprintf(buf, `<cbc:ID>%s</cbc:ID>`, addr.CityCode)
	}
	if addr.City != "" {
		fmt.Fprintf(buf, `<cbc:CityName>%s</cbc:CityName>`, escapeXML(addr.City))
	}
	if addr.PostalCode != "" {
		fmt.Fprintf(buf, `<cbc:PostalZone>%s</cbc:PostalZone>`, addr.PostalCode)
	}
	if addr.DeptCode != "" {
		fmt.Fprintf(buf, `<cbc:CountrySubentity>%s</cbc:CountrySubentity>`, escapeXML(addr.Department))
		fmt.Fprintf(buf, `<cbc:CountrySubentityCode>%s</cbc:CountrySubentityCode>`, addr.DeptCode)
	}
	if addr.Street != "" {
		fmt.Fprintf(buf, `<cac:AddressLine><cbc:Line>%s</cbc:Line></cac:AddressLine>`, escapeXML(addr.Street))
	}

	countryCode := addr.CountryCode
	if countryCode == "" {
		countryCode = "CO" // Colombia default
	}
	countryName := addr.Country
	if countryName == "" {
		countryName = codes.CountryName("CO")
	}
	fmt.Fprintf(buf, `<cac:Country><cbc:IdentificationCode>%s</cbc:IdentificationCode><cbc:Name languageID="es">%s</cbc:Name></cac:Country>`,
		countryCode, countryName)

	buf.WriteString(`</cac:Address></cac:PhysicalLocation>`)
}

func writePartyTaxScheme(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:PartyTaxScheme>`)
	fmt.Fprintf(buf, `<cbc:RegistrationName>%s</cbc:RegistrationName>`, escapeXML(party.Name))

	docType := party.DocType
	if docType == "" {
		docType = string(codes.IDNIT) // "31" NIT
	}
	fmt.Fprintf(buf, `<cbc:CompanyID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="%s" schemeName="%s">%s</cbc:CompanyID>`,
		party.DV, docType, party.NIT)

	// Tax responsibilities
	if len(party.TaxResponsibilities) > 0 {
		for _, resp := range party.TaxResponsibilities {
			fmt.Fprintf(buf, `<cbc:TaxLevelCode listName="48">%s</cbc:TaxLevelCode>`, resp)
		}
	} else {
		buf.WriteString(`<cbc:TaxLevelCode listName="48">R-99-PN</cbc:TaxLevelCode>`)
	}

	fmt.Fprintf(buf, `<cac:TaxScheme><cbc:ID>%s</cbc:ID><cbc:Name>%s</cbc:Name></cac:TaxScheme>`,
		string(codes.TaxIVA), codes.TaxTypeName(string(codes.TaxIVA)))
	buf.WriteString(`</cac:PartyTaxScheme>`)
}

func writePartyLegalEntity(buf *bytes.Buffer, party *Party) {
	buf.WriteString(`<cac:PartyLegalEntity>`)
	fmt.Fprintf(buf, `<cbc:RegistrationName>%s</cbc:RegistrationName>`, escapeXML(party.Name))

	docType := party.DocType
	if docType == "" {
		docType = string(codes.IDNIT) // "31" NIT
	}
	fmt.Fprintf(buf, `<cbc:CompanyID schemeAgencyID="195" schemeAgencyName="CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)" schemeID="%s" schemeName="%s">%s</cbc:CompanyID>`,
		party.DV, docType, party.NIT)
	buf.WriteString(`</cac:PartyLegalEntity>`)
}

func writePaymentMeans(buf *bytes.Buffer, payment *Payment, issueDate time.Time) {
	method := payment.Method
	if method == "" {
		method = string(codes.PaymentCash) // "1" Contado
	}
	means := payment.Means
	if means == "" {
		means = string(codes.MeansCash) // "10" Efectivo
	}
	dueDate := issueDate
	if payment.DueDate != nil {
		dueDate = *payment.DueDate
	}

	buf.WriteString(`<cac:PaymentMeans>`)
	fmt.Fprintf(buf, `<cbc:ID>%s</cbc:ID>`, method)
	fmt.Fprintf(buf, `<cbc:PaymentMeansCode>%s</cbc:PaymentMeansCode>`, means)
	fmt.Fprintf(buf, `<cbc:PaymentDueDate>%s</cbc:PaymentDueDate>`, dueDate.Format("2006-01-02"))
	buf.WriteString(`</cac:PaymentMeans>`)
	buf.WriteByte('\n')
}

func writeTaxTotals(buf *bytes.Buffer, lines []InvoiceLine, currency string) {
	// Aggregate taxes by type
	taxByType := make(map[string]struct {
		amount      float64
		taxableBase float64
		percent     float64
	})

	for _, line := range lines {
		for _, tax := range line.Taxes {
			t := taxByType[tax.Type]
			t.amount += tax.Amount
			t.taxableBase += tax.TaxableBase
			t.percent = tax.Percent
			taxByType[tax.Type] = t
		}
	}

	// Write tax totals
	for taxType, t := range taxByType {
		buf.WriteString(`<cac:TaxTotal>`)
		fmt.Fprintf(buf, `<cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount>`, currency, t.amount)
		buf.WriteString(`<cac:TaxSubtotal>`)
		fmt.Fprintf(buf, `<cbc:TaxableAmount currencyID="%s">%.2f</cbc:TaxableAmount>`, currency, t.taxableBase)
		fmt.Fprintf(buf, `<cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount>`, currency, t.amount)
		buf.WriteString(`<cac:TaxCategory>`)
		fmt.Fprintf(buf, `<cbc:Percent>%.2f</cbc:Percent>`, t.percent)
		fmt.Fprintf(buf, `<cac:TaxScheme><cbc:ID>%s</cbc:ID><cbc:Name>%s</cbc:Name></cac:TaxScheme>`, taxType, getTaxName(taxType))
		buf.WriteString(`</cac:TaxCategory>`)
		buf.WriteString(`</cac:TaxSubtotal>`)
		buf.WriteString(`</cac:TaxTotal>`)
		buf.WriteByte('\n')
	}
}

func writeLegalMonetaryTotal(buf *bytes.Buffer, lineExtension, taxExclusive, taxInclusive, payable float64, currency string) {
	buf.WriteString(`<cac:LegalMonetaryTotal>`)
	fmt.Fprintf(buf, `<cbc:LineExtensionAmount currencyID="%s">%.2f</cbc:LineExtensionAmount>`, currency, lineExtension)
	fmt.Fprintf(buf, `<cbc:TaxExclusiveAmount currencyID="%s">%.2f</cbc:TaxExclusiveAmount>`, currency, taxExclusive)
	fmt.Fprintf(buf, `<cbc:TaxInclusiveAmount currencyID="%s">%.2f</cbc:TaxInclusiveAmount>`, currency, taxInclusive)
	fmt.Fprintf(buf, `<cbc:PayableAmount currencyID="%s">%.2f</cbc:PayableAmount>`, currency, payable)
	buf.WriteString(`</cac:LegalMonetaryTotal>`)
	buf.WriteByte('\n')
}

func writeInvoiceLine(buf *bytes.Buffer, line *InvoiceLine, number int, currency string) {
	lineTotal := line.Quantity * line.UnitPrice

	buf.WriteString(`<cac:InvoiceLine>`)
	fmt.Fprintf(buf, `<cbc:ID>%d</cbc:ID>`, number)
	fmt.Fprintf(buf, `<cbc:InvoicedQuantity unitCode="%s">%.2f</cbc:InvoicedQuantity>`, line.UnitCode, line.Quantity)
	fmt.Fprintf(buf, `<cbc:LineExtensionAmount currencyID="%s">%.2f</cbc:LineExtensionAmount>`, currency, lineTotal)

	// Line taxes
	for _, tax := range line.Taxes {
		buf.WriteString(`<cac:TaxTotal>`)
		fmt.Fprintf(buf, `<cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount>`, currency, tax.Amount)
		buf.WriteString(`<cac:TaxSubtotal>`)
		fmt.Fprintf(buf, `<cbc:TaxableAmount currencyID="%s">%.2f</cbc:TaxableAmount>`, currency, tax.TaxableBase)
		fmt.Fprintf(buf, `<cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount>`, currency, tax.Amount)
		buf.WriteString(`<cac:TaxCategory>`)
		fmt.Fprintf(buf, `<cbc:Percent>%.2f</cbc:Percent>`, tax.Percent)
		fmt.Fprintf(buf, `<cac:TaxScheme><cbc:ID>%s</cbc:ID><cbc:Name>%s</cbc:Name></cac:TaxScheme>`, tax.Type, getTaxName(tax.Type))
		buf.WriteString(`</cac:TaxCategory>`)
		buf.WriteString(`</cac:TaxSubtotal>`)
		buf.WriteString(`</cac:TaxTotal>`)
	}

	// Item
	buf.WriteString(`<cac:Item>`)
	fmt.Fprintf(buf, `<cbc:Description>%s</cbc:Description>`, escapeXML(line.Description))
	if line.ProductCode != "" {
		fmt.Fprintf(buf, `<cac:StandardItemIdentification><cbc:ID schemeID="999">%s</cbc:ID></cac:StandardItemIdentification>`, line.ProductCode)
	}
	buf.WriteString(`</cac:Item>`)

	// Price
	buf.WriteString(`<cac:Price>`)
	fmt.Fprintf(buf, `<cbc:PriceAmount currencyID="%s">%.2f</cbc:PriceAmount>`, currency, line.UnitPrice)
	fmt.Fprintf(buf, `<cbc:BaseQuantity unitCode="%s">1.00</cbc:BaseQuantity>`, line.UnitCode)
	buf.WriteString(`</cac:Price>`)

	buf.WriteString(`</cac:InvoiceLine>`)
	buf.WriteByte('\n')
}

func writeDocumentReference(buf *bytes.Buffer, ref *DocumentReference) {
	buf.WriteString(`<cac:BillingReference><cac:InvoiceDocumentReference>`)
	fmt.Fprintf(buf, `<cbc:ID>%s</cbc:ID>`, ref.Number)
	if ref.CUFE != "" {
		fmt.Fprintf(buf, `<cbc:UUID schemeName="CUFE-SHA384">%s</cbc:UUID>`, ref.CUFE)
	}
	fmt.Fprintf(buf, `<cbc:IssueDate>%s</cbc:IssueDate>`, ref.IssueDate.Format("2006-01-02"))
	buf.WriteString(`</cac:InvoiceDocumentReference></cac:BillingReference>`)
	buf.WriteByte('\n')
}

func getTaxName(taxType string) string {
	name := codes.TaxTypeName(taxType)
	if name == "" {
		return codes.TaxTypeName(string(codes.TaxIVA)) // Default to IVA
	}
	return name
}

func escapeXML(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
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
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
