//go:build windows

package main

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const maintenanceMessagePipe = 4

type maintenanceWindowsInput struct {
	file     *os.File
	handle   syscall.Handle
	peek     *syscall.Proc
	dll      *syscall.DLL
	once     sync.Once
	closeErr error
}

func openMaintenanceInput(input *os.File) (maintenanceInputOps, error) {
	if input == nil {
		return nil, maintenanceFail("maintenance.invalid_request", nil)
	}
	handle := syscall.Handle(input.Fd())
	kind, err := syscall.GetFileType(handle)
	if err != nil {
		return nil, maintenanceFail("maintenance.failed", err)
	}
	if kind != syscall.FILE_TYPE_PIPE {
		return nil, maintenanceFail("maintenance.invalid_request", nil)
	}
	// Go's syscall package registers this exact name for the System32 loader.
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return nil, maintenanceFail("maintenance.failed", err)
	}
	ops, err := prepareMaintenanceWindowsInput(input, handle, dll)
	if err != nil {
		_ = dll.Release()
		return nil, err
	}
	return ops, nil
}

func prepareMaintenanceWindowsInput(input *os.File, handle syscall.Handle, dll *syscall.DLL) (*maintenanceWindowsInput, error) {
	info, err := dll.FindProc("GetNamedPipeInfo")
	if err != nil {
		return nil, maintenanceFail("maintenance.failed", err)
	}
	peek, err := dll.FindProc("PeekNamedPipe")
	if err != nil {
		return nil, maintenanceFail("maintenance.failed", err)
	}
	var flags uint32
	ok, _, callErr := info.Call(uintptr(handle), uintptr(unsafe.Pointer(&flags)), 0, 0, 0)
	runtime.KeepAlive(input)
	if ok == 0 {
		return nil, maintenanceFail("maintenance.failed", callErr)
	}
	if flags&maintenanceMessagePipe != 0 {
		return nil, maintenanceFail("maintenance.invalid_request", nil)
	}
	return &maintenanceWindowsInput{file: input, handle: handle, peek: peek, dll: dll}, nil
}

func (input *maintenanceWindowsInput) Peek() (uint32, bool, error) {
	var available uint32
	ok, _, err := input.peek.Call(uintptr(input.handle), 0, 0, 0, uintptr(unsafe.Pointer(&available)), 0)
	runtime.KeepAlive(input.file)
	if ok != 0 {
		return available, false, nil
	}
	if err == syscall.ERROR_BROKEN_PIPE || err == syscall.Errno(233) {
		return 0, true, nil
	}
	return 0, false, err
}

func (input *maintenanceWindowsInput) ReadAvailable(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, maintenanceFail("maintenance.failed", nil)
	}
	var read uint32
	err := syscall.ReadFile(input.handle, buffer, &read, nil)
	runtime.KeepAlive(input.file)
	return int(read), err
}

func (input *maintenanceWindowsInput) Close() error {
	input.once.Do(func() {
		input.closeErr = input.file.Close()
		if err := input.dll.Release(); input.closeErr == nil {
			input.closeErr = err
		}
	})
	return input.closeErr
}
