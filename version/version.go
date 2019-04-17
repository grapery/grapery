package version

import "fmt"

// Version information.
var (
	BuildTS   = "None"
	GitHash   = "None"
	GitBranch = "None"
	Version   = "None"
)

//GetVersion Printer print build version
func GetVersion() string {
	if GitHash != "" {
		h := GitHash
		if len(h) > 7 {
			h = h[:7]
		}
		return fmt.Sprintf("%s-%s", Version, h)
	}
	return Version
}

//PrintFullVersionInfo ...
func PrintFullVersionInfo() {
	fmt.Println("Version:          ", GetVersion())
	fmt.Println("Git Branch:       ", GitBranch)
	fmt.Println("Git Commit:       ", GitHash)
	fmt.Println("Build Time (UTC): ", BuildTS)
}
