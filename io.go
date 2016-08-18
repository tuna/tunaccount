// import and export data

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// import models from json file
func importJSON(filename string, m *mongoCtx) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %s", filename, err.Error())
	}

	var dump DBDump
	if err := json.Unmarshal(buf, &dump); err != nil {
		return fmt.Errorf("failed to unmarsal json: %s", err.Error())
	}

	if m == nil {
		m = getMongo()
		defer m.Close()
	}

	for _, user := range dump.Users {
		m.ensureCounterMin("uid", user.UID)
		m.UserColl().Insert(user)
	}
	for _, group := range dump.PosixGroups {
		m.ensureCounterMin("gid", group.GID)
		m.PosixGroupColl().Insert(group)
	}

	return nil
}
