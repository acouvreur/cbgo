package cbgo

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/tinygo-org/cbgo/internal/heap"
)

// Diagnostic only: C heap growth per batch of calls, to tell one-time growth
// from a per-call leak on the CI runner.
func TestDiag(t *testing.T) {
	out, _ := exec.Command("sw_vers").CombinedOutput()
	t.Logf("GOARCH=%s\n%s", runtime.GOARCH, out)

	u16 := UUID16(0xfe40)
	u128 := MustParseUUID("00467768-6228-2272-4663-277478268000")
	value := make([]byte, 244)
	chr := NewMutableCharacteristic(u128, CharacteristicPropertyRead, nil, AttributePermissionsReadable)
	format := NewMutableDescriptor(UUID16(0x2904), make([]byte, 7))
	cases := []struct {
		name string
		fn   func()
	}{
		{"noop", func() {}},
		{"pool", func() { popPool(pushPool()) }},
		{"write", func() { Peripheral{}.WriteCharacteristic(value, Characteristic{}, false) }},
		{"scan", func() {
			CentralManager{}.Scan([]UUID{u16, u128}, &CentralManagerScanOpts{SolicitedServiceUUIDs: []UUID{u16, u128}})
		}},
		{"scan16", func() { CentralManager{}.Scan([]UUID{u16}, nil) }},
		{"scan128", func() { CentralManager{}.Scan([]UUID{u128}, nil) }},
		{"scanNil", func() { CentralManager{}.Scan(nil, nil) }},
		{"beacon", func() { PeripheralManager{}.StartAdvertising(AdvData{IBeaconData: make([]byte, 21)}) }},
		{"update", func() { PeripheralManager{}.UpdateValue(value, Characteristic{}, nil) }},
		{"descriptors", func() { chr.SetDescriptors([]MutableDescriptor{format}) }},
		{"chrUUID", func() { chr.Characteristic().UUID() }},
	}
	for _, c := range cases {
		runtime.LockOSThread()
		c.fn()
		var growth []int
		for b := 0; b < 6; b++ {
			before := heap.InUse()
			for i := 0; i < 2000; i++ {
				c.fn()
			}
			growth = append(growth, heap.InUse()-before)
		}
		runtime.UnlockOSThread()
		t.Logf("%-12s %v", c.name, growth)
	}
}
