//账户、账户状态
//关于账户状态的一些基本操作

package core

import (
	"blockEmulator/utils"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"
)

type Account struct { //Account结构包含账户的各种信息
	AcAddress utils.Address //AcAddress：该变量似乎代表账户的地址
	PublicKey []byte        //PublicKey：该变量似乎代表账户的公钥
}

// AccoutState记录帐户的详细信息，它将保存在状态树中
type AccountState struct { //AccountState结构包含账户状态的各种信息
	AcAddress   utils.Address //AcAddress：该变量似乎代表账户的地址
	Nonce       uint64        //Nonce：该变量似乎代表账户的随机数
	Balance     *big.Int      //Balance：该变量似乎代表账户的余额
	StorageRoot []byte        //代表账户的存储根，仅适用于智能合约账户
	CodeHash    []byte        //代表账户的代码哈希，仅适用于智能合约账户
}

// 减少账户余额
func (as *AccountState) Deduct(val *big.Int) bool { //Deduct方法用于减少账户余额
	if as.Balance.Cmp(val) < 0 {
		return false
	} //如果账户余额小于val，则返回false
	as.Balance.Sub(as.Balance, val) //否则，减少账户余额
	return true
}

// 增加账户余额
func (s *AccountState) Deposit(value *big.Int) {
	s.Balance.Add(s.Balance, value)
}

// 对 AccountState 进行编码以便存储在 MPT 中
func (as *AccountState) Encode() []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(as)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

// 解码帐户状态
func DecodeAS(b []byte) *AccountState {
	var as AccountState

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&as)
	if err != nil {
		log.Panic(err)
	}
	return &as
}

// 用于计算 MPT 根的哈希 AccountState
func (as *AccountState) Hash() []byte {
	h := sha256.Sum256(as.Encode())
	return h[:]
}
