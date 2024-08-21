package metrics

import (
    "log"
    "github.com/shirou/gopsutil/cpu"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/host"
    "github.com/shirou/gopsutil/mem"
    "github.com/shirou/gopsutil/net"
)

type NetworkStat struct {
    Name       string
    BytesSent  uint64
    BytesRecv  uint64
}

type ConnectionStat struct {
    LocalAddr  string
    LocalPort  uint32
    RemoteAddr string
    RemotePort uint32
}

func GatherBasicMetrics() (float64, uint64, error) {
    // Gather CPU percentage
    cpuPercents, err := cpu.Percent(0, false)
    if err != nil {
        log.Printf("Error gathering CPU percentage: %v", err)
        return 0, 0, err
    }
    cpuPercent := cpuPercents[0]

    // Gather Memory usage
    vmStat, err := mem.VirtualMemory()
    if err != nil {
        log.Printf("Error gathering Memory usage: %v", err)
        return 0, 0, err
    }
    memoryUsage := vmStat.Used

    return cpuPercent, memoryUsage, nil
}

func GatherNetworkMetrics() ([]NetworkStat, []ConnectionStat, error) {
    // Gather Network I/O counters
    netIOCounters, err := net.IOCounters(false)
    if err != nil {
        log.Printf("Error gathering Network I/O counters: %v", err)
        return nil, nil, err
    }

    // Convert the data into our custom NetworkStat struct
    var netStats []NetworkStat
    for _, io := range netIOCounters {
        netStats = append(netStats, NetworkStat{
            Name:      io.Name,
            BytesSent: io.BytesSent,
            BytesRecv: io.BytesRecv,
        })
    }

    // Gather Open Ports and Active Connections
    netConnections, err := net.Connections("inet")
    if err != nil {
        log.Printf("Error gathering Network connections: %v", err)
        return nil, nil, err
    }

    var connStats []ConnectionStat
    for _, conn := range netConnections {
        connStats = append(connStats, ConnectionStat{
            LocalAddr:  conn.Laddr.IP,
            LocalPort:  conn.Laddr.Port,
            RemoteAddr: conn.Raddr.IP,
            RemotePort: conn.Raddr.Port,
        })
    }

    return netStats, connStats, nil
}

func GatherOSInfo() (string, string, string) {
    info, err := host.Info()
    if err != nil {
        log.Printf("Error gathering OS information: %v", err)
        return "", "", ""
    }
    return info.Platform, info.PlatformVersion, info.KernelVersion
}

func GatherCPUInfo() ([]cpu.InfoStat, error) {
    cpuInfo, err := cpu.Info()
    if err != nil {
        log.Printf("Error gathering CPU information: %v", err)
        return nil, err
    }
    return cpuInfo, nil
}

func GatherDiskIOInfo() (map[string]disk.IOCountersStat, error) {
    ioCounters, err := disk.IOCounters()
    if err != nil {
        log.Printf("Error gathering Disk I/O information: %v", err)
        return nil, err
    }
    return ioCounters, nil
}

