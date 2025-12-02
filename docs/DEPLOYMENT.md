# EDR Windows Process Collector - Deployment Guide

## System Requirements

### Operating System
- **Windows 10** (1809 or later) / **Windows Server 2016** or later
- **64-bit** architecture required
- **Administrator privileges** required for ETW access

### Hardware Requirements
- **CPU**: 2+ cores recommended
- **Memory**: 512MB minimum, 1GB recommended
- **Disk**: 100MB for installation

### Software Dependencies
- **Visual C++ Redistributable 2015-2022** (x64)
- **Windows SDK** (for development builds)

---

## Installation

### Option 1: Pre-built Binary (Recommended)

1. **Download Release Package**
   ```powershell
   # Download from GitHub Releases
   Invoke-WebRequest -Uri "https://github.com/houzhh15/EDR-POC/releases/latest/download/edr-agent-windows-amd64.zip" -OutFile "edr-agent.zip"
   
   # Extract
   Expand-Archive -Path "edr-agent.zip" -DestinationPath "C:\Program Files\EDR-Agent"
   ```

2. **Verify Installation**
   ```powershell
   cd "C:\Program Files\EDR-Agent"
   .\edr-agent.exe --version
   ```

3. **Install as Windows Service** (Optional)
   ```powershell
   # Run as Administrator
   .\edr-agent.exe install
   ```

---

### Option 2: Build from Source

#### Prerequisites

**Required Tools:**
- **CMake** 3.15+
- **Visual Studio 2019/2022** with C++ Build Tools
  - OR **MinGW-w64** (GCC 9.0+)
- **Go** 1.20+

#### Build Steps

1. **Clone Repository**
   ```bash
   git clone https://github.com/houzhh15/EDR-POC.git
   cd EDR-POC/agent
   ```

2. **Build C Layer**
   ```bash
   # Using Visual Studio
   cd core-c
   mkdir build && cd build
   cmake .. -G "Visual Studio 17 2022" -A x64
   cmake --build . --config Release
   
   # OR using MinGW
   cmake .. -G "MinGW Makefiles" -DCMAKE_BUILD_TYPE=Release
   cmake --build .
   ```

3. **Build Go Layer**
   ```bash
   cd ../../main-go
   
   # Set CGO environment
   set CGO_ENABLED=1
   set CGO_CFLAGS=-I../core-c/include
   set CGO_LDFLAGS=-L../core-c/build/Release -ledr_core
   
   # Build
   go build -o edr-agent.exe ./cmd/agent
   ```

4. **Verify Build**
   ```bash
   .\edr-agent.exe --version
   ```

---

## Configuration

### Configuration File

Create `configs/agent.yaml`:

```yaml
# EDR Agent Configuration

collector:
  process:
    enabled: true
    poll_interval_ms: 10      # Poll interval (milliseconds)
    batch_size: 100           # Batch size for event retrieval
    channel_size: 1000        # Event channel buffer size

logging:
  level: info                 # Logging level: debug, info, warn, error
  output: file                # Output: stdout, file
  file_path: logs/edr-agent.log
  max_size_mb: 100            # Log rotation size
  max_backups: 10             # Number of backups to keep

output:
  type: elasticsearch         # Output type: elasticsearch, kafka, file
  elasticsearch:
    hosts:
      - "http://localhost:9200"
    index: "edr-events"
    username: ""
    password: ""
  
  # Fallback to local file if network unavailable
  fallback:
    enabled: true
    path: "data/events.jsonl"
```

### Environment Variables

```powershell
# Set environment variables
$env:EDR_CONFIG_PATH = "C:\Program Files\EDR-Agent\configs\agent.yaml"
$env:EDR_LOG_LEVEL = "info"
$env:EDR_DATA_DIR = "C:\ProgramData\EDR-Agent"
```

---

## Running the Agent

### Manual Start

```powershell
# Run as Administrator
cd "C:\Program Files\EDR-Agent"
.\edr-agent.exe
```

### Windows Service

```powershell
# Install service (as Administrator)
.\edr-agent.exe install

# Start service
Start-Service EDR-Agent

# Check status
Get-Service EDR-Agent

# Stop service
Stop-Service EDR-Agent

# Uninstall service
.\edr-agent.exe uninstall
```

### Command-Line Options

```
Usage: edr-agent.exe [OPTIONS]

Options:
  --config PATH        Config file path (default: configs/agent.yaml)
  --log-level LEVEL    Log level: debug, info, warn, error
  --version            Show version and exit
  --help               Show help message
  
Service Management:
  install              Install as Windows service
  uninstall            Uninstall Windows service
  start                Start service
  stop                 Stop service
```

---

## Verification

### Check Agent Status

```powershell
# Check process
Get-Process edr-agent

# Check service status
Get-Service EDR-Agent

# Check logs
Get-Content "logs\edr-agent.log" -Tail 50
```

### Test Event Collection

```powershell
# Generate test events
Start-Process notepad.exe
Start-Sleep -Seconds 2
Stop-Process -Name notepad

# Check event output
Get-Content "data\events.jsonl" -Tail 10
```

---

## Troubleshooting

### Common Issues

#### 1. **ETW Access Denied (Error -101)**

**Symptoms:**
```
Failed to start collector: ERROR_ACCESS_DENIED (-101)
```

**Solutions:**
- Run as Administrator
- Check user has "Performance Log Users" group membership
- Verify SeSecurityPrivilege is granted

```powershell
# Add user to Performance Log Users
Add-LocalGroupMember -Group "Performance Log Users" -Member "YourUsername"
```

---

#### 2. **DLL Not Found**

**Symptoms:**
```
The code execution cannot proceed because edr_core.dll was not found
```

**Solutions:**
- Ensure `edr_core.dll` is in same directory as `edr-agent.exe`
- Check Visual C++ Redistributable is installed
- Add DLL directory to PATH

```powershell
$env:PATH += ";C:\Program Files\EDR-Agent"
```

---

#### 3. **High CPU Usage**

**Symptoms:**
- Agent using >5% CPU continuously

**Solutions:**
- Increase `poll_interval_ms` in config (e.g., 50ms)
- Reduce `batch_size` if processing is slow
- Check for event processing bottlenecks

---

#### 4. **Events Dropped**

**Symptoms:**
```
Warning: Events dropped: 100
```

**Solutions:**
- Increase `channel_size` in config
- Optimize event processing speed
- Check network connectivity (if using Elasticsearch)
- Enable fallback file output

---

#### 5. **Service Won't Start**

**Symptoms:**
```
Service failed to start: error 1053
```

**Solutions:**
- Check config file exists and is valid
- Verify log directory is writable
- Check Windows Event Viewer for details

```powershell
Get-EventLog -LogName Application -Source "EDR-Agent" -Newest 10
```

---

### Diagnostic Commands

```powershell
# Check ETW sessions
logman query -ets | Select-String "EDR-Process"

# Check open handles
handle.exe edr-agent.exe

# Monitor performance
Get-Counter "\Process(edr-agent)\% Processor Time"
Get-Counter "\Process(edr-agent)\Working Set - Private"

# Enable debug logging
.\edr-agent.exe --log-level debug
```

---

## Upgrade

### In-Place Upgrade

```powershell
# Stop service
Stop-Service EDR-Agent

# Backup config
Copy-Item "configs\agent.yaml" "configs\agent.yaml.bak"

# Replace binaries
Copy-Item "path\to\new\edr-agent.exe" -Destination "C:\Program Files\EDR-Agent" -Force
Copy-Item "path\to\new\edr_core.dll" -Destination "C:\Program Files\EDR-Agent" -Force

# Start service
Start-Service EDR-Agent

# Verify
Get-Service EDR-Agent
.\edr-agent.exe --version
```

---

## Uninstallation

### Remove Service

```powershell
# Stop service
Stop-Service EDR-Agent

# Uninstall service
.\edr-agent.exe uninstall

# Remove files
Remove-Item -Path "C:\Program Files\EDR-Agent" -Recurse -Force
Remove-Item -Path "C:\ProgramData\EDR-Agent" -Recurse -Force
```

### Clean Registry (Optional)

```powershell
# Remove service registry keys (if needed)
Remove-Item -Path "HKLM:\SYSTEM\CurrentControlSet\Services\EDR-Agent" -Recurse -Force
```

---

## Security Considerations

### Permissions
- Agent requires **Administrator** privileges for ETW access
- Config file should have restricted permissions (ACL)
- Log files may contain sensitive process information

### Network Security
- Use HTTPS for Elasticsearch connections
- Configure firewall rules for outbound connections
- Consider using mutual TLS authentication

### Data Privacy
- Process command lines may contain sensitive data
- Consider masking sensitive arguments (passwords, keys)
- Implement data retention policies

---

## Performance Tuning

### High-Volume Environments

```yaml
collector:
  process:
    poll_interval_ms: 5       # Faster polling
    batch_size: 200           # Larger batches
    channel_size: 5000        # Bigger buffer

output:
  elasticsearch:
    bulk_size: 500            # Bulk indexing
    flush_interval_sec: 5
```

### Low-Resource Systems

```yaml
collector:
  process:
    poll_interval_ms: 50      # Slower polling
    batch_size: 50            # Smaller batches
    channel_size: 500         # Smaller buffer
```

---

## Monitoring

### Metrics to Monitor

- **Events per second**: `TotalEventsCollected` / time
- **Dropped events**: `TotalEventsDropped` (should be 0)
- **CPU usage**: < 2% under normal load
- **Memory usage**: ~50MB baseline
- **Network throughput**: Depends on event volume

### Integration with Monitoring Systems

**Prometheus Example:**
```yaml
# Agent exposes metrics on :9090/metrics
scrape_configs:
  - job_name: 'edr-agent'
    static_configs:
      - targets: ['localhost:9090']
```

---

## Support

- **Documentation**: https://github.com/houzhh15/EDR-POC/docs
- **Issues**: https://github.com/houzhh15/EDR-POC/issues
- **Email**: support@example.com

---

## Appendix

### Directory Structure

```
C:\Program Files\EDR-Agent\
├── edr-agent.exe           # Main executable
├── edr_core.dll            # C library (Windows)
├── configs\
│   └── agent.yaml          # Configuration
├── logs\
│   └── edr-agent.log       # Log files
└── data\
    └── events.jsonl        # Fallback event storage

C:\ProgramData\EDR-Agent\   # Runtime data
├── cache\
└── temp\
```

### Registry Keys

```
HKLM\SYSTEM\CurrentControlSet\Services\EDR-Agent
├── ImagePath                    # Service executable path
├── DisplayName                  # Service display name
└── Description                  # Service description
```

---

**Version**: 1.0  
**Last Updated**: 2025-12-02
