package main

import (
    "fmt"
    "log"
    "os"
    "sync/atomic"
    "time"

    "github.com/dickiesanders/go-agent/internal/metrics"
    "github.com/shirou/gopsutil/process"
)

var isPaused int32 // 0 = running, 1 = paused

func gatherBasicMetrics() (float64, uint64) {
    if atomic.LoadInt32(&isPaused) == 1 {
        fmt.Println("Data collection paused due to high resource usage")
        return 0, 0
    }

    cpuPercent, memoryUsage, err := metrics.GatherBasicMetrics()
    if err != nil {
        log.Printf("Error gathering basic metrics: %v", err)
    }
    return cpuPercent, memoryUsage
}

func gatherNetworkMetrics() ([]metrics.NetworkStat, []metrics.ConnectionStat, error) {
    if atomic.LoadInt32(&isPaused) == 1 {
        return nil, nil, fmt.Errorf("data collection paused")
    }

    netStats, connStats, err := metrics.GatherNetworkMetrics()
    if err != nil {
        log.Printf("Error gathering network metrics: %v", err)
    }
    return netStats, connStats, err
}

func gatherSystemInfo() {
    // Gather OS Info
    platform, platformVersion, kernelVersion := metrics.GatherOSInfo()
    if platform == "" {
        log.Printf("Failed to gather OS info")
    } else {
        fmt.Printf("Platform: %s\nVersion: %s\nKernel: %s\n", platform, platformVersion, kernelVersion)
    }

    // Gather CPU Info
    cpuInfo, err := metrics.GatherCPUInfo()
    if err != nil {
        log.Printf("Failed to gather CPU info")
    } else {
        for _, cpu := range cpuInfo {
            fmt.Printf("CPU Model: %s, Cores: %d, Vendor: %s\n", cpu.ModelName, cpu.Cores, cpu.VendorID)
        }
    }

    // Gather Disk I/O Info
    diskIO, err := metrics.GatherDiskIOInfo()
    if err != nil {
        log.Printf("Failed to gather Disk I/O info")
    } else {
        for name, io := range diskIO {
            fmt.Printf("Disk: %s, ReadBytes: %d, WriteBytes: %d\n", name, io.ReadBytes, io.WriteBytes)
        }
    }
}

func watchdog(proc *process.Process) {
    for {
        // Monitor CPU usage
        cpuPercent, err := proc.CPUPercent()
        if err != nil {
            log.Printf("Error getting CPU usage: %v", err)
            continue
        }

        // Monitor memory usage
        memInfo, err := proc.MemoryInfo()
        if err != nil {
            log.Printf("Error getting memory usage: %v", err)
            continue
        }

        // Define thresholds (3-5%)
        if cpuPercent > 5 || float64(memInfo.RSS)/float64(memInfo.VMS)*100 > 5 {
            fmt.Println("Pausing data collection due to high resource usage")
            atomic.StoreInt32(&isPaused, 1) // Pause data collection
        } else if cpuPercent < 3 && float64(memInfo.RSS)/float64(memInfo.VMS)*100 < 3 {
            fmt.Println("Resuming data collection")
            atomic.StoreInt32(&isPaused, 0) // Resume data collection
        }

        time.Sleep(5 * time.Second) // Check every 5 seconds
    }
}

func main() {
    // Get the current process using the PID
    pid := int32(os.Getpid())
    proc, err := process.NewProcess(pid)
    if err != nil {
        log.Fatal("Failed to create process object", err)
    }

    // Start the watchdog goroutine
    go watchdog(proc)

    // Gather system information
    gatherSystemInfo()

    // Main loop for data collection
    for {
        cpuPercent, memoryUsage := gatherBasicMetrics()
        fmt.Printf("CPU Usage: %.2f%%\n", cpuPercent)
        fmt.Printf("Memory Usage: %d bytes\n", memoryUsage)

        netStats, connStats, err := gatherNetworkMetrics()
        if err != nil {
            fmt.Println(err)
        } else {
            // Print Network I/O statistics
            fmt.Printf("\nNetwork I/O Statistics:\n")
            for _, io := range netStats {
                fmt.Printf("Interface: %s - Bytes Sent: %d, Bytes Received: %d\n", io.Name, io.BytesSent, io.BytesRecv)
            }

            // Print Active Connections
            fmt.Printf("\nActive Network Connections:\n")
            for _, conn := range connStats {
                fmt.Printf("Local Address: %s:%d -> Remote Address: %s:%d\n", conn.LocalAddr, conn.LocalPort, conn.RemoteAddr, conn.RemotePort)
            }
        }

        time.Sleep(5 * time.Second) // Sleep before the next collection cycle
    }
}

