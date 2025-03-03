// Definition of block

package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// 区块头的定义
type BlockHeader struct { //BlockHeader结构包含区块头的各种信息
	ParentBlockHash []byte    //ParentBlockHash：该变量似乎代表父区块的哈希值
	StateRoot       []byte    //StateRoot：该变量似乎代表状态树的根
	TxRoot          []byte    //TxRoot：该变量似乎代表交易树的根
	Number          uint64    //Number：该变量似乎代表区块的编号
	Time            time.Time //Time：该变量似乎代表区块的时间
	Miner           uint64    //Miner：该变量似乎代表区块的旷工
}

// 对 blockHeader 进行编码以进一步存储
func (bh *BlockHeader) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(bh)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

// 解码区块头
func DecodeBH(b []byte) *BlockHeader {
	var blockHeader BlockHeader

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&blockHeader)
	if err != nil {
		log.Panic(err)
	}

	return &blockHeader
}

// 哈希区块头
func (bh *BlockHeader) Hash() []byte {
	hash := sha256.Sum256(bh.Encode())
	return hash[:]
}

// 打印区块头
func (bh *BlockHeader) PrintBlockHeader() string {
	vals := []interface{}{
		hex.EncodeToString(bh.ParentBlockHash),
		hex.EncodeToString(bh.StateRoot),
		hex.EncodeToString(bh.TxRoot),
		bh.Number,
		bh.Time,
	}
	res := fmt.Sprintf("%v\n", vals)
	return res
}

// 区块的定义
type Block struct {
	Header *BlockHeader   //Header：该变量似乎代表区块头
	Body   []*Transaction //Body：该变量似乎代表区块体
	Hash   []byte         //Hash：该变量似乎代表区块的哈希值
}

func NewBlock(bh *BlockHeader, bb []*Transaction) *Block {
	//NewBlock()函数用于创建区块。它需要两个参数： bh（类型为*BlockHeader）：这是一个指向BlockHeader结构的指针。 bb（类型为[]*Transaction）：这是一个指向Transaction结构的指针的切片。
	return &Block{Header: bh, Body: bb}
}

func (b *Block) PrintBlock() string {
	//PrintBlock()函数用于打印区块。它需要一个参数： b（类型为*Block）：这是一个指向Block结构的指针。
	vals := []interface{}{
		b.Header.Number,
		b.Hash,
	}
	res := fmt.Sprintf("%v\n", vals)
	fmt.Println(res)
	return res
}

// 用于存储的编码块
func (b *Block) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

// 解码区块
func DecodeB(b []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}
