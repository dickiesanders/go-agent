package main

import (
    "fmt"
    "log"
    "time"
    "sync/atomic"
    "os"

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
        log.Fatal(err)
    }
    return cpuPercent, memoryUsage
}

func gatherNetworkMetrics() ([]metrics.NetworkStat, []metrics.ConnectionStat, error) {
    if atomic.LoadInt32(&isPaused) == 1 {
        return nil, nil, fmt.Errorf("data collection paused")
    }

    netStats, connStats, err := metrics.GatherNetworkMetrics()
    if err != nil {
        return nil, nil, err
    }
    return netStats, connStats, nil
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

