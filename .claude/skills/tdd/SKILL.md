---
name: tdd
description: Test-driven development with red-green-refactor loop. Use when user says /tdd or asks to implement with TDD.
---

# Test-Driven Development

Implement features using RED → GREEN → REFACTOR.

## Philosophy
- Tests verify behavior through public interfaces
- One behavior at a time — not all tests first

## Anti-Pattern

WRONG: test1, test2, test3 → impl1, impl2, impl3
RIGHT: test1 → impl1 → test2 → impl2 → test3 → impl3

## Process

### 1. Identify task
- /tdd #44 → fetch gh issue view 44
- /tdd (no number) → list issues and ask

### 2. Plan first

Output before coding:

**Task:** [summary]
**Approach:** [steps]
**Tricky parts:** [gotchas]
**Files:** [what to touch]
**First test:** [RED test]

Ask: "Does this plan look right?" Wait for approval.

### 3. TDD Loop

For EACH behavior:

RED: Write ONE failing test, run → FAIL
GREEN: Write MINIMAL code, run → PASS

Repeat.

### 4. Refactor
After all pass: simplify, extract duplication, run tests after each change.

## Go Patterns

### Test Naming
Test[Function]_[Scenario]_[Expected]

### Table-Driven Tests
func TestCreateStore_Validation(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateStoreInput
        wantErr bool
    }{
        {"valid", CreateStoreInput{Name: "Shop"}, false},
        {"empty name", CreateStoreInput{Name: ""}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewService(mockRepo)
            _, err := svc.Create(ctx, tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

## Commands

make test
go test ./... -v
go test ./path/to/package -run TestName -v

## Commit Pattern

After completing the issue:

git commit -m "feat(scope): description (#issue-number)"

Examples:
feat(stores): add migration and model (#2)
feat(stores): POST /stores endpoint (#3)
fix(stores): handle duplicate name error (#3)
