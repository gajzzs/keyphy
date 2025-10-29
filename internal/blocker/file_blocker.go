package blocker

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type FileBlocker struct {
	blockedPaths map[string]bool
	originalPerms map[string]os.FileMode
}

func NewFileBlocker() *FileBlocker {
	return &FileBlocker{
		blockedPaths: make(map[string]bool),
		originalPerms: make(map[string]os.FileMode),
	}
}

func (fb *FileBlocker) BlockPath(path string) error {
	fb.blockedPaths[path] = true
	
	// Store original permissions
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	fb.originalPerms[path] = info.Mode()
	
	// Remove all permissions
	if err := os.Chmod(path, 0000); err != nil {
		return fmt.Errorf("failed to change permissions: %v", err)
	}
	
	// Set immutable attribute if supported
	return fb.setImmutable(path)
}

func (fb *FileBlocker) UnblockPath(path string) error {
	delete(fb.blockedPaths, path)
	
	// Remove immutable attribute
	if err := fb.removeImmutable(path); err != nil {
		return fmt.Errorf("failed to remove immutable: %v", err)
	}
	
	// Restore original permissions
	if perm, exists := fb.originalPerms[path]; exists {
		if err := os.Chmod(path, perm); err != nil {
			return fmt.Errorf("failed to restore permissions: %v", err)
		}
		delete(fb.originalPerms, path)
	}
	
	return nil
}

func (fb *FileBlocker) IsBlocked(path string) bool {
	return fb.blockedPaths[path]
}

func (fb *FileBlocker) setImmutable(path string) error {
	// Use chattr to set immutable attribute on ext filesystems
	return syscall.Syscall(syscall.SYS_IOCTL, 
		uintptr(0), 
		uintptr(0x40086602), // FS_IOC_SETFLAGS
		uintptr(0x00000010)) // FS_IMMUTABLE_FL
}

func (fb *FileBlocker) removeImmutable(path string) error {
	// Remove immutable attribute
	return syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(0),
		uintptr(0x40086602), // FS_IOC_SETFLAGS  
		uintptr(0x00000000)) // Remove flags
}

func (fb *FileBlocker) BlockDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return fb.BlockPath(path)
	})
}

func (fb *FileBlocker) UnblockDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fb.IsBlocked(path) {
			return fb.UnblockPath(path)
		}
		return nil
	})
}

func (fb *FileBlocker) GetBlockedPaths() []string {
	var paths []string
	for path := range fb.blockedPaths {
		paths = append(paths, path)
	}
	return paths
}