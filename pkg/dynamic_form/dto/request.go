package dto

// ValidateFormRequest 用于表单验证 API
type ValidateFormRequest struct {
	FormID string                 `json:"form_id"`
	Input  map[string]interface{} `json:"input"`
}

// SubmitFormRequest 用于表单提交 API
type SubmitFormRequest struct {
	FormID string                 `json:"form_id"`
	Input  map[string]interface{} `json:"input"`
	// 可选：是否触发 side-effect，比如直接调用某个 tool
	Trigger string `json:"trigger,omitempty"`
}
