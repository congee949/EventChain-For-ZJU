package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&ActivityContract{})
	if err != nil {
		log.Fatalf("create activity chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Fatalf("start activity chaincode: %v", err)
	}
}
