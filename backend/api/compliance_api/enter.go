package compliance_api

import "backend/compliance"

// API 包装
type API struct {
	Service *compliance.Service
}
