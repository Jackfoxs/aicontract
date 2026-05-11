package contract_api

import (
	"backend/knowledge"
)

// ContractAPI 合同审核API
type ContractAPI struct {
	ContractService *knowledge.ContractService
}

// NewContractAPI 创建合同审核API实例
func NewContractAPI() *ContractAPI {
	return &ContractAPI{
		ContractService: knowledge.NewContractService(),
	}
}
