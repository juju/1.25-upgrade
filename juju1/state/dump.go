// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import "github.com/juju/errors"

// DumpAll returns a map of collection names to a slice of documents
// in that collection. Every document that is related to the current
// model is returned in the map.
func (st *State) DumpAll() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for name, _ := range allCollections() {
		// Just skip the mgo txn collections
		if name == txnsC || name == txnLogC {
			continue
		}

		docs, err := getAllCollectionDocs(st, name)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(docs) > 0 {
			result[name] = docs
		}
	}
	return result, nil
}

func getAllCollectionDocs(st *State, collectionName string) ([]map[string]interface{}, error) {
	coll, closer := st.getRawCollection(collectionName)
	defer closer()

	var (
		result []map[string]interface{}
		doc    map[string]interface{}
	)
	// Always output in id order.
	iter := coll.Find(nil).Sort("_id").Iter()
	defer iter.Close()
	for iter.Next(&doc) {
		delete(doc, "txn-revno")
		delete(doc, "txn-queue")
		result = append(result, doc)
		doc = nil
	}

	if err := iter.Err(); err != nil {
		return nil, errors.Annotatef(err, "reading collection %q", collectionName)
	}
	return result, nil
}
