package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Persistent interface {
	Save(dataSourceName string, req RpcRequest) (countSaved int64)
	Len() int
}

type RpcRequest struct {
	AsOfBlock uint64
	Query     interface{}
	method    string
	endpoint  string
}

type RpcResponse struct {
	persistent     Persistent
	dataSourceName string
}

func FromHex(hex string) (value uint64) {
	s := strings.Replace(hex, "0x", "", -1)
	value, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		log.Printf("cannot FromHex: %s\n", err)
	}
	return
}

func ToHex(value uint64) (hex string) {
	hex = fmt.Sprintf("0x%x", value)
	return
}
