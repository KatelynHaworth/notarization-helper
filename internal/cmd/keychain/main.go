package main

/*
#cgo LDFLAGS: -framework CoreFoundation -framework SecurityFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
*/
import "C"
import (
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"

	"github.com/LiamHaworth/macos-golang/coreFoundation"
)

func main() {
	label := toCFString("FamilyZoneNotarize")
	if label == 0 {
		panic("create label string")
	}
	defer C.CFRelease(C.CFTypeRef(label))

	query := mapToCFDictionary(map[C.CFTypeRef]C.CFTypeRef{
		C.CFTypeRef(C.kSecClass):      C.CFTypeRef(C.kSecClassGenericPassword),
		C.CFTypeRef(C.kSecAttrLabel):  C.CFTypeRef(label),
		C.CFTypeRef(C.kSecMatchLimit): C.CFTypeRef(C.kSecMatchLimitOne),
		C.CFTypeRef(C.kSecReturnData): C.CFTypeRef(C.kCFBooleanTrue),
	})
	if query == 0 {
		panic("failed to make query dict")
	}
	defer C.CFRelease(C.CFTypeRef(query))

	var (
		resultRef   C.CFTypeRef
		secOsStatus C.OSStatus
	)

	secOsStatus = C.SecItemCopyMatching(query, &resultRef)
	if secOsStatus != 0 {
		stringRef := C.SecCopyErrorMessageString(secOsStatus, nil)
		if msg, err := coreFoundation.FromCFString(coreFoundation.StringRef(stringRef)); err != nil {
			panic(err)
		} else {
			panic(msg)
		}
	}
	defer C.CFRelease(resultRef)

	description := C.CFCopyTypeIDDescription(C.CFGetTypeID(resultRef))
	if description == 0 {
		panic("get type description")
	}
	defer C.CFRelease(C.CFTypeRef(description))

	fmt.Println(fromCFString(description))

	data, err := fromCFData(C.CFDataRef(resultRef))
	if err != nil {
		panic("copy result data")
	}
	fmt.Println(hex.Dump(data))
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

	/*
		Check if the CFStringRef is a plain C
		string pointer, if so just convert directly
		from the pointer
	*/
	if p := C.CFStringGetCStringPtr(ref, C.kCFStringEncodingUTF8); p != nil {
		return C.GoString(p), nil
	}

	/*
		The CFStringRef isn't a plain CString so
		it has to be converted by copying the bytes
		and crafting a new Golang string
	*/
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

func fromCFData(ref C.CFDataRef) ([]byte, error) {
	if ref == 0 {
		return []byte{}, nil
	}

	size := C.CFDataGetLength(ref)
	data := make([]byte, size)

	C.CFDataGetBytes(ref, C.CFRange{0, size}, (*C.UInt8)(&data[0]))

	return data, nil
}
