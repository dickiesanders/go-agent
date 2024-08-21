package main

import (
    "fmt"
    "log"
    "github.com/dickiesanders/go-agent/internal/metrics"
)

func gatherBasicMetrics() (float64, uint64) {
    // Gather CPU percentage and Memory usage using the metrics package
    cpuPercent, memoryUsage, err := metrics.GatherBasicMetrics()
    if err != nil {
        log.Fatal(err)
    }
    return cpuPercent, memoryUsage
}

func gatherNetworkMetrics() ([]metrics.NetworkStat, []metrics.ConnectionStat, error) {
    // Gather Network I/O counters and open network connections using the metrics package
    netStats, connStats, err := metrics.GatherNetworkMetrics()
    if err != nil {
        return nil, nil, err
    }
    return netStats, connStats, nil
}

func main() {
    // Gather basic metrics (CPU and Memory)
    cpuPercent, memoryUsage := gatherBasicMetrics()

    // Print CPU and Memory metrics
    fmt.Printf("CPU Usage: %.2f%%\n", cpuPercent)
    fmt.Printf("Memory Usage: %d bytes\n", memoryUsage)

    // Gather network metrics (Network I/O and Active Connections)
    netStats, connStats, err := gatherNetworkMetrics()
    if err != nil {
        log.Fatal(err)
    }

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

