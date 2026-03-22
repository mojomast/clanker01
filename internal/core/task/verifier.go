package task

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type Verifier struct {
	schemaValidator *SchemaValidator
	customCheckers  map[string]CustomChecker
}

type CustomChecker func(ctx context.Context, task *Task) error

type VerificationResult struct {
	TaskID     TaskID            `json:"task_id"`
	Valid      bool              `json:"valid"`
	Assertions []AssertionResult `json:"assertions"`
	Failures   []AssertionResult `json:"failures"`
	CheckedAt  time.Time         `json:"checked_at"`
	Error      string            `json:"error,omitempty"`
}

type AssertionResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Actual   any    `json:"actual,omitempty"`
	Expected any    `json:"expected"`
	Message  string `json:"message"`
}

func NewVerifier(schemaValidator *SchemaValidator, customCheckers map[string]CustomChecker) *Verifier {
	if schemaValidator == nil {
		schemaValidator = &SchemaValidator{}
	}
	if customCheckers == nil {
		customCheckers = make(map[string]CustomChecker)
	}
	return &Verifier{
		schemaValidator: schemaValidator,
		customCheckers:  customCheckers,
	}
}

func (v *Verifier) Verify(ctx context.Context, task *Task) *VerificationResult {
	if task == nil {
		return &VerificationResult{
			CheckedAt: time.Now(),
			Valid:     false,
			Error:     "task is nil",
		}
	}

	result := &VerificationResult{
		TaskID:    task.ID,
		CheckedAt: time.Now(),
		Valid:     true,
	}

	if task.Verification == nil {
		return result
	}

	for _, assertion := range task.Verification.Assertions {
		checkResult := v.checkAssertion(ctx, task, assertion)
		result.Assertions = append(result.Assertions, checkResult)
		if !checkResult.Passed {
			result.Valid = false
			result.Failures = append(result.Failures, checkResult)
		}
	}

	return result
}

func (v *Verifier) checkAssertion(ctx context.Context, task *Task, a Assertion) AssertionResult {
	result := AssertionResult{
		Name:     a.Name,
		Expected: a.Expected,
	}

	if task.Output == nil {
		result.Passed = false
		result.Message = "task output is nil"
		return result
	}

	actual := getJSONPath(task.Output, a.Path)
	result.Actual = actual

	switch a.Type {
	case "equals":
		result.Passed = deepEqual(actual, a.Expected)
	case "not_equals":
		result.Passed = !deepEqual(actual, a.Expected)
	case "contains":
		result.Passed = contains(actual, a.Expected)
	case "greater_than":
		result.Passed = compareNumbers(actual, a.Expected) > 0
	case "less_than":
		result.Passed = compareNumbers(actual, a.Expected) < 0
	case "regex":
		matched, _ := regexp.MatchString(fmt.Sprint(a.Expected), fmt.Sprint(actual))
		result.Passed = matched
	case "schema":
		schemaMap, ok := a.Expected.(map[string]any)
		if !ok {
			result.Passed = false
			result.Message = fmt.Sprintf("schema assertion expected map[string]any, got %T", a.Expected)
		} else {
			err := v.schemaValidator.Validate(schemaMap, actual)
			result.Passed = err == nil
			if err != nil {
				result.Message = err.Error()
			}
		}
	case "custom":
		checker := v.customCheckers[a.Name]
		if checker != nil {
			err := checker(ctx, task)
			result.Passed = err == nil
			if err != nil {
				result.Message = err.Error()
			}
		} else {
			result.Passed = false
			result.Message = fmt.Sprintf("custom checker %s not found", a.Name)
		}
	default:
		result.Passed = false
		result.Message = fmt.Sprintf("unknown assertion type: %s", a.Type)
	}

	if !result.Passed && result.Message == "" {
		result.Message = fmt.Sprintf("%s: expected %v, got %v", a.Path, a.Expected, actual)
	}

	return result
}

func getJSONPath(obj map[string]any, path string) any {
	if path == "" || path == "." {
		return obj
	}

	keys := parsePath(path)
	current := any(obj)

	for _, key := range keys {
		switch v := current.(type) {
		case map[string]any:
			current = v[key]
		case []any:
			index := 0
			fmt.Sscanf(key, "[%d]", &index)
			if index < len(v) {
				current = v[index]
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

func parsePath(path string) []string {
	parts := []string{}
	current := ""
	for _, r := range path {
		if r == '.' || r == '[' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			if r == '[' {
				current += string(r)
			}
		} else if r == ']' {
			current += string(r)
			parts = append(parts, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func deepEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

func contains(container, item any) bool {
	switch v := container.(type) {
	case []any:
		for _, i := range v {
			if deepEqual(i, item) {
				return true
			}
		}
	case string:
		strItem, ok := item.(string)
		if ok {
			return strings.Contains(v, strItem)
		}
	case map[string]any:
		if subMap, ok := item.(map[string]any); ok {
			for k, val := range subMap {
				if !deepEqual(v[k], val) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func compareNumbers(a, b any) int {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)

	if aFloat < bFloat {
		return -1
	} else if aFloat > bFloat {
		return 1
	}
	return 0
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	default:
		return 0
	}
}

type SchemaValidator struct{}

func (s *SchemaValidator) Validate(schema map[string]any, data any) error {
	schemaType, _ := schema["type"].(string)

	switch schemaType {
	case "object":
		return s.validateObject(schema, data)
	case "array":
		return s.validateArray(schema, data)
	case "string":
		return s.validateString(schema, data)
	case "number", "integer":
		return s.validateNumber(schema, data)
	case "boolean":
		return s.validateBoolean(schema, data)
	default:
		return nil
	}
}

func (s *SchemaValidator) validateObject(schema map[string]any, data any) error {
	obj, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("expected object, got %T", data)
	}

	required, _ := schema["required"].([]any)
	properties, _ := schema["properties"].(map[string]any)

	for _, r := range required {
		req, ok := r.(string)
		if !ok {
			continue
		}
		if _, exists := obj[req]; !exists {
			return fmt.Errorf("missing required field: %s", req)
		}
	}

	for key, propSchema := range properties {
		if val, exists := obj[key]; exists {
			propSchemaMap, ok := propSchema.(map[string]any)
			if !ok {
				continue
			}
			if err := s.Validate(propSchemaMap, val); err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
		}
	}

	return nil
}

func (s *SchemaValidator) validateArray(schema map[string]any, data any) error {
	arr, ok := data.([]any)
	if !ok {
		return fmt.Errorf("expected array, got %T", data)
	}

	var minItems float64
	switch v := schema["minItems"].(type) {
	case float64:
		minItems = v
	case int:
		minItems = float64(v)
	}

	if minItems > 0 && float64(len(arr)) < minItems {
		return fmt.Errorf("array must have at least %d items", int(minItems))
	}

	itemsSchema, _ := schema["items"].(map[string]any)
	if itemsSchema != nil {
		for _, item := range arr {
			if err := s.Validate(itemsSchema, item); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SchemaValidator) validateString(schema map[string]any, data any) error {
	str, ok := data.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", data)
	}

	var minLength float64
	switch v := schema["minLength"].(type) {
	case float64:
		minLength = v
	case int:
		minLength = float64(v)
	}

	if minLength > 0 && float64(len(str)) < minLength {
		return fmt.Errorf("string must be at least %d characters", int(minLength))
	}

	return nil
}

func (s *SchemaValidator) validateNumber(schema map[string]any, data any) error {
	switch data.(type) {
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return nil
	default:
		return fmt.Errorf("expected number, got %T", data)
	}
}

func (s *SchemaValidator) validateBoolean(schema map[string]any, data any) error {
	_, ok := data.(bool)
	if !ok {
		return fmt.Errorf("expected boolean, got %T", data)
	}
	return nil
}

func (v *Verifier) VerifyOutput(task *Task) error {
	if task.Output == nil {
		return fmt.Errorf("task output is nil")
	}

	if task.Verification != nil {
		for _, assertion := range task.Verification.Assertions {
			result := v.checkAssertion(context.Background(), task, assertion)
			if !result.Passed {
				return fmt.Errorf("assertion failed: %s", result.Message)
			}
		}
	}

	return nil
}

func (v *Verifier) VerifyWithSchema(task *Task, schema map[string]any) error {
	if task.Output == nil {
		return fmt.Errorf("task output is nil")
	}

	return v.schemaValidator.Validate(schema, task.Output)
}

func (v *Verifier) ValidateJSON(task *Task) error {
	if task.Output == nil {
		return fmt.Errorf("task output is nil")
	}

	_, err := json.Marshal(task.Output)
	if err != nil {
		return fmt.Errorf("output is not valid JSON: %w", err)
	}

	return nil
}
