# Table-Driven Test Standard

All unit tests must use the table-driven pattern with `t.Run` subtests. This ensures consistent test structure, easy extensibility, and compatibility with mutation testing.

## Standard Pattern

```go
func TestFunctionName_Scenario(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
        wantErr error
    }{
        {
            name:    "happy path",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        {
            name:    "edge case empty input",
            input:   emptyInput,
            want:    zeroValue,
            wantErr: false,
        },
        {
            name:    "error invalid input",
            input:   invalidInput,
            wantErr: true,
            wantErr: types.ErrInvalidArgument,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionUnderTest(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                if tt.wantErr != nil {
                    assert.ErrorIs(t, err, tt.wantErr)
                }
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Rules

1. **Test function naming**: `Test<Function>_<Category>` (e.g., `TestValidateDAG_Diamond`)
2. **Table structure**: Named `tests` with anonymous struct. Include at minimum `name`, input fields, and expected output/error fields
3. **Subtests**: Every case runs via `t.Run(tt.name, ...)`
4. **Assertions**: Use `testify/require` for fatal assertions, `testify/assert` for non-fatal
5. **Error cases**: Always test error paths — nil input, invalid input, boundary conditions
6. **Happy path first**: Place the primary success case as the first entry
7. **Mutation score**: All table-driven tests must achieve >= 60% mutation score in their package

## When NOT to Use Table-Driven

- Integration tests using `testify/suite.Suite` with container setup/teardown
- Tests that require per-case setup that cannot be expressed in a struct
- Generated test files

## Mutation Testing

Mutation testing measures test quality by injecting bugs (mutations) into source code and checking if tests catch them. The survival rate is the percentage of mutations that tests successfully detect and kill.

**Threshold**: Packages must achieve >= 60% mutation score. CI will enforce this via `gremlins`.

**Running mutation tests**:
```bash
go tool task test:mutation        # Full suite with threshold check
go tool task test:mutation:pkg    # pkg/ only
go tool task test:mutation:score  # Score-only, no threshold enforcement
go tool task test:mutation:report # Generate JSON report
```

**Understanding results**:
- **KILLED**: Mutant was detected by tests — good
- **LIVED**: Mutant survived all tests — needs attention
- **NOT COVERED**: Line has no test coverage at all
- **TIMED OUT**: Test took too long running against the mutant
- **NOT VIABLE**: Mutant could not be compiled

**CI integration**: The `testing.yml` workflow runs `go tool gremlins unleash --threshold-efficacy=0.60 --threshold-mcover=0.60 ./pkg/...` after unit tests. PRs that fall below the threshold will fail CI.
