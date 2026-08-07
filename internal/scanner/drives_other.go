//go:build !windows

package scanner

func localDriveRoots() []root {
	return []root{{path: "/", source: "Disco local"}}
}
