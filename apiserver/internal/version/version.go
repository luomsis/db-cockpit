// Package version 保存编译期注入的构建信息（Dockerfile ldflags -X 注入）。
package version

// 通过 ldflags 注入，例如：
// go build -ldflags "-X db-cockpit/apiserver/internal/version.GitSHA=${GIT_SHA} -X db-cockpit/apiserver/internal/version.BuildTime=${BUILD_TIME}"
// 本地 go run / 未注入时保持 unknown。
var (
	GitSHA    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Service   string `json:"service"`
	GitSHA    string `json:"gitSHA"`
	BuildTime string `json:"buildTime"`
}

func Get() Info {
	return Info{Service: "apiserver", GitSHA: GitSHA, BuildTime: BuildTime}
}
