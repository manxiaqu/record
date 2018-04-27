package ethereum

import (
    "reflect"
    "strings"
    "testing"
    "fmt"
    "encoding/hex"
    "math/big"
    
    "github.com/ethereum/go-ethereum/crypto/sha3"
)
//TODO Add tools to convert inputdata to original data

func TestDecode(t *testing.T) {
    inputData := "0xa9059cbb0000000000000000000000001062a747393198f70f71ec65a582423dba7e5ab3000000000000000000000000000000000000000000000005010b44f3834b6000"
    methodString := "transfer(address dst, uint256 wad)"
    
    values, _ := DecodeInputData(inputData, methodString)
    fmt.Println(values)
}

// Decode inputData to original data list
// InputData: data need to decode
// methodString: the function of contract you called; like "transfer(address dst, uint256 wad)"
func DecodeInputData(inputData string, methodString string) ([]interface{}, error) {
    if strings.HasPrefix(inputData, "0x") {
        inputData = strings.TrimPrefix(inputData, "0x")
    }
    
    //
    methodID, typeNames, _ := decodeFunctionString(methodString)
    if methodID != inputData[0:8] {
        return nil, nil
    }
    
    //
    hexValues, _ := GetHexData(inputData[8:])
    
    // TODO add support for dynamic type
    typeValues := GetTypeValues(typeNames, hexValues)
    return typeValues, nil
}

func decodeFunctionString(methodString string) (string, []string, error) {
    // Trim space of string
    methodString = strings.TrimSpace(methodString)
    
    start := strings.Index(methodString, "(")
    if start <= 0 {
        return "", []string{}, nil
    }
    
    end := strings.Index(methodString, ")")
    if end != len(methodString) - 1 {
        return "", []string{}, nil
    }
    
    // Get method name and params' type
    methodName := methodString[0 : start]
    typeStrings := methodString[start + 1 : end]
    
    typeNames := make([]string, 0)
    
    for _, typeStr := range strings.Split(typeStrings, ",") {
        typeStr = strings.TrimSpace(typeStr)
        typeEnd := strings.Index(typeStr, " ")
        typeName := typeStr[0: typeEnd]
        typeNames = append(typeNames, typeName)
    }
    
    toBeSigned := GetStrToBeSigned(methodName, typeNames)
    signed := GetSigned(toBeSigned)
    fmt.Println(signed)
    methodID := signed[0:8]
    fmt.Println(methodID)
    
    
    return methodID, typeNames, nil
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


//
func GetTypeValues(typeNames []string, hexValue []string) []interface{} {
    res := make([]interface{}, 0)
    
    // index of type and value should be equal
    for i, typeName := range typeNames {
        value := GetTypeValue(typeName, hexValue[i])
        res = append(res, value)
    }
    
    return res
}

func GetTypeValue(typeName string, hexValue string) (interface{}) {
    //
    switch {
    case "bool" == typeName:
        return reflect.Bool
        
    case strings.HasPrefix(typeName, "uint"):
        return ConvertToBigInt(hexValue)

    case "address" == typeName:
        return ConvertToAddress(hexValue)
    }
    
    return hexValue
}

//
func ConvertToBigInt(hexValue string) *big.Int {
    //
    value, _ := hex.DecodeString(hexValue)
    // TODO Change hex to uint256
    fmt.Println(string(value))
    return nil
}


func ConvertToAddress(hexValue string) string {
    return "0x" + hexValue[24:]
}


func GetHexData(hexValues string) ([]string, error) {
    
    if len(hexValues) % 64 != 0 || len(hexValues) == 0{
        return []string{}, nil
    }
    
    res := make([]string, 0)
    
    paramLen := len(hexValues) / 64
    start := 0
    end := 64
    
    for i := 0; i < paramLen; i++ {
        res = append(res, hexValues[start : end])
        start += 64
        end += 64
    }
    
    return res, nil
}

