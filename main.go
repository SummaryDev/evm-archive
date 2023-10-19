package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	//_ "github.com/lib/pq"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	rpc "github.com/ybbus/jsonrpc/v3"
)

func getArgs() (endpoint string, dataSourceName string, contracts []string, tokens []string, fromBlock uint64, toBlock uint64, blockStep uint64, sleepSeconds uint64) {

	endpoint = os.Getenv("EVM_ARCHIVE_ENDPOINT")
	if endpoint == "" {
		log.Println("EVM_ARCHIVE_ENDPOINT not set using default 'http://localhost:8545'")
		endpoint = "http://localhost:8545"
	}

	schema := os.Getenv("EVM_ARCHIVE_SCHEMA")
	if schema == "" {
		log.Println("EVM_ARCHIVE_SCHEMA not set using default 'public'")
		schema = "public"
	}

	dataSourceName = fmt.Sprintf("host=%v dbname=%v user=%v password=%v search_path=%v", os.Getenv("PGHOST"), os.Getenv("PGDATABASE"), os.Getenv("PGUSER"), os.Getenv("PGPASSWORD"), schema)
	log.Println(dataSourceName)

	contractString := os.Getenv("EVM_ARCHIVE_CONTRACTS")
	if contractString != "" {
		contracts = strings.Split(contractString, ",")
	}

	tokenString := os.Getenv("EVM_ARCHIVE_TOKENS")
	if tokenString != "" {
		tokens = strings.Split(tokenString, ",")
	}

	var err error

	fromBlockString := os.Getenv("EVM_ARCHIVE_FROM_BLOCK")
	if fromBlockString != "" {
		fromBlock, err = strconv.ParseUint(fromBlockString, 10, 64)
		if err != nil {
			log.Fatalf("cannot parse EVM_ARCHIVE_FROM_BLOCK for %v", err)
		}
	}

	toBlockString := os.Getenv("EVM_ARCHIVE_TO_BLOCK")
	if toBlockString != "" {
		toBlock, err = strconv.ParseUint(toBlockString, 10, 64)
		if err != nil {
			log.Fatalf("cannot parse EVM_ARCHIVE_TO_BLOCK for %v", err)
		}
	} else {
		toBlock = ^uint64(0) // infinity
	}

	blockStepString := os.Getenv("EVM_ARCHIVE_BLOCK_STEP")
	if blockStepString != "" {
		blockStep, err = strconv.ParseUint(blockStepString, 10, 64)
		if err != nil {
			log.Fatalf("cannot parse EVM_ARCHIVE_BLOCK_STEP for %v", err)
		}
	} else {
		blockStep = 100
	}

	sleepSecondsString := os.Getenv("EVM_ARCHIVE_SLEEP_SECONDS")
	if sleepSecondsString != "" {
		sleepSeconds, err = strconv.ParseUint(sleepSecondsString, 10, 64)
		if err != nil {
			log.Fatalf("cannot parse EVM_ARCHIVE_SLEEP_SECONDS for %v", err)
		}
	} else {
		sleepSeconds = 5
	}

	return
}

func call(req RpcRequest, res RpcResponse) {
	client := rpc.NewClient(req.endpoint)

	params := make([]interface{}, 0)

	if req.Query != nil {
		params = append(params, req.Query)
	}

	if req.AsOfBlock > 0 {
		params = append(params, ToHex(req.AsOfBlock))
	}

	// keep retrying to overcome recoverable comm errors
	retry := true

	// sleep for 10s between some retries
	sleep := time.Duration(10) * time.Second

	for retry {
		log.Printf("call %v", req)

		response, err := client.Call(context.Background(), req.method, params)

		// log.Printf("response %v", response)

		if err != nil {
			retry = true

			switch e := err.(type) {
			case *rpc.HTTPError:
				if e.Code == 429 {
					log.Printf("sleeping for %v then retrying after Call failed with too many requests HTTPError=%v", sleep, err)
					time.Sleep(sleep)
				} else if e.Code == 503 || e.Code == 504 {
					log.Printf("sleeping for %v then retrying after Call failed with server overloaded HTTPError=%v", sleep, err)
					time.Sleep(sleep)
				} else {
					log.Printf("retrying after Call failed with HTTPError=%v", err)
				}
			default:
				log.Printf("sleeping for %v then retrying after Call failed with err=%v\n", sleep, err)
				time.Sleep(sleep)
			}
		} else if response == nil {
			retry = true
			log.Println("retrying after Call failed with nil response")
		} else if response.Error != nil {
			if response.Error.Code == -32602 {
				retry = false
				log.Printf("not retrying after Call failed with response.Error %v", response.Error)
			} else {
				log.Fatalf("exiting after Call failed with unhandled response.Error %v", response.Error)
			}
		} else {
			retry = false

			err := response.GetObject(&res.persistent)
			if err != nil {
				log.Fatalf("cannot GetObject %v\n", err)
			}

			log.Printf("%v responded with %v records", req.method, res.persistent.Len())

			res.persistent.Save(res.dataSourceName, req)
		}
	}
}

func getDatabaseBlockNumber(dataSourceName string) (blockNumber uint64) {
	db, err := sqlx.Open("pgx", dataSourceName) // postgres
	if err != nil {
		log.Fatalf("sqlx.Open %v", err)
	}
	defer db.Close()

	err = db.QueryRow("select max(block_number) from logs").Scan(&blockNumber)
	if err != nil {
		blockNumber = 0
	}

	return /* uint64(18322923) */
}

func getNetworkBlockNumber(endpoint string) uint64 {
	getBlockNumberResponse := NewGetBlockNumberResponse()

	call(RpcRequest{0, nil, "eth_blockNumber", endpoint}, RpcResponse{getBlockNumberResponse, ""})

	return getBlockNumberResponse.ToNumber()
}

func getLogs(contracts []string, fromBlock uint64, toBlock uint64, endpoint string, dataSourceName string) {
	log.Printf("query for logs in blocks from %v to %v", fromBlock, toBlock)

	call(RpcRequest{0, NewGetLogsRequest(contracts, fromBlock, toBlock), "eth_getLogs", endpoint}, RpcResponse{NewGetLogsResponse(), dataSourceName})
}

func getPrices(tokens []string, fromBlock uint64, endpoint string, dataSourceName string) {
	for _, token := range tokens {
		log.Printf("query for price of %v as of block %v", token, fromBlock)

		call(RpcRequest{fromBlock, NewGetPriceRequest(token), "eth_call", endpoint}, RpcResponse{NewGetPriceResponse(), dataSourceName})
	}
}

func main() {
	endpoint, dataSourceName, contracts, tokens, fromBlockArg, toBlockArg, blockStep, sleepSeconds := getArgs()

	sleep := time.Duration(sleepSeconds) * time.Second

	// get the latest block number in the db
	databaseBlockNumber := getDatabaseBlockNumber(dataSourceName)

	var fromBlock, toBlock uint64

	// start querying from the block right after the one we have in the db or from the number specified in the arguments if it's higher than the db
	if fromBlockArg <= databaseBlockNumber {
		fromBlock = databaseBlockNumber + 1
	} else {
		fromBlock = fromBlockArg
	}

	for fromBlock <= toBlockArg {
		// get the latest block number in the network
		networkBlockNumber := getNetworkBlockNumber(endpoint)

		if fromBlock > networkBlockNumber {
			log.Printf("sleeping for %v as intended block to index %v is higher than network block %v", sleep, fromBlock, networkBlockNumber)

			time.Sleep(sleep)
			continue
		}

		toBlock = fromBlock + blockStep - 1

		// if the window specified by block step is beyond the latest block in the network then shrink the window
		if toBlock > networkBlockNumber {
			toBlock = networkBlockNumber
		}

		getLogs(contracts, fromBlock, toBlock, endpoint, dataSourceName)

		getPrices(tokens, fromBlock, endpoint, dataSourceName)

		// progress to the block right after the window specified by block step
		fromBlock = toBlock + 1
	}
}
