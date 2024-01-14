package indexer

import (
	"fmt"
	"log"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"slices"

	"github.com/shtirlic/knot/internal/store"
)

const (
	DirType  store.ItemType = "dir"
	FileType store.ItemType = "file"
)

type FileSystemIndexer struct {
	RootPath           string
	ExcludeDirFilters  []string
	ExcludeFileFilters []string
}

func (indexer *FileSystemIndexer) NewItemInfo() store.ItemInfo {
	return store.ItemInfo{}
}
func (indexer *FileSystemIndexer) ModifiedIndex(s store.Store) {
	// TODO implement me
	panic("implement me")
}

func (indexer *FileSystemIndexer) Config() *Config {
	return &Config{Name: "fs indexer", Params: map[string]string{}}
}

func DefaultExcludeDirFilters() []string {
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
func DefaultExcludeFileFilters() []string {
	return []string{
		"*~",
		"*.part",

		// temporary build files
		"*.o",
		"*.la",
		"*.lo",
		"*.loT",
		"*.moc",
		"moc_*.cpp",
		"qrc_*.cpp",
		"ui_*.h",
		"cmake_install.cmake",
		"CMakeCache.txt",
		"CTestTestfile.cmake",
		"libtool",
		"config.status",
		"confdefs.h",
		"autom4te",
		"conftest",
		"confstat",
		"Makefile.am",
		"*.gcode", // CNC machine/3D printer toolpath files
		".ninja_deps",
		".ninja_log",
		"build.ninja",

		// misc
		"*.csproj",
		"*.m4",
		"*.rej",
		"*.gmo",
		"*.pc",
		"*.omf",
		"*.aux",
		"*.tmp",
		"*.po",
		"*.vm*",
		"*.nvram",
		"*.rcore",
		"*.swp",
		"*.swap",
		"lzo",
		"litmain.sh",
		"*.orig",
		".histfile.*",
		".xsession-errors*",
		"*.map",
		"*.so",
		"*.a",
		"*.db",
		"*.qrc",
		"*.ini",
		"*.init",
		"*.img",      // typical extension for raw disk images
		"*.vdi",      // Virtualbox disk images
		"*.vbox*",    // Virtualbox VM files
		"vbox.log",   // Virtualbox log files
		"*.qcow2",    // QEMU QCOW2 disk images
		"*.vmdk",     // VMware disk images
		"*.vhd",      // Hyper-V disk images
		"*.vhdx",     // Hyper-V disk images
		"*.sql",      // SQL database dumps
		"*.sql.gz",   // Compressed SQL database dumps
		"*.ytdl",     // youtube-dl temp files
		"*.tfstate*", // Terraform state files

		// Bytecode files
		"*.class", // Java
		"*.pyc",   // Python
		"*.pyo",   // More Python
		"*.elc",   // Emacs Lisp
		"*.qmlc",  // QML
		"*.jsc",   // Javascript

		// files known in bioinformatics containing huge amount of unindexable data
		"*.fastq",
		"*.fq",
		"*.gb",
		"*.fasta",
		"*.fna",
		"*.gbff",
		"*.faa",
		"*.fna",
	}
}

func NewFileSystemIndexer(rootPath string, excludeDirFilter []string, excludeFileFilter []string) Indexer {

	if len(excludeDirFilter) == 0 {
		excludeDirFilter = DefaultExcludeDirFilters()
	}
	if len(excludeFileFilter) == 0 {
		excludeFileFilter = DefaultExcludeFileFilters()
	}

	return &FileSystemIndexer{
		RootPath:           filepath.Clean(rootPath),
		ExcludeDirFilters:  excludeDirFilter,
		ExcludeFileFilters: excludeFileFilter,
	}
}

func (indexer *FileSystemIndexer) NewIndex(s store.Store) {

	var err error
	path, err := os.Stat(indexer.RootPath)
	if err != nil {
		log.Println("Can't get fileinfo:", err, path)
	}

	rootFileInfo := store.ItemInfo{
		Name: path.Name(),
		Path: indexer.RootPath,
		Type: func() store.ItemType {
			if path.IsDir() {
				return DirType
			}
			return FileType
		}(),
	}
	if s.Find(rootFileInfo) != nil {
		slog.Info("already in index, skipping", "key", rootFileInfo.KeyName())
		return
	} else {
		slog.Info("not found in index, adding", "key", rootFileInfo.KeyName())
	}

	idxSize := 0
	idxDirSize := 0
	idxFileSize := 0
	itemList := make(map[string]store.ItemInfo)
	err = filepath.Walk(indexer.RootPath, func(path string, info os.FileInfo, err error) error {

		// TODO: Add to failed item list
		// Skip access denied etc.
		if err != nil {
			log.Println("Inside Walk:", err, path)
			return filepath.SkipDir
		}

		// Skip dirs via exclude dir filters
		if info.IsDir() && slices.Contains(indexer.ExcludeDirFilters, info.Name()) {
			return filepath.SkipDir
		}

		// TODO: Try to use Match/Glob for file masks
		// Skip files via exclude file filters
		if !info.IsDir() && slices.Contains(indexer.ExcludeFileFilters, info.Name()) {
			return nil
		}

		objInfo := store.ItemInfo{
			Hash:    "",
			Name:    info.Name(),
			Path:    path,
			ModTime: info.ModTime(),
			Size:    info.Size(),
			Type: func() store.ItemType {
				if info.IsDir() {
					idxDirSize++
					return DirType
				}
				idxFileSize++
				return FileType
			}(),
		}

		// Get the mimetype for file by extension
		if objInfo.Type == FileType {
			objInfo.MimeType = mime.TypeByExtension(filepath.Ext(path))
		}
		// Calc hash
		objInfo.Hash = objInfo.XXhash()

		// Add to items list
		itemList[objInfo.KeyName()] = objInfo
		idxSize++

		// Add items to store in batch per 100 items
		if len(itemList) > 100 {
			s.Add(itemList)
			clear(itemList)
		}
		return err
	})
	if err != nil {
		log.Println("After Walk:", err)
	}

	// Add remaining items after batch inserts
	if len(itemList) > 0 {
		s.Add(itemList)
	}

	addInfo := fmt.Sprintf("All: %d, Files: %d, Dirs: %d \n", idxSize, idxFileSize, idxDirSize)
	slog.Info(addInfo)

	if err != nil {
		log.Println(err)
	}
	slog.Debug(s.Info())
}
