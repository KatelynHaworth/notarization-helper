package code_directory

import (
	"fmt"
	"strings"
)

type CodeDirectoryFlag uint32

const (
	CodeDirectoryFlagNone         CodeDirectoryFlag = 0x00000000 /* no flags set */
	CodeDirectoryFlagValid        CodeDirectoryFlag = 0x00000001 /* dynamically valid */
	CodeDirectoryFlagAdhoc        CodeDirectoryFlag = 0x00000002 /* ad hoc signed */
	CodeDirectoryFlagGetTaskAllow CodeDirectoryFlag = 0x00000004 /* has get-task-allow entitlement */
	CodeDirectoryFlagInstaller    CodeDirectoryFlag = 0x00000008 /* has installer entitlement */

	CodeDirectoryFlagForcedLV       CodeDirectoryFlag = 0x00000010 /* Library Validation required by Hardened System Policy */
	CodeDirectoryFlagInvalidAllowed CodeDirectoryFlag = 0x00000020 /* (macOS Only) Page invalidation allowed by task port policy */

	CodeDirectoryFlagHard            CodeDirectoryFlag = 0x00000100 /* don't load invalid pages */
	CodeDirectoryFlagKill            CodeDirectoryFlag = 0x00000200 /* kill process if it becomes invalid */
	CodeDirectoryFlagCheckExpiration CodeDirectoryFlag = 0x00000400 /* force expiration checking */
	CodeDirectoryFlagRestrict        CodeDirectoryFlag = 0x00000800 /* tell dyld to treat restricted */

	CodeDirectoryFlagEnforcement           CodeDirectoryFlag = 0x00001000 /* require enforcement */
	CodeDirectoryFlagRequireLV             CodeDirectoryFlag = 0x00002000 /* require library validation */
	CodeDirectoryFlagEntitlementsValidated CodeDirectoryFlag = 0x00004000 /* code signature permits restricted entitlements */
	CodeDirectoryFlagNVRAMUnrestricted     CodeDirectoryFlag = 0x00008000 /* has com.apple.rootless.restricted-nvram-variables.heritable entitlement */

	CodeDirectoryFlagRuntime      CodeDirectoryFlag = 0x00010000 /* Apply hardened runtime policies */
	CodeDirectoryFlagLinkerSigned CodeDirectoryFlag = 0x00020000 /* Automatically signed by the linker */

	CodeDirectoryFlagExecSetHard        CodeDirectoryFlag = 0x00100000 /* set CodeDirectoryFlagHard on any exec'ed process */
	CodeDirectoryFlagExecSetKill        CodeDirectoryFlag = 0x00200000 /* set CodeDirectoryFlagKill on any exec'ed process */
	CodeDirectoryFlagExecSetEnforcement CodeDirectoryFlag = 0x00400000 /* set CodeDirectoryFlagEnforcement on any exec'ed process */
	CodeDirectoryFlagExecInheritSIP     CodeDirectoryFlag = 0x00800000 /* set CodeDirectoryFlagInstaller on any exec'ed process */

	CodeDirectoryFlagKilled             CodeDirectoryFlag = 0x01000000                          /* was killed by kernel for invalidity */
	CodeDirectoryFlagNoUntrustedHelpers CodeDirectoryFlag = 0x02000000                          /* kernel did not load a non-platform-binary dyld or Rosetta runtime */
	CodeDirectoryFlagDyldPlatform                         = CodeDirectoryFlagNoUntrustedHelpers /* old name */
	CodeDirectoryFlagPlatformBinary     CodeDirectoryFlag = 0x04000000                          /* this is a platform binary */
	CodeDirectoryFlagPlatformPath       CodeDirectoryFlag = 0x08000000                          /* platform binary by the fact of path (osx only) */

	CodeDirectoryFlagDebugged            CodeDirectoryFlag = 0x10000000 /* process is currently or has previously been debugged and allowed to run with invalid pages */
	CodeDirectoryFlagSigned              CodeDirectoryFlag = 0x20000000 /* process has a signature (may have gone invalid) */
	CodeDirectoryFlagDevCode             CodeDirectoryFlag = 0x40000000 /* code is dev signed, cannot be loaded into prod signed code (will go away with rdar://problem/28322552) */
	CodeDirectoryFlagDataVaultController CodeDirectoryFlag = 0x80000000 /* has Data Vault controller entitlement */

	CodeDirectoryFlagAllowedMachO     = CodeDirectoryFlagAdhoc | CodeDirectoryFlagHard | CodeDirectoryFlagKill | CodeDirectoryFlagCheckExpiration | CodeDirectoryFlagRestrict | CodeDirectoryFlagEnforcement | CodeDirectoryFlagRequireLV | CodeDirectoryFlagRuntime | CodeDirectoryFlagLinkerSigned
	CodeDirectoryFlagEntitlementFlags = CodeDirectoryFlagGetTaskAllow | CodeDirectoryFlagInstaller | CodeDirectoryFlagDataVaultController | CodeDirectoryFlagNVRAMUnrestricted
)

var cdFlagToName = map[CodeDirectoryFlag]string{
	CodeDirectoryFlagValid:                 "valid",
	CodeDirectoryFlagAdhoc:                 "adhoc",
	CodeDirectoryFlagGetTaskAllow:          "get_task_allow",
	CodeDirectoryFlagInstaller:             "installer",
	CodeDirectoryFlagForcedLV:              "forced_lv",
	CodeDirectoryFlagInvalidAllowed:        "invalid_allowed",
	CodeDirectoryFlagHard:                  "hard",
	CodeDirectoryFlagKill:                  "kill",
	CodeDirectoryFlagCheckExpiration:       "check_expiration",
	CodeDirectoryFlagRestrict:              "restrict",
	CodeDirectoryFlagEnforcement:           "enforcement",
	CodeDirectoryFlagRequireLV:             "require_lv",
	CodeDirectoryFlagEntitlementsValidated: "entitlements_validated",
	CodeDirectoryFlagNVRAMUnrestricted:     "nvram_unrestricted",
	CodeDirectoryFlagRuntime:               "runtime",
	CodeDirectoryFlagLinkerSigned:          "linker_signed",
	CodeDirectoryFlagExecSetHard:           "set_hard",
	CodeDirectoryFlagExecSetKill:           "set_kill",
	CodeDirectoryFlagExecSetEnforcement:    "set_enforcement",
	CodeDirectoryFlagExecInheritSIP:        "inherit_sip",
	CodeDirectoryFlagKilled:                "killed",
	CodeDirectoryFlagNoUntrustedHelpers:    "no_untrusted_helpers",
	CodeDirectoryFlagPlatformBinary:        "platform_binary",
	CodeDirectoryFlagPlatformPath:          "platform_path",
	CodeDirectoryFlagDebugged:              "debugged",
	CodeDirectoryFlagSigned:                "signed",
	CodeDirectoryFlagDevCode:               "dev_code",
	CodeDirectoryFlagDataVaultController:   "data_vault_controller",
}

func (flags CodeDirectoryFlag) String() string {
	if flags == CodeDirectoryFlagNone {
		return fmt.Sprintf("0x%x (none)", uint32(flags))
	}

	var flagNames []string
	for flag, name := range cdFlagToName {
		if flags&flag == flag {
			flagNames = append(flagNames, name)
		}
	}

	return fmt.Sprintf("0x%x (%s)", uint32(flags), strings.Join(flagNames, ","))
}
