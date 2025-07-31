//go:build windows
// +build windows

package config

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// Windows-specific implementation using Credential Manager

var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procCredRead   = advapi32.NewProc("CredReadW")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

const (
	CRED_TYPE_GENERIC          = 1
	CRED_PERSIST_LOCAL_MACHINE = 2
)

// CREDENTIAL structure for Windows Credential Manager
type CREDENTIAL struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

// isPlatformStorageAvailable checks if Windows Credential Manager is available
func isPlatformStorageAvailable() bool {
	// Check if we can load the required DLL procedures
	return procCredRead.Find() == nil
}

// getPlatformAPIKey retrieves API key from Windows Credential Manager
func getPlatformAPIKey(provider string) (string, error) {
	targetName := fmt.Sprintf("%s:%s", GetServiceName(), provider)
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return "", err
	}

	var credPtr *CREDENTIAL
	ret, _, err := procCredRead.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		CRED_TYPE_GENERIC,
		0,
		uintptr(unsafe.Pointer(&credPtr)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to read credential: %v", err)
	}

	defer procCredFree.Call(uintptr(unsafe.Pointer(credPtr)))

	// Extract the credential blob as string
	if credPtr.CredentialBlobSize == 0 {
		return "", errors.New("empty credential")
	}

	key := C.GoStringN((*C.char)(unsafe.Pointer(credPtr.CredentialBlob)), C.int(credPtr.CredentialBlobSize))
	return key, nil
}

// setPlatformAPIKey stores API key in Windows Credential Manager
func setPlatformAPIKey(provider string, key string) error {
	targetName := fmt.Sprintf("%s:%s", GetServiceName(), provider)
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return err
	}

	comment := fmt.Sprintf("CODA API key for %s", provider)
	commentPtr, err := syscall.UTF16PtrFromString(comment)
	if err != nil {
		return err
	}

	cred := CREDENTIAL{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetNamePtr,
		Comment:            commentPtr,
		CredentialBlobSize: uint32(len(key)),
		CredentialBlob:     &[]byte(key)[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
	}

	ret, _, err := procCredWrite.Call(
		uintptr(unsafe.Pointer(&cred)),
		0,
	)

	if ret == 0 {
		return fmt.Errorf("failed to write credential: %v", err)
	}

	return nil
}

// deletePlatformAPIKey removes API key from Windows Credential Manager
func deletePlatformAPIKey(provider string) error {
	targetName := fmt.Sprintf("%s:%s", GetServiceName(), provider)
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return err
	}

	ret, _, err := procCredDelete.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		CRED_TYPE_GENERIC,
		0,
	)

	if ret == 0 {
		// Check if credential doesn't exist
		if err == syscall.ERROR_NOT_FOUND {
			return nil // Nothing to delete
		}
		return fmt.Errorf("failed to delete credential: %v", err)
	}

	return nil
}

// listPlatformProviders lists providers in Windows Credential Manager
func listPlatformProviders() ([]string, error) {
	// TODO: Implement credential enumeration
	// For now, return empty list
	return []string{}, nil
}
