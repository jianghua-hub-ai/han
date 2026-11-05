//go:build !windows

package main

import "os"

func openMaintenanceInput(*os.File) (maintenanceInputOps, error) {
	return nil, maintenanceFail("maintenance.unsupported_platform", nil)
}
