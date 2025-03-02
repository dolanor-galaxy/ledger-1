package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// Transactions struct
type Transactions struct {
	Transactions []TransactionData `json:"transactions" binding:"required,dive"`
}

type TransactionData struct {
	Postings  Postings `json:"postings"`
	Reference string   `json:"reference"`
	Metadata  Metadata `json:"metadata" swaggertype:"object"`
}

type Transaction struct {
	TransactionData
	ID        int64  `json:"txid"`
	Timestamp string `json:"timestamp"`
	Hash      string `json:"hash" swaggerignore:"true"`
}

func (t *Transaction) AppendPosting(p Posting) {
	t.Postings = append(t.Postings, p)
}

func (t *Transaction) Reverse() TransactionData {
	postings := t.Postings
	postings.Reverse()

	ret := TransactionData{
		Postings: postings,
	}
	if t.Reference != "" {
		ret.Reference = "revert_" + t.Reference
	}
	return ret
}

func Hash(t1 *Transaction, t2 *Transaction) string {
	b1, _ := json.Marshal(t1)
	b2, _ := json.Marshal(t2)

	h := sha256.New()
	h.Write(b1)
	h.Write(b2)

	return fmt.Sprintf("%x", h.Sum(nil))
}
