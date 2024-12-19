package main

import (
	"bufio"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"io"
	"math/big"
	"os"
	"testing"
)

const (
	blockBloomFile = "block_blooms.dat"
	// file is a simple binary file with sequentially stored Bloom filters,
	// each 256 bits, repeated for the number of blocks

	address          = "0x0c0e5f2ff0ff18a3be9b835635039256dc4b4963"
	topic0           = "0xa307f5d0802489baddec443058a63ce115756de9020e2b07d3e2cd2f21269e2a"
	fromBlock uint64 = 17371439
)

func Test_BloomSearch(t *testing.T) {

	file, err := os.Open(blockBloomFile)
	require.NoError(t, err)
	defer file.Close()

	// position the file pointer at the block where the search begins
	p := int64(fromBlock * 256)
	n, err := file.Seek(p, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, p, n)
	r := bufio.NewReader(file)

	// perform a bitwise OR for the hashes of the data being searched
	// this allows for a fast negative lookup using the Bloom filters
	search := bloom9(common.HexToAddress(address).Bytes())
	search.Or(search, bloom9(common.HexToHash(topic0).Bytes()))

	var bloomData = make([]byte, 256)
	count := 0

	for {
		n, err := r.Read(bloomData)
		if err != nil && errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		require.Equal(t, 256, n)
		bloom := new(big.Int).SetBytes(bloomData)

		if new(big.Int).And(bloom, search).Cmp(search) == 0 {
			// data we are searching matches the Bloom filter
			count++
		}
	}
	t.Log(count)
}

func bloom9(b []byte) *big.Int {
	b = crypto.Keccak256(b)

	r := new(big.Int)

	for i := 0; i < 6; i += 2 {
		t := big.NewInt(1)
		b := (uint(b[i+1]) + (uint(b[i]) << 8)) & 2047
		r.Or(r, t.Lsh(t, b))
	}

	return r
}
