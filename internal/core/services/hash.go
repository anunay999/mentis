package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type HashService struct{}

func NewHashService() *HashService {
	return &HashService{}
}

func (s *HashService) ComputeContentHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func (s *HashService) ComputeInputHash(input interface{}) string {
	data, err := json.Marshal(input)
	if err != nil {
		// Fallback to string representation
		data = []byte(fmt.Sprintf("%v", input))
	}
	return s.ComputeContentHash(data)
}