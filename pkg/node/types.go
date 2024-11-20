package node

import (
	"encoding/json"
	"math"

	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

// from the blockchain node
type Block struct {
	Hash           Bytes         `json:"hash"`
	Size           int64         `json:"size"`
	StrippedSize   int64         `json:"strippedsize"`
	Weight         int32         `json:"weight"`
	Height         int32         `json:"height"`
	Version        int32         `json:"version"`
	HashMerkleRoot Bytes         `json:"merkleRoot"`
	Transactions   []Transaction `json:"tx"`
	Time           int32         `json:"time"`
	MedianTime     int32         `json:"mediantime"`
	Nonce          int64         `json:"nonce"`
	Bits           Bytes         `json:"bits"`
	Difficulty     float64       `json:"difficulty"`
	Chainwork      Bytes         `json:"chainwork"`
	PrevBlockHash  Bytes         `json:"previousblockhash"`
	NextBlockHash  Bytes         `json:"nextblockhash,omitempty"`
}

type Transaction struct {
	Txid     Bytes   `json:"txid"`
	Hash     Bytes   `json:"hash"`
	Version  int     `json:"version"`
	Size     int     `json:"size"`
	VSize    int     `json:"vsize"`
	Weight   int     `json:"weight"`
	LockTime uint32  `json:"locktime"`
	Vin      []Vin   `json:"vin"`
	Vout     []Vout  `json:"vout"`
	FloatFee float64 `json:"fee,omitempty"`
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	type TxAlias Transaction
	aux := &struct {
		LocktimeAlt *uint32 `json:"lock_time"`
		*TxAlias
	}{
		TxAlias: (*TxAlias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.LocktimeAlt != nil {
		t.LockTime = *aux.LocktimeAlt
	}
	return nil
}

func (tx *Transaction) TxHash() Bytes {
	return tx.Hash
}

func (tx *Transaction) Fee() int {
	return int(math.Round(tx.FloatFee * 1e8))
}

type Vin struct {
	HashPrevout  *Bytes     `json:"txid,omitempty"`
	IndexPrevout int        `json:"vout,omitempty"`
	ScriptSig    *ScriptSig `json:"scriptSig,omitempty"`
	Coinbase     *Bytes     `json:"coinbase,omitempty"`
	TxinWitness  []Bytes    `json:"txinwitness,omitempty"`
	Sequence     uint32     `json:"sequence"`
}

type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type Vout struct {
	FloatValue       float64      `json:"value"`
	Index            int          `json:"n"`
	NodeScriptPubKey ScriptPubKey `json:"scriptPubKey"`
}

func (vout *Vout) Value() int {
	return int(math.Round(vout.FloatValue * 1e8))
}

type ScriptPubKey struct {
	Asm     string `json:"asm"`
	Desc    string `json:"desc"`
	Hex     Bytes  `json:"hex"`
	Address string `json:"address"`
	Type    string `json:"type"`
}

// Spaces types
type Tip struct {
	Hash   Bytes `json:"hash"`
	Height int   `json:"height"`
}

type ServerInfo struct {
	Chain string `json:"chain"`
	Tip   Tip    `json:"tip"`
}

type RollOutSpace struct {
	Name  string `json:"space"`
	Value int    `json:"value"`
}

type Covenant struct {
	Type          string      `json:"type"`
	BurnIncrement *int        `json:"burn_increment,omitempty"`
	Signature     Bytes       `json:"signature"`
	TotalBurned   *int        `json:"total_burned,omitempty"`
	ClaimHeight   *int        `json:"claim_height,omitempty"`
	ExpireHeight  *int        `json:"expire_height,omitempty"`
	Data          interface{} `json:"data,omitempty"`
}

type SpacesBlock struct {
	Transactions []MetaTransaction `json:"tx_meta"`
}

type MetaTransaction struct {
	TxID   Bytes `json:"txid"`
	Spends []struct {
		N           int          `json:"n"`
		ScriptError *ScriptError `json:"script_error,omitempty"`
	} `json:"spends"`
	Creates []CreateMeta `json:"creates"`
	Updates []UpdateMeta `json:"updates"`
}

type ScriptError struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type CreateMeta struct {
	N            int      `json:"n"`
	Name         string   `json:"name,omitempty"`
	Covenant     Covenant `json:"covenant,omitempty"`
	Value        int      `json:"value"`
	ScriptPubKey Bytes    `json:"script_pubkey"`
}

type UpdateMeta struct {
	Type     string     `json:"type"`
	Priority int        `json:"priority,omitempty"`
	Output   OutputMeta `json:"output"`
	Reason   string     `json:"reason,omitempty"`
}

type OutputMeta struct {
	TxID         Bytes    `json:"txid"`
	N            int      `json:"n"`
	Covenant     Covenant `json:"covenant"`
	Value        int      `json:"value"`
	Name         string   `json:"name,omitempty"`
	ScriptPubKey Bytes    `json:"script_pubkey"`
}

func (vout *Vout) Scriptpubkey() *Bytes {
	return &vout.NodeScriptPubKey.Hex
}
