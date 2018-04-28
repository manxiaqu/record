package ethereum

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

type FunctionCall struct {
	Method string  `json:"method"`
	Params []Param `json:"params"`
}

type Param struct {
	ParamName string      `json:"paramName"`
	Value     interface{} `json:"value"`
}

var (
	ErrMethodIDNotEqual       = errors.New("method id not equal")
	ErrParamValueSizeNotEqual = errors.New("size of params and values not equal")
	ErrMethodStringInvalid    = errors.New("method string invalid")
)

// Select transaction from etherscan.io randomly and test
func TestDecode(t *testing.T) {
	test1(t)
	test2(t)
}

func test1(t *testing.T) {
	// Based on https://etherscan.io/tx/0x5f44b56ed2f21412fedcafac96fae4d7711caa14676c730296e0b16fb8d2d7c8

	// Function: transfer(address dst, uint256 wad)
	// MethodID: 0xa9059cbb
	// [0]:  0000000000000000000000001062a747393198f70f71ec65a582423dba7e5ab3
	// [1]:  000000000000000000000000000000000000000000000005010b44f3834b6000

	// inputData: 0xa9059cbb0000000000000000000000001062a747393198f70f71ec65a582423dba7e5ab3000000000000000000000000000000000000000000000005010b44f3834b6000
	inputData := "0xa9059cbb0000000000000000000000001062a747393198f70f71ec65a582423dba7e5ab3000000000000000000000000000000000000000000000005010b44f3834b6000"
	methodString := "transfer(address dst, uint256 wad)"

	res, err := DecodeInputData(inputData, methodString)
	if err != nil {
		t.Fatal(err)
	}

	// Print Out the Value
	resJsonByte, _ := json.Marshal(res)
	fmt.Println(string(resJsonByte))
}

func test2(t *testing.T) {
	// Based on https://etherscan.io/tx/0xac2892bfa9893dc8aa776fc3fc74d7fa6d23a77517be244adda66c5f91b2a9e5

	// Function: breedWithAuto(uint256 _matronId, uint256 _sireId)
	// MethodID: 0xf7d8c883
	// [0]:  00000000000000000000000000000000000000000000000000000000000a85f2
	// [1]:  00000000000000000000000000000000000000000000000000000000000a8daf

	// inputData: 0xf7d8c88300000000000000000000000000000000000000000000000000000000000a85f200000000000000000000000000000000000000000000000000000000000a8daf

	inputData := "0xf7d8c88300000000000000000000000000000000000000000000000000000000000a85f200000000000000000000000000000000000000000000000000000000000a8daf"
	methodString := "breedWithAuto(uint256 _matronId, uint256 _sireId)"

	res, err := DecodeInputData(inputData, methodString)
	if err != nil {
		t.Fatal(err)
	}

	// Print Out the Value
	resJsonByte, _ := json.Marshal(res)
	fmt.Println(string(resJsonByte))
}

// Decode inputData to original data list
// InputData: data need to decode
// methodString: the function of contract you called; like "transfer(address dst, uint256 wad)"
func DecodeInputData(inputData string, methodString string) (FunctionCall, error) {
	if strings.HasPrefix(inputData, "0x") {
		inputData = strings.TrimPrefix(inputData, "0x")
	}

	//
	methodName, methodID, typeNames, paramNames, _ := decodeFunctionString(methodString)
	if methodID != inputData[0:8] {
		return FunctionCall{}, ErrMethodIDNotEqual
	}

	hexValues, err := GetHexData(inputData[8:])
	if err != nil {
		return FunctionCall{}, err
	}

	// TODO add support for dynamic type
	typeValues := GetTypeValues(typeNames, hexValues)
	if len(paramNames) != len(typeValues) {
		return FunctionCall{}, ErrParamValueSizeNotEqual
	}

	var res FunctionCall
	res.Method = methodName
	for i := 0; i < len(paramNames); i++ {
		res.Params = append(res.Params, Param{typeNames[i], typeValues[i]})
	}
	return res, nil
}

func decodeFunctionString(methodString string) (string, string, []string, []string, error) {
	// Trim space of string
	methodString = strings.TrimSpace(methodString)

	start := strings.Index(methodString, "(")
	if start <= 0 {
		return "", "", []string{}, []string{}, ErrMethodStringInvalid
	}

	end := strings.Index(methodString, ")")
	if end != len(methodString)-1 {
		return "", "", []string{}, []string{}, ErrMethodStringInvalid
	}

	// Get method name and params' type
	methodName := methodString[0:start]
	typeStrings := methodString[start+1 : end]

	typeNames := make([]string, 0)
	paramNames := make([]string, 0)

	for _, typeStr := range strings.Split(typeStrings, ",") {
		typeStr = strings.TrimSpace(typeStr)
		typeEnd := strings.Index(typeStr, " ")
		typeName := typeStr[0:typeEnd]
		typeNames = append(typeNames, typeName)
		paramNames = append(paramNames, typeStr[typeEnd+1:])
	}

	toBeSigned := GetStrToBeSigned(methodName, typeNames)
	signed := GetSigned(toBeSigned)
	methodID := signed[0:8]

	return methodName, methodID, typeNames, paramNames, nil
}

func GetStrToBeSigned(methodName string, typeNames []string) string {
	var res string
	res = res + methodName + "("

	for _, typeName := range typeNames {
		res = res + typeName + ","
	}

	res = strings.TrimSuffix(res, ",")
	res = res + ")"
	return res
}

// Use keccak256 encode the method
func GetSigned(src string) string {

	hash := sha3.NewKeccak256()

	var buf []byte
	hash.Write([]byte(src))
	buf = hash.Sum(buf)

	return hex.EncodeToString(buf)
}

func GetTypeValues(typeNames []string, hexValue []string) []interface{} {
	res := make([]interface{}, 0)

	// index of type and value should be equal
	for i, typeName := range typeNames {
		value := GetTypeValue(typeName, hexValue[i])
		res = append(res, value)
	}

	return res
}

// TODO support for dynamic field
func GetTypeValue(typeName string, hexValue string) interface{} {

	switch {
	case "bool" == typeName:
		return reflect.Bool

	case strings.HasPrefix(typeName, "uint"):
		return ConvertToBigInt(hexValue)

	case "address" == typeName:
		return ConvertToAddress(hexValue)

		// TODO add support
	case strings.HasPrefix(typeName, "bytes"):

	case "string" == typeName:
		return ConvertToUTF8(hexValue)

	}

	return hexValue
}

func ConvertToBigInt(hexValue string) *big.Int {
	res := new(big.Int)
	res.SetString(hexValue, 16)
	return res
}

func ConvertToAddress(hexValue string) string {
	return "0x" + hexValue[24:]
}

func GetHexData(hexValues string) ([]string, error) {

	if len(hexValues)%64 != 0 || len(hexValues) == 0 {
		return []string{}, nil
	}

	res := make([]string, 0)

	paramLen := len(hexValues) / 64
	start := 0
	end := 64

	for i := 0; i < paramLen; i++ {
		res = append(res, hexValues[start:end])
		start += 64
		end += 64
	}

	return res, nil
}

func ConvertToUTF8(hexValue string) string {
	return hex.EncodeToString([]byte(hexValue))
}

// TODO
func ConvertToBool(hex string) bool {
	return true
}
