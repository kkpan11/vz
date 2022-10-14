package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import (
	"fmt"
	"os"
	"runtime"
)

// BootLoader is the interface of boot loader definitions.
// see: LinuxBootLoader
type BootLoader interface {
	NSObject

	bootLoader()
}

type baseBootLoader struct{}

func (*baseBootLoader) bootLoader() {}

var _ BootLoader = (*LinuxBootLoader)(nil)

// LinuxBootLoader Boot loader configuration for a Linux kernel.
type LinuxBootLoader struct {
	vmlinuzPath string
	initrdPath  string
	cmdLine     string
	pointer

	*baseBootLoader
}

func (b *LinuxBootLoader) String() string {
	return fmt.Sprintf(
		"vmlinuz: %q, initrd: %q, command-line: %q",
		b.vmlinuzPath,
		b.initrdPath,
		b.cmdLine,
	)
}

type LinuxBootLoaderOption func(b *LinuxBootLoader) error

// WithCommandLine sets the command-line parameters.
// see: https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html
func WithCommandLine(cmdLine string) LinuxBootLoaderOption {
	return func(b *LinuxBootLoader) error {
		b.cmdLine = cmdLine
		cs := charWithGoString(cmdLine)
		defer cs.Free()
		C.setCommandLineVZLinuxBootLoader(b.Ptr(), cs.CString())
		return nil
	}
}

// WithInitrd sets the optional initial RAM disk.
func WithInitrd(initrdPath string) LinuxBootLoaderOption {
	return func(b *LinuxBootLoader) error {
		if _, err := os.Stat(initrdPath); err != nil {
			return fmt.Errorf("invalid initial RAM disk path: %w", err)
		}
		b.initrdPath = initrdPath
		cs := charWithGoString(initrdPath)
		defer cs.Free()
		C.setInitialRamdiskURLVZLinuxBootLoader(b.Ptr(), cs.CString())
		return nil
	}
}

// NewLinuxBootLoader creates a LinuxBootLoader with the Linux kernel passed as Path.
//
// This is only supported on macOS 11 and newer, ErrUnsupportedOSVersion will
// be returned on older versions.
func NewLinuxBootLoader(vmlinuz string, opts ...LinuxBootLoaderOption) (*LinuxBootLoader, error) {
	if macosMajorVersionLessThan(11) {
		return nil, ErrUnsupportedOSVersion
	}
	if _, err := os.Stat(vmlinuz); err != nil {
		return nil, fmt.Errorf("invalid linux kernel path: %w", err)
	}

	vmlinuzPath := charWithGoString(vmlinuz)
	defer vmlinuzPath.Free()
	bootLoader := &LinuxBootLoader{
		vmlinuzPath: vmlinuz,
		pointer: pointer{
			ptr: C.newVZLinuxBootLoader(
				vmlinuzPath.CString(),
			),
		},
	}
	runtime.SetFinalizer(bootLoader, func(self *LinuxBootLoader) {
		self.Release()
	})
	for _, opt := range opts {
		if err := opt(bootLoader); err != nil {
			return nil, err
		}
	}
	return bootLoader, nil
}
