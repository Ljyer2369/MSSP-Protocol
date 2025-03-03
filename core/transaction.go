// Definition of transaction

package core

import (
	"blockEmulator/utils"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
	"math/big"
	"time"
)

type Transaction struct { //Transaction结构包含交易的各种信息
	Sender    utils.Address //Sender：该变量似乎代表交易的发送者
	Recipient utils.Address //Recipient：该变量似乎代表交易的接收者
	Nonce     uint64        //Nonce：该变量似乎代表交易的随机数
	Signature []byte        //Signature：该变量似乎代表交易的签名
	Value     *big.Int      //Value：该变量似乎代表交易的价值
	TxHash    []byte        //TxHash：该变量似乎代表交易的哈希值

	Time time.Time //交易添加到池的时间

	//用于交易中继
	Relayed bool //Relayed：该变量似乎代表交易是否被中继
	//在broker中使用时，如果tx不是broker1或broker2 tx，这些值应该为空。
	HasBroker      bool
	SenderIsBroker bool
	OriginalSender utils.Address
	FinalRecipient utils.Address
	RawTxHash      []byte
}

func (tx *Transaction) PrintTx() string { //PrintTx方法用于打印交易
	vals := []interface{}{ //将交易的各种信息转换为接口类型
		tx.Sender[:],
		tx.Recipient[:],
		tx.Value,
		string(tx.TxHash[:]),
	}
	res := fmt.Sprintf("%v\n", vals) //将交易的各种信息转换为字符串
	return res
}

// 对交易进行编码以进行存储
func (tx *Transaction) Encode() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// 解码交易
func DecodeTx(to_decode []byte) *Transaction {
	var tx Transaction

	decoder := gob.NewDecoder(bytes.NewReader(to_decode))
	err := decoder.Decode(&tx)
	if err != nil {
		log.Panic(err)
	}

	return &tx
}

// new a transaction
func NewTransaction(sender, recipient string, value *big.Int, nonce uint64) *Transaction {
	//NewTransaction方法用于创建交易
	tx := &Transaction{ //创建交易
		Sender:    sender, //Sender: sender 中的冒号(:) 是用于创建一个结构体（struct）字面量的语法。它表示将一个字段名与相应的值关联在一起，以初始化结构体的字段。在这种情况下，Sender 是结构体的字段名，而 sender 是赋给该字段的值
		Recipient: recipient,
		Value:     value,
		Nonce:     nonce,
	}

	hash := sha256.Sum256(tx.Encode()) //对交易进行哈希
	tx.TxHash = hash[:]
	tx.Relayed = false     //设置交易的中继状态为false，表示默认情况下不中继（转发）交易
	tx.FinalRecipient = "" //设置交易的最终接收者为空
	tx.OriginalSender = "" //设置交易的原始发送者为空
	tx.RawTxHash = nil
	tx.HasBroker = false
	tx.SenderIsBroker = false
	return tx
}

//上面的函数本质上是使用提供的发送者、接收者、值和随机数创建并初始化交易，然后计算交易数据的哈希值。生成的Transaction实例已准备好在您的区块链或加密货币系统中使用。
