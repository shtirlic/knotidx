package indexer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"
)

const (
	// DirItemType represents the item type for directories.
	DirItemType store.ItemType = "dir"

	// FileItemType represents the item type for files.
	FileItemType store.ItemType = "file"

	// FileSystemIndexerType represents the type of the FileSystemIndexer.
	FileSystemIndexerType IndexerType = "fs"
)

// FileSystemIndexer represents an indexer implementation for the filesystem.
type FileSystemIndexer struct {
	RootPath           string                  // RootPath is the root directory path to be indexed.
	ExcludeDirFilters  []string                // ExcludeDirFilters contains filters for excluding directories.
	ExcludeFileFilters []string                // ExcludeFileFilters contains filters for excluding files.
	Store              store.Store             // Store is the data store to index items.
	watcher            *fsnotify.Watcher       // watcher is used to monitor file system events.
	config             config.IndexerConfig    // config is the configuration for the indexer.
	ctx                context.Context         // ctx is the cancel conext.
	info               info                    // runtime info
	feedback           chan IndexerRuntimeInfo // feedback channel
}

// Feedback returns feeback channel.
func (idx *FileSystemIndexer) Feedback() chan IndexerRuntimeInfo {
	return idx.feedback
}

type info struct {
	Dirs   int
	Files  int
	Failed int
	Total  int
	IndexerRuntimeInfo
}

// Info returns runtime info.
func (idx *FileSystemIndexer) Info() IndexerRuntimeInfo {
	return idx.info.IndexerRuntimeInfo
}

// Watch monitors the file system for events and updates the index accordingly.
func (idx *FileSystemIndexer) Watch() {

	// Check if the watcher is initialized.
	if idx.watcher == nil {
		return
	}
	// Close the watcher when the function exits.
	defer idx.watcher.Close()

	slog.Debug("Watcher events select", "idx RootPath", idx.RootPath)

	// Infinite loop to continuously monitor file system events.
	for {
		select {
		// Handle file system events.
		case event, ok := <-idx.watcher.Events:
			if !ok {
				return
			}
			slog.Debug("Watcher", "event", event)

			// Process different types of file system events.
			switch event.Op {
			case fsnotify.Chmod, fsnotify.Write, fsnotify.Create:
				idx.addPath(event.Name) // Add or update the path in the index.
			case fsnotify.Remove, fsnotify.Rename:
				idx.removePath(event.Name) // Remove the path from the index.
			}
		// Handle errors from the watcher.
		case err, ok := <-idx.watcher.Errors:
			if !ok {
				return
			}
			slog.Debug("Watcher", "error", err)

		// check for contex done channel for cancel.
		case <-idx.ctx.Done():
			slog.Debug("Quit watcher", "idx RootPath", idx.RootPath)
			return
		}
	}
}

// Type returns the type of the FileSystemIndexer.
func (idx *FileSystemIndexer) Type() IndexerType {
	return FileSystemIndexerType
}

// Config returns the configuration of the FileSystemIndexer.
func (idx *FileSystemIndexer) Config() config.IndexerConfig {
	return idx.config
}

// NewFileSystemIndexer creates a new instance of FileSystemIndexer with the provided store,
// rootPath, and configuration. It returns an Indexer interface.
func NewFileSystemIndexer(ctx context.Context, store store.Store, rootPath string, c config.IndexerConfig) Indexer {

	// Initialize excludeDirFilters and excludeFileFilters with default values if not provided.
	excludeDirFilters := c.ExcludeDirFilters
	excludeFileFilters := c.ExcludeFileFilters

	if len(excludeDirFilters) == 0 {
		excludeDirFilters = DefaultExcludeDirFilters()
	}
	if len(excludeFileFilters) == 0 {
		excludeFileFilters = DefaultExcludeFileFilters()
	}

	// Create a new FileSystemIndexer instance.
	fsi := &FileSystemIndexer{
		RootPath:           filepath.Clean(rootPath),
		ExcludeDirFilters:  excludeDirFilters,
		ExcludeFileFilters: excludeFileFilters,
		Store:              store,
		config:             c,
		ctx:                ctx,
		info: info{Dirs: 0, Files: 0, Failed: 0, Total: 0,
			IndexerRuntimeInfo: IndexerRuntimeInfo{Status: "Created"}},
		feedback: make(chan IndexerRuntimeInfo, 1),
	}

	// Enable fsnotify watcher if Notify is true in the configuration.
	if c.Notify {
		fsi.watcher, _ = fsnotify.NewWatcher()
	}
	// TODO: Handle error returned from fsnotify.NewWatcher().

	// Returning the created FileSystemIndexer.
	return fsi
}

// ItemType returns the appropriate store.ItemType based on the given boolean value isDir.
// If isDir is true, it returns DirItemType; otherwise, it returns FileItemType.
func ItemType(isDir bool) store.ItemType {
	if isDir {
		return DirItemType
	}
	return FileItemType
}

// CleanIndex removes items from the index that match the specified prefix.
// It iterates through the keys in the store with the given prefix and validates
// whether the corresponding paths still exist on the file system. If not, it deletes
// the corresponding key from the store.
func (idx *FileSystemIndexer) CleanIndex(prefix string) error {
	// Add indexer type prefix to the provided prefix.
	prefix = string(idx.Type()) + "_" + prefix

	// Iterate through keys with the specified prefix.
	for _, key := range idx.Store.Keys(prefix, "", 0) {
		// Split the key into components.
		item := strings.SplitN(key, "_", 3)
		path := item[2]

		// Retrieve file info for the path.
		fi, err := os.Lstat(path)
		if err != nil {
			slog.Debug("CleanIndex", "key", key, "path", path, "err", err)
			if err = idx.Store.Delete(key); err != nil {
				return err
			} else {
				continue
			}
		}

		switch store.ItemType(item[1]) {
		case DirItemType:
			if !fi.IsDir() {
				slog.Debug("CleanIndex", "key", key, "path", path, "err", err)

				// Delete the key from the store if the types do not match.
				if err = idx.Store.Delete(key); err != nil {
					return err
				}
			}
		case FileItemType:
			if fi.IsDir() && store.ItemType(item[1]) != DirItemType {
				slog.Debug("CleanIndex", "key", key, "path", path, "err", err)

				// Delete the key from the store if the types do not match.
				if err = idx.Store.Delete(key); err != nil {
					return err
				}
			}
		}
	}

	// Perform maintenance operations on the store.
	idx.Store.Maintenance()

	// Return nil to indicate a successful cleaning operation.
	return nil
}

// UpdateIndex updates the index by first cleaning it to remove stale entries
// and then adding the paths starting from the root path.
func (idx *FileSystemIndexer) UpdateIndex() (time.Duration, error) {

	startTime := time.Now()
	idx.info.StartTime = startTime
	idx.info.Status = "Started"
	idx.feedback <- idx.Info()

	// Clean the index to remove stale entries.
	if err := idx.CleanIndex(""); err != nil {
		return 0, err
	}

	// Add the root path and its subdirectories to the index.
	if err := idx.addPath(idx.RootPath); err != nil {
		return 0, err
	}

	idx.info.Status = "Finished"
	idx.feedback <- idx.Info()
	close(idx.feedback)

	// Return nil to indicate a successful update.
	idx.info.FinishTime = time.Now()
	idx.info.Duration = idx.info.FinishTime.Sub(idx.info.StartTime)
	return time.Since(startTime), nil
}

// removePath removes entries from the index associated with the specified path.
// It cleans both directory and file entries for the given path.
func (idx *FileSystemIndexer) removePath(path string) {
	path = filepath.Clean(path)

	// Clean directory entries associated with the path.
	idx.CleanIndex("dir_" + path)

	// Clean file entries associated with the path.
	idx.CleanIndex("file_" + path)
}

// addPath recursively traverses the file system starting from the specified path and adds
// directory and file entries to the index. It skips directories based on exclude directory filters
// and files based on exclude file filters. The function also handles batch insertion of items into
// the store to improve efficiency.
func (idx *FileSystemIndexer) addPath(newPath string) (err error) {

	// If newPath is empty, use the root path of the indexer.
	if newPath == "" {
		newPath = idx.RootPath
	}
	newPath = filepath.Clean(newPath)

	// Retrieve file info for the specified path.
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

	// Initialize counters for directory and file sizes in the index.
	idxSize := 0
	idxDirSize := 0
	idxFileSize := 0

	// Map for storing file/dir entries.
	itemList := make(map[string]store.ItemInfo)

	// List for failed file/dir items.
	var failedItems []string

	// Walk the file system starting from the specified path.
	err = filepath.Walk(newPath, func(path string, info os.FileInfo, err error) error {

		idx.info.Duration = time.Since(idx.info.StartTime)
		// check for contex done channel for cancel.
		select {
		case <-idx.ctx.Done():
			return idx.ctx.Err()
		case <-idx.feedback:
		default:
			idx.feedback <- idx.Info()
		}

		// Skip access denied, etc., and add to the failed list.
		if err != nil {
			slog.Debug("Inside Walk", "err", err, "path", path)
			failedItems = append(failedItems, path)
			idx.info.Failed = len(failedItems)
			return filepath.SkipDir
		}

		// Skip dirs via exclude dir filters.
		if info.IsDir() && slices.Contains(idx.ExcludeDirFilters, info.Name()) {
			return filepath.SkipDir
		}

		// TODO: Try to use Match/Glob for file masks
		// NOTICE: file masks are not working now
		// Skip files via exclude file filters.
		if !info.IsDir() && slices.Contains(idx.ExcludeFileFilters, info.Name()) {
			return nil
		}

		// Create ItemInfo for index addition.
		itemInfo := store.NewItemInfo(
			info.Name(),
			path,
			info.ModTime(),
			info.Size(),
			ItemType(info.IsDir()))

		// Get the mimetype for files by extension.
		if itemInfo.Type == FileItemType {
			idxFileSize++
			idx.info.Files = idxFileSize
			itemInfo.MimeType = mime.TypeByExtension(filepath.Ext(path))
		} else {
			// Add directories to the file system watcher.
			if idx.watcher != nil {
				idx.watcher.Add(path)
			}
			idxDirSize++
			idx.info.Dirs = idxDirSize
		}
		// Calculate the hash.
		itemInfo.Hash = itemInfo.XXhash()

		// Create the key for the item in the format "indexerType_path".
		key := fmt.Sprintf("%s_%s", idx.Type(), itemInfo.KeyName())
		// Add the item to the items list.
		itemList[key] = itemInfo
		idxSize++
		idx.info.Total = idxSize

		// Add items to the store in batches.
		if len(itemList) > store.BatchCount {
			err = idx.Store.Add(itemList)
			if err != nil {
				slog.Error("can't add items to store")
				return err
			}
			// Clear the items list after successful batch insertion.
			clear(itemList)
		}
		return err
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Debug("After Walk", "err", err)
		return
	}

	// Add remaining items after batch inserts.
	if len(itemList) > 0 {
		err = idx.Store.Add(itemList)
		if err != nil {
			slog.Error("can't add items to store")
			return
		}
	}

	addInfo := fmt.Sprintf("Total: %d, Files: %d, Dirs: %d, Failed: %d, rootPath: %s", idx.info.Total, idx.info.Files, idx.info.Dirs, idx.info.Failed, idx.RootPath)
	slog.Debug(addInfo)

	if errors.Is(err, context.Canceled) {
		err = nil
	}
	return
}
