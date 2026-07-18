package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&FinanceContract{})
	if err != nil {
		log.Fatalf("create finance chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Fatalf("start finance chaincode: %v", err)
	}
}
