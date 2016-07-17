package main

import (
	ldapMsg "github.com/vjeantet/goldap/message"
	"gopkg.in/mgo.v2/bson"
)

func queryLdapUsers(filter ldapMsg.Filter, tag string) []User {
	query := ldapQueryToBson(filter, userldap2bson)
	if tag != "" {
		query["tags"] = bson.M{"$elemMatch": bson.M{"$eq": tag}}
	}

	var results []User

	m := getMongo()
	defer m.Close()

	m.UserColl().Find(query).All(&results)
	return results
}
