package main

import (
	"encoding/json"
	"log"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type LogRpc struct {
	Address          string    `json:"address"`
	Topics           [4]string `json:"topics"`
	Data             string    `json:"data"`
	BlockHash        string    `json:"blockHash"`
	BlockNumber      string    `json:"blockNumber"`
	TransactionHash  string    `json:"transactionHash"`
	TransactionIndex string    `json:"transactionIndex"`
	LogIndex         string    `json:"logIndex"`
	Removed          bool      `json:"removed"`
}

type LogDb struct {
	Address          string
	Topic0           string
	Topic1           string
	Topic2           string
	Topic3           string
	Data             *string
	BlockHash        string
	BlockNumber      uint64
	TransactionHash  string
	TransactionIndex uint64
	LogIndex         uint64
	Removed          bool
}

func NewLogDb(r LogRpc) (d LogDb) {
	d.Address = r.Address
	d.Topic0 = r.Topics[0]
	d.Topic1 = r.Topics[1]
	d.Topic2 = r.Topics[2]
	d.Topic3 = r.Topics[3]
	if r.Data != "0x" && r.Data != "" {
		d.Data = &r.Data
	}
	d.BlockHash = r.BlockHash
	d.BlockNumber = FromHex(r.BlockNumber)
	d.TransactionHash = r.TransactionHash
	d.TransactionIndex = FromHex(r.TransactionIndex)
	d.LogIndex = FromHex(r.LogIndex)
	d.Removed = r.Removed

	return
}

type GetLogsRequest struct {
	Address   []string `json:"address,omitempty"`
	FromBlock string   `json:"fromBlock,omitempty"`
	ToBlock   string   `json:"toBlock,omitempty"`
}

func (t *GetLogsRequest) ToJson() (s string) {
	b, _ := json.Marshal(t)
	s = string(b)
	return
}

func NewGetLogsRequest(contracts []string, fromBlock uint64, toBlock uint64) *GetLogsRequest {
	q := &GetLogsRequest{}
	if len(contracts) > 0 {
		q.Address = contracts
	}
	if fromBlock > 0 {
		q.FromBlock = ToHex(fromBlock)
	}
	if toBlock >= fromBlock {
		q.ToBlock = ToHex(toBlock)
	}
	log.Printf("from %v to %v with %v", fromBlock, toBlock, q.ToJson())
	return q
}

type GetLogsResponse []LogRpc

func NewGetLogsResponse() *GetLogsResponse {
	return &GetLogsResponse{}
}

func (t *GetLogsResponse) Len() int {
	logs := *t
	return len(logs)
}

func (t *GetLogsResponse) Save(dataSourceName string, req RpcRequest) (countSaved int64) {
	logs := *t

	if len(logs) == 0 {
		log.Println("no logs in the response")
		return
	}

	logsDb := make([]LogDb, 0)

	for _, r := range logs {
		d := NewLogDb(r)
		logsDb = append(logsDb, d) //todo don't append if removed then drop column removed
	}

	lastLog := logsDb[len(logsDb)-1]

	db, err := sqlx.Open("pgx", dataSourceName) // postgres
	if err != nil {
		log.Fatalf("sqlx.Open %v", err)
	}
	defer db.Close()

	insertQuery := "insert into logs (address, topic0, topic1, topic2, topic3, data, block_hash, block_number, transaction_hash, transaction_index, log_index, removed) " +
		"values (:address, :topic0, :topic1, :topic2, :topic3, :data, :blockhash, :blocknumber, :transactionhash, :transactionindex, :logindex, :removed) " +
		"on conflict on constraint logs_pkey do nothing" //todo on conflict on constraint logs_pkey update

	result, err := db.NamedExec(insertQuery, logsDb)
	if err != nil {
		log.Fatalf("db.NamedExec %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("result.RowsAffected %v", err)
	}

	log.Printf("inserted %v rows out of %v records with last block number %v", rows, len(logsDb), lastLog.BlockNumber)

	countSaved = rows

	return
}
