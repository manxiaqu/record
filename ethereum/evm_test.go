package ethereum

import (
    "testing"
    "fmt"
    
    "github.com/ethereum/go-ethereum/common"
)

func TestEVM(t *testing.T) {
    code := "60056004016000526001601ff3"
    newCode := common.Hex2Bytes(code)
    fmt.Printf("%x\n", newCode)
    fmt.Println(uint64(len(newCode)))
}
