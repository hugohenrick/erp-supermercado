package pkcs12

import (
	"crypto/x509"
	"encoding/pem"

	"software.sslmate.com/src/go-pkcs12"
)

// ToPEM converte um certificado PKCS12 para blocos PEM
func ToPEM(pfxData []byte, password string) ([]*pem.Block, error) {
	// Decodificar o arquivo PKCS12
	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		return nil, err
	}

	// Criar slice para armazenar os blocos PEM
	var blocks []*pem.Block

	// Adicionar o certificado principal
	if certificate != nil {
		blocks = append(blocks, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificate.Raw,
		})
	}

	// Adicionar certificados da cadeia (CA)
	for _, cert := range caCerts {
		blocks = append(blocks, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
	}

	// Adicionar chave privada se disponível
	if privateKey != nil {
		pkData, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: pkData,
		})
	}

	return blocks, nil
}
