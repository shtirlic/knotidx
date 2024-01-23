package indexer

import (
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/shtirlic/knotidx/internal/store"
)

const (
	DirItemType  store.ItemType = "dir"
	FileItemType store.ItemType = "file"

	FsIndexerType IndexerType = "fs"
)

type FileSystemIndexer struct {
	RootPath           string
	ExcludeDirFilters  []string
	ExcludeFileFilters []string
	idxType            IndexerType
	Store              store.Store
	watcher            *fsnotify.Watcher
}

func (idx *FileSystemIndexer) Watch(quit chan bool) {

	if idx.watcher == nil {
		return
	}

	defer idx.watcher.Close()

	slog.Debug("Watcher events select", "idx RootPath", idx.RootPath)

	for {
		select {
		case event, ok := <-idx.watcher.Events:
			if !ok {
				return
			}
			slog.Debug("Watcher", "event", event)

			switch event.Op {
			case fsnotify.Chmod, fsnotify.Write, fsnotify.Create:
				idx.addPath(event.Name)
			case fsnotify.Remove, fsnotify.Rename:
				idx.removePath(event.Name)
			}
		case err, ok := <-idx.watcher.Errors:
			if !ok {
				return
			}
			slog.Debug("Watcher", "error", err)
		case <-quit:
			slog.Debug("Quit watcher", "idx RootPath", idx.RootPath)
			return
		}
	}
}

func (idx *FileSystemIndexer) Type() IndexerType {
	return idx.idxType
}

func (indexer *FileSystemIndexer) Config() *Config {
	return &Config{Name: "fs indexer", Params: map[string]string{}}
}

func NewFileSystemIndexer(store store.Store, rootPath string, notify bool, excludeDirFilters []string, excludeFileFilters []string) Indexer {

	if len(excludeDirFilters) == 0 {
		excludeDirFilters = DefaultExcludeDirFilters()
	}
	if len(excludeFileFilters) == 0 {
		excludeFileFilters = DefaultExcludeFileFilters()
	}

	fsi := &FileSystemIndexer{
		RootPath:           filepath.Clean(rootPath),
		ExcludeDirFilters:  excludeDirFilters,
		ExcludeFileFilters: excludeFileFilters,
		idxType:            FsIndexerType,
		Store:              store,
	}
	// TODO: handle error
	// Enbale fs notify watcher
	if notify {
		fsi.watcher, _ = fsnotify.NewWatcher()
	}
	return fsi
}

func ItemType(isDir bool) store.ItemType {
	if isDir {
		return DirItemType
	}
	return FileItemType
}

func (idx *FileSystemIndexer) CleanIndex(prefix string) error {

	prefix = string(idx.idxType) + "_" + prefix

	for _, key := range idx.Store.Keys(prefix) {
		item := strings.SplitN(key, "_", 3)
		path := item[2]
		fi, err := os.Lstat(path)
		if err != nil {
			slog.Debug("CleanIndex", "key", key, "path", path, "err", err)
			if err = idx.Store.Delete(key); err != nil {
				return err
			} else {
				continue
			}
		}
		if fi.IsDir() && store.ItemType(item[1]) != DirItemType {
			slog.Debug("CleanIndex", "key", key, "path", path, "err", err)
			if err = idx.Store.Delete(key); err != nil {
				return err
			}
		}
		if !fi.IsDir() && store.ItemType(item[1]) != FileItemType {
			slog.Debug("CleanIndex", "key", key, "path", path, "err", err)
			if err = idx.Store.Delete(key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (idx *FileSystemIndexer) UpdateIndex() (err error) {
	err = idx.CleanIndex("")
	if err != nil {
		return
	}

	err = idx.addPath(idx.RootPath)
	if err != nil {
		return
	}

	idx.Store.Maintenance()
	return
}

func (idx *FileSystemIndexer) removePath(path string) {
	path = filepath.Clean(path)
	idx.CleanIndex("dir_" + path)
	idx.CleanIndex("file_" + path)
}

func (idx *FileSystemIndexer) addPath(newPath string) (err error) {

	if newPath == "" {
		newPath = idx.RootPath
	}
	newPath = filepath.Clean(newPath)

	// Return if no access to path
	path, err := os.Stat(newPath)
	if err != nil {
		slog.Error("Can't get fileinfo for path:", "error", err, "path", path)
		return
	}

	slog.Debug("fs indexer addPath", "path", path.Name())

	// // Create temp ItemInfo for searching
	// rootFileInfo := store.ItemInfo{
	// 	Name: path.Name(),
	// 	Path: idx.RootPath,
	// 	Type: ItemType(path.IsDir()),
	// }

	// // Search for root path in index
	// if idx.Store.Find(rootFileInfo) != nil {
	// 	slog.Info("already in index, skipping", "key", rootFileInfo.KeyName())
	// 	return
	// } else {
	// 	slog.Info("not found in index, adding", "key", rootFileInfo.KeyName())
	// }

	idxSize := 0
	idxDirSize := 0
	idxFileSize := 0

	// Map for storing file/dir entries
	itemList := make(map[string]store.ItemInfo)

	// List for failed file/dir items
	var failedItems []string

	// Walking from the root path recursevly
	err = filepath.Walk(newPath, func(path string, info os.FileInfo, err error) error {

		// Skip access denied etc and add to failed list
		if err != nil {
			slog.Debug("Inside Walk", "err", err, "path", path)
			failedItems = append(failedItems, path)
			return filepath.SkipDir
		}

		// Skip dirs via exclude dir filters
		if info.IsDir() && slices.Contains(idx.ExcludeDirFilters, info.Name()) {
			return filepath.SkipDir
		}

		// TODO: Try to use Match/Glob for file masks
		// NOTICE: file masks are not working now
		// Skip files via exclude file filters
		if !info.IsDir() && slices.Contains(idx.ExcludeFileFilters, info.Name()) {
			return nil
		}

		// Create ItemInfo for index addtion
		itemInfo := store.NewItemInfo(
			info.Name(),
			path,
			info.ModTime(),
			info.Size(),
			ItemType(info.IsDir()))

		// Get the mimetype for file by extension
		if itemInfo.Type == FileItemType {
			idxFileSize++
			itemInfo.MimeType = mime.TypeByExtension(filepath.Ext(path))
		} else {
			if idx.watcher != nil {
				idx.watcher.Add(path)
			}
			idxDirSize++
		}
		// Calc hash
		itemInfo.Hash = itemInfo.XXhash()

		key := fmt.Sprintf("%s_%s", idx.idxType, itemInfo.KeyName())
		// Add to items list
		itemList[key] = itemInfo
		idxSize++

		// Add items to store in batches
		if len(itemList) > store.BatchCount {
			err = idx.Store.Add(itemList)
			if err != nil {
				slog.Error("can't add items to store")
				return err
			}
			clear(itemList)
		}
		return err
	})
	if err != nil {
		slog.Debug("After Walk", "err", err)
		return
	}

	// Add remaining items after batch inserts
	if len(itemList) > 0 {
		err = idx.Store.Add(itemList)
		if err != nil {
			slog.Error("can't add items to store")
			return
		}
	}

	addInfo := fmt.Sprintf("All: %d, Files: %d, Dirs: %d, Failed: %d, rootPath: %s", idxSize, idxFileSize, idxDirSize, len(failedItems), idx.RootPath)
	slog.Info(addInfo)

	slog.Debug(idx.Store.Info())
	return
}
