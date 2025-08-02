package contract

import "errors"

var (
	ErrEmptyIntent      = errors.New("intent is empty")
	ErrPlanGeneration   = errors.New("failed to generate execution plan")
	ErrExecutionFailure = errors.New("execution failed")
	ErrInvalidPlan      = errors.New("execution plan invalid")
	ErrFeedbackConflict = errors.New("feedback causes conflict in plan")
)
