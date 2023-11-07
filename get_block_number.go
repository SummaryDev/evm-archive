package main

type GetBlockNumberResponse string

func NewGetBlockNumberResponse() *GetBlockNumberResponse {
	var newGetBlockNumberResponse GetBlockNumberResponse = ""
	return &newGetBlockNumberResponse
}

func (t *GetBlockNumberResponse) Len() int {
	blocknumber := *t
	if len(blocknumber) >= 2 {
		return 1
	} else {
		return 0
	}
}

func (t *GetBlockNumberResponse) ToNumber() uint64 {
	s := string(*t)
	return FromHex(s)
}

func (t *GetBlockNumberResponse) Save(dataSourceName string, req RpcRequest) (countSaved int64) {
	return
}
