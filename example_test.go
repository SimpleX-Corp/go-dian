package dian_test

import (
	"fmt"

	_ "github.com/SimpleX-Corp/go-dian"
)

func Example() {
	// Create client for habilitacion environment
	// client, err := dian.NewClient("cert.p12", "password", dian.Habilitacion)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer client.Close()

	// // Check certificate info
	// info := client.CertificateInfo()
	// fmt.Printf("Certificate: %s (valid: %v)\n", info.Subject, info.IsValid)

	// // Sign invoice (XAdES-EPES)
	// signedXML, err := client.SignInvoice(invoiceXML)
	// if err != nil {
	//     log.Fatal(err)
	// }

	// // Create ZIP with signed invoice
	// zipContent := createZip(signedXML)

	// // Send to DIAN
	// response, err := client.SendBillSync("900123456-01-FV01-1.zip", zipContent)
	// if err != nil {
	//     log.Fatal(err)
	// }

	// // Check response
	// if response.IsSuccess() {
	//     fmt.Println("CUFE:", response.DocumentKey)
	// } else if response.IsPending() {
	//     fmt.Println("Processing, track ID needed")
	// } else {
	//     fmt.Println("Error:", response.StatusDescription)
	//     for _, msg := range response.ErrorMessages {
	//         fmt.Println(" -", msg)
	//     }
	// }

	fmt.Println("DIAN client example")
	// Output:
	// DIAN client example
}

func Example_getStatus() {
	// client, _ := dian.NewClient("cert.p12", "password", dian.Habilitacion)
	// defer client.Close()

	// // Check document status
	// response, err := client.GetStatus("track-id-from-previous-submission")
	// if err != nil {
	//     log.Fatal(err)
	// }

	// fmt.Printf("Status: %s - %s\n", response.StatusCode, response.StatusDescription)

	fmt.Println("GetStatus example")
	// Output:
	// GetStatus example
}

func Example_getNumberingRange() {
	// client, _ := dian.NewClient("cert.p12", "password", dian.Habilitacion)
	// defer client.Close()

	// // Query numbering ranges
	// response, err := client.GetNumberingRange(
	//     "900123456",           // NIT empresa
	//     "900123456",           // NIT proveedor tecnológico
	//     "software-id-hash",    // ID del software
	// )
	// if err != nil {
	//     log.Fatal(err)
	// }

	// // Response contains authorized ranges in XmlBase64Bytes
	// if response.IsSuccess() {
	//     decoded, _ := base64.StdEncoding.DecodeString(response.RawXML)
	//     fmt.Println(string(decoded))
	// }

	fmt.Println("GetNumberingRange example")
	// Output:
	// GetNumberingRange example
}

func Example_produccion() {
	// For production, just change the environment
	// client, _ := dian.NewClient("cert.p12", "password", dian.Produccion)

	fmt.Println("Production environment example")
	// Output:
	// Production environment example
}

func Example_withOptions() {
	// Configure timeouts and retries
	// client, _ := dian.NewClient("cert.p12", "password", dian.Habilitacion,
	//     dian.Options{
	//         Timeout:    60 * time.Second,
	//         MaxRetries: 3,
	//         RetryDelay: 2 * time.Second,
	//     },
	// )

	fmt.Println("Client with options example")
	// Output:
	// Client with options example
}
