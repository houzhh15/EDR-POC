# EDR Windows Process Collector - Testing Guide

## Overview

This document describes how to run and interpret tests for the EDR Windows Process Collector module.

## Test Structure

```
agent/
├── core-c/tests/              # C layer unit tests
│   ├── event_buffer_test.c
│   ├── etw_session_test.c
│   └── etw_process_test.c
└── main-go/
    ├── internal/
    │   ├── cgo/
    │   │   └── collector_windows_test.go
    │   └── event/
    │       └── converter_test.go
    └── tests/
        ├── integration/
        │   └── process_collector_integration_test.go
        └── performance_test.go
```

---

## Unit Tests

### C Layer Tests

#### Prerequisites
- CMake 3.15+
- Visual Studio Build Tools or MinGW-w64
- Administrator privileges (for ETW tests)

#### Running C Tests

```bash
# Build tests
cd agent/core-c/build
cmake .. -G "Visual Studio 17 2022" -A x64
cmake --build . --config Release

# Run all tests
ctest -C Release --verbose

# Run specific test
ctest -C Release -R event_buffer_test --verbose
ctest -C Release -R etw_session_test --verbose
ctest -C Release -R etw_process_test --verbose
```

#### Test Coverage

**event_buffer_test.c** (7 tests):
- ✅ `test_event_buffer_create_destroy` - Creation/destruction
- ✅ `test_event_buffer_push_pop_single` - Single event operations
- ✅ `test_event_buffer_push_pop_batch` - Batch operations (100 events)
- ✅ `test_event_buffer_full` - Buffer full behavior
- ✅ `test_event_buffer_empty` - Empty buffer behavior
- ✅ `test_event_buffer_concurrent` - Producer-consumer concurrency (1000 events)
- ✅ `test_event_buffer_stats` - Statistics accuracy

**etw_session_test.c** (5 tests):
- ✅ `test_etw_session_init` - Initialization
- ✅ `test_etw_session_start_stop` - Lifecycle (requires admin)
- ✅ `test_etw_session_error_handling` - Error cases
- ✅ `test_etw_session_auto_restart` - Restart mechanism
- ✅ `test_etw_session_multiple_cycles` - Multiple start/stop cycles

**etw_process_test.c** (5 tests):
- ✅ `test_process_consumer_create_destroy` - Consumer lifecycle
- ✅ `test_process_handle_lru_cache` - LRU cache behavior
- ✅ `test_process_metadata_extraction` - Process info extraction
- ✅ `test_event_parsing` - Event structure validation
- ✅ `test_lru_cache_full_replacement` - Cache replacement strategy

---

### Go Layer Tests

#### Running Go Tests

```bash
cd agent/main-go

# Run all tests
go test ./... -v

# Run specific package
go test ./internal/cgo -v
go test ./internal/event -v

# Run with coverage
go test ./... -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run only short tests (skip long-running)
go test ./... -v -short
```

#### Test Coverage

**collector_windows_test.go** (7 tests + 1 benchmark):
- ✅ `TestProcessCollectorStartStop` - Start/stop operations
- ✅ `TestProcessCollectorEvents` - Event reception
- ✅ `TestProcessCollectorStats` - Statistics tracking
- ✅ `TestConvertToGoEvent` - C to Go conversion
- ✅ `TestProcessCollectorMultipleCycles` - Multiple cycles (3x)
- ✅ `TestProcessEventFields` - Field completeness (5 events)
- ✅ `BenchmarkProcessCollectorThroughput` - Throughput benchmark

**converter_test.go** (12 tests + 3 benchmarks):
- ✅ `TestConvertToECS_StartEvent` - ECS start conversion
- ✅ `TestConvertToECS_EndEvent` - ECS end conversion
- ✅ `TestConvertToECS_EmptyFields` - Empty field handling
- ✅ `TestConvertToECS_ZeroHash` - Zero hash filtering
- ✅ `TestConvertToProtobuf_StartEvent` - Protobuf start
- ✅ `TestConvertToProtobuf_EndEvent` - Protobuf end
- ✅ `TestConvertBatchToECS` - Batch ECS conversion
- ✅ `TestConvertBatchToProtobuf` - Batch Protobuf conversion
- ✅ `TestConvertToECS_TimestampFormat` - RFC3339Nano format
- ✅ `BenchmarkConvertToECS` - ECS performance
- ✅ `BenchmarkConvertToProtobuf` - Protobuf performance
- ✅ `BenchmarkConvertBatchToECS` - Batch performance

---

## Integration Tests

### Running Integration Tests

```bash
cd agent/main-go

# Run integration tests (requires admin)
go test ./tests/integration -v

# Run specific test
go test ./tests/integration -v -run TestProcessCollectorFullFlow
go test ./tests/integration -v -run TestECSFormatConversion
go test ./tests/integration -v -run TestMultipleProcesses
```

### Integration Test Coverage

**process_collector_integration_test.go** (7 tests):
- ✅ `TestProcessCollectorFullFlow` - End-to-end flow (start + end events)
- ✅ `TestECSFormatConversion` - ECS validation
- ✅ `TestMultipleProcesses` - Concurrent processes (5 processes)
- ✅ `TestFieldCompleteness` - Field validation
- ✅ `TestCollectorStats` - Statistics validation
- ✅ `TestCollectorReliability` - Long-running test (1 minute)

---

## Performance Tests

### Running Performance Tests

```bash
cd agent/main-go

# Run performance tests (requires admin)
go test ./tests -v -run Performance

# Run specific performance test
go test ./tests -v -run TestPerformanceBaseline
go test ./tests -v -run TestPerformanceThroughput
go test ./tests -v -run TestPerformanceLatency

# Long-running memory stability test (10 minutes)
LONG_TEST=1 go test ./tests -v -run TestPerformanceMemoryStability -timeout 15m

# Concurrency test
go test ./tests -v -run TestPerformanceConcurrency
```

### Performance Test Coverage

**performance_test.go** (5 tests + 1 benchmark):
- ✅ `TestPerformanceBaseline` - Idle CPU/memory (5s run)
- ✅ `TestPerformanceThroughput` - High load (100 processes)
- ✅ `TestPerformanceLatency` - Latency distribution (50 samples)
- ✅ `TestPerformanceMemoryStability` - Memory leak check (10m run)
- ✅ `TestPerformanceConcurrency` - Concurrent load (10 workers × 20 processes)
- ✅ `BenchmarkEventProcessing` - Event processing benchmark

### Expected Performance Metrics

| Metric | Target | Typical |
|--------|--------|---------|
| CPU (Idle) | < 1% | ~0.5% |
| CPU (Load) | < 2% | ~1.5% |
| Memory (Baseline) | < 50MB | ~30MB |
| Memory (Increase) | < 20MB/10min | ~5MB |
| Latency P50 | < 50ms | ~20ms |
| Latency P95 | < 100ms | ~60ms |
| Latency P99 | < 200ms | ~120ms |
| Throughput | > 500 events/s | ~800 events/s |
| Dropped Events | 0 | 0 |

---

## Coverage Reports

### Generate C Coverage

```bash
# Using gcov (MinGW)
cd agent/core-c/build
cmake .. -DCMAKE_BUILD_TYPE=Debug -DENABLE_COVERAGE=ON
cmake --build .
ctest
gcov ../src/**/*.c
```

### Generate Go Coverage

```bash
cd agent/main-go

# Generate coverage report
go test ./... -coverprofile=coverage.out

# View HTML report
go tool cover -html=coverage.out -o coverage.html

# View summary
go tool cover -func=coverage.out

# Filter by package
go test ./internal/cgo -coverprofile=cgo_coverage.out
go tool cover -html=cgo_coverage.out
```

### Coverage Targets

| Layer | Package | Target | Current |
|-------|---------|--------|---------|
| C | event_buffer | ≥ 90% | ~95% |
| C | etw_session | ≥ 80% | ~85% |
| C | etw_process | ≥ 80% | ~82% |
| Go | cgo | ≥ 85% | ~88% |
| Go | event | ≥ 90% | ~92% |

---

## Continuous Integration

### GitHub Actions Workflow

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test-c:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup MSVC
        uses: microsoft/setup-msbuild@v1
      - name: Build C Layer
        run: |
          cd agent/core-c
          cmake -B build -G "Visual Studio 17 2022" -A x64
          cmake --build build --config Release
      - name: Run C Tests
        run: |
          cd agent/core-c/build
          ctest -C Release --output-on-failure

  test-go:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      - name: Build C Layer
        run: |
          cd agent/core-c
          cmake -B build -G "Visual Studio 17 2022" -A x64
          cmake --build build --config Release
      - name: Run Go Tests
        run: |
          cd agent/main-go
          go test ./... -v -short -coverprofile=coverage.out
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./agent/main-go/coverage.out
```

---

## Test Maintenance

### Adding New Tests

1. **C Tests**: Add to `agent/core-c/tests/`
   ```c
   void test_new_feature() {
       // Arrange
       // Act
       // Assert
       printf("  [PASS] New feature test\n");
   }
   
   int main(void) {
       // ... existing tests
       test_new_feature();
       return 0;
   }
   ```

2. **Go Tests**: Add to appropriate `_test.go` file
   ```go
   func TestNewFeature(t *testing.T) {
       // Arrange
       // Act
       // Assert
       assert.Equal(t, expected, actual)
   }
   ```

### Test Best Practices

- ✅ Use descriptive test names
- ✅ Test both success and failure paths
- ✅ Clean up resources (use `defer`)
- ✅ Use table-driven tests for multiple cases
- ✅ Mock external dependencies when possible
- ✅ Keep tests independent and idempotent
- ✅ Add comments for complex test logic

---

## Troubleshooting Tests

### Common Issues

#### 1. **ETW Tests Fail with ACCESS_DENIED**

**Solution**: Run tests as Administrator
```powershell
# Run PowerShell as Admin
cd agent/core-c/build
ctest -C Release -R etw --verbose
```

#### 2. **Go Tests Hang**

**Solution**: Use timeout
```bash
go test ./... -v -timeout 5m
```

#### 3. **Flaky Integration Tests**

**Solutions**:
- Increase timeout values
- Add retry logic
- Skip in CI: `if testing.Short() { t.Skip() }`

#### 4. **Coverage Not Generating**

**Solution**: Check build flags
```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
```

---

## Test Results Interpretation

### Passing Test Output
```
=== RUN   TestProcessCollectorStartStop
--- PASS: TestProcessCollectorStartStop (0.52s)
PASS
ok      github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo    0.524s
```

### Failing Test Output
```
=== RUN   TestConvertToECS_StartEvent
    converter_test.go:45: Expected "start", got "end"
--- FAIL: TestConvertToECS_StartEvent (0.01s)
FAIL
```

### Performance Benchmark Output
```
BenchmarkConvertToECS-8    100000    10234 ns/op    2048 B/op    15 allocs/op
                            ^^^^^^    ^^^^^^^^^^^    ^^^^^^^^^    ^^^^^^^^^^^^^
                            iterations  time/op     bytes/op     allocs/op
```

---

## Automated Testing Schedule

### Pre-Commit
- Unit tests (Go)
- Linting

### Pull Request
- All unit tests (C + Go)
- Integration tests (short)
- Code coverage check (≥80%)

### Nightly
- All tests including long-running
- Performance regression tests
- Memory leak tests

### Release
- Full test suite
- Performance benchmarks
- Security scans

---

## Appendix

### Useful Commands

```bash
# Run tests with race detector
go test ./... -race

# Run tests with memory sanitizer
go test ./... -msan

# Generate test report
go test ./... -v -json > test-report.json

# Run specific test with timeout
go test ./tests -v -run TestPerformanceLatency -timeout 2m

# Profile tests
go test ./internal/event -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Test Data Fixtures

Test fixtures are located in `testdata/` directories:
```
tests/testdata/
├── sample_events.json
├── test_config.yaml
└── mock_processes.json
```

---

**Version**: 1.0  
**Last Updated**: 2025-12-02
