// +build windows

package metrics

import (
    "github.com/StackExchange/wmi"
    "fmt"
    "log"
    // other imports...
)

type win32_ComputerSystem struct {
    Manufacturer string
    Model        string
}

func getWindowsMetrics() bool {
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
