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

func query(method string, endpoint string, dataSourceName string, req Request, res Persistent) {
	client := rpc.NewClient(endpoint)

	// keep retrying to overcome recoverable comm errors
	retry := true

	for retry {
		params := append(make([]interface{}, 0), req.Query)
		if req.AsOfBlock > 0 {
			params = append(params, ToHex(req.AsOfBlock))
		}

		log.Printf("call %v with %v %v as of %v\n", endpoint, method, req.Query, req.AsOfBlock)

		response, err := client.Call(context.Background(), method, params)

		// log.Printf("response %v", response)

		if err != nil {
			retry = true

			switch e := err.(type) {
			case *rpc.HTTPError:
				if e.Code == 429 {
					log.Printf("sleeping for 10s then retrying after Call failed with too many requests HTTPError=%v\n", err)
					time.Sleep(time.Second * 10)
				} else if e.Code == 503 || e.Code == 504 {
					log.Printf("sleeping for 5s then retrying after Call failed with server overloaded HTTPError=%v\n", err)
					time.Sleep(time.Second * 5)
				} else {
					log.Printf("retrying after Call failed with HTTPError=%v\n", err)
				}
			default:
				log.Printf("retrying after Call failed with err=%v\n", err)
			}
		} else if response == nil {
			retry = true
			log.Println("retrying after Call failed with nil response")
		} else if response.Error != nil {
			if response.Error.Code == -32602 {
				retry = false
				log.Printf("not retrying after Call failed with response.Error %v\n", response.Error)
			} else {
				log.Fatalf("exiting after Call failed with unhandled response.Error %v", response.Error)
			}
		} else {
			retry = false

			err := response.GetObject(&res)
			if err != nil {
				log.Fatalf("cannot GetObject %v\n", err)
			}

			log.Printf("%v responded with %v records", method, res.Len())

			res.Save(dataSourceName, req)
		}
	}
}

func getMaxBlockNumber(dataSourceName string) (blockNumber uint64) {
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

func main() {
	endpoint, dataSourceName, contracts, tokens, fromBlockArg, toBlockArg, blockStep, sleepSeconds := getArgs()

	sleep := time.Duration(sleepSeconds) * time.Second

	for blockNumber := fromBlockArg; blockNumber <= toBlockArg; {
		// get the latest block number from the db
		maxBlockNumber := getMaxBlockNumber(dataSourceName)

		if blockNumber <= maxBlockNumber {
			blockNumber = maxBlockNumber + 1
		}

		fromBlock := blockNumber
		toBlock := fromBlock + blockStep - 1

		log.Printf("query for logs in blocks from %v to %v then sleep for %v", fromBlock, toBlock, sleep)

		query("eth_getLogs", endpoint, dataSourceName, Request{0, NewGetLogsRequest(contracts, fromBlock, toBlock)}, NewGetLogsResponse())

		for _, token := range tokens {
			query("eth_call", endpoint, dataSourceName, Request{fromBlock, NewGetPriceRequest(token)}, NewGetPriceResponse())
		}

		time.Sleep(sleep)
	}
}
