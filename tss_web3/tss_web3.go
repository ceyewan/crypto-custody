package main

import (
	"crypto/ecdsa"
	"math/big"

	// 这里使用 binance-chain/tss-lib 作为示例
	// 实际项目中可能需要选择其他实现或自己实现

	"github.com/binance-chain/tss-lib/tss"
)

// 参与方结构
type Party struct {
	ID        *tss.PartyID
	Share     *big.Int // 私钥分片
	PublicKey *ecdsa.PublicKey
}

// 初始化参与方
func initializeParties(n int) []*Party {
	parties := make([]*Party, n)
	for i := 0; i < n; i++ {
		parties[i] = &Party{
			ID: tss.NewPartyID(
				string(i),
				string(i),
				big.NewInt(int64(i)),
			),
		}
	}
	return parties
}
