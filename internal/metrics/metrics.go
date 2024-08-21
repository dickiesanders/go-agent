package metrics

import (
    "fmt" // Import fmt to fix the undefined error
    "log"

    "github.com/klauspost/cpuid/v2"     // For CPU information
    "github.com/shirou/gopsutil/cpu"    // Keep for CPU percentage collection
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/host"
    "github.com/shirou/gopsutil/mem"
    "github.com/shirou/gopsutil/net"
    "github.com/shirou/gopsutil/process"
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

// CPUInfo holds the basic CPU information using klauspost/cpuid
type CPUInfo struct {
    BrandName     string
    PhysicalCores int
    ThreadsPerCore int
    VendorID      string
    CacheLine     int
    Features      []string
}

// ProcessInfo holds information about a single process
type ProcessInfo struct {
    PID         int32
    Name        string
    CPUPercent  float64
    MemoryUsage uint64
}

func GatherBasicMetrics() (float64, uint64, error) {
    // Gather CPU percentage using gopsutil/cpu
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

// GatherCPUInfo collects detailed CPU information using klauspost/cpuid
func GatherCPUInfo() (*CPUInfo, error) {
    cpu := cpuid.CPU

    // Check if the CPU information is valid
    if cpu.BrandName == "" {
        log.Printf("Unable to gather CPU information")
        return nil, fmt.Errorf("unable to gather CPU information")
    }

    // Collect detailed CPU info
    cpuInfo := &CPUInfo{
        BrandName:    cpu.BrandName,
        PhysicalCores: cpu.PhysicalCores,
        ThreadsPerCore: cpu.ThreadsPerCore,
        VendorID:     cpu.VendorID.String(),
        CacheLine:    cpu.CacheLine,
        Features:     cpu.FeatureSet(),
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

// GatherProcessMetrics collects information about running processes, including CPU and memory usage
func GatherProcessMetrics() ([]ProcessInfo, error) {
    processes, err := process.Processes()
    if err != nil {
        log.Printf("Error gathering process information: %v", err)
        return nil, err
    }

    var processInfoList []ProcessInfo

    for _, proc := range processes {
        // Attempt to get the process name
        name, err := proc.Name()
        if err != nil {
            // Skip processes that we cannot read or access
            if err.Error() == "EOF" || err.Error() == "operation not permitted" {
                continue
            }
            log.Printf("Error getting process name for PID %d: %v", proc.Pid, err)
            continue
        }

        // Attempt to get CPU usage
        cpuPercent, err := proc.CPUPercent()
        if err != nil {
            log.Printf("Error getting process CPU usage for PID %d: %v", proc.Pid, err)
            continue
        }

        // Attempt to get memory usage
        memInfo, err := proc.MemoryInfo()
        if err != nil {
            log.Printf("Error getting process memory usage for PID %d: %v", proc.Pid, err)
            continue
        }

        processInfoList = append(processInfoList, ProcessInfo{
            PID:         proc.Pid,
            Name:        name,
            CPUPercent:  cpuPercent,
            MemoryUsage: memInfo.RSS,
        })
    }

    return processInfoList, nil
}
