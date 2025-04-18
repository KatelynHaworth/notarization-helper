package super_blob

import "fmt"

// Slot represents a 32-bit unsigned
// integer used to describe the type
// of Code Signature blobs.Blob store
// in a slot of a SuperBlob.
type Slot uint32

const (
	SlotCodeDirectory               Slot = 0x0 /* slot index for CodeDirectory */
	SlotInfo                             = 0x1
	SlotRequirements                     = 0x2
	SlotResourceDir                      = 0x3
	SlotApplication                      = 0x4
	SlotEntitlements                     = 0x5
	SlotDerEntitlements                  = 0x7
	SlotLaunchConstraintSelf             = 0x8
	SlotLaunchConstraintParent           = 0x9
	SlotLaunchConstraintResponsible      = 0xa
	SlotSignature                        = 0x10000 /* CMS Signature */
	SlotIdentification                   = 0x10001
	SlotTicket                           = 0x10002

	SlotAlternativeCodeDirectories    Slot = 0x1000 /* first alternate CodeDirectory, if any */
	SlotAlternativeCodeDirectoryMax   Slot = 0x5    /* max number of alternate CD slots */
	SlotAlternativeCodeDirectoryLimit Slot = SlotAlternativeCodeDirectories + SlotAlternativeCodeDirectoryMax
)

var (
	slotToName = map[Slot]string{
		SlotCodeDirectory:               "CSSLOT_CODEDIRECTORY",
		SlotInfo:                        "CSSLOT_INFOSLOT",
		SlotRequirements:                "CSSLOT_REQUIREMENTS",
		SlotResourceDir:                 "CSSLOT_RESOURCEDIR",
		SlotApplication:                 "CSSLOT_APPLICATION",
		SlotEntitlements:                "CSSLOT_ENTITLEMENTS",
		SlotDerEntitlements:             "CSSLOT_DER_ENTITLEMENTS",
		SlotLaunchConstraintSelf:        "CSSLOT_LAUNCH_CONSTRAINT_SELF",
		SlotLaunchConstraintParent:      "CSSLOT_LAUNCH_CONSTRAINT_PARENT",
		SlotLaunchConstraintResponsible: "CSSLOT_LAUNCH_CONSTRAINT_RESPONSIBLE",
		SlotSignature:                   "CSSLOT_SIGNATURESLOT",
		SlotIdentification:              "CSSLOT_IDENTIFICATIONSLOT",
		SlotTicket:                      "CSSLOT_TICKETSLOT",
	}
)

// String returns the name of a Slot
// if it is known, otherwise the hex
// encoding is returned.
func (slot Slot) String() string {
	if name, known := slotToName[slot]; known {
		return name
	}

	if SlotAlternativeCodeDirectories <= slot && slot < SlotAlternativeCodeDirectoryLimit {
		return "CSSLOT_ALTERNATE_CODEDIRECTORY"
	}

	return fmt.Sprintf("0x%x", uint32(slot))
}
