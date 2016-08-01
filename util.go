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

func getApiClient(clientName, privateKeyPath, chefServerUrl string) chef.Client {
	privateKey := getPrivateKey(privateKeyPath)

	client, err := chef.NewClient(&chef.Config{
		Name:    clientName,
		Key:     privateKey,
		BaseURL: chefServerUrl,
		SkipSSL: true,
	})
	if err != nil {
		fmt.Println("Issue setting up client:", err)
	}
	return *client
}

func setupChefLoad(nodeName string, config chefLoadConfig, sem chan int) {
	sem <- 1
	adminClient := getApiClient(config.ClientName, config.ClientKey, config.ChefServerUrl)

	// TODO add error handling here
	fmt.Println(nodeName, "Bootstrapping")
	err := adminClient.Nodes.Delete(nodeName)
	if err != nil {
		fmt.Println(err)
	}
	err = adminClient.Clients.Delete(nodeName)
	if err != nil {
		fmt.Println(err)
	}
	createClient(adminClient, nodeName, getPublicKey(config.ClientKey))
	<-sem
	quit <- 1
}

func createClient(adminClient chef.Client, clientName, publicKey string) {
	apiClient := chef.ApiClient{
		Name:       clientName,
		ClientName: clientName,
		PublicKey:  publicKey,
		Admin:      false,
		Validator:  false,
	}
	data, err := chef.JSONReader(apiClient)
	if err != nil {
		return
	}
	req, err := adminClient.NewRequest("POST", "clients", data)
	res, err := adminClient.Do(req, nil)
	if err != nil {
		// can't print res here if it is nil
		// fmt.Println(res.StatusCode)
		// TODO: need to handle errors better
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
}

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
