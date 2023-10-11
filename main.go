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
	} else {
		toBlock = ^uint64(0) // infinity
	}

	blockStepString := os.Getenv("EVM_ARCHIVE_BLOCK_STEP")
	if blockStepString != "" {
		blockStep, err = strconv.ParseUint(blockStepString, 10, 64)
		if err != nil {
			panic(err)
		}
	} else {
		blockStep = 100
	}

	sleepSecondsString := os.Getenv("EVM_ARCHIVE_SLEEP_SECONDS")
	if sleepSecondsString != "" {
		sleepSeconds, err = strconv.ParseUint(sleepSecondsString, 10, 64)
		if err != nil {
			panic(err)
		}
	} else {
		sleepSeconds = 5
	}

	return
}

func query(endpoint string, dataSourceName string, query interface{}) {
	const method = "eth_getLogs"

	client := rpc.NewClient(endpoint)

	// keep retrying to overcome recoverable comm errors
	failed := true

	for failed {
		log.Printf("query %v with %v %v\n", endpoint, method, query)

		params := append(make([]interface{}, 0), query)

		response, err := client.Call(context.Background(), method, params)

		// log.Printf("response %v", response)

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
			log.Println("retrying immediately after Call failed with nil response")
		} else if response.Error != nil {
			panic(fmt.Sprintf("exiting immediately after Call failed with response.Error %v", response.Error))
		} else {
			failed = false

			var getLogsResponse *GetLogsResponse

			err := response.GetObject(&getLogsResponse)
			if err != nil {
				log.Fatalf("cannot GetObject for GetLogsResponse %v\n", err)
			}

			log.Printf("responded with %v records", getLogsResponse.Len())

			getLogsResponse.Save(dataSourceName)
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

	return
}

func main() {
	endpoint, dataSourceName, contracts, fromBlockArg, toBlockArg, blockStep, sleepSeconds := getArgs()

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

		query(endpoint, dataSourceName, NewGetLogsRequest(contracts, fromBlock, toBlock))

		time.Sleep(sleep)
	}
}
