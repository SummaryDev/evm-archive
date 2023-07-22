package main

import (
	"encoding/json"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"log"
)

/*
	{
	  "address": "0x985bca32293a7a496300a48081947321177a86fd",
	  "topics": [
		"0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9",
		"0x00000000000000000000000027e4ecf13634f28588e00059594df8465e3b32e9",
		"0x000000000000000000000000acc15dc74880c9944775448304b263d191c6077f"
	  ],
	  "data": "0x0000000000000000000000004293fb54fca9e1490ac4675e0a1a35f97c4ee06800000000000000000000000000000000000000000000000000000000000000a1",
	  "blockHash": "0x14c58621a13d5ede27e71fa5f338a485d76c1557402a4255953a9b293f21e26f",
	  "blockNumber": "0x3d2e29",
	  "transactionHash": "0x75a22452f56b0d69834374817fc222d072108d8bf1475e2562181e18dd5d9c30",
	  "transactionIndex": "0x5",
	  "logIndex": "0x2f",
	  "transactionLogIndex": "0x0",
	  "removed": false
	}
*/

type LogRpc struct {
	Address             string    `json:"address"`
	Topics              [4]string `json:"topics"`
	Data                string    `json:"data"`
	BlockHash           string    `json:"blockHash"`
	BlockNumber         string    `json:"blockNumber"`
	TransactionHash     string    `json:"transactionHash"`
	TransactionIndex    string    `json:"transactionIndex"`
	LogIndex            string    `json:"logIndex"`
	TransactionLogIndex string    `json:"transactionLogIndex"`
	Removed             bool      `json:"removed"`
}

/*
create table if not exists Log
(

	address             text,
	topic0              text,
	topic1              text,
	topic2              text,
	topic3              text,
	data                text,
	blockHash           text,
	blockNumber         text,
	transactionHash     text,
	transactionIndex    text,
	logIndex            text,
	transactionLogIndex text,
	removed             boolean,
	primary key (blockNumber, transactionHash, logIndex)

);
*/

type LogDb struct {
	Address             string
	Topic0              string
	Topic1              string
	Topic2              string
	Topic3              string
	Data                string
	BlockHash           string
	BlockNumber         string
	TransactionHash     string
	TransactionIndex    string
	LogIndex            string
	TransactionLogIndex string
	Removed             bool
}

func NewLogDb(r LogRpc) (d LogDb) {
	d.Address = r.Address
	d.Topic0 = r.Topics[0]
	d.Topic1 = r.Topics[1]
	d.Topic2 = r.Topics[2]
	d.Topic3 = r.Topics[3]
	d.Data = r.Data
	d.BlockHash = r.BlockHash
	d.BlockNumber = r.BlockNumber
	d.TransactionHash = r.TransactionHash
	d.TransactionIndex = r.TransactionIndex
	d.LogIndex = r.LogIndex
	d.TransactionLogIndex = r.TransactionLogIndex
	d.Removed = r.Removed

	return
}

type GetLogsRequest struct {
	Filter Filter `json:"filter"`
}

func (t *GetLogsRequest) ToJson() (s string) {
	b, _ := json.Marshal(t)
	s = string(b)
	return
}

type Filter struct {
	Address   []string `json:"address,omitempty"`
	FromBlock uint64   `json:"fromBlock,omitempty"`
	ToBlock   uint64   `json:"toBlock,omitempty"`
}

func NewGetLogsRequest(contracts []string, fromBlock uint64, toBlock uint64) *GetLogsRequest {
	q := &GetLogsRequest{}
	if len(contracts) > 0 {
		q.Filter.Address = contracts
	}
	if fromBlock > 0 {
		q.Filter.FromBlock = fromBlock
	}
	if toBlock >= fromBlock {
		q.Filter.ToBlock = toBlock
	}
	log.Println(q.ToJson())
	return q
}

type GetLogsResponse []LogRpc

func (t *GetLogsResponse) Save(dataSourceName string) (countSaved int64) {
	logs := *t

	if len(logs) == 0 {
		log.Println("no logs found")
		return
	}

	logsDb := make([]LogDb, 0)

	for _, r := range logs {
		d := NewLogDb(r)
		logsDb = append(logsDb, d)
	}

	db, err := sqlx.Open("pgx", dataSourceName) // postgres
	if err != nil {
		log.Fatal(err)
	}

	insertQuery := "insert into log (address, topic0, topic1, topic2, topic3, data, blockHash, blockNumber, transactionHash, transactionIndex, logIndex, transactionLogIndex, removed) " +
		"values (:address, :topic0, :topic1, :topic2, :topic3, :data, :blockhash, :blocknumber, :transactionhash, :transactionindex, :logindex, :transactionlogindex, :removed) " +
		"on conflict on constraint log_pkey do nothing" // todo on conflict on constraint Log_pkey update

	result, err := db.NamedExec(insertQuery, logsDb)
	if err != nil {
		log.Fatalf("%w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("inserted %v rows out of %v data", rows, len(logsDb))

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}

	countSaved = rows

	return
}
