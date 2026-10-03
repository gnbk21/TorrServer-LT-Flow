package settings

import "errors"

// Maintenance operations must distinguish a durable write from a cached value.
// Legacy callers retain Set; every production backend also supplies PutChecked.
func putChecked(db TorrServerDB, path, name string, value []byte) error {
	writer, ok := db.(interface {
		PutChecked(string, string, []byte) error
	})
	if !ok {
		return errors.New("database does not support verified writes")
	}
	return writer.PutChecked(path, name, value)
}
func (v *XPathDBRouter) PutChecked(path, name string, value []byte) error {
	return putChecked(v.getDBForXPath(path), path, name, value)
}
func (v *DBReadCache) PutChecked(path, name string, value []byte) error {
	if ReadOnly {
		return errors.New("database is read-only")
	}
	if v.db == nil {
		return errors.New("database is closed")
	}
	if err := putChecked(v.db, path, name, value); err != nil {
		return err
	}
	v.dataCacheMutex.Lock()
	if v.dataCache != nil {
		v.dataCache[v.makeDataCacheKey(path, name)] = append([]byte(nil), value...)
	}
	v.dataCacheMutex.Unlock()
	v.listCacheMutex.Lock()
	delete(v.listCache, path)
	v.listCacheMutex.Unlock()
	return nil
}
