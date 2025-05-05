//go:build !noembeds

package embeds

import "embed"

//go:embed files/bin
var bin embed.FS
