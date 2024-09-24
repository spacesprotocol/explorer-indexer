package node

import (
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
	// WitnessRoot    Bytes         `json:"witnessRoot"`
	// TreeRoot       Bytes         `json:"treeRoot"`
	// ReservedRoot   Bytes         `json:"reservedRoot"`
	// Mask           Bytes         `json:"mask"`
	Time          int32   `json:"time"`
	MedianTime    int32   `json:"mediantime"`
	Nonce         int64   `json:"nonce"`
	Bits          Bytes   `json:"bits"`
	Difficulty    float64 `json:"difficulty"`
	Chainwork     Bytes   `json:"chainwork"`
	PrevBlockHash Bytes   `json:"previousblockhash"`
	NextBlockHash Bytes   `json:"nextblockhash,omitempty"`
	// ExtraNonce     Bytes         `json:"extraNonce"`
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
	FloatFee float64 `json:"fee,omitempty"` // Fee is optional
	// Vmetaout []Vmetaout `json:"vmetaout"`
}

func (tx *Transaction) TxHash() Bytes {
	return tx.Hash
}

func (tx *Transaction) Fee() int {
	return int(math.Round(tx.FloatFee * 1e8))
}

// type Covenant struct {
// 	Type          string      `json:"type"`
// 	BurnIncrement int         `json:"burn_increment"`
// 	Signature     string      `json:"signature"`
// 	TotalBurned   int         `json:"total_burned"`
// 	ClaimHeight   int         `json:"claim_height"`
// 	ExpireHeight  int         `json:"expire_height"`
// 	Data          interface{} `json:"data"` // To handle null or any type
// }

type Vin struct {
	HashPrevout  Bytes      `json:"txid,omitempty"`
	IndexPrevout int        `json:"vout,omitempty"`
	ScriptSig    *ScriptSig `json:"scriptSig,omitempty"`
	Coinbase     Bytes      `json:"coinbase,omitempty"`
	TxinWitness  []Bytes    `json:"txinwitness,omitempty"`
	Sequence     uint32     `json:"sequence"`
}

type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type Vout struct {
	FloatValue   float64      `json:"value"`
	Index        int          `json:"n"`
	ScriptPubKey ScriptPubKey `json:"scriptPubKey"`
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

//	type Space struct {
//		Outpoint     string   `json:"outpoint"`
//		Value        int      `json:"value"`
//		ScriptPubKey string   `json:"script_pubkey"`
//		Name         string   `json:"name"`
//		Covenant     Covenant `json:"covenant"`
//	}

// Spaces types
type Tip struct {
	Hash   Bytes `json:"hash"`
	Height int   `json:"height"`
}

type ServerInfo struct {
	Chain string `json:"chain"`
	Tip   Tip    `json:"tip"`
}

type Space struct {
	Outpoint     string   `json:"outpoint"`
	Value        int      `json:"value"`
	ScriptPubKey Bytes    `json:"script_pubkey"`
	Name         string   `json:"name"`
	Covenant     Covenant `json:"covenant"`
}

// Define the Covenant struct
type Covenant struct {
	Type          string      `json:"type"`
	BurnIncrement *int        `json:"burn_increment,omitempty"` // Use pointer for optional fields
	Signature     *string     `json:"signature,omitempty"`      // Use pointer for optional fields
	TotalBurned   int         `json:"total_burned"`
	ClaimHeight   *int        `json:"claim_height,omitempty"`  // Use pointer for optional fields
	ExpireHeight  *int        `json:"expire_height,omitempty"` // Use pointer for optional fields
	Data          interface{} `json:"data,omitempty"`          // Use interface{} for generic fields
}

type SpacesBlock struct {
	TxData []SpacesTx `json:"tx_data"`
}

type SpacesTx struct {
	Version  int         `json:"version"`
	TxID     Bytes       `json:"txid"`
	LockTime int         `json:"lock_time"`
	Vin      []SpacesVin `json:"vin"`
	Vout     []Vout      `json:"vout"`
	VMetaOut []VMetaOut  `json:"vmetaout"`
}

// Define the Vin struct for inputs
type SpacesVin struct {
	PreviousOutput string   `json:"previous_output"`
	ScriptSig      string   `json:"script_sig"`
	Sequence       int      `json:"sequence"`
	Witness        []string `json:"witness"`
	ScriptError    *string  `json:"script_error,omitempty"` // Optional field
}

// Define the Vout struct for outputs
// type Vout struct {
// 	Value        int    `json:"value"`
// 	ScriptPubKey string `json:"script_pubkey"`
// }

// Define the VMetaOut struct for meta outputs
type VMetaOut struct {
	Outpoint     string   `json:"outpoint"`
	Value        int      `json:"value"`
	ScriptPubKey string   `json:"script_pubkey"`
	Name         string   `json:"name"`
	Covenant     Covenant `json:"covenant"`
}
