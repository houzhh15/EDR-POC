# EDR Windows Process Collector API Documentation

## Overview

This document describes the API interfaces for the Windows ETW Process Event Collector module.

## C Layer API

### Core Functions

#### `edr_core_init()`
```c
edr_error_t edr_core_init(void);
```
Initialize the EDR core library. Must be called before any other operations.

**Returns:**
- `EDR_OK` (0): Success
- `EDR_ERR_ALREADY_INITIALIZED` (-5): Already initialized
- `EDR_ERR_NO_MEMORY` (-3): Memory allocation failed

**Example:**
```c
edr_error_t result = edr_core_init();
if (result != EDR_OK) {
    fprintf(stderr, "Failed to initialize: %s\n", edr_error_string(result));
    return -1;
}
```

---

#### `edr_start_process_collector()`
```c
int edr_start_process_collector(void** out_handle);
```
Start the Windows ETW process event collector.

**Parameters:**
- `out_handle`: Pointer to receive the collector session handle

**Returns:**
- `EDR_SUCCESS` (0): Success
- `EDR_ERR_NOT_INITIALIZED`: Core library not initialized
- `EDR_ERR_INVALID_PARAM`: Invalid parameter
- `EDR_ERR_NO_MEMORY`: Memory allocation failed
- `EDR_ERROR_ETW_CREATE_FAILED`: ETW session creation failed
- `EDR_ERROR_ETW_ACCESS_DENIED`: Requires administrator privileges

**Example:**
```c
void* handle = NULL;
int result = edr_start_process_collector(&handle);
if (result != EDR_SUCCESS) {
    fprintf(stderr, "Failed to start collector: %d\n", result);
    return -1;
}
```

---

#### `edr_poll_process_events()`
```c
int edr_poll_process_events(
    void* handle,
    edr_process_event_t* events,
    int max_count,
    int* out_count
);
```
Poll process events from the collector (batch retrieval).

**Parameters:**
- `handle`: Collector session handle
- `events`: Array to receive events (caller allocated)
- `max_count`: Maximum number of events to retrieve
- `out_count`: Pointer to receive actual event count

**Returns:**
- `EDR_SUCCESS` (0): Success
- `EDR_ERR_INVALID_PARAM`: Invalid parameter

**Example:**
```c
#define BATCH_SIZE 100
edr_process_event_t events[BATCH_SIZE];
int count = 0;

int result = edr_poll_process_events(handle, events, BATCH_SIZE, &count);
if (result == EDR_SUCCESS) {
    printf("Polled %d events\n", count);
    for (int i = 0; i < count; i++) {
        printf("Event: PID=%d, Type=%d, Name=%s\n",
               events[i].pid, events[i].event_type, events[i].process_name);
    }
}
```

---

#### `edr_stop_process_collector()`
```c
int edr_stop_process_collector(void* handle);
```
Stop the process event collector and release resources.

**Parameters:**
- `handle`: Collector session handle

**Returns:**
- `EDR_SUCCESS` (0): Success
- `EDR_ERR_INVALID_PARAM`: Invalid handle

**Example:**
```c
int result = edr_stop_process_collector(handle);
if (result != EDR_SUCCESS) {
    fprintf(stderr, "Failed to stop collector: %d\n", result);
}
```

---

#### `edr_core_cleanup()`
```c
void edr_core_cleanup(void);
```
Clean up and shutdown the EDR core library.

**Example:**
```c
edr_core_cleanup();
```

---

### Data Structures

#### `edr_process_event_t`
```c
typedef struct {
    uint64_t timestamp;                   // Event timestamp (UTC, 100ns units)
    uint32_t pid;                         // Process ID
    uint32_t ppid;                        // Parent process ID
    char process_name[256];               // Process name
    char executable_path[MAX_PATH];       // Full executable path
    char command_line[4096];              // Command line arguments
    char username[128];                   // Username
    uint8_t sha256[32];                   // SHA256 hash (binary)
    edr_process_event_type_t event_type;  // Event type (START/END)
    int32_t exit_code;                    // Exit code (END events only)
    uint32_t reserved[4];                 // Reserved for future use
} edr_process_event_t;
```

#### `edr_process_event_type_t`
```c
typedef enum {
    EDR_PROCESS_START = 1,  // Process creation event
    EDR_PROCESS_END = 2     // Process termination event
} edr_process_event_type_t;
```

---

## Go Layer API

### Types

#### `ProcessCollector`
```go
type ProcessCollector struct {
    // Internal fields (not exported)
}
```

#### `ProcessEvent`
```go
type ProcessEvent struct {
    Timestamp          time.Time  // Event timestamp
    EventType          string     // "start" or "end"
    ProcessPID         uint32     // Process ID
    ProcessPPID        uint32     // Parent process ID
    ProcessName        string     // Process name
    ProcessPath        string     // Full executable path
    ProcessCommandLine string     // Command line
    ProcessHash        string     // SHA256 hash (hex string)
    UserName           string     // Username
    UserDomain         string     // User domain
    ExitCode           *int32     // Exit code (end events only)
}
```

#### `CollectorStats`
```go
type CollectorStats struct {
    TotalEventsCollected uint64    // Total events collected from C layer
    TotalEventsProcessed uint64    // Total events sent to channel
    TotalEventsDropped   uint64    // Events dropped (channel full)
    LastPollTime         time.Time // Last poll timestamp
}
```

---

### Functions

#### `StartProcessCollector()`
```go
func StartProcessCollector() (*ProcessCollector, error)
```
Start the process event collector.

**Returns:**
- `*ProcessCollector`: Collector instance
- `error`: Error if startup fails

**Example:**
```go
collector, err := cgo.StartProcessCollector()
if err != nil {
    log.Fatalf("Failed to start collector: %v", err)
}
defer collector.StopProcessCollector()
```

---

#### `(*ProcessCollector) Events()`
```go
func (pc *ProcessCollector) Events() <-chan *ProcessEvent
```
Get the read-only event channel.

**Returns:**
- `<-chan *ProcessEvent`: Event channel (capacity 1000)

**Example:**
```go
for event := range collector.Events() {
    fmt.Printf("Event: %s PID=%d Name=%s\n",
        event.EventType, event.ProcessPID, event.ProcessName)
}
```

---

#### `(*ProcessCollector) GetStats()`
```go
func (pc *ProcessCollector) GetStats() CollectorStats
```
Get collector statistics.

**Returns:**
- `CollectorStats`: Current statistics

**Example:**
```go
stats := collector.GetStats()
fmt.Printf("Collected: %d, Processed: %d, Dropped: %d\n",
    stats.TotalEventsCollected,
    stats.TotalEventsProcessed,
    stats.TotalEventsDropped)
```

---

#### `(*ProcessCollector) StopProcessCollector()`
```go
func (pc *ProcessCollector) StopProcessCollector() error
```
Stop the collector and clean up resources.

**Returns:**
- `error`: Error if stop fails

**Example:**
```go
err := collector.StopProcessCollector()
if err != nil {
    log.Printf("Warning: Stop failed: %v", err)
}
```

---

## Event Conversion API

### ECS Format Conversion

#### `ConvertToECS()`
```go
func ConvertToECS(event *cgo.ProcessEvent) map[string]interface{}
```
Convert ProcessEvent to Elastic Common Schema (ECS) format.

**Example:**
```go
ecs := event.ConvertToECS(event)
// Output example:
// {
//   "@timestamp": "2025-12-02T15:30:45.123456789Z",
//   "event": {
//     "category": ["process"],
//     "type": ["start"],
//     "created": "2025-12-02T15:30:45.123456789Z"
//   },
//   "process": {
//     "pid": 1234,
//     "parent": {"pid": 5678},
//     "name": "notepad.exe",
//     "executable": "C:\\Windows\\System32\\notepad.exe",
//     "command_line": "notepad.exe test.txt",
//     "hash": {"sha256": "abc123..."}
//   },
//   "user": {
//     "name": "john",
//     "domain": "CORPORATE"
//   }
// }
```

---

#### `ConvertToProtobuf()`
```go
func ConvertToProtobuf(event *cgo.ProcessEvent, agentID string, agentVersion string) map[string]interface{}
```
Convert ProcessEvent to Protobuf-compatible format.

**Example:**
```go
pb := event.ConvertToProtobuf(event, "agent-001", "1.0.0")
```

---

#### `ConvertBatchToECS()`
```go
func ConvertBatchToECS(events []*cgo.ProcessEvent) []map[string]interface{}
```
Batch convert multiple events to ECS format.

**Example:**
```go
ecsArray := event.ConvertBatchToECS(events)
```

---

## Complete Usage Example

### C Language
```c
#include "edr_core.h"
#include <stdio.h>

int main() {
    // Initialize
    if (edr_core_init() != EDR_OK) {
        fprintf(stderr, "Init failed\n");
        return 1;
    }
    
    // Start collector
    void* handle = NULL;
    if (edr_start_process_collector(&handle) != EDR_SUCCESS) {
        fprintf(stderr, "Start failed\n");
        edr_core_cleanup();
        return 1;
    }
    
    // Poll events
    edr_process_event_t events[100];
    int count;
    
    for (int i = 0; i < 10; i++) {
        if (edr_poll_process_events(handle, events, 100, &count) == EDR_SUCCESS) {
            printf("Polled %d events\n", count);
            for (int j = 0; j < count; j++) {
                printf("  PID=%d Name=%s\n", events[j].pid, events[j].process_name);
            }
        }
        Sleep(1000);
    }
    
    // Cleanup
    edr_stop_process_collector(handle);
    edr_core_cleanup();
    
    return 0;
}
```

### Go Language
```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
    "github.com/houzhh15/EDR-POC/agent/main-go/internal/event"
)

func main() {
    // Start collector
    collector, err := cgo.StartProcessCollector()
    if err != nil {
        log.Fatalf("Failed to start: %v", err)
    }
    defer collector.StopProcessCollector()
    
    // Process events
    go func() {
        for evt := range collector.Events() {
            fmt.Printf("[%s] %s PID=%d PPID=%d %s\n",
                evt.Timestamp.Format("15:04:05"),
                evt.EventType,
                evt.ProcessPID,
                evt.ProcessPPID,
                evt.ProcessName)
            
            // Convert to ECS
            ecs := event.ConvertToECS(evt)
            fmt.Printf("  ECS: %+v\n", ecs)
        }
    }()
    
    // Run for 1 minute
    time.Sleep(1 * time.Minute)
    
    // Print stats
    stats := collector.GetStats()
    fmt.Printf("Stats: Collected=%d Processed=%d Dropped=%d\n",
        stats.TotalEventsCollected,
        stats.TotalEventsProcessed,
        stats.TotalEventsDropped)
}
```

---

## Error Handling

### Error Codes
```c
EDR_OK                        = 0
EDR_ERR_UNKNOWN              = -1
EDR_ERR_INVALID_PARAM        = -2
EDR_ERR_NO_MEMORY            = -3
EDR_ERR_NOT_INITIALIZED      = -4
EDR_ERR_ALREADY_INITIALIZED  = -5
EDR_ERROR_ETW_CREATE_FAILED  = -100
EDR_ERROR_ETW_ACCESS_DENIED  = -101
EDR_ERROR_BUFFER_EMPTY       = -300
```

### Common Issues

**ERROR_ACCESS_DENIED (-101)**
- Cause: Insufficient privileges
- Solution: Run as Administrator

**EDR_ERR_NOT_INITIALIZED (-4)**
- Cause: `edr_core_init()` not called
- Solution: Initialize before use

**Channel full / Events dropped**
- Cause: Consumer too slow
- Solution: Process events faster or increase channel size

---

## Performance Considerations

- **Polling Interval**: Default 10ms, adjustable via `CollectorConfig`
- **Batch Size**: Default 100 events per poll
- **Channel Capacity**: 1000 events
- **Memory Usage**: ~24MB for ring buffer + Go overhead
- **Expected Latency**: P95 < 100ms
- **Throughput**: 1000+ events/sec

---

## Thread Safety

- C layer: Ring buffer is lock-free (SPSC)
- Go layer: Goroutine-safe, uses channels
- CGO calls: Sequential, no concurrent access required

---

## Version

- Library Version: 0.1.0
- API Stability: Beta
- Last Updated: 2025-12-02
