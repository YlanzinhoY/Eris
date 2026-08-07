//go:build windows

package scanner

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func localDriveRoots() []root {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}

	roots := make([]root, 0)
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}

		letter := byte('A' + index)
		path := fmt.Sprintf("%c:\\", letter)
		pathPointer, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		driveType := windows.GetDriveType(pathPointer)
		if driveType != windows.DRIVE_FIXED && driveType != windows.DRIVE_REMOVABLE {
			continue
		}

		roots = append(roots, root{
			path:   path,
			source: fmt.Sprintf("Disco %c:", letter),
		})
	}
	return roots
}
