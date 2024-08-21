package main

import (
    "flag"
    "fmt"
    "log"
    "net"
    "os"
    "sync/atomic"
    "time"
    "runtime"

    "github.com/dickiesanders/go-agent/internal/metrics"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/host"
    "github.com/shirou/gopsutil/process"
    "github.com/StackExchange/wmi"
)

var isPaused int32 // 0 = running, 1 = paused

// A struct to hold the collected metrics data
type MetricsData struct {
    CPUPercent    float64
    MemoryUsage   uint64
    ProcessInfo   []metrics.ProcessInfo
    NetworkStats  []metrics.NetworkStat
    ConnStats     []metrics.ConnectionStat
    DiskIOStats   map[string]disk.IOCountersStat // Correct type here
    Timestamp     time.Time
}

// OneTimeHostInfo holds information that is sent when the agent first registers
type OneTimeHostInfo struct {
    Hostname     string
    FQDN         string
    CPUInfo      *metrics.CPUInfo
    IP           string
    IsVirtual    bool
}

type win32_ComputerSystem struct {
    Manufacturer string
    Model        string
}

// Simulate sending one-time host information to the mothership
func registerAgentWithHostInfo(hostInfo OneTimeHostInfo, consoleFlag bool) {
    if consoleFlag {
        fmt.Println("\nRegistering agent with the following host information:")
        fmt.Printf("Hostname: %s\n", hostInfo.Hostname)
        fmt.Printf("FQDN: %s\n", hostInfo.FQDN)
        fmt.Printf("IP Address: %s\n", hostInfo.IP)
        
        // Improved formatting for CPU Info
        if hostInfo.CPUInfo != nil {
            fmt.Println("CPU Information:")
            fmt.Printf("  Brand Name: %s\n", hostInfo.CPUInfo.BrandName)
            fmt.Printf("  Physical Cores: %d\n", hostInfo.CPUInfo.PhysicalCores)
            fmt.Printf("  Threads per Core: %d\n", hostInfo.CPUInfo.ThreadsPerCore)
            fmt.Printf("  Vendor ID: %s\n", hostInfo.CPUInfo.VendorID)
            fmt.Printf("  Cache Line Size: %d bytes\n", hostInfo.CPUInfo.CacheLine)
            fmt.Printf("  Features: %v\n", hostInfo.CPUInfo.Features)
        } else {
            fmt.Println("CPU Information: Not available")
        }

        fmt.Printf("Is Virtual: %v\n", hostInfo.IsVirtual)
    }

    // Logic to send the host information to the mothership
}


// Simulate pushing data to the server
func pushDataToServer(data []MetricsData) {
    fmt.Printf("\nPushing %d metrics to the server...\n", len(data))
    // Here you would add your logic to send the data to a remote server.
}

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

func gatherProcessMetrics() []metrics.ProcessInfo {
    processInfoList, err := metrics.GatherProcessMetrics()
    if err != nil {
        log.Printf("Error gathering process metrics: %v", err)
        return nil
    }
    return processInfoList
}

func gatherNetworkMetrics() ([]metrics.NetworkStat, []metrics.ConnectionStat) {
    netStats, connStats, err := metrics.GatherNetworkMetrics()
    if err != nil {
        log.Printf("Error gathering network metrics: %v", err)
        return nil, nil
    }
    return netStats, connStats
}

func gatherDiskMetrics() map[string]disk.IOCountersStat {
    diskStats, err := metrics.GatherDiskIOInfo()
    if err != nil {
        log.Printf("Error gathering disk metrics: %v", err)
        return nil
    }
    return diskStats
}

// Gather one-time host information when the agent starts
func gatherOneTimeHostInfo() OneTimeHostInfo {
    // Gather Hostname and FQDN
    hostname, err := os.Hostname()
    if err != nil {
        log.Printf("Error gathering hostname: %v", err)
    }

    // Assuming FQDN is the same as hostname on most systems
    fqdn := hostname

    // Gather CPU Information
    cpuInfo, err := metrics.GatherCPUInfo()
    if err != nil {
        log.Printf("Error gathering CPU info: %v", err)
    }

    // Get IP address
    ip := getLocalIP()

    // Check if the system is virtual
    isVirtual := checkIfVirtual()

    return OneTimeHostInfo{
        Hostname:  hostname,
        FQDN:      fqdn,
        CPUInfo:   cpuInfo,
        IP:        ip,
        IsVirtual: isVirtual,
    }
}

// Get the local IP address
func getLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        log.Printf("Error getting IP address: %v", err)
        return ""
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return ""
}

// Check if the system is virtual by querying host info
func checkIfVirtual() bool {
    // Check if we're on Linux, macOS, or Windows
    switch runtime.GOOS {
    case "linux", "darwin":
        info, err := host.Info()
        if err != nil {
            log.Printf("Error checking if system is virtual: %v", err)
            return false
        }
        return info.VirtualizationSystem != ""
    
    case "windows":
        return checkIfVirtualWindows()
    
    default:
        log.Printf("Unsupported OS: %s", runtime.GOOS)
        return false
    }
}

// Check if the system is virtual on Windows
func checkIfVirtualWindows() bool {
    var cs []win32_ComputerSystem
    query := wmi.CreateQuery(&cs, "")
    err := wmi.Query(query, &cs)
    if err != nil {
        log.Printf("Error checking if system is virtual on Windows: %v", err)
        return false
    }

    if len(cs) > 0 {
        // Look for common VM-related strings in the manufacturer or model
        manufacturer := cs[0].Manufacturer
        model := cs[0].Model
        if manufacturer == "Microsoft Corporation" && (model == "Virtual Machine" || model == "Hyper-V") {
            return true
        }
        if manufacturer == "VMware, Inc." || manufacturer == "Xen" || manufacturer == "QEMU" {
            return true
        }
    }

    return false
}

func main() {
    // Define the console flag
    consoleFlag := flag.Bool("console", false, "Enable console output for collected data")
    flag.Parse()

    // Debug: Print the console flag status
    fmt.Println("Console flag enabled:", *consoleFlag)

    // Register the agent with the mothership and send one-time host information
    hostInfo := gatherOneTimeHostInfo()
    registerAgentWithHostInfo(hostInfo, *consoleFlag)

    // Get the current process using the PID
    pid := int32(os.Getpid())
    proc, err := process.NewProcess(pid)
    if err != nil {
        log.Fatal("Failed to create process object", err)
    }

    // Start the watchdog goroutine
    go watchdog(proc)

    // Buffer to hold collected metrics every 30 seconds
    var metricsBuffer []MetricsData

    // Create a ticker for collecting data every 30 seconds
    dataCollectionTicker := time.NewTicker(30 * time.Second)
    // Create another ticker for pushing data every 5 minutes
    dataPushTicker := time.NewTicker(5 * time.Minute)

    for {
        select {
        case <-dataCollectionTicker.C:
            // Debug: Confirm the data collection ticker is triggering
            fmt.Println("Data collection tick")

            // Collect metrics every 30 seconds
            cpuPercent, memoryUsage := gatherBasicMetrics()
            processInfo := gatherProcessMetrics()
            netStats, connStats := gatherNetworkMetrics()
            diskStats := gatherDiskMetrics()

            metricsData := MetricsData{
                CPUPercent:  cpuPercent,
                MemoryUsage: memoryUsage,
                ProcessInfo: processInfo,
                NetworkStats: netStats,
                ConnStats:   connStats,
                DiskIOStats: diskStats, // Added disk metrics
                Timestamp:   time.Now(),
            }

            // Add the collected data to the buffer
            metricsBuffer = append(metricsBuffer, metricsData)

            // Print data to the console if the console flag is enabled
            if *consoleFlag {
                fmt.Printf("\nCollected Metrics at %s:\n", metricsData.Timestamp)
                fmt.Printf("CPU Usage: %.2f%%\n", metricsData.CPUPercent)
                fmt.Printf("Memory Usage: %d bytes\n", metricsData.MemoryUsage)
                fmt.Println("Process Information:")
                for _, proc := range metricsData.ProcessInfo {
                    fmt.Printf("PID: %d, Name: %s, CPU: %.2f%%, Memory: %d bytes\n",
                        proc.PID, proc.Name, proc.CPUPercent, proc.MemoryUsage)
                }
                fmt.Println("\nDisk I/O Statistics:")
                for name, io := range metricsData.DiskIOStats {
                    fmt.Printf("Disk: %s, ReadBytes: %d, WriteBytes: %d\n", name, io.ReadBytes, io.WriteBytes)
                }
                fmt.Println("\nNetwork I/O Statistics:")
                for _, io := range metricsData.NetworkStats {
                    fmt.Printf("Interface: %s - Bytes Sent: %d, Bytes Received: %d\n", io.Name, io.BytesSent, io.BytesRecv)
                }
                fmt.Println("\nActive Network Connections:")
                for _, conn := range metricsData.ConnStats {
                    fmt.Printf("Local Address: %s:%d -> Remote Address: %s:%d\n", conn.LocalAddr, conn.LocalPort, conn.RemoteAddr, conn.RemotePort)
                }
            }

        case <-dataPushTicker.C:
            // Push data to the server every 5 minutes
            if len(metricsBuffer) > 0 {
                pushDataToServer(metricsBuffer)
                metricsBuffer = nil // Clear the buffer after pushing
            } else {
                fmt.Println("No data to push to the server.")
            }
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
