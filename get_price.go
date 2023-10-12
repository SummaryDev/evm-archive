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
	To   string `json:"to,omitempty"`
	Data string `json:"data,omitempty"`
}

func (t *GetPriceRequest) ToJson() (s string) {
	b, _ := json.Marshal(t)
	s = string(b)
	return
}

var oracles = map[string]string{
	"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": "0xf4030086522a5beea4988f8ca5b36dbc97bee88c", // WBTC
	"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": "0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419", // WETH
}

var tokens = map[string]string{
	"0xf4030086522a5beea4988f8ca5b36dbc97bee88c": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
	"0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
}

func NewGetPriceRequest(token string) *GetPriceRequest {
	q := &GetPriceRequest{}
	q.To = oracles[token]
	q.Data = "0x50d25bcd" // latestAnswer encoded as function selector
	log.Printf("from %v with %v", token, q.ToJson())
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

func (t *GetPriceResponse) Save(dataSourceName string, req Request) (countSaved int64) {
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
	token := tokens[getPriceRequest.To]

	priceDb := PriceDb{Address: token, BlockNumber: blockNumber, Price: price}

	db, err := sqlx.Open("pgx", dataSourceName) // postgres
	if err != nil {
		log.Fatalf("sqlx.Open %v", err)
	}
	defer db.Close()

	insertQuery := "insert into price (address, block_number, price) " +
		"values (:address, :blocknumber, :price) " +
		"on conflict on constraint price_pkey do nothing" //todo on conflict on constraint price_pkey update

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
