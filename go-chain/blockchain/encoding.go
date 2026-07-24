package blockchain

import (
	"encoding/binary"
	"errors"
)

func EncodeTransactionBinary(tx Transaction) ([]byte, error) {
	b := make([]byte, 0)
	b = appendString(b, tx.ID)
	b = appendString(b, tx.From)
	b = appendString(b, tx.FromPubKey)
	b = appendString(b, tx.To)
	b = appendUint64(b, tx.Amount)
	b = appendUint64(b, tx.Fee)
	b = appendUint64(b, tx.Nonce)
	b = appendString(b, string(tx.TxType))
	b = appendString(b, tx.Payload)
	b = appendString(b, tx.Signature)
	b = appendInt64(b, tx.Timestamp)
	b = appendString(b, tx.ChainID)
	return b, nil
}

func DecodeTransactionBinary(data []byte) (Transaction, error) {
	var tx Transaction
	var err error
	tx.ID, data, err = popString(data)
	if err != nil {
		return tx, err
	}
	tx.From, data, err = popString(data)
	if err != nil {
		return tx, err
	}
	tx.FromPubKey, data, err = popString(data)
	if err != nil {
		return tx, err
	}
	tx.To, data, err = popString(data)
	if err != nil {
		return tx, err
	}
	tx.Amount, data, err = popUint64(data)
	if err != nil {
		return tx, err
	}
	tx.Fee, data, err = popUint64(data)
	if err != nil {
		return tx, err
	}
	tx.Nonce, data, err = popUint64(data)
	if err != nil {
		return tx, err
	}
	txTypeStr, data, err := popString(data)
	if err != nil {
		return tx, err
	}
	tx.TxType = TransactionType(txTypeStr)
	tx.Payload, data, err = popString(data)
	if err != nil {
		return tx, err
	}
	tx.Signature, data, err = popString(data)
	if err != nil {
		return tx, err
	}
	tx.Timestamp, data, err = popInt64(data)
	if err != nil {
		return tx, err
	}
	tx.ChainID, _, err = popString(data)
	return tx, err
}

func EncodeBlockBinary(block Block) ([]byte, error) {
	b := make([]byte, 0)
	b = appendUint64(b, block.Index)
	b = appendString(b, block.Author)
	b = appendString(b, block.MinerAddress)
	b = appendString(b, block.PreviousHash)
	b = appendInt64(b, block.Timestamp)
	b = appendBytes(b, []byte(block.BlockHash))
	if block.TxMerkleRoot != "" {
		b = appendBytes(b, []byte(block.TxMerkleRoot))
	} else {
		b = appendBytes(b, []byte{})
	}
	b = appendUint64(b, uint64(len(block.Transactions)))
	for _, tx := range block.Transactions {
		encoded, err := EncodeTransactionBinary(tx)
		if err != nil {
			return nil, err
		}
		b = appendBytes(b, encoded)
	}
	b = appendUint64(b, block.Nonce)
	return b, nil
}

func DecodeBlockBinary(data []byte) (Block, error) {
	var block Block
	var err error
	block.Index, data, err = popUint64(data)
	if err != nil {
		return block, err
	}
	block.Author, data, err = popString(data)
	if err != nil {
		return block, err
	}
	block.MinerAddress, data, err = popString(data)
	if err != nil {
		return block, err
	}
	block.PreviousHash, data, err = popString(data)
	if err != nil {
		return block, err
	}
	block.Timestamp, data, err = popInt64(data)
	if err != nil {
		return block, err
	}
	hashBytes, data, err := popBytes(data)
	if err != nil {
		return block, err
	}
	block.BlockHash = string(hashBytes)
	merkleBytes, data, err := popBytes(data)
	if err != nil {
		return block, err
	}
	if len(merkleBytes) > 0 {
		block.TxMerkleRoot = string(merkleBytes)
	}
	txCount, data, err := popUint64(data)
	if err != nil {
		return block, err
	}
	block.Transactions = make([]Transaction, txCount)
	for i := uint64(0); i < txCount; i++ {
		txBytes, rest, err := popBytes(data)
		if err != nil {
			return block, err
		}
		tx, err := DecodeTransactionBinary(txBytes)
		if err != nil {
			return block, err
		}
		block.Transactions[i] = tx
		data = rest
	}
	block.Nonce, _, err = popUint64(data)
	return block, err
}

func appendString(b []byte, s string) []byte {
	b = appendUint64(b, uint64(len(s)))
	b = append(b, []byte(s)...)
	return b
}

func appendBytes(b []byte, data []byte) []byte {
	b = appendUint64(b, uint64(len(data)))
	b = append(b, data...)
	return b
}

func appendUint64(b []byte, v uint64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendInt64(b []byte, v int64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], uint64(v))
	return append(b, tmp[:]...)
}

func popString(data []byte) (string, []byte, error) {
	s, rest, err := popBytes(data)
	if err != nil {
		return "", nil, err
	}
	return string(s), rest, nil
}

func popBytes(data []byte) ([]byte, []byte, error) {
	if len(data) < 8 {
		return nil, nil, errors.New("insufficient data for length prefix")
	}
	length := binary.BigEndian.Uint64(data[:8])
	if uint64(len(data)) < 8+length {
		return nil, nil, errors.New("insufficient data for payload")
	}
	rest := data[8+length:]
	return data[8 : 8+length], rest, nil
}

func popUint64(data []byte) (uint64, []byte, error) {
	if len(data) < 8 {
		return 0, nil, errors.New("insufficient data for uint64")
	}
	v := binary.BigEndian.Uint64(data[:8])
	return v, data[8:], nil
}

func popInt64(data []byte) (int64, []byte, error) {
	v, rest, err := popUint64(data)
	return int64(v), rest, err
}
