/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var accountPrefix = "acct:"
var securityToken = "D44867B6ADB93F15D3DD77C323BF6"

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

// comment
type Account struct {
	id      string  `json:"id"`
	balance float64 `json:"balance"`
	loyalty int64   `json:"loyalty"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Initialize the collection of commercial paper keys
	fmt.Println("Initializing accountIds collection")
	var blank map[string]int
	blankBytes, _ := json.Marshal(&blank)
	err := stub.PutState("AccountIds", blankBytes)
	if err != nil {
		fmt.Println("Failed to initialize paper key collection")
	}

	fmt.Println("Initialization complete")
	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" { //initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "transfer" {
		return t.transfer(stub, args)
	} else if function == "registerAccounts" {
		return t.registerAccounts(stub, args)
	} else if function == "addLoyalty" {
		return t.addLoyalty(stub, args)
	} else if function == "removeLoyalty" {
		return t.removeLoyalty(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation: " + function)
}

func (t *SimpleChaincode) registerAccounts(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Creating accounts")

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting account numbers")
	}

	//var err error
	var ids []string
	err := json.Unmarshal([]byte(args[0]), &ids)
	if err != nil {
		fmt.Println("error creating accounts with input")
		return nil, errors.New("registerAccounts accepts an array of account ids")
	}

	registeredIds, err := GetAllAccountIds(stub)
	var newIds []string

	if registeredIds == nil {
		registeredIds = make(map[string]int)
	}

	//create a bunch of accounts
	for _, id := range ids {
		if registeredIds[id] != 1 {
			newIds = append(newIds, id)
		}
		registeredIds[id] = 1
	}

	registeredIdsBytes, err := json.Marshal(&registeredIds)
	if err != nil {
		fmt.Println("error marshaling accounts")
		return nil, errors.New("Error marshaling accounts")
	}

	err = stub.PutState("AccountIds", registeredIdsBytes)
	if err != nil {
		fmt.Println("error putting accounts")
		return nil, errors.New("Error putting accounts")
	}

	var account Account
	for _, newId := range newIds {
		account = Account{id: newId, balance: 10000.0, loyalty: 0}
		fmt.Println("New Account" + account.id)
		accountBytes, err := json.Marshal(&account)
		err = stub.PutState(accountPrefix+newId, accountBytes)
		if err != nil {
			fmt.Println("error putting account balance")
			return nil, errors.New("Error putting account balance")
		}
	}

	fmt.Println("Accounts created")
	return nil, nil

}

func (t *SimpleChaincode) transfer(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Handling a transfer")

	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting from, to, amount and token")
	}

	if args[3] != securityToken {
		fmt.Println("security token does not match")
		return nil, errors.New("security token does not match")
	}

	registeredIds, err := GetAllAccountIds(stub)
	if err != nil {
		fmt.Println("error gettng account ids")
		return nil, errors.New("Error gettng account ids")
	}

	var fromId = args[0]
	var toId = args[1]

	amount, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		fmt.Println("Amount is not a number")
		return nil, errors.New("Amount is not a number")
	}

	if registeredIds[fromId] != 1 {
		fmt.Println("error: from account is not registered")
		return nil, errors.New("Erro: from account is not registered")
	}

	if registeredIds[toId] != 1 {
		fmt.Println("error: to account is not registered")
		return nil, errors.New("Error: to account is not registered")
	}

	fromAccount, err := GetAccount(stub, fromId)
	if err != nil {
		fmt.Println("error gettng from account")
		return nil, errors.New("Error getting from account")
	}

	if fromAccount.balance < amount {
		fmt.Println("error not enough resources on from account")
		return nil, errors.New("error not enough resources on from account")
	}

	toAccount, err := GetAccount(stub, toId)
	if err != nil {
		fmt.Println("error gettng to Account")
		return nil, errors.New("Error getting to account")
	}

	fromAccount.balance = fromAccount.balance - amount
	toAccount.balance = toAccount.balance + amount

	fromAccountBytes, err := json.Marshal(&fromAccount)
	err = stub.PutState(accountPrefix+fromId, fromAccountBytes)
	if err != nil {
		fmt.Println("error putting from account")
		return nil, errors.New("Error putting from account")
	}

	toAccountBytes, err := json.Marshal(&toAccount)
	err = stub.PutState(accountPrefix+toId, toAccountBytes)
	if err != nil {
		fmt.Println("error putting to account")
		return nil, errors.New("Error putting to account")
	}

	return nil, nil
}

func (t *SimpleChaincode) addLoyalty(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Handling a adding loyalty points")

	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting account id, number of points and a token")
	}

	if args[2] != securityToken {
		fmt.Println("security token does not match")
		return nil, errors.New("security token does not match")
	}

	registeredIds, err := GetAllAccountIds(stub)
	if err != nil {
		fmt.Println("error gettng account ids")
		return nil, errors.New("Error gettng account ids")
	}

	var accountId = args[0]
	points, err := strconv.ParseInt(args[2], 10, 0)
	if err != nil {
		fmt.Println("Number of loyalty points is not a number")
		return nil, errors.New("Number of loyalty points  is not a number")
	}

	if registeredIds[accountId] != 1 {
		fmt.Println("error: account is not registered")
		return nil, errors.New("Erro: account is not registered")
	}

	account, err := GetAccount(stub, accountId)
	if err != nil {
		fmt.Println("error gettng account")
		return nil, errors.New("Error getting account")
	}

	account.loyalty = account.loyalty + points

	accountBytes, err := json.Marshal(&account)
	err = stub.PutState(accountPrefix+accountId, accountBytes)
	if err != nil {
		fmt.Println("error putting from account balance")
		return nil, errors.New("Error putting from account balance")
	}

	return nil, nil
}

func (t *SimpleChaincode) removeLoyalty(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Handling a adding loyalty points")

	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting account id, number of points and a token")
	}

	if args[2] != securityToken {
		fmt.Println("security token does not match")
		return nil, errors.New("security token does not match")
	}

	registeredIds, err := GetAllAccountIds(stub)
	if err != nil {
		fmt.Println("error gettng account ids")
		return nil, errors.New("Error gettng account ids")
	}

	var accountId = args[0]
	points, err := strconv.ParseInt(args[2], 10, 0)
	if err != nil {
		fmt.Println("Number of loyalty points is not a number")
		return nil, errors.New("Number of loyalty points  is not a number")
	}

	if registeredIds[accountId] != 1 {
		fmt.Println("error: account is not registered")
		return nil, errors.New("Erro: account is not registered")
	}

	account, err := GetAccount(stub, accountId)
	if err != nil {
		fmt.Println("error gettng account")
		return nil, errors.New("Error getting account")
	}

	if account.loyalty < points {
		fmt.Println("not enough loyalty points")
		return nil, errors.New("not enough loyalty points")
	}

	account.loyalty = account.loyalty - points

	accountBytes, err := json.Marshal(&account)
	err = stub.PutState(accountPrefix+accountId, accountBytes)
	if err != nil {
		fmt.Println("error putting from account balance")
		return nil, errors.New("Error putting from account balance")
	}

	return nil, nil
}

func GetAllAccountIds(stub shim.ChaincodeStubInterface) (map[string]int, error) {

	var accountIds map[string]int

	// Get list of all the keys
	idsBytes, err := stub.GetState("AccountIds")
	if err != nil {
		fmt.Println("Error retrieving account Ids")
		return nil, errors.New("Error retrieving account Ids")
	}

	err = json.Unmarshal(idsBytes, &accountIds)
	if err != nil {
		fmt.Println("Error unmarshalling account Ids")
		return nil, errors.New("Error unmarshalling account Ids")
	}

	return accountIds, nil
}

func GetAccount(stub shim.ChaincodeStubInterface, id string) (Account, error) {

	var account Account
	accountBytes, err := stub.GetState(accountPrefix + id)
	if err != nil {
		fmt.Println("Error retrieving account Ids")
		return account, errors.New("Error retrieving account Ids")
	}
	err = json.Unmarshal(accountBytes, &account)
	if err != nil {
		fmt.Println("Error unmarshalling account Ids")
		return account, errors.New("Error unmarshalling account Ids")
	}

	return account, nil
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "GetAllAccountIds" { //read a variable
		fmt.Println("Getting all accounts")
		registeredIds, err := GetAllAccountIds(stub)
		if err != nil {
			fmt.Println("error gettng account ids")
			return nil, err
		}

		registeredIdsBytes, err1 := json.Marshal(&registeredIds)
		if err1 != nil {
			fmt.Println("Error marshalling registeredIds")
			return nil, err1
		}

		fmt.Println("All success, returning accounts")
		return registeredIdsBytes, nil
	} else if function == "GetAccountDetails" { //read a variable
		fmt.Println("Getting account")

		account, err := GetAccount(stub, args[0])
		if err != nil {
			fmt.Println("error gettng account")
			return nil, err
		}

		accountBytes, err1 := json.Marshal(&account)
		if err1 != nil {
			fmt.Println("Error marshalling accountBytes")
			return nil, err1
		}

		fmt.Println("All success, returning account")
		return accountBytes, nil
	}

	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Received unknown function query: " + function)
}
