package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"server/diagnostics"
	"strings"
	"sync"
	"time"

	"server/log"
)

type JsonDB struct {
	Path              string
	filenameDelimiter string
	filenameExtension string
	fileMode          fs.FileMode
	xPathDelimeter    string
}

var globalJsonDB TorrServerDB
var jsonDbLocks = make(map[string]*sync.Mutex)
var jsonDbLocksMutex sync.Mutex

func NewJsonDB() TorrServerDB {
	if globalJsonDB != nil {
		return globalJsonDB
	}
	// NB: assign to the package-level singleton, not `:=` — a `:=` here shadows
	// globalJsonDB with a local, so every call returned a fresh JsonDB (each with
	// its own per-file lock map, defeating the intended mutual exclusion).
	globalJsonDB = &JsonDB{
		Path:              Path,
		filenameDelimiter: ".",
		filenameExtension: ".json",
		fileMode:          fs.FileMode(0o666),
		xPathDelimeter:    "/",
	}
	return globalJsonDB
}

func (v *JsonDB) CloseDB() {
	// Not necessary
}

func (v *JsonDB) Set(xPath, name string, value []byte) {
	if err := v.PutChecked(xPath, name, value); err != nil {
		v.log("Set: error writing entry", err)
	}
}

func (v *JsonDB) PutChecked(xPath, name string, value []byte) error {
	var object map[string]interface{}
	if err := json.Unmarshal(value, &object); err != nil {
		return err
	}
	filename, err := v.xPathToFilename(xPath)
	if err != nil {
		return err
	}
	v.lock(filename)
	defer v.unlock(filename)
	root, err := v.readJsonFileAsMap(filename)
	if err != nil {
		// Explicit Apply may repair a corrupt settings document after loading a
		// known-good/default configuration. Preserve the original privately first.
		if xPath != "Settings" || SettingsRecovery().Issue != "SETTINGS_UNREADABLE" {
			return err
		}
		original, readErr := os.ReadFile(filepath.Join(v.Path, filename))
		if readErr != nil {
			return err
		}
		backup := filepath.Join(v.Path, fmt.Sprintf("flow-corrupt-settings-%d.json", time.Now().UnixNano()))
		if backupErr := diagnostics.WritePrivateFile(backup, original); backupErr != nil {
			return backupErr
		}
		root = map[string]interface{}{}
	}
	root[name] = object
	return v.writeMapAsJsonFile(filename, root)
}

func (v *JsonDB) Get(xPath, name string) []byte {
	var err error = nil
	if filename, err := v.xPathToFilename(xPath); err == nil {
		v.lock(filename)
		defer v.unlock(filename)
		if root, err := v.readJsonFileAsMap(filename); err == nil {
			if jsonData, ok := root[name]; ok {
				if byteData, err := json.Marshal(jsonData); err == nil {
					// Return a copy to be safe
					data := make([]byte, len(byteData))
					copy(data, byteData)
					return data
				}
			} else {
				// We assume this is not 'error' but 'no entry' which is normal
				return nil
			}
		}
	}
	v.log(fmt.Sprintf("Get: error reading entry %s->%s", xPath, name), err)
	return nil
}

func (v *JsonDB) List(xPath string) []string {
	var err error = nil
	if filename, err := v.xPathToFilename(xPath); err == nil {
		v.lock(filename)
		defer v.unlock(filename)
		if root, err := v.readJsonFileAsMap(filename); err == nil {
			nameList := make([]string, 0, len(root))
			for k := range root {
				nameList = append(nameList, k)
			}
			return nameList
		}
	}
	v.log(fmt.Sprintf("List: error reading entries in xPath %s", xPath), err)
	return nil
}

func (v *JsonDB) Rem(xPath, name string) {
	var err error = nil
	if filename, err := v.xPathToFilename(xPath); err == nil {
		v.lock(filename)
		defer v.unlock(filename)
		if root, err := v.readJsonFileAsMap(filename); err == nil {
			delete(root, name)
			if err = v.writeMapAsJsonFile(filename, root); err == nil {
				return
			}
		}
	}
	v.log(fmt.Sprintf("Rem: error removing entry %s->%s", xPath, name), err)
}

func (v *JsonDB) Clear(xPath string) {
	filename, err := v.xPathToFilename(xPath)
	if err != nil {
		v.log(fmt.Sprintf("Clear: error converting xPath %s to filename: %v", xPath, err))
		return
	}

	v.lock(filename)
	defer v.unlock(filename)

	if err := v.writeMapAsJsonFile(filename, map[string]interface{}{}); err != nil {
		v.log(fmt.Sprintf("Clear: error writing empty file for xPath %s: %v", xPath, err))
	}
}

func (v *JsonDB) lock(filename string) {
	jsonDbLocksMutex.Lock()
	mtx, ok := jsonDbLocks[filename]
	if !ok {
		mtx = &sync.Mutex{}
		jsonDbLocks[filename] = mtx
	}
	jsonDbLocksMutex.Unlock()
	mtx.Lock()
}

func (v *JsonDB) unlock(filename string) {
	jsonDbLocksMutex.Lock()
	if mtx, ok := jsonDbLocks[filename]; ok {
		mtx.Unlock()
	}
	jsonDbLocksMutex.Unlock()
}

func (v *JsonDB) xPathToFilename(xPath string) (string, error) {
	if pathComponents := strings.Split(xPath, v.xPathDelimeter); len(pathComponents) > 0 {
		return strings.ToLower(strings.Join(pathComponents, v.filenameDelimiter) + v.filenameExtension), nil
	}
	return "", errors.New("xPath has no components")
}

func (v *JsonDB) readJsonFileAsMap(filename string) (map[string]interface{}, error) {
	jsonData := map[string]interface{}{}
	path := filepath.Join(v.Path, filename)
	fileData, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return jsonData, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(fileData, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON database: %w", err)
	}
	if jsonData == nil {
		return nil, errors.New("JSON database must contain an object")
	}
	return jsonData, nil
}

func (v *JsonDB) writeMapAsJsonFile(filename string, o map[string]interface{}) error {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(v.Path, ".flow-json-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = temporary.Chmod(v.fileMode); err == nil {
		_, err = temporary.Write(data)
	}
	if err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(v.Path, filename))
}

func (v *JsonDB) log(s string, params ...interface{}) {
	if len(params) > 0 {
		log.TLogln(fmt.Sprintf("JsonDB: %s: %s", s, fmt.Sprint(params...)))
	} else {
		log.TLogln(fmt.Sprintf("JsonDB: %s", s))
	}
}
