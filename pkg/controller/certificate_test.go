package controller

import (
	"testing"
	"k8s.io/client-go/util/cert"

	"crypto/x509"
	"fmt"
)

func TestNew(t *testing.T) {

	caKey, err := cert.NewPrivateKey()
	if err != nil {
		return
	}
	fmt.Println(string(x509.MarshalPKCS1PrivateKey(caKey)))
}