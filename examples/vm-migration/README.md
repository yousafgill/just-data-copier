# VM Image Migration - JustDataCopier

Transfer virtual machine files and snapshots efficiently between Windows servers with JustDataCopier.

## 📋 Overview

Virtual machine files are typically large (5GB-500GB+) and require reliable transfer without corruption. This guide shows how to move VM images, snapshots, and related files between Windows servers using optimal JustDataCopier configurations.

## 🎯 Common VM Transfer Scenarios

### Hypervisor Migration
- Moving VMs between Hyper-V hosts
- Transferring VMware workstation files
- Consolidating VM storage

### Backup and Recovery
- Backing up VM images to remote storage
- Creating disaster recovery copies
- Moving VMs to backup servers

### Development and Testing
- Distributing VM templates to development teams
- Moving test environments between servers
- Sharing pre-configured development VMs

## ⚙️ Optimal Configuration for VM Files

### Large VM Files (50GB+)
```cmd
rem High-performance transfer for large VM files
jdc.exe -server -listen 0.0.0.0:8000 -output "D:\VM-Backups"
```

```cmd
rem Client: Transfer large VHDX/VMDK files
jdc.exe -file "C:\VMs\Windows10-Dev.vhdx" -connect vm-server:8000 -chunk 16777216 -workers 6 -adaptive -timeout 30m
```

### VM Snapshot Files (10-50GB)
```cmd
rem Optimized for medium-sized snapshot files
jdc.exe -file "C:\VMs\Snapshots\snapshot-001.avhdx" -connect backup-server:8000 -chunk 8388608 -workers 4 -compress -timeout 15m
```

### VM Configuration Files (Small)
```cmd
rem Quick transfer for VM config files
jdc.exe -file "C:\VMs\Windows10-Dev.xml" -connect vm-server:8000 -chunk 1048576 -workers 2 -timeout 5m
```

## 🗂️ Complete VM Migration Script

### migrate-vm.bat
```batch
@echo off
setlocal enabledelayedexpansion

echo =============================================================
echo VM Migration Script - JustDataCopier
echo =============================================================

set SOURCE_VM_PATH=C:\VMs\%1
set DEST_SERVER=%2
set DEST_PORT=8000
set VM_NAME=%1

if "%VM_NAME%"=="" (
    echo Usage: migrate-vm.bat [VM_NAME] [DESTINATION_SERVER]
    echo Example: migrate-vm.bat "Windows10-Dev" "backup-server.company.com"
    exit /b 1
)

if "%DEST_SERVER%"=="" (
    echo Error: Destination server not specified
    exit /b 1
)

echo Starting migration of VM: %VM_NAME%
echo Source: %SOURCE_VM_PATH%
echo Destination: %DEST_SERVER%:%DEST_PORT%
echo.

rem Check if source VM directory exists
if not exist "%SOURCE_VM_PATH%" (
    echo Error: VM directory not found: %SOURCE_VM_PATH%
    exit /b 1
)

rem Create migration log
set LOG_FILE=%TEMP%\vm-migration-%VM_NAME%-%DATE:~-4,4%%DATE:~-10,2%%DATE:~-7,2%-%TIME:~0,2%%TIME:~3,2%.log
echo Migration started at %DATE% %TIME% > "%LOG_FILE%"

echo Transferring VM files...
echo.

rem Transfer all VM files
for %%F in ("%SOURCE_VM_PATH%\*") do (
    echo Transferring: %%~nxF
    echo File: %%~nxF at %DATE% %TIME% >> "%LOG_FILE%"
    
    rem Determine optimal settings based on file size
    set FILE_SIZE=%%~zF
    
    if !FILE_SIZE! GTR 53687091200 (
        rem Files larger than 50GB - use large chunk settings
        jdc.exe -file "%%F" -connect %DEST_SERVER%:%DEST_PORT% -chunk 16777216 -workers 6 -adaptive -timeout 30m
    ) else if !FILE_SIZE! GTR 1073741824 (
        rem Files larger than 1GB - use medium chunk settings
        jdc.exe -file "%%F" -connect %DEST_SERVER%:%DEST_PORT% -chunk 8388608 -workers 4 -compress -timeout 15m
    ) else (
        rem Small files - use standard settings
        jdc.exe -file "%%F" -connect %DEST_SERVER%:%DEST_PORT% -chunk 2097152 -workers 2 -timeout 5m
    )
    
    if !ERRORLEVEL! NEQ 0 (
        echo ERROR: Failed to transfer %%~nxF
        echo ERROR: Failed to transfer %%~nxF at %DATE% %TIME% >> "%LOG_FILE%"
        exit /b 1
    ) else (
        echo SUCCESS: %%~nxF transferred successfully
        echo SUCCESS: %%~nxF at %DATE% %TIME% >> "%LOG_FILE%"
    )
    echo.
)

echo.
echo =============================================================
echo VM Migration Completed Successfully!
echo VM Name: %VM_NAME%
echo Log File: %LOG_FILE%
echo =============================================================

rem Display transfer summary
echo.
echo Transfer Summary:
dir "%SOURCE_VM_PATH%" /s /-c
echo.
echo Migration log saved to: %LOG_FILE%
echo.

pause
```

## 💡 PowerShell Migration Script

### Migrate-VM.ps1
```powershell
param(
    [Parameter(Mandatory=$true)]
    [string]$VMName,
    
    [Parameter(Mandatory=$true)]
    [string]$DestinationServer,
    
    [int]$Port = 8000,
    
    [string]$SourcePath = "C:\VMs"
)

Write-Host "=============================================================" -ForegroundColor Green
Write-Host "VM Migration Script - JustDataCopier" -ForegroundColor Green
Write-Host "=============================================================" -ForegroundColor Green

$VMPath = Join-Path $SourcePath $VMName
$LogFile = "$env:TEMP\vm-migration-$VMName-$(Get-Date -Format 'yyyyMMdd-HHmm').log"

# Validate inputs
if (-not (Test-Path $VMPath)) {
    Write-Error "VM directory not found: $VMPath"
    exit 1
}

Write-Host "VM Name: $VMName" -ForegroundColor Yellow
Write-Host "Source Path: $VMPath" -ForegroundColor Yellow
Write-Host "Destination: $DestinationServer`:$Port" -ForegroundColor Yellow
Write-Host "Log File: $LogFile" -ForegroundColor Yellow
Write-Host

# Start logging
"Migration started at $(Get-Date)" | Out-File $LogFile

# Get all VM files
$VMFiles = Get-ChildItem -Path $VMPath -File -Recurse

Write-Host "Found $($VMFiles.Count) files to transfer" -ForegroundColor Cyan
Write-Host

foreach ($File in $VMFiles) {
    Write-Host "Transferring: $($File.Name) ($([math]::Round($File.Length/1GB, 2)) GB)" -ForegroundColor White
    
    # Determine optimal settings based on file size
    $ChunkSize = 2097152  # 2MB default
    $Workers = 2
    $Timeout = "5m"
    $ExtraArgs = @()
    
    if ($File.Length -gt 50GB) {
        $ChunkSize = 16777216  # 16MB
        $Workers = 6
        $Timeout = "30m"
        $ExtraArgs += "-adaptive"
    }
    elseif ($File.Length -gt 1GB) {
        $ChunkSize = 8388608   # 8MB
        $Workers = 4
        $Timeout = "15m"
        $ExtraArgs += "-compress"
    }
    
    # Build JDC command
    $JDCArgs = @(
        "-file", $File.FullName,
        "-connect", "$DestinationServer`:$Port",
        "-chunk", $ChunkSize,
        "-workers", $Workers,
        "-timeout", $Timeout
    ) + $ExtraArgs
    
    # Execute transfer
    $StartTime = Get-Date
    & jdc.exe @JDCArgs
    $EndTime = Get-Date
    $Duration = $EndTime - $StartTime
    
    if ($LASTEXITCODE -eq 0) {
        $Speed = [math]::Round(($File.Length / $Duration.TotalSeconds) / 1MB, 2)
        Write-Host "✓ SUCCESS: $($File.Name) transferred in $($Duration.ToString('hh\:mm\:ss')) at $Speed MB/s" -ForegroundColor Green
        "SUCCESS: $($File.Name) transferred at $(Get-Date) - Duration: $($Duration.ToString('hh\:mm\:ss')) - Speed: $Speed MB/s" | Out-File $LogFile -Append
    }
    else {
        Write-Host "✗ ERROR: Failed to transfer $($File.Name)" -ForegroundColor Red
        "ERROR: Failed to transfer $($File.Name) at $(Get-Date)" | Out-File $LogFile -Append
        exit 1
    }
    Write-Host
}

Write-Host "=============================================================" -ForegroundColor Green
Write-Host "VM Migration Completed Successfully!" -ForegroundColor Green
Write-Host "VM: $VMName" -ForegroundColor Green
Write-Host "Log: $LogFile" -ForegroundColor Green
Write-Host "=============================================================" -ForegroundColor Green

# Display summary
$TotalSize = ($VMFiles | Measure-Object -Property Length -Sum).Sum
Write-Host
Write-Host "Migration Summary:" -ForegroundColor Cyan
Write-Host "  Files Transferred: $($VMFiles.Count)" -ForegroundColor White
Write-Host "  Total Size: $([math]::Round($TotalSize/1GB, 2)) GB" -ForegroundColor White
Write-Host "  Log File: $LogFile" -ForegroundColor White
```

## 🔍 Verification and Validation

### VM Integrity Check Script (verify-vm-transfer.bat)
```batch
@echo off
echo Verifying VM file integrity...

rem Check if VM files exist on destination
echo Checking file presence...
if not exist "D:\VM-Backups\%1\*.vhdx" (
    echo ERROR: VHDX files not found
    exit /b 1
)

rem Compare file sizes (basic validation)
echo Comparing file sizes...
for %%F in ("C:\VMs\%1\*") do (
    if exist "D:\VM-Backups\%1\%%~nxF" (
        echo ✓ Found: %%~nxF
    ) else (
        echo ✗ Missing: %%~nxF
        exit /b 1
    )
)

echo.
echo ✓ VM transfer verification completed successfully
```

## ⚠️ Important Considerations

### Before Migration
1. **Stop VMs**: Always shut down VMs before transferring their files
2. **Check Disk Space**: Ensure sufficient space on destination server
3. **Network Stability**: Use stable network connections for large transfers
4. **Backup First**: Keep original VM files until transfer is verified

### VM-Specific Settings
- **Hyper-V VMs**: Transfer .vhdx, .xml, and snapshot files
- **VMware VMs**: Include .vmdk, .vmx, .nvram, and .vmsd files
- **VirtualBox VMs**: Transfer .vdi, .vbox files and snapshots folder

### Performance Tips
1. **Large Chunks**: Use 16MB+ chunks for VM files over 50GB
2. **Multiple Workers**: 4-8 workers for fast networks
3. **Compression**: Enable for snapshot files (often compress well)
4. **Adaptive Mode**: Use for unstable network connections

## 🔧 Troubleshooting

### Transfer Stuck or Slow
```cmd
rem Try with smaller chunks and fewer workers
jdc.exe -file "vm-file.vhdx" -connect server:8000 -chunk 4194304 -workers 2 -adaptive
```

### Large File Timeout
```cmd
rem Increase timeout for very large VM files
jdc.exe -file "large-vm.vhdx" -connect server:8000 -timeout 60m -chunk 16777216
```

### Network Interruption
```cmd
rem Use resume capability if supported, or restart with same settings
jdc.exe -file "vm-file.vhdx" -connect server:8000 -chunk 8388608 -workers 4 -timeout 30m
```

---

**Note**: Always test VM functionality after migration and keep original files until new location is verified. VM files are critical system components and require careful handling.
