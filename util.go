package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/go-chef/chef"
)

func getStatusCode(err error) int {
	errFields := strings.Fields(err.Error())
	statusCode, _ := strconv.Atoi(errFields[len(errFields)-1])
	return statusCode
}

func getPrivateKey(privateKeyPath string) string {
	fileContent, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		fmt.Println("Couldn't read private key:", err)
	}
	privateKey := string(fileContent)
	return privateKey
}

func getPublicKey(privateKeyPath string) string {
	privateKey := getPrivateKey(privateKeyPath)
	rsaPrivateKey, _ := chef.PrivateKeyFromString([]byte(privateKey))
	rsaPublicKey := rsaPrivateKey.Public()
	PubASN1, _ := x509.MarshalPKIXPublicKey(rsaPublicKey)
	publicKey := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: PubASN1,
	}))
	return publicKey
}
