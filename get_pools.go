package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

type record struct {
	Address     common.Address `json:"address"`
	Input       hexutil.Bytes  `json:"input,omitempty"`
	Topics      []common.Hash  `json:"topics"`
	Data        hexutil.Bytes  `json:"data"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	TxHash      common.Hash    `json:"transactionHash"`
}

type data struct {
	Block   uint64   `json:"block"`
	Records []record `json:"records"`
}

type config map[string]struct {
	Rpc     string `yaml:"Rpc"`
	LogLen  uint64 `yaml:"LogLen"`
	Sources map[string]struct {
		Factory common.Address `yaml:"Factory"`
		Call    *struct {
			Selectors []hexutil.Bytes `yaml:"Selectors"`
		} `yaml:"Call"`
		Topics    [][]common.Hash `yaml:"Topics"`
		FromBlock uint64          `yaml:"FromBlock"`
	} `yaml:"Sources"`
}

func main() {
	cfgData, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var cfg config

	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		log.Fatal(err)
	}

	for _, chainCfg := range cfg {
		cli, err := ethclient.Dial(chainCfg.Rpc)
		if err != nil {
			log.Fatal(err)
		}
		b, err := cli.HeaderByNumber(context.Background(), nil)
		if err != nil {
			log.Fatal(err)
		}

		for source, sourceCfg := range chainCfg.Sources {
			var xxx data

			file := fmt.Sprintf("%s.json", source)

			jsonData, err := os.ReadFile(file)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				if err != nil {
					log.Fatal(err)
				}
			}
			if len(jsonData) > 0 {
				if err := json.Unmarshal(jsonData, &xxx); err != nil {
					log.Fatal(err)
				}
			}

			from := sourceCfg.FromBlock
			if xxx.Block > sourceCfg.FromBlock {
				from = xxx.Block + 1
			}

			for i := from; i <= b.Number.Uint64(); i += chainCfg.LogLen {
				ll, err := cli.FilterLogs(context.Background(), ethereum.FilterQuery{
					FromBlock: new(big.Int).SetUint64(i),
					ToBlock:   new(big.Int).SetUint64(i + chainCfg.LogLen),
					Addresses: []common.Address{sourceCfg.Factory},
					Topics:    sourceCfg.Topics,
				})
				if err != nil {
					log.Fatal(err)
				}

				for _, l := range ll {
					rec := record{
						Address:     l.Address,
						Topics:      l.Topics,
						Data:        l.Data,
						BlockNumber: hexutil.Uint64(l.BlockNumber),
						TxHash:      l.TxHash,
					}
					if sourceCfg.Call != nil {
						fmt.Println("SELECTORS", sourceCfg.Call.Selectors)
						rec.Input, err = fetchInput(cli, l.TxHash, sourceCfg.Factory, sourceCfg.Call.Selectors)
						if err != nil {
							log.Fatal(err)
						}
					}
					xxx.Records = append(xxx.Records, rec)
				}
				fmt.Println(i, len(ll), len(xxx.Records))
				time.Sleep(time.Second)
			}

			data1, err := json.MarshalIndent(data{
				Block:   b.Number.Uint64(),
				Records: xxx.Records,
			}, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(file, data1, 0644); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func fetchInput(cli *ethclient.Client, txHash common.Hash, factory common.Address, selectors []hexutil.Bytes) ([]byte, error) {
	time.Sleep(time.Second)
	tx, _, err := cli.TransactionByHash(context.Background(), txHash)
	if err != nil {
		return nil, err
	}
	if *tx.To() == factory && selectorsContain(selectors, tx.Data()) {
		return tx.Data(), nil
	}
	fmt.Println(tx.To().String(), factory.String(), "0x"+hex.EncodeToString(tx.Data()[:4]), selectors[0].String())

	return traceTx(txHash, factory, selectors)
}

func selectorsContain(selectors []hexutil.Bytes, data []byte) bool {
	if len(data) < 4 {
		return false
	}
	for _, s := range selectors {
		if s[0] == data[0] && s[1] == data[1] && s[2] == data[2] && s[3] == data[3] {
			return true
		}
	}
	return false
}

func traceTx(txHash common.Hash, factory common.Address, selectors []hexutil.Bytes) ([]byte, error) {
	r, err := http.Post("https://docs-demo.quiknode.pro/", "application/json",
		strings.NewReader(fmt.Sprintf(`{"method":"debug_traceTransaction","params":["%s", {"tracer": "callTracer"}], "id":1,"jsonrpc":"2.0"}`, txHash.String())))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var x struct {
		Result call
	}

	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		return nil, err
	}

	return x.Result.find(factory, selectors)
}

type call struct {
	From   common.Address
	To     common.Address
	Input  hexutil.Bytes
	Output hexutil.Bytes
	Calls  []call
	Type   string
}

func (c *call) find(factory common.Address, selectors []hexutil.Bytes) ([]byte, error) {
	if c.To == factory && selectorsContain(selectors, c.Input) {
		return c.Input, nil
	}
	for _, cc := range c.Calls {
		if d, err := cc.find(factory, selectors); err == nil {
			return d, nil
		}
	}
	return nil, errors.New("couldn't find")
}
