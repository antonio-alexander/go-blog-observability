package authz

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

func getChecksum(item any) (string, error) {
	bytes, err := json.Marshal(item)
	if err != nil {
		return "", err
	}
	hashBytes := md5.Sum([]byte(bytes))
	return hex.EncodeToString(hashBytes[:]), nil
}
