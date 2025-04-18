//go:build darwin && !disable_keychain

package config

/*
#cgo LDFLAGS: -framework CoreFoundation -framework SecurityFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

func getPasswordFromKeychain(label string) (string, error) {
	cfLabel := toCFString(label)
	if cfLabel == 0 {
		return "", errors.New("covert label to CFString")
	}
	defer C.CFRelease(C.CFTypeRef(cfLabel))

	query := mapToCFDictionary(map[C.CFTypeRef]C.CFTypeRef{
		C.CFTypeRef(C.kSecClass):      C.CFTypeRef(C.kSecClassGenericPassword),
		C.CFTypeRef(C.kSecAttrLabel):  C.CFTypeRef(cfLabel),
		C.CFTypeRef(C.kSecMatchLimit): C.CFTypeRef(C.kSecMatchLimitOne),
		C.CFTypeRef(C.kSecReturnData): C.CFTypeRef(C.kCFBooleanTrue),
	})
	if query == 0 {
		return "", errors.New("create query CFDictionary")
	}
	defer C.CFRelease(C.CFTypeRef(query))

	var resultRef C.CFTypeRef
	if status := osStatus(C.SecItemCopyMatching(query, &resultRef)); status != 0 {
		return "", fmt.Errorf("copy item from keychain: %w", status)
	}
	defer C.CFRelease(resultRef)

	return string(fromCFData(C.CFDataRef(resultRef))), nil
}

func mapToCFDictionary(gomap map[C.CFTypeRef]C.CFTypeRef) C.CFDictionaryRef {
	var (
		n      = len(gomap)
		keys   = make([]unsafe.Pointer, 0, n)
		values = make([]unsafe.Pointer, 0, n)
	)

	for k, v := range gomap {
		keys = append(keys, unsafe.Pointer(k))
		values = append(values, unsafe.Pointer(v))
	}

	return C.CFDictionaryCreate(0, &keys[0], &values[0], C.CFIndex(n), nil, nil)
}

func toCFString(s string) C.CFStringRef {
	data := make([]byte, len(s))
	copy(data, s)

	return C.CFStringCreateWithBytes(0, *(**C.UInt8)(unsafe.Pointer(&data)), C.CFIndex(len(s)), C.kCFStringEncodingUTF8, 0)
}

func fromCFString(ref C.CFStringRef) (string, error) {
	if ref == 0 {
		return "", nil
	}

	if p := C.CFStringGetCStringPtr(ref, C.kCFStringEncodingUTF8); p != nil {
		return C.GoString(p), nil
	}

	length := C.CFStringGetLength(ref)
	if length == 0 {
		// String is already empty
		return "", nil
	}

	bufferLength := C.CFStringGetMaximumSizeForEncoding(length, C.kCFStringEncodingUTF8)
	if bufferLength == 0 {
		return "", errors.New("string is not encoded with UTF-8, unable to convert to Golang string")
	}

	var bufferFilled C.CFIndex
	buffer := make([]byte, bufferLength)

	C.CFStringGetBytes(ref, C.CFRange{0, length}, C.kCFStringEncodingUTF8, C.UInt8(0), C.false, (*C.UInt8)(&buffer[0]), bufferLength, &bufferFilled)

	return unsafe.String(&buffer[0], int(bufferFilled)), nil
}

func fromCFData(ref C.CFDataRef) []byte {
	if ref == 0 {
		return []byte{}
	}

	size := C.CFDataGetLength(ref)
	data := make([]byte, size)

	C.CFDataGetBytes(ref, C.CFRange{0, size}, (*C.UInt8)(&data[0]))
	return data
}

type osStatus C.OSStatus

func (status osStatus) Error() string {
	stringRef := C.SecCopyErrorMessageString(C.OSStatus(status), nil)
	defer C.CFRelease(C.CFTypeRef(stringRef))

	msg, err := fromCFString(stringRef)
	if err != nil {
		return fmt.Sprintf("0x%x", status)
	}

	return msg
}
