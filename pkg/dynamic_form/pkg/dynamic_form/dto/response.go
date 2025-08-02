package dto

// FieldError 细粒度字段错误
type FieldError struct {
	Field  string   `json:"field"`
	Errors []string `json:"errors"`
}

// ValidateFormResponse 验证结果
type ValidateFormResponse struct {
	ValidatedInputs map[string]interface{} `json:"validated_inputs"`
	FieldErrors     map[string][]string    `json:"field_errors"`
	VisibleFields   []string               `json:"visible_fields"`
	EnabledFields   []string               `json:"enabled_fields"`
	Variables       map[string]interface{} `json:"variables,omitempty"`
}

// SubmitFormResponse 提交后返回（可以扩展包含 submission ID 等）
type SubmitFormResponse struct {
	ValidateFormResponse
	SubmissionID string `json:"submission_id,omitempty"`
}
