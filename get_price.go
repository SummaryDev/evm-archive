package main

import (
	"encoding/json"
	"log"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type PriceRpc struct {
	Address     string `json:"address"`
	BlockNumber string `json:"blockNumber"`
	Price       string `json:"priceIndex"`
}

type PriceDb struct {
	Address     string
	BlockNumber uint64
	Price       uint64
}

func NewPriceDb(r PriceRpc) (d PriceDb) {
	d.Address = r.Address
	d.BlockNumber = FromHex(r.BlockNumber)
	d.Price = FromHex(r.Price)

	return
}

type GetPriceRequest struct {
	To    string `json:"to,omitempty"`
	Data  string `json:"data,omitempty"`
	Token string
}

func (t *GetPriceRequest) ToJson() (s string) {
	b, _ := json.Marshal(t)
	s = string(b)
	return
}

func NewGetPriceRequest(token string, oracle string) *GetPriceRequest {
	q := &GetPriceRequest{}
	q.To = oracle
	q.Data = "0x50d25bcd" // latestAnswer encoded as function selector
	q.Token = token
	log.Printf("for token %v from oracle %v with %v", token, oracle, q.ToJson())
	return q
}

type GetPriceResponse string

func NewGetPriceResponse() *GetPriceResponse {
	var newGetPriceResponse GetPriceResponse = ""
	return &newGetPriceResponse
}

func (t *GetPriceResponse) Len() int {
	price := *t
	if len(price) == 66 {
		return 1
	} else {
		return 0
	}
}

func (t *GetPriceResponse) ToNumber() uint64 {
	s := string(*t)
	return FromHex(s)
}

func (t *GetPriceResponse) Save(dataSourceName string, req RpcRequest) (countSaved int64) {
	if t.Len() == 0 {
		log.Println("no price in the response")
		return
	}

	getPriceRequest, ok := req.Query.(*GetPriceRequest)
	if !ok {
		log.Fatal("cannot cast to GetPriceRequest")
	}

	price := t.ToNumber()
	blockNumber := req.AsOfBlock
	token := getPriceRequest.Token

	priceDb := PriceDb{Address: token, BlockNumber: blockNumber, Price: price}

	db, err := sqlx.Open("pgx", dataSourceName) // postgres
	if err != nil {
		log.Fatalf("sqlx.Open %v", err)
	}
	defer db.Close()

	insertQuery := "insert into price (address, block_number, price) values (:address, :blocknumber, :price) on conflict on constraint price_pkey do nothing" //todo on conflict on constraint price_pkey update

	result, err := db.NamedExec(insertQuery, priceDb)
	if err != nil {
		log.Fatalf("db.NamedExec %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("result.RowsAffected %v", err)
	}

	log.Printf("inserted %v rows out of %v records with last block number %v", rows, t.Len(), blockNumber)

	countSaved = rows

	return
}
