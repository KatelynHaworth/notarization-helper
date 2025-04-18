package hash

import (
	"fmt"
	"hash"
	"math"
	"slices"
	"strconv"
)

type (
	Type uint8

	Metadata struct {
		Priority   int
		Name       string
		Size       uint8
		DigestSize uint8
		New        func() hash.Hash

		inUse bool
	}
)

var registry [math.MaxUint8]Metadata

func RegisterHashType(id uint8, metadata Metadata) Type {
	if meta := registry[id]; meta.inUse == true {
		panic(fmt.Sprintf("hash type 0x%x already registred to '%s'", id, meta.Name))
	}

	i := slices.IndexFunc(registry[:], func(meta Metadata) bool {
		return meta.inUse && meta.Priority == metadata.Priority
	})

	if i != -1 {
		panic(fmt.Errorf("priority conflict with hash type '%s'", registry[i].Name))
	}

	metadata.inUse = true
	registry[id] = metadata

	return Type(id)
}

// Priority returns the "rank" of this hash
// type to allow the "best" hash type to be
// found in a set of hash types.
//
// A priority of zero or less means this
// hash MUST not be used.
func (hashType Type) Priority() int {
	if meta := registry[hashType]; meta.inUse {
		return meta.Priority
	}

	return -1
}

func (hashType Type) String() string {
	if meta := registry[hashType]; meta.inUse {
		return meta.Name
	}

	return strconv.Itoa(int(hashType))
}

func (hashType Type) Size() uint8 {
	if meta := registry[hashType]; meta.inUse {
		return meta.Size
	}

	return 0
}

func (hashType Type) DigestSize() uint8 {
	if meta := registry[hashType]; meta.inUse {
		return meta.DigestSize
	}

	return 0
}

func (hashType Type) New() hash.Hash {
	if meta := registry[hashType]; meta.inUse {
		return meta.New()
	}

	return nil
}

func (hashType Type) Valid() bool {
	if hashType == TypeInvalid {
		return false
	}

	return registry[hashType].inUse
}
