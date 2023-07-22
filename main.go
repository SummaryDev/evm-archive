package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	//_ "github.com/lib/pq"
	rpc "github.com/ybbus/jsonrpc/v3"
	"log"
	"os"
	"time"
)

func getArgs() (endpoint string, dataSourceName string, contracts []string, fromBlock uint64, toBlock uint64, blockStep uint64, sleepSeconds uint64) {

	endpoint = os.Getenv("EVM_ARCHIVE_ENDPOINT")
	if endpoint == "" {
		panic("please set env EVM_ARCHIVE_ENDPOINT")
	}

	schema := os.Getenv("EVM_ARCHIVE_SCHEMA")
	if schema == "" {
		panic("please set env EVM_ARCHIVE_SCHEMA")
	}

	dataSourceName = fmt.Sprintf("host=%v dbname=%v user=%v password=%v search_path=%v", os.Getenv("PGHOST"), os.Getenv("PGDATABASE"), os.Getenv("PGUSER"), os.Getenv("PGPASSWORD"), schema)
	log.Println(dataSourceName)

	contractString := os.Getenv("EVM_ARCHIVE_CONTRACTS")
	if contractString != "" {
		contracts = strings.Split(contractString, ",")
	}

	var err error

	fromBlockString := os.Getenv("EVM_ARCHIVE_FROM_BLOCK")
	if fromBlockString != "" {
		fromBlock, err = strconv.ParseUint(fromBlockString, 10, 64)
		if err != nil {
			panic(err)
		}
	}

	toBlockString := os.Getenv("EVM_ARCHIVE_TO_BLOCK")
	if toBlockString != "" {
		toBlock, err = strconv.ParseUint(toBlockString, 10, 64)
		if err != nil {
			panic(err)
		}
	}

	blockStepString := os.Getenv("EVM_ARCHIVE_BLOCK_STEP")
	if blockStepString != "" {
		blockStep, err = strconv.ParseUint(blockStepString, 10, 64)
		if err != nil {
			panic(err)
		}
	} else {
		blockStep = 1000
	}

	sleepSecondsString := os.Getenv("EVM_ARCHIVE_SLEEP_SECONDS")
	if sleepSecondsString != "" {
		sleepSeconds, err = strconv.ParseUint(sleepSecondsString, 10, 64)
		if err != nil {
			panic(err)
		}
	} else {
		sleepSeconds = 10
	}

	return
}

func query(endpoint string, dataSourceName string, query interface{}) {
	const method = "eth_getLogs"

	client := rpc.NewClient(endpoint)

	failed := true

	for failed {
		log.Printf("query %v with %v %v\n", endpoint, method, query)

		response, err := client.Call(context.Background(), method, query)

		//log.Printf("response %v", response)

		if err != nil {
			failed = true

			switch e := err.(type) {
			case *rpc.HTTPError:
				if e.Code == 429 {
					log.Printf("sleeping for 10s then retrying after Call failed with too many requests HTTPError=%v\n", err)
					time.Sleep(time.Second * 10)
				} else if e.Code == 503 || e.Code == 504 {
					log.Printf("sleeping for 5s then retrying after Call failed with server overloaded HTTPError=%v\n", err)
					time.Sleep(time.Second * 5)
				} else {
					log.Printf("retrying immediately after Call failed with HTTPError=%v\n", err)
				}
			default:
				log.Printf("retrying immediately after Call failed with err=%v\n", err)
			}

		} else if response == nil {
			failed = true
			log.Printf("retrying immediately after Call failed with response=%v\n", response)
		} else if response.Error != nil {
			failed = true
			log.Printf("retrying immediately after Call failed with %v\n", response.Error)
		} else {
			failed = false

			var getLogsResponse *GetLogsResponse

			err := response.GetObject(&getLogsResponse)
			if err != nil {
				log.Fatalf("cannot GetObject for GetLogsResponse %v\n", err)
			}

			getLogsResponse.Save(dataSourceName)
		}
	}
}

func main() {
	endpoint, dataSourceName, contracts, fromBlock, toBlock, blockStep, sleepSeconds := getArgs()

	sleep := time.Duration(sleepSeconds) * time.Second
	const maxUint = ^uint64(0)

	var q *GetLogsRequest

	if fromBlock > 0 {
		if toBlock < fromBlock {
			toBlock = maxUint
		}

		log.Printf("repeating query for logs in each block from %v to %v", fromBlock, toBlock)

		for blockNumber := fromBlock; blockNumber <= toBlock; blockNumber += blockStep + 1 {
			q = NewGetLogsRequest(contracts, blockNumber, blockNumber+blockStep)
			query(endpoint, dataSourceName, q)
		}
	} else {
		for {
			log.Printf("repeating query for logs in the latest block with sleep %v in between", sleep)

			q = NewGetLogsRequest(contracts, 0, 0)
			query(endpoint, dataSourceName, q)

			time.Sleep(sleep)
		}
	}
}
