package indexer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/cespare/xxhash/v2"

	"github.com/shtirlic/knot/internal/store"
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
	itemList := make(map[string]store.ItemInfo)
	err := filepath.Walk(indexer.RootPath, func(path string, info os.FileInfo, err error) error {
		if slices.Contains(indexer.ExcludeFilters, info.Name()) {
			return filepath.SkipDir
		}
		objInfo := store.ItemInfo{
			Hash:    "",
			Name:    info.Name(),
			Path:    path,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}
		if info.IsDir() {
			objInfo.Type = store.DIR
			idxDirSize++
		} else {
			objInfo.Type = store.FILE
			idxFileSize++
		}
		objInfo.Hash = strconv.FormatUint(xxhash.Sum64String(objInfo.String()), 10)
		keyName := fmt.Sprintf("%s_%s", objInfo.Type, path)
		itemList[keyName] = objInfo

		idxSize++
		return err
	})
	if err != nil {
		fmt.Println(err)
	}

	var s store.Store = store.NewInMemoryBadgerStore()
	s.Add(itemList)

	log.Printf("All: %d, Files: %d, Dirs: %d \n", idxSize, idxFileSize, idxDirSize)

	all, err := s.GetAll()
	if err != nil {
		log.Println(err)
	}
	log.Println(all)
}
