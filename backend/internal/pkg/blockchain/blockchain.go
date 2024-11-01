package blockchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	client *ethclient.Client
}

func NewClient(url string) (*Client, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	return &Client{client: client}, nil
}

func (c *Client) GetBalance(address string) (*big.Int, error) {
	account := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (c *Client) SendTransaction(tx *types.Transaction) error {
	return c.client.SendTransaction(context.Background(), tx)
}
