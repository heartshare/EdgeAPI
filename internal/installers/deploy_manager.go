package installers

import (
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"regexp"
)

var SharedDeployManager = NewDeployManager()

type DeployFile struct {
	OS      string
	Arch    string
	Version string
	Path    string
}

type DeployManager struct {
	dir string
}

func NewDeployManager() *DeployManager {
	return &DeployManager{
		dir: Tea.Root + "/deploy",
	}
}

// 加载所有文件
func (this *DeployManager) LoadFiles() []*DeployFile {
	keyMap := map[string]*DeployFile{} // key => File

	reg := regexp.MustCompile(`(\w+)-(\w+)-v([0-9.]+)\.zip`)
	for _, file := range files.NewFile(this.dir).List() {
		name := file.Name()
		if !reg.MatchString(name) {
			continue
		}
		matches := reg.FindStringSubmatch(name)
		osName := matches[1]
		arch := matches[2]
		version := matches[3]

		key := osName + "_" + arch
		oldFile, ok := keyMap[key]
		if ok && stringutil.VersionCompare(oldFile.Version, version) > 0 {
			continue
		}
		keyMap[key] = &DeployFile{
			OS:      osName,
			Arch:    arch,
			Version: version,
			Path:    file.Path(),
		}
	}

	result := []*DeployFile{}
	for _, v := range keyMap {
		result = append(result, v)
	}
	return result
}
