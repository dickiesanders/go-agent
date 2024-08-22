package main

import (
    "crypto/sha256"
    "encoding/hex"
    "flag"
    "log"
    "net"
    "os"
    "sync/atomic"
    "time"
    "runtime"
    "io"
    "encoding/json"
    "net/http"
    "fmt"
    "strings"

    "github.com/dickiesanders/go-agent/internal/metrics"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/host"
    "github.com/shirou/gopsutil/process"
    // "github.com/yusufpapurcu/wmi" wmic
)

var isPaused int32 // 0 = running, 1 = paused

// A struct to hold the collected metrics data
type MetricsData struct {
    CPUPercent    float64                    `json:"cpu_percent"`
    MemoryUsage   uint64                     `json:"memory_usage"`
    ProcessInfo   []metrics.ProcessInfo      `json:"process_info"`
    NetworkStats  []metrics.NetworkStat      `json:"network_stats"`
    ConnStats     []metrics.ConnectionStat   `json:"conn_stats"`
    DiskIOStats   map[string]disk.IOCountersStat `json:"disk_io_stats"`
    Timestamp     time.Time                  `json:"timestamp"`
    UniqueID        string                     `json:"unique_id"`
}

// OneTimeHostInfo holds information that is sent when the agent first registers
type OneTimeHostInfo struct {
    Hostname     string
    FQDN         string
    CPUInfo      *metrics.CPUInfo
    IP           string
    IsVirtual    bool
    UniqueID     string
}

// GenerateUniqueID creates a unique, reproducible ID based on the API key, hostname, and IP.
func GenerateUniqueID(apiKey, hostname, ip string) string {
    data := apiKey + hostname + ip
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

// type win32_ComputerSystem struct {
//     Manufacturer string
//     Model        string
// }

// Simulate sending one-time host information to the mothership
func registerAgentWithHostInfo(hostInfo OneTimeHostInfo, consoleFlag bool, logger *log.Logger) {
    logger.Println("Registering agent with the following host information:")
    logger.Printf("Hostname: %s\n", hostInfo.Hostname)
    logger.Printf("FQDN: %s\n", hostInfo.FQDN)
    logger.Printf("IP Address: %s\n", hostInfo.IP)
    
    // Improved formatting for CPU Info
    if hostInfo.CPUInfo != nil {
        logger.Println("CPU Information:")
        logger.Printf("  Brand Name: %s\n", hostInfo.CPUInfo.BrandName)
        logger.Printf("  Physical Cores: %d\n", hostInfo.CPUInfo.PhysicalCores)
        logger.Printf("  Threads per Core: %d\n", hostInfo.CPUInfo.ThreadsPerCore)
        logger.Printf("  Vendor ID: %s\n", hostInfo.CPUInfo.VendorID)
        logger.Printf("  Cache Line Size: %d bytes\n", hostInfo.CPUInfo.CacheLine)
        logger.Printf("  Features: %v\n", hostInfo.CPUInfo.Features)
    } else {
        logger.Println("CPU Information: Not available")
    }

    logger.Printf("Is Virtual: %v\n", hostInfo.IsVirtual)
    logger.Printf("Unique Client ID: %s\n", hostInfo.UniqueID) // Log the unique client ID

    // Logic to send the host information to the mothership
}

// Simulate pushing data to the server
func pushDataToServer(apiKey string, data []MetricsData, logger *log.Logger) {
    // Define the API endpoint and the authorization token
    apiEndpoint := "http://localhost:8080/receive"
    authToken := apiKey

    // Iterate over the collected metrics data
    for _, metricsData := range data {
        jsonData, err := json.Marshal(metricsData)
        if err != nil {
            logger.Printf("Error marshalling metrics data: %v", err)
            continue
        }

        // Prepare the request body with Action and MessageBody as URL-encoded parameters
        requestData := fmt.Sprintf("Action=SendMessage&MessageBody=%s", string(jsonData))

        // Create a new HTTP POST request
        req, err := http.NewRequest("POST", apiEndpoint, strings.NewReader(requestData))
        if err != nil {
            logger.Printf("Error creating HTTP request: %v", err)
            continue
        }

        // Set necessary headers
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("Authorization", authToken)

        // Send the HTTP request
        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            logger.Printf("Error sending data to server: %v", err)
            continue
        }
        defer resp.Body.Close()

        // Check the response status code
        if resp.StatusCode != http.StatusOK {
            logger.Printf("Failed to push data to server. Status Code: %d", resp.StatusCode)
        } else {
            logger.Println("Data successfully pushed to the server")
        }
    }
}

func gatherBasicMetrics(logger *log.Logger) (float64, uint64) {
    if atomic.LoadInt32(&isPaused) == 1 {
        logger.Println("Data collection paused due to high resource usage")
        return 0, 0
    }

    cpuPercent, memoryUsage, err := metrics.GatherBasicMetrics()
    if err != nil {
        logger.Printf("Error gathering basic metrics: %v", err)
    }
    return cpuPercent, memoryUsage
}

func gatherProcessMetrics(logger *log.Logger) []metrics.ProcessInfo {
    processInfoList, err := metrics.GatherProcessMetrics()
    if err != nil {
        logger.Printf("Error gathering process metrics: %v", err)
        return nil
    }
    return processInfoList
}

func gatherNetworkMetrics(logger *log.Logger) ([]metrics.NetworkStat, []metrics.ConnectionStat) {
    netStats, connStats, err := metrics.GatherNetworkMetrics()
    if err != nil {
        logger.Printf("Error gathering network metrics: %v", err)
        return nil, nil
    }
    return netStats, connStats
}

func gatherDiskMetrics(logger *log.Logger) map[string]disk.IOCountersStat {
    diskStats, err := metrics.GatherDiskIOInfo()
    if err != nil {
        logger.Printf("Error gathering disk metrics: %v", err)
        return nil
    }
    return diskStats
}

// Gather one-time host information when the agent starts
func gatherOneTimeHostInfo(logger *log.Logger, apiKey string) OneTimeHostInfo {
    // Gather Hostname and FQDN
    hostname, err := os.Hostname()
    if err != nil {
        logger.Printf("Error gathering hostname: %v", err)
    }

    // Assuming FQDN is the same as hostname on most systems
    fqdn := hostname

    // Gather CPU Information
    cpuInfo, err := metrics.GatherCPUInfo()
    if err != nil {
        logger.Printf("Error gathering CPU info: %v", err)
    }

    // Get IP address
    ip := getLocalIP(logger)
    // Check if the system is virtual
    isVirtual := checkIfVirtual(logger)

    // Generate a unique ID based on the API key, hostname, and IP
    uniqueID := GenerateUniqueID(apiKey, hostname, ip)

    return OneTimeHostInfo{
        Hostname:  hostname,
        FQDN:      fqdn,
        CPUInfo:   cpuInfo,
        IP:        ip,
        IsVirtual: isVirtual,
        UniqueID:  uniqueID,
    }
}

// Get the local IP address
func getLocalIP(logger *log.Logger) string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        logger.Printf("Error getting IP address: %v", err)
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
func checkIfVirtual(logger *log.Logger) bool {
    // Check if we're on Linux, macOS, or Windows
    switch runtime.GOOS {
    case "linux", "darwin":
        info, err := host.Info()
        if err != nil {
            logger.Printf("Error checking if system is virtual: %v", err)
            return false
        }
        return info.VirtualizationSystem != ""
    
    case "windows":
        return checkIfVirtualWindows(logger)
    
    default:
        logger.Printf("Unsupported OS: %s", runtime.GOOS)
        return false
    }
}

// Check if the system is virtual on Windows
func checkIfVirtualWindows(logger *log.Logger) bool {
    // Uncomment the following block if WMI checks are enabled
    /*
    var cs []win32_ComputerSystem
    query := wmi.CreateQuery(&cs, "")
    err := wmi.Query(query, &cs)
    if err != nil {
        logger.Printf("Error checking if system is virtual on Windows: %v", err)
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
    */

    return false
}

func main() {
    // Define the console flag
    consoleFlag := flag.Bool("console", false, "Enable console output for collected data")
    tokenFlag := flag.String("token", "1234567890", "Provide client authentication token")
    flag.Parse()

    var logger *log.Logger
    // Always create a log file to store the output
    file, err := os.OpenFile("console_output.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatalf("Failed to create log file: %v", err)
    }
    defer file.Close()

    if *consoleFlag {
        // Create a multi-writer to write to both stdout and the log file
        multiWriter := io.MultiWriter(os.Stdout, file)
        logger = log.New(multiWriter, "", log.LstdFlags)
        logger.Println("Console flag enabled: true")
    } else {
        logger = log.New(file, "", log.LstdFlags)
    }

    logger.Println("Starting the agent...")

    // Example API key for generating unique client ID
    apiKey := *tokenFlag


    // Register the agent with the mothership and send one-time host information
    hostInfo := gatherOneTimeHostInfo(logger, apiKey)
    registerAgentWithHostInfo(hostInfo, *consoleFlag, logger)

    // Get the current process using the PID
    pid := int32(os.Getpid())
    proc, err := process.NewProcess(pid)
    if err != nil {
        logger.Fatal("Failed to create process object", err)
    }

    // Start the watchdog goroutine
    go watchdog(proc, logger)

    // Buffer to hold collected metrics every 30 seconds
    var metricsBuffer []MetricsData

    // Create a ticker for collecting data every 30 seconds
    dataCollectionTicker := time.NewTicker(30 * time.Second)
    // Create another ticker for pushing data every 5 minutes
    dataPushTicker := time.NewTicker(5 * time.Minute)

    for {
        select {
        case <-dataCollectionTicker.C:
            logger.Println("Data collection tick")
            // Collect metrics every 30 seconds
            cpuPercent, memoryUsage := gatherBasicMetrics(logger)
            processInfo := gatherProcessMetrics(logger)
            netStats, connStats := gatherNetworkMetrics(logger)
            diskStats := gatherDiskMetrics(logger)

            metricsData := MetricsData{
                CPUPercent: cpuPercent,
                MemoryUsage: memoryUsage,
                ProcessInfo: processInfo,
                NetworkStats: netStats,
                ConnStats: connStats,
                DiskIOStats: diskStats,
                Timestamp: time.Now(),
                UniqueID: hostInfo.UniqueID,
            }

            // Add the collected data to the buffer
            metricsBuffer = append(metricsBuffer, metricsData)

            // Log collected data
            logMetrics(metricsData, logger)

        case <-dataPushTicker.C:
            // Push data to the server every 5 minutes
            if len(metricsBuffer) > 0 {
                pushDataToServer(apiKey, metricsBuffer, logger)
                metricsBuffer = nil // Clear the buffer after pushing
            } else {
                logger.Println("No data to push to the server.")
            }
        }
    }
}

// Log collected metrics data
func logMetrics(metricsData MetricsData, logger *log.Logger) {
    logger.Printf("Collected Metrics at %s:\n", metricsData.Timestamp)
    logger.Printf("CPU Usage: %.2f%%\n", metricsData.CPUPercent)
    logger.Printf("Memory Usage: %d bytes\n", metricsData.MemoryUsage)
    logger.Println("Process Information:")
    for _, proc := range metricsData.ProcessInfo {
        logger.Printf("PID: %d, Name: %s, CPU: %.2f%%, Memory: %d bytes\n",
            proc.PID, proc.Name, proc.CPUPercent, proc.MemoryUsage)
    }
    logger.Println("Disk I/O Statistics:")
    for name, io := range metricsData.DiskIOStats {
        logger.Printf("Disk: %s, ReadBytes: %d, WriteBytes: %d\n", name, io.ReadBytes, io.WriteBytes)
    }
    logger.Println("Network I/O Statistics:")
    for _, io := range metricsData.NetworkStats {
        logger.Printf("Interface: %s - Bytes Sent: %d, Bytes Received: %d\n", io.Name, io.BytesSent, io.BytesRecv)
    }
    logger.Println("Active Network Connections:")
    for _, conn := range metricsData.ConnStats {
        logger.Printf("Local Address: %s:%d -> Remote Address: %s:%d\n", conn.LocalAddr, conn.LocalPort, conn.RemoteAddr, conn.RemotePort)
    }
}

func watchdog(proc *process.Process, logger *log.Logger) {
    for {
        // Monitor CPU usage
        cpuPercent, err := proc.CPUPercent()
        if err != nil {
            logger.Printf("Error getting CPU usage: %v", err)
            continue
        }

        // Monitor memory usage
        memInfo, err := proc.MemoryInfo()
        if err != nil {
            logger.Printf("Error getting memory usage: %v", err)
            continue
        }

        // Define thresholds (3-5%)
        if cpuPercent > 30 || float64(memInfo.RSS)/float64(memInfo.VMS)*100 > 5 {
            logger.Println("Pausing data collection due to high resource usage")
            atomic.StoreInt32(&isPaused, 1) // Pause data collection
        } else if cpuPercent < 25 && float64(memInfo.RSS)/float64(memInfo.VMS)*100 < 3 {
            logger.Println("Resuming data collection")
            atomic.StoreInt32(&isPaused, 0) // Resume data collection
        }

        time.Sleep(5 * time.Second) // Check every 5 seconds
    }
}
