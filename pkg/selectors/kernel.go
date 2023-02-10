// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

package selectors

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/cilium/tetragon/api/v1/tetragon"
	"github.com/cilium/tetragon/pkg/generictypes"
	"github.com/cilium/tetragon/pkg/k8s/apis/cilium.io/v1alpha1"
	"github.com/cilium/tetragon/pkg/kernels"
	"github.com/cilium/tetragon/pkg/reader/namespace"
)

const (
	ActionTypePost       = 0
	ActionTypeFollowFd   = 1
	ActionTypeSigKill    = 2
	ActionTypeUnfollowFd = 3
	ActionTypeOverride   = 4
	ActionTypeCopyFd     = 5
	ActionTypeGetUrl     = 6
	ActionTypeDnsLookup  = 7
)

var actionTypeTable = map[string]uint32{
	"post":       ActionTypePost,
	"followfd":   ActionTypeFollowFd,
	"unfollowfd": ActionTypeUnfollowFd,
	"sigkill":    ActionTypeSigKill,
	"override":   ActionTypeOverride,
	"copyfd":     ActionTypeCopyFd,
	"geturl":     ActionTypeGetUrl,
	"dnslookup":  ActionTypeDnsLookup,
}

var actionTypeStringTable = map[uint32]string{
	ActionTypePost:       "post",
	ActionTypeFollowFd:   "followfd",
	ActionTypeUnfollowFd: "unfollowfd",
	ActionTypeSigKill:    "sigkill",
	ActionTypeOverride:   "override",
	ActionTypeCopyFd:     "copyfd",
	ActionTypeGetUrl:     "geturl",
	ActionTypeDnsLookup:  "dnslookup",
}

func MatchActionSigKill(spec interface{}) bool {
	var sels []v1alpha1.KProbeSelector
	switch s := spec.(type) {
	case *v1alpha1.KProbeSpec:
		sels = s.Selectors
	case *v1alpha1.TracepointSpec:
		sels = s.Selectors
	default:
		return false
	}

	for _, s := range sels {
		for _, act := range s.MatchActions {
			if strings.ToLower(act.Action) == actionTypeStringTable[ActionTypeSigKill] {
				return true
			}
		}
	}
	return false
}

func GetUrls(spec *v1alpha1.KProbeSpec) []string {
	var urls []string
	sels := spec.Selectors
	for _, s := range sels {
		for _, act := range s.MatchActions {
			if strings.ToLower(act.Action) == actionTypeStringTable[ActionTypeGetUrl] {
				if len(act.ArgUrl) > 0 {
					urls = append(urls, act.ArgUrl)
				}
			}
		}
	}
	return urls
}

func GetDnsFQDNs(spec *v1alpha1.KProbeSpec) []string {
	var fqdns []string
	sels := spec.Selectors
	for _, s := range sels {
		for _, act := range s.MatchActions {
			if strings.ToLower(act.Action) == actionTypeStringTable[ActionTypeDnsLookup] {
				if len(act.ArgFqdn) > 0 {
					fqdns = append(fqdns, act.ArgFqdn)
				}
			}
		}
	}
	return fqdns
}

const (
	namespaceTypeUts             = 0
	namespaceTypeIpc             = 1
	namespaceTypeMnt             = 2
	namespaceTypePid             = 3
	namespaceTypePidForChildren  = 4
	namespaceTypeNet             = 5
	namespaceTypeTime            = 6
	namespaceTypeTimeForChildren = 7
	namespaceTypeCgroup          = 8
	namespaceTypeUser            = 9
)

var namespaceTypeTable = map[string]uint32{
	"uts":             namespaceTypeUts,
	"ipc":             namespaceTypeIpc,
	"mnt":             namespaceTypeMnt,
	"pid":             namespaceTypePid,
	"pidforchildren":  namespaceTypePidForChildren,
	"net":             namespaceTypeNet,
	"time":            namespaceTypeTime,
	"timeforchildren": namespaceTypeTimeForChildren,
	"cgroup":          namespaceTypeCgroup,
	"user":            namespaceTypeUser,
}

const (
	capsPermitted   = 0
	capsEffective   = 1
	capsInheritable = 2
)

var capabilitiesTypeTable = map[string]uint32{
	"effective":   capsEffective,
	"inheritable": capsInheritable,
	"permitted":   capsPermitted,
}

const (
	argTypeUrl  = 18
	argTypeFqdn = 19
)

var argTypeTable = map[string]uint32{
	"int":        generictypes.GenericIntType,
	"uint32":     generictypes.GenericU32Type,
	"int32":      generictypes.GenericS32Type,
	"uint64":     generictypes.GenericU64Type,
	"int64":      generictypes.GenericS64Type,
	"char_buf":   generictypes.GenericCharBuffer,
	"char_iovec": generictypes.GenericCharIovec,
	"sizet":      generictypes.GenericSizeType,
	"skb":        generictypes.GenericSkbType,
	"string":     generictypes.GenericStringType,
	"fd":         generictypes.GenericFdType,
	"file":       generictypes.GenericFileType,
	"sock":       generictypes.GenericSockType,
	"url":        argTypeUrl,
	"fqdn":       argTypeFqdn,
}

var argTypeStringTable = map[uint32]string{
	generictypes.GenericIntType:    "int",
	generictypes.GenericU32Type:    "uint32",
	generictypes.GenericS32Type:    "int32",
	generictypes.GenericU64Type:    "uint64",
	generictypes.GenericS64Type:    "int64",
	generictypes.GenericCharBuffer: "char_buf",
	generictypes.GenericCharIovec:  "char_iovec",
	generictypes.GenericSizeType:   "sizet",
	generictypes.GenericSkbType:    "skb",
	generictypes.GenericStringType: "string",
	generictypes.GenericFdType:     "fd",
	generictypes.GenericFileType:   "file",
	generictypes.GenericSockType:   "sock",
	argTypeUrl:                     "url",
	argTypeFqdn:                    "fqdn",
}

const (
	selectorOpGT  = 1
	selectorOpLT  = 2
	selectorOpEQ  = 3
	selectorOpNEQ = 4
	// Pid and Namespace ops
	selectorOpIn    = 5
	selectorOpNotIn = 6
	// String ops
	selectorOpPrefix  = 8
	selectorOpPostfix = 9
	// Map ops
	selectorInMap    = 10
	selectorNotInMap = 11
)

func selectorOp(op string) (uint32, error) {
	switch op {
	case "gt":
		return selectorOpGT, nil
	case "lt":
		return selectorOpLT, nil
	case "eq", "Equal":
		return selectorOpEQ, nil
	case "neq":
		return selectorOpNEQ, nil
	case "In":
		return selectorOpIn, nil
	case "NotIn":
		return selectorOpNotIn, nil
	case "prefix", "Prefix":
		return selectorOpPrefix, nil
	case "postfix", "Postfix":
		return selectorOpPostfix, nil
	case "InMap":
		return selectorInMap, nil
	case "NotInMap":
		return selectorNotInMap, nil
	}

	return 0, fmt.Errorf("Unknown op '%s'", op)
}

const (
	pidNamespacePid = 0x1
	pidFollowForks  = 0x2
)

func pidSelectorFlags(pid *v1alpha1.PIDSelector) uint32 {
	flags := uint32(0)

	if pid.IsNamespacePID {
		flags |= pidNamespacePid
	}
	if pid.FollowForks {
		flags |= pidFollowForks
	}
	return flags
}

func pidSelectorValue(pid *v1alpha1.PIDSelector) ([]byte, uint32) {
	b := make([]byte, len(pid.Values)*4)

	for i, v := range pid.Values {
		off := i * 4
		binary.LittleEndian.PutUint32(b[off:], v)
	}
	return b, uint32(len(b))
}

func parseMatchPid(k *KernelSelectorState, pid *v1alpha1.PIDSelector) error {
	op, err := selectorOp(pid.Operator)
	if err != nil {
		return fmt.Errorf("matchpid error: %w", err)
	}
	WriteSelectorUint32(k, op)

	flags := pidSelectorFlags(pid)
	WriteSelectorUint32(k, flags)

	value, size := pidSelectorValue(pid)
	WriteSelectorUint32(k, size/4)
	WriteSelectorByteArray(k, value, size)
	return nil
}

func parseMatchPids(k *KernelSelectorState, matchPids []v1alpha1.PIDSelector) error {
	loff := AdvanceSelectorLength(k)
	for _, p := range matchPids {
		if err := parseMatchPid(k, &p); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func kprobeArgType(t string) uint32 {
	return argTypeTable[t]
}

func ArgTypeToString(t uint32) string {
	return argTypeStringTable[t]
}

func argSelectorType(arg *v1alpha1.ArgSelector, sig []v1alpha1.KProbeArg) (uint32, error) {
	for _, s := range sig {
		if arg.Index == s.Index {
			// TBD: We shouldn't get this far with invalid KProbe args
			// KProbe args have already been validated
			return kprobeArgType(s.Type), nil
		}
	}
	return 0, fmt.Errorf("argFilter for unknown index")
}

func writeMatchValuesInMap(k *KernelSelectorState, values []string, ty uint32) error {
	mid, m := k.newValueMap()
	for _, v := range values {
		var val [8]byte
		switch ty {
		case generictypes.GenericS64Type, generictypes.GenericIntType:
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("MatchArgs value %s invalid: %x", v, err)
			}
			binary.LittleEndian.PutUint64(val[:], uint64(i))
		case generictypes.GenericU64Type:
			i, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return fmt.Errorf("MatchArgs value %s invalid: %x", v, err)
			}
			binary.LittleEndian.PutUint64(val[:], uint64(i))
		default:
			return fmt.Errorf("Unknown type: %d", ty)
		}
		m[val] = struct{}{}

	}
	// write the map id into the selector
	WriteSelectorUint32(k, mid)
	return nil
}

func writeMatchValues(k *KernelSelectorState, values []string, ty uint32) error {
	for _, v := range values {
		switch ty {
		case generictypes.GenericFdType, generictypes.GenericFileType:
			value, size := ArgSelectorValue(v)
			WriteSelectorUint32(k, size)
			WriteSelectorByteArray(k, value, size)
		case generictypes.GenericStringType, generictypes.GenericCharBuffer:
			value, size := ArgSelectorValue(v)
			WriteSelectorUint32(k, size)
			WriteSelectorByteArray(k, value, size)
		case generictypes.GenericS32Type, generictypes.GenericIntType, generictypes.GenericSizeType:
			i, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				return fmt.Errorf("MatchArgs value %s invalid: %x", v, err)
			}
			WriteSelectorInt32(k, int32(i))
		case generictypes.GenericU32Type:
			i, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return fmt.Errorf("MatchArgs value %s invalid: %x", v, err)
			}
			WriteSelectorUint32(k, uint32(i))
		case generictypes.GenericS64Type:
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("MatchArgs value %s invalid: %x", v, err)
			}
			WriteSelectorInt64(k, int64(i))
		case generictypes.GenericU64Type:
			i, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return fmt.Errorf("MatchArgs value %s invalid: %x", v, err)
			}
			WriteSelectorUint64(k, uint64(i))
		case generictypes.GenericSockType, generictypes.GenericSkbType, generictypes.GenericCharIovec:
			return fmt.Errorf("MatchArgs values %s unsupported", v)
		}
	}
	return nil
}

func parseMatchArg(k *KernelSelectorState, arg *v1alpha1.ArgSelector, sig []v1alpha1.KProbeArg) error {
	WriteSelectorUint32(k, arg.Index)

	op, err := selectorOp(arg.Operator)
	if err != nil {
		return fmt.Errorf("matcharg error: %w", err)
	}
	WriteSelectorUint32(k, op)
	moff := AdvanceSelectorLength(k)
	ty, err := argSelectorType(arg, sig)
	if err != nil {
		return fmt.Errorf("argSelector error: %w", err)
	}
	WriteSelectorUint32(k, ty)
	switch op {
	case selectorInMap, selectorNotInMap:
		err := writeMatchValuesInMap(k, arg.Values, ty)
		if err != nil {
			return fmt.Errorf("writeMatchValuesInMap error: %w", err)
		}
	default:
		err = writeMatchValues(k, arg.Values, ty)
		if err != nil {
			return fmt.Errorf("writeMatchValues error: %w", err)
		}
	}

	WriteSelectorLength(k, moff)
	return nil
}
func parseMatchArgs(k *KernelSelectorState, args []v1alpha1.ArgSelector, sig []v1alpha1.KProbeArg) error {
	loff := AdvanceSelectorLength(k)
	for _, a := range args {
		if err := parseMatchArg(k, &a, sig); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func parseMatchAction(k *KernelSelectorState, action *v1alpha1.ActionSelector) error {
	act, ok := actionTypeTable[strings.ToLower(action.Action)]
	if !ok {
		return fmt.Errorf("parseMatchAction: ActionType %s unknown", action.Action)
	}
	WriteSelectorUint32(k, act)
	switch act {
	case ActionTypeFollowFd, ActionTypeCopyFd:
		WriteSelectorUint32(k, action.ArgFd)
		WriteSelectorUint32(k, action.ArgName)
	case ActionTypeOverride:
		WriteSelectorInt32(k, action.ArgError)
	case ActionTypeGetUrl:
		WriteSelectorByteArray(k, []byte(action.ArgUrl), uint32(len(action.ArgUrl)))
	case ActionTypeDnsLookup:
		WriteSelectorByteArray(k, []byte(action.ArgFqdn), uint32(len(action.ArgFqdn)))
	}
	return nil
}

func parseMatchActions(k *KernelSelectorState, actions []v1alpha1.ActionSelector) error {
	loff := AdvanceSelectorLength(k)
	for _, a := range actions {
		if err := parseMatchAction(k, &a); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func namespaceSelectorValue(ns *v1alpha1.NamespaceSelector, nstype string) ([]byte, uint32, error) {
	b := make([]byte, len(ns.Values)*4)

	if len(ns.Values) > 4 { // 4 should match the number of iterations in selector_match() in pfilter.h
		return b, 0, fmt.Errorf("matchNamespace supports up to 4 values per filter (current number of values is %d)", len(ns.Values))
	}
	for i, v := range ns.Values {
		val, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			// the only case that we can accept and is not a uint32 is "<host_ns>"
			// in this case we should replace that with the approproate value
			if v == "host_ns" {
				val = uint64(namespace.GetHostNsInode(nstype))
			} else {
				return b, 0, fmt.Errorf("Values for matchNamespace can only be numeric or \"host_ns\". (%w)", err)
			}
		}

		off := i * 4
		binary.LittleEndian.PutUint32(b[off:], uint32(val))
	}
	return b, uint32(len(b)), nil
}

func parseMatchNamespace(k *KernelSelectorState, action *v1alpha1.NamespaceSelector) error {
	nsstr := strings.ToLower(action.Namespace)
	// write namespace type
	ns, ok := namespaceTypeTable[nsstr]
	if !ok {
		return fmt.Errorf("parseMatchNamespace: actionType %s unknown", action.Namespace)
	}
	WriteSelectorUint32(k, ns)

	// write operator
	op, err := selectorOp(action.Operator)
	if err != nil {
		return fmt.Errorf("matchNamespace error: %w", err)
	}
	if (op != selectorOpIn) && (op != selectorOpNotIn) {
		return fmt.Errorf("matchNamespace supports only In and NotIn operators")
	}
	WriteSelectorUint32(k, op)

	// write values
	value, size, err := namespaceSelectorValue(action, nsstr)
	if err != nil {
		return err
	}
	WriteSelectorUint32(k, size/4)
	WriteSelectorByteArray(k, value, size)
	return nil
}

func parseMatchNamespaces(k *KernelSelectorState, actions []v1alpha1.NamespaceSelector) error {
	max_nactions := 4 // 4 should match the value of the NUM_NS_FILTERS_SMALL in pfilter.h
	if kernels.EnableLargeProgs() {
		max_nactions = 10 // 10 should match the value of ns_max_types in hubble_msg.h
	}
	if len(actions) > max_nactions {
		return fmt.Errorf("matchNamespace supports up to %d filters (current number of filters is %d)", max_nactions, len(actions))
	}
	loff := AdvanceSelectorLength(k)
	// maybe write the number of namespace matches
	for _, a := range actions {
		if err := parseMatchNamespace(k, &a); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func parseMatchNamespaceChange(k *KernelSelectorState, action *v1alpha1.NamespaceChangesSelector) error {
	// write operator
	op, err := selectorOp(action.Operator)
	if err != nil {
		return fmt.Errorf("matchNamespaceChanges error: %w", err)
	}
	if (op != selectorOpIn) && (op != selectorOpNotIn) {
		return fmt.Errorf("matchNamespaceChanges supports only In and NotIn operators")
	}
	WriteSelectorUint32(k, op)

	// process and write values
	nsval := uint32(0)
	for _, v := range action.Values {
		nsstr := strings.ToLower(v)
		ns, ok := namespaceTypeTable[nsstr]
		if !ok {
			return fmt.Errorf("parseMatchNamespaceChange: actionType %s unknown", v)
		}
		nsval |= (1 << ns)
	}
	WriteSelectorUint32(k, nsval)
	return nil
}

func parseMatchNamespaceChanges(k *KernelSelectorState, actions []v1alpha1.NamespaceChangesSelector) error {
	if len(actions) > 1 {
		return fmt.Errorf("matchNamespaceChanges supports only a single filter (current number of filters is %d)", len(actions))
	}
	if (len(actions) == 1) && (kernels.EnableLargeProgs() == false) {
		return fmt.Errorf("matchNamespaceChanges is only supported in kernels >= 5.3")
	}
	loff := AdvanceSelectorLength(k)
	// maybe write the number of namespace matches
	for _, a := range actions {
		if err := parseMatchNamespaceChange(k, &a); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func parseMatchCaps(k *KernelSelectorState, action *v1alpha1.CapabilitiesSelector) error {
	// type
	tystr := strings.ToLower(action.Type)
	ty, ok := capabilitiesTypeTable[tystr]
	if !ok {
		return fmt.Errorf("parseMatchCapability: actionType %s unknown", action.Type)
	}
	WriteSelectorUint32(k, ty)

	// operator
	op, err := selectorOp(action.Operator)
	if err != nil {
		return fmt.Errorf("matchCapabilities error: %w", err)
	}
	if (op != selectorOpIn) && (op != selectorOpNotIn) {
		return fmt.Errorf("matchCapabilities supports only In and NotIn operators")
	}
	WriteSelectorUint32(k, op)

	// isnamespacecapability
	isns := uint32(0) // false by default
	if action.IsNamespaceCapability {
		// If IsNamespaceCapability == true will try to match the capabilities
		//     only when current_user_namespace != host_user_namespace.
		// If IsNamespaceCapability == false will try to match the capabilities
		//     ignoring the user_namespace value.
		// To implement this we pass the "/proc/1/ns/user" value as the host
		// user namespace to compare with that inside the kernel.
		isns = namespace.GetPidNsInode(1, "user")
	}
	WriteSelectorUint32(k, isns)

	// values
	caps := uint64(0)
	for _, v := range action.Values {
		valstr := strings.ToUpper(v)
		c, ok := tetragon.CapabilitiesType_value[valstr]
		if !ok {
			return fmt.Errorf("parseMatchCapability: value %s unknown", valstr)
		}
		caps |= (1 << c)
	}
	WriteSelectorUint64(k, caps)

	return nil
}

func parseMatchCapabilities(k *KernelSelectorState, actions []v1alpha1.CapabilitiesSelector) error {
	loff := AdvanceSelectorLength(k)
	for _, a := range actions {
		if err := parseMatchCaps(k, &a); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func parseMatchCapabilityChanges(k *KernelSelectorState, actions []v1alpha1.CapabilitiesSelector) error {
	loff := AdvanceSelectorLength(k)
	for _, a := range actions {
		if err := parseMatchCaps(k, &a); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func parseMatchBinary(k *KernelSelectorState, index uint32, b *v1alpha1.BinarySelector) error {
	op, err := selectorOp(b.Operator)
	if err != nil {
		return fmt.Errorf("matchpid error: %w", err)
	}
	WriteSelectorUint32(k, op)
	WriteSelectorUint32(k, index)
	WriteSelectorUint32(k, index)
	WriteSelectorUint32(k, index)
	WriteSelectorUint32(k, index)
	return nil
}

func parseMatchBinaries(k *KernelSelectorState, binarys []v1alpha1.BinarySelector) error {
	loff := AdvanceSelectorLength(k)
	if len(binarys) > 1 {
		return fmt.Errorf("Only support single binary selector")
	} else if len(binarys) == 0 {
		// To aid verifier we always zero in binary fields to allow
		// BPF to assume the values exist.
		WriteSelectorUint32(k, 0)
		WriteSelectorUint32(k, 0)
		WriteSelectorUint32(k, 0)
		WriteSelectorUint32(k, 0)
		WriteSelectorUint32(k, 0)
	} else {
		if err := parseMatchBinary(k, 1, &binarys[0]); err != nil {
			return err
		}
	}
	WriteSelectorLength(k, loff)
	return nil
}

func parseSelector(
	k *KernelSelectorState,
	selectors *v1alpha1.KProbeSelector,
	args []v1alpha1.KProbeArg) error {
	if err := parseMatchPids(k, selectors.MatchPIDs); err != nil {
		return fmt.Errorf("parseMatchPids error: %w", err)
	}
	if err := parseMatchNamespaces(k, selectors.MatchNamespaces); err != nil {
		return fmt.Errorf("parseMatchNamespaces error: %w", err)
	}
	if err := parseMatchCapabilities(k, selectors.MatchCapabilities); err != nil {
		return fmt.Errorf("parseMatchCapabilities error: %w", err)
	}
	if err := parseMatchNamespaceChanges(k, selectors.MatchNamespaceChanges); err != nil {
		return fmt.Errorf("parseMatchNamespaceChanges error: %w", err)
	}
	if err := parseMatchCapabilityChanges(k, selectors.MatchCapabilityChanges); err != nil {
		return fmt.Errorf("parseMatchCapabilityChanges error: %w", err)
	}
	if err := parseMatchBinaries(k, selectors.MatchBinaries); err != nil {
		return fmt.Errorf("parseMatchBinaries error: %w", err)
	}
	if err := parseMatchArgs(k, selectors.MatchArgs, args); err != nil {
		return fmt.Errorf("parseMatchArgs  error: %w", err)
	}
	if err := parseMatchActions(k, selectors.MatchActions); err != nil {
		return fmt.Errorf("parseMatchActions error: %w", err)
	}
	return nil
}

// The byte array storing the selector configuration has the following format
// array := [N][S1_off][S2_off]...[SN_off][S1][S2][...][SN]
//
//	N: is the number of selectors (u32)
//	Sx_off: is the relative offset of  selector x (diff of Sx to Sx_off)
//	Sx: holds the data for the selector
//
// Each selector x starts with its length in bytes, and then stores a number of sections for the
// different matchers. Each section will typically starts with its length in bytes.
//
// Sx := [length]
//
//	[matchPIDs]
//	[matchNamespaces]
//	[matchCapabilities]
//	[matchNamespaceChanges]
//	[matchCapabilityChanges]
//	[matchBinaries]
//	[matchArgs]
//	[matchActions]
//
// matchPIDs := [length][PID1][PID2]...[PIDn]
// matchNamespaces := [length][NSx][NSy]...[NSn]
// matchCapabilities := [length][CAx][CAy]...[CAn]
// matchNamespaceChanges := [length][NCx][NCy]...[NCn]
// matchCapabilityChanges := [length][CAx][CAy]...[CAn]
// matchBinaries := [length][op][Index]...[Index]
// matchArgs := [length][ARGx][ARGy]...[ARGn]
// PIDn := [op][flags][nValues][v1]...[vn]
// Argn := [index][op][valueGen]
// NSn := [namespace][op][valueInt]
// NCn := [op][valueInt]
// CAn := [type][op][namespacecap][valueInt]
// valueGen := [type][len][v]
// valueInt := [len][v]
//
// For some examples, see kernel_test.go
func InitKernelSelectors(selectors []v1alpha1.KProbeSelector, args []v1alpha1.KProbeArg) ([4096]byte, error) {
	kernelSelectors, err := InitKernelSelectorState(selectors, args)
	if err != nil {
		return [4096]byte{}, err
	}
	return kernelSelectors.e, nil
}

func InitKernelSelectorState(selectors []v1alpha1.KProbeSelector, args []v1alpha1.KProbeArg) (*KernelSelectorState, error) {
	kernelSelectors := &KernelSelectorState{}

	WriteSelectorUint32(kernelSelectors, uint32(len(selectors)))
	soff := make([]uint32, len(selectors))
	for i := range selectors {
		soff[i] = AdvanceSelectorLength(kernelSelectors)
	}
	for i, s := range selectors {
		WriteSelectorLength(kernelSelectors, soff[i])
		loff := AdvanceSelectorLength(kernelSelectors)
		if err := parseSelector(kernelSelectors, &s, args); err != nil {
			return nil, err
		}
		WriteSelectorLength(kernelSelectors, loff)
	}
	return kernelSelectors, nil
}

func HasOverride(spec *v1alpha1.KProbeSpec) bool {
	for _, s := range spec.Selectors {
		for _, action := range s.MatchActions {
			act, _ := actionTypeTable[strings.ToLower(action.Action)]
			if act == ActionTypeOverride {
				return true
			}
		}
	}
	return false
}
