module github.com/SimpleX-Corp/go-dian

go 1.26.1

require (
	github.com/SimpleX-Corp/go-dian-client v0.0.0
	github.com/SimpleX-Corp/go-soap-builder v0.0.0
	github.com/SimpleX-Corp/go-wssecurity v0.0.0
	github.com/SimpleX-Corp/go-xades-dian v0.0.0
)

require (
	golang.org/x/crypto v0.25.0 // indirect
	software.sslmate.com/src/go-pkcs12 v0.7.0 // indirect
)

replace (
	github.com/SimpleX-Corp/go-dian-client => ../go-dian-client
	github.com/SimpleX-Corp/go-soap-builder => ../go-soap-builder
	github.com/SimpleX-Corp/go-wssecurity => ../go-wssecurity
	github.com/SimpleX-Corp/go-xades-dian => ../go-xades-dian
)
