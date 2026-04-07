package dian

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// CUFE calculates the CUFE (Código Único de Facturación Electrónica) for an invoice.
// CUFE = SHA384(NumFac + FecFac + HorFac + ValFac + CodImp1 + ValImp1 + CodImp2 + ValImp2 + CodImp3 + ValImp3 + ValTot + NitOFE + NumAdq + ClTec + TipoAmbiente)
func CalculateCUFE(req *InvoiceRequest, env Environment) string {
	// Calculate totals
	var valFac float64  // Valor factura sin impuestos
	var valImp1 float64 // IVA
	var valImp2 float64 // INC
	var valImp3 float64 // ICA
	var valTot float64  // Total

	for _, line := range req.Lines {
		lineTotal := line.Quantity * line.UnitPrice
		valFac += lineTotal

		for _, tax := range line.Taxes {
			switch tax.Type {
			case "01": // IVA
				valImp1 += tax.Amount
			case "04": // INC
				valImp2 += tax.Amount
			case "03": // ICA
				valImp3 += tax.Amount
			}
		}
	}
	valTot = valFac + valImp1 + valImp2 + valImp3

	// Get dates
	now := time.Now()
	issueDate := now
	issueTime := now
	if req.IssueDate != nil {
		issueDate = *req.IssueDate
	}
	if req.IssueTime != nil {
		issueTime = *req.IssueTime
	}

	// Build CUFE string
	// NumFac: número factura (prefijo + número)
	numFac := req.Prefix + req.Number

	// FecFac: fecha factura YYYY-MM-DD
	fecFac := issueDate.Format("2006-01-02")

	// HorFac: hora factura HH:MM:SS-05:00
	horFac := issueTime.Format("15:04:05-07:00")

	// ValFac: valor sin impuestos (2 decimales, sin separador miles)
	valFacStr := formatAmount(valFac)

	// Impuestos
	codImp1 := "01" // IVA
	valImp1Str := formatAmount(valImp1)
	codImp2 := "04" // INC
	valImp2Str := formatAmount(valImp2)
	codImp3 := "03" // ICA
	valImp3Str := formatAmount(valImp3)

	// ValTot
	valTotStr := formatAmount(valTot)

	// NitOFE: NIT emisor
	nitOFE := req.Issuer.NIT

	// NumAdq: NIT/documento adquiriente
	numAdq := req.Customer.NIT

	// ClTec: Clave técnica (de la resolución)
	clTec := req.TechnicalKey
	if clTec == "" && req.Resolution != nil {
		clTec = req.Resolution.TechnicalKey
	}

	// TipoAmbiente: 1=Producción, 2=Habilitación
	tipoAmbiente := "2"
	if env == Produccion {
		tipoAmbiente = "1"
	}

	// Concatenar
	cufeString := numFac + fecFac + horFac + valFacStr + codImp1 + valImp1Str + codImp2 + valImp2Str + codImp3 + valImp3Str + valTotStr + nitOFE + numAdq + clTec + tipoAmbiente

	// SHA384
	hash := sha512.Sum384([]byte(cufeString))
	return strings.ToLower(hex.EncodeToString(hash[:]))
}

// CalculateCUDE calculates the CUDE for credit/debit notes.
// Similar to CUFE but uses different technical key.
func CalculateCUDE(req *InvoiceRequest, env Environment) string {
	// CUDE uses same algorithm as CUFE
	// The technical key for notes comes from resolution
	return CalculateCUFE(req, env)
}

// CalculateSoftwareSecurityCode calculates the software security code.
// SHA384(SoftwareID + PIN + NIT)
func CalculateSoftwareSecurityCode(softwareID, pin, nit string) string {
	data := softwareID + pin + nit
	hash := sha512.Sum384([]byte(data))
	return strings.ToLower(hex.EncodeToString(hash[:]))
}

func formatAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}
