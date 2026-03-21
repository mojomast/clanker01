package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewVerifier(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	assert.NotNil(t, verifier)
	assert.NotNil(t, verifier.schemaValidator)
	assert.NotNil(t, verifier.customCheckers)
}

func TestVerifier_Verify_NoVerificationSpec(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:      "task-1",
		Name:    "Task 1",
		Status:  StatusCompleted,
		Output:  map[string]any{"result": "success"},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Assertions)
}

func TestVerifier_Verify_Equals(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "success"},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check result",
					Type:     "equals",
					Path:     "result",
					Expected: "success",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
	assert.Len(t, result.Assertions, 1)
	assert.True(t, result.Assertions[0].Passed)
}

func TestVerifier_Verify_Equals_Fail(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "failure"},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check result",
					Type:     "equals",
					Path:     "result",
					Expected: "success",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.False(t, result.Valid)
	assert.Len(t, result.Assertions, 1)
	assert.False(t, result.Assertions[0].Passed)
}

func TestVerifier_Verify_NotEquals(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "success"},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check result not failure",
					Type:     "not_equals",
					Path:     "result",
					Expected: "failure",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_Contains(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"items": []any{"a", "b", "c"}},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check contains",
					Type:     "contains",
					Path:     "items",
					Expected: "b",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_GreaterThan(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"count": 10},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check greater than",
					Type:     "greater_than",
					Path:     "count",
					Expected: 5,
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_LessThan(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"count": 3},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check less than",
					Type:     "less_than",
					Path:     "count",
					Expected: 5,
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_Regex(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"email": "test@example.com"},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check email format",
					Type:     "regex",
					Path:     "email",
					Expected: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_Schema(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{
			"name":  "John",
			"age":   30,
			"email": "john@example.com",
		},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name: "Check schema",
					Type: "schema",
					Path: "",
					Expected: map[string]any{
						"type":     "object",
						"required": []any{"name", "age"},
						"properties": map[string]any{
							"name":  map[string]any{"type": "string"},
							"age":   map[string]any{"type": "number"},
							"email": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_Schema_MissingRequired(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{
			"name": "John",
		},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name: "Check schema",
					Type: "schema",
					Path: "",
					Expected: map[string]any{
						"type":     "object",
						"required": []any{"name", "age"},
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
							"age":  map[string]any{"type": "number"},
						},
					},
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Failures[0].Message, "missing required field")
}

func TestVerifier_Verify_Custom(t *testing.T) {
	customCheckers := map[string]CustomChecker{
		"custom_check": func(ctx context.Context, task *Task) error {
			if task.Output == nil {
				return assert.AnError
			}
			return nil
		},
	}

	verifier := NewVerifier(nil, customCheckers)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "success"},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name: "custom_check",
					Type: "custom",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
}

func TestVerifier_Verify_Custom_Fail(t *testing.T) {
	customCheckers := map[string]CustomChecker{
		"custom_check": func(ctx context.Context, task *Task) error {
			return assert.AnError
		},
	}

	verifier := NewVerifier(nil, customCheckers)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "success"},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name: "Custom check",
					Type: "custom",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.False(t, result.Valid)
	assert.Len(t, result.Failures, 1)
}

func TestVerifier_Verify_MultipleAssertions(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{
			"count": 10,
			"name":  "test",
		},
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check count",
					Type:     "greater_than",
					Path:     "count",
					Expected: 5,
				},
				{
					Name:     "Check name",
					Type:     "equals",
					Path:     "name",
					Expected: "test",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.True(t, result.Valid)
	assert.Len(t, result.Assertions, 2)
}

func TestVerifier_Verify_NilOutput(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Verification: &VerificationSpec{
			Assertions: []Assertion{
				{
					Name:     "Check result",
					Type:     "equals",
					Path:     "result",
					Expected: "success",
				},
			},
		},
		Timeout: time.Minute,
	}

	result := verifier.Verify(context.Background(), task)
	assert.False(t, result.Valid)
}

func TestVerifier_VerifyOutput(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "success"},
	}

	err := verifier.VerifyOutput(task)
	assert.NoError(t, err)
}

func TestVerifier_VerifyOutput_NilOutput(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
	}

	err := verifier.VerifyOutput(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestVerifier_ValidateJSON(t *testing.T) {
	verifier := NewVerifier(nil, nil)

	task := &Task{
		ID:     "task-1",
		Name:   "Task 1",
		Status: StatusCompleted,
		Output: map[string]any{"result": "success", "count": 10},
	}

	err := verifier.ValidateJSON(task)
	assert.NoError(t, err)
}

func TestSchemaValidator_ValidateObject(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type":     "object",
		"required": []any{"name", "age"},
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
	}

	data := map[string]any{
		"name": "John",
		"age":  30,
	}

	err := validator.Validate(schema, data)
	assert.NoError(t, err)
}

func TestSchemaValidator_ValidateObject_InvalidType(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type": "object",
	}

	data := "not an object"

	err := validator.Validate(schema, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected object")
}

func TestSchemaValidator_ValidateArray(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type":     "array",
		"minItems": 2,
	}

	data := []any{1, 2, 3}

	err := validator.Validate(schema, data)
	assert.NoError(t, err)
}

func TestSchemaValidator_ValidateArray_TooFew(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type":     "array",
		"minItems": 5,
	}

	data := []any{1, 2}

	err := validator.Validate(schema, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 5 items")
}

func TestSchemaValidator_ValidateString(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type":      "string",
		"minLength": 3,
	}

	data := "test"

	err := validator.Validate(schema, data)
	assert.NoError(t, err)
}

func TestSchemaValidator_ValidateString_TooShort(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type":      "string",
		"minLength": 10,
	}

	data := "test"

	err := validator.Validate(schema, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 10 characters")
}

func TestSchemaValidator_ValidateNumber(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type": "number",
	}

	data := 42.5

	err := validator.Validate(schema, data)
	assert.NoError(t, err)
}

func TestSchemaValidator_ValidateNumber_InvalidType(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type": "number",
	}

	data := "not a number"

	err := validator.Validate(schema, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected number")
}

func TestSchemaValidator_ValidateBoolean(t *testing.T) {
	validator := &SchemaValidator{}

	schema := map[string]any{
		"type": "boolean",
	}

	data := true

	err := validator.Validate(schema, data)
	assert.NoError(t, err)
}

func TestGetJSONPath(t *testing.T) {
	obj := map[string]any{
		"user": map[string]any{
			"name": "John",
			"age":  30,
		},
	}

	result := getJSONPath(obj, "user.name")
	assert.Equal(t, "John", result)

	result = getJSONPath(obj, "user.age")
	assert.Equal(t, 30, result)
}

func TestParsePath(t *testing.T) {
	parts := parsePath("user.profile.name")
	assert.Equal(t, []string{"user", "profile", "name"}, parts)

	parts = parsePath("items[0].name")
	assert.Equal(t, []string{"items", "[0]", "name"}, parts)
}

func TestDeepEqual(t *testing.T) {
	assert.True(t, deepEqual("test", "test"))
	assert.True(t, deepEqual(123, 123))
	assert.True(t, deepEqual([]int{1, 2, 3}, []int{1, 2, 3}))
	assert.False(t, deepEqual("test", "other"))
}

func TestCompareNumbers(t *testing.T) {
	assert.Equal(t, -1, compareNumbers(1, 2))
	assert.Equal(t, 0, compareNumbers(5, 5))
	assert.Equal(t, 1, compareNumbers(10, 5))
}
