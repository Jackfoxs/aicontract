package compliance_api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

func getUserID(ctx *app.RequestContext) uint64 {
	if v, ok := ctx.Get("user_id"); ok {
		switch val := v.(type) {
		case uint64:
			return val
		case int:
			if val >= 0 {
				return uint64(val)
			}
		case int64:
			if val >= 0 {
				return uint64(val)
			}
		case string:
			if id, err := strconv.ParseUint(val, 10, 64); err == nil {
				return id
			}
		}
	}
	return 0
}

func parseQueryInt(ctx *app.RequestContext, key string, def int) int {
	value := ctx.Query(key)
	if len(value) == 0 {
		return def
	}
	if n, err := strconv.Atoi(string(value)); err == nil {
		return n
	}
	return def
}

func parseParamUint64(ctx *app.RequestContext, key string) (uint64, error) {
	value := ctx.Param(key)
	if len(value) == 0 {
		return 0, fmt.Errorf("missing param %s", key)
	}
	return strconv.ParseUint(string(value), 10, 64)
}

func parseStringID(field, value string) (uint64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("%s 不能为空", field)
	}
	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s 格式错误", field)
	}
	return id, nil
}

func parseOptionalStringID(value string) (uint64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	return strconv.ParseUint(trimmed, 10, 64)
}

func parseStringIDSlice(values []string) ([]uint64, error) {
	if len(values) == 0 {
		return []uint64{}, nil
	}
	result := make([]uint64, 0, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		id, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("selected_rule_ids 包含无效值: %s", raw)
		}
		result = append(result, id)
	}
	return result, nil
}

func formatOptionalID(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
}
