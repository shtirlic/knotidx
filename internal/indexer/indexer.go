package indexer

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"knot/internal/item"
	"knot/internal/store"
)

type Indexer struct {
	RootPath       string
	ExcludeFilters []string
}

func defaultExcludeFilters() []string {
	return []string{
		"po",

		// VCS
		"CVS",
		".svn",
		".git",
		"_darcs",
		".bzr",
		".hg",

		// development
		"CMakeFiles",
		"CMakeTmp",
		"CMakeTmpQmake",
		".moc",
		".obj",
		".pch",
		".uic",
		".npm",
		".yarn",
		".yarn-cache",
		"__pycache__",
		"node_modules",
		"node_packages",
		"nbproject",
		".terraform",
		".venv",
		"venv",
		".rbenv",
		".bundle",
		".conda",
		".cargo",
		".vscode",

		// misc
		"core-dumps",
		"lost+found",
		"drive_c", // wine giant dirs
		".wine",
		".mozilla",
		".thunderbird",

		// cache dirs
		".cache",
		"CachedData",
		"CacheStorage",
		"Cache_Data",
		"Code Cache",
		"ScriptCache",

		// do not use in production
		".local",
		".config",
	}
}

func NewIndexer(rootPath string, excludeFilter []string) *Indexer {

	if len(excludeFilter) == 0 {
		excludeFilter = defaultExcludeFilters()
	}
	return &Indexer{
		RootPath:       rootPath,
		ExcludeFilters: excludeFilter,
	}
}

func (indexer *Indexer) Run() {

	idxSize := 0
	idxDirSize := 0
	idxFileSize := 0
	fileList := make(map[string]item.ItemInfo)
	dirList := make(map[string]item.ItemInfo)
	err := filepath.Walk(indexer.RootPath, func(path string, info os.FileInfo, err error) error {
		if slices.Contains(indexer.ExcludeFilters, info.Name()) {
			return filepath.SkipDir
		}
		objInfo := item.ItemInfo{
			Hash:    "",
			Name:    info.Name(),
			Path:    path,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}
		// fmt.Println(path)
		if info.IsDir() {
			objInfo.Type = item.DIR
			idxDirSize++
			dirList[fmt.Sprintf("dir_%s", path)] = objInfo
		} else {
			objInfo.Type = item.FILE
			idxFileSize++
			fileList[fmt.Sprintf("file_%s", path)] = objInfo
		}

		idxSize++
		return err
	})
	if err != nil {
		fmt.Println(err)
	}

	store.GlobalStore.Add(dirList)
	store.GlobalStore.Add(fileList)

	_, err = fmt.Printf("All: %d, Files: %d, Dirs: %d \n", idxSize, idxFileSize, idxDirSize)

	store.GlobalStore.List()

	if err != nil {
		fmt.Println(err)
	}
}

func hash(s string) string {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		return ""
	}
	return strconv.Itoa(int(h.Sum32()))
}
