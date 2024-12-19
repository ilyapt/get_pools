package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
	"math/big"
	"os"
	"strings"
	"testing"
)

func Test_Unpack(t *testing.T) {
	triCryptoAbi := `[{"name":"TricryptoPoolDeployed","inputs":[{"name":"pool","type":"address","indexed":false},{"name":"name","type":"string","indexed":false},{"name":"symbol","type":"string","indexed":false},{"name":"weth","type":"address","indexed":false},{"name":"coins","type":"address[3]","indexed":false},{"name":"math","type":"address","indexed":false},{"name":"salt","type":"bytes32","indexed":false},{"name":"packed_precisions","type":"uint256","indexed":false},{"name":"packed_A_gamma","type":"uint256","indexed":false},{"name":"packed_fee_params","type":"uint256","indexed":false},{"name":"packed_rebalancing_params","type":"uint256","indexed":false},{"name":"packed_prices","type":"uint256","indexed":false},{"name":"deployer","type":"address","indexed":false}],"anonymous":false,"type":"event"},{"stateMutability":"nonpayable","type":"function","name":"deploy_pool","inputs":[{"name":"_name","type":"string"},{"name":"_symbol","type":"string"},{"name":"_coins","type":"address[3]"},{"name":"_weth","type":"address"},{"name":"implementation_id","type":"uint256"},{"name":"A","type":"uint256"},{"name":"gamma","type":"uint256"},{"name":"mid_fee","type":"uint256"},{"name":"out_fee","type":"uint256"},{"name":"fee_gamma","type":"uint256"},{"name":"allowed_extra_profit","type":"uint256"},{"name":"adjustment_step","type":"uint256"},{"name":"ma_exp_time","type":"uint256"},{"name":"initial_prices","type":"uint256[2]"}],"outputs":[{"name":"","type":"address"}]}]`

	type record struct {
		Address     common.Address `json:"address"`
		Input       hexutil.Bytes  `json:"input,omitempty"`
		Topics      []common.Hash  `json:"topics"`
		Data        hexutil.Bytes  `json:"data"`
		BlockNumber hexutil.Uint64 `json:"blockNumber"`
		TxHash      common.Hash    `json:"transactionHash"`
	}

	type input struct {
		Name               string
		Symbol             string
		Coins              [3]common.Address
		Weth               common.Address
		ImplementationId   *big.Int
		A                  *big.Int
		Gamma              *big.Int
		MidFee             *big.Int
		OutFee             *big.Int
		FeeGamma           *big.Int
		AllowedExtraProfit *big.Int
		AdjustmentStep     *big.Int
		MaExpTime          *big.Int
		InitialPrices      [2]*big.Int
	}

	type event struct {
		Pool                    common.Address
		Name                    string
		Symbol                  string
		Weth                    common.Address
		Coins                   [3]common.Address
		Math                    common.Address
		Salt                    [32]uint8
		PackedPrecisions        *big.Int
		PackedAGamma            *big.Int
		PackedFeeParams         *big.Int
		PackedRebalancingParams *big.Int
		PackedPrices            *big.Int
		Deployer                common.Address
	}

	a, err := abi.JSON(strings.NewReader(triCryptoAbi))
	require.NoError(t, err)

	data, err := os.ReadFile("ETH_TRICRYPTO_NG.json")
	require.NoError(t, err)
	var tmp struct {
		Records []record `json:"records"`
	}
	require.NoError(t, json.Unmarshal(data, &tmp))

	m := a.Methods["deploy_pool"]
	e := a.Events["TricryptoPoolDeployed"]
	for _, l := range tmp.Records {

		var xx input
		require.NoError(t, m.Inputs.Unpack(&xx, l.Input[4:]))
		data1, err := json.MarshalIndent(xx, "", "  ")
		require.NoError(t, err)
		fmt.Println(string(data1))

		var yy event
		require.NoError(t, e.Inputs.Unpack(&yy, l.Data))
		data2, err := json.MarshalIndent(yy, "", "  ")
		require.NoError(t, err)
		fmt.Println(string(data2))
	}
}
