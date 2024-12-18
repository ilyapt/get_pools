package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"math/big"
	"os"
	"time"
)

type config map[string]struct {
	Rpc     string `yaml:"Rpc"`
	LogLen  uint64 `yaml:"LogLen"`
	Sources map[string]struct {
		Factory   common.Address  `yaml:"Factory"`
		Topics    [][]common.Hash `yaml:"Topics"`
		FromBlock uint64          `yaml:"FromBlock"`
	} `yaml:"Sources"`
}

func main() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var cfg config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
			var xxx struct {
				Block uint64
				Logs  []types.Log
			}

			file := fmt.Sprintf("%s.json", source)

			data, err := os.ReadFile(file)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				if err != nil {
					log.Fatal(err)
				}
			}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &xxx); err != nil {
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

				xxx.Logs = append(xxx.Logs, ll...)
				fmt.Println(i, len(ll), len(xxx.Logs))
				time.Sleep(time.Second)
			}

			data1, err := json.MarshalIndent(struct {
				Block uint64
				Logs  []types.Log
			}{
				Block: b.Number.Uint64(),
				Logs:  xxx.Logs,
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
