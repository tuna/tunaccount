// mongodb related functions
package main

import (
	"fmt"

	ldapMsg "github.com/vjeantet/goldap/message"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type mongoCtx struct {
	session *mgo.Session
	dbname  string
}

var (
	_mongo *mongoCtx

	mgoIndexes = map[string]([]mgo.Index){
		mgoUserColl: []mgo.Index{
			mgo.Index{
				Key:    []string{"username"},
				Unique: true,
			},
			mgo.Index{
				Key:    []string{"email"},
				Unique: true,
			},
			mgo.Index{
				Key: []string{"tags"},
			},
		},
		mgoPosixGroupColl: []mgo.Index{
			mgo.Index{
				Key:    []string{"tag", "gid"},
				Unique: true,
			},
			mgo.Index{
				Key:    []string{"tag", "name"},
				Unique: true,
			},
		},
	}
)

func getMongo() *mongoCtx {
	return &mongoCtx{_mongo.session.Copy(), _mongo.dbname}
}

func (m *mongoCtx) Close() {
	m.session.Close()
}

func (m *mongoCtx) UserColl() *mgo.Collection {
	return m.session.DB(m.dbname).C(mgoUserColl)
}

func (m *mongoCtx) PosixGroupColl() *mgo.Collection {
	return m.session.DB(m.dbname).C(mgoPosixGroupColl)
}

func (m *mongoCtx) CounterColl() *mgo.Collection {
	return m.session.DB(m.dbname).C(mgoCounterColl)
}

// FindUsers returns the user list that matches filter and has a specified tag
func (m *mongoCtx) FindUsers(filter bson.M, tag string) []User {
	var results []User

	filters := []bson.M{filter, bson.M{"is_active": true}}

	if tag != "" {
		filters = append(filters, bson.M{"tags": tag})
	}

	m.UserColl().Find(bson.M{"$and": filters}).All(&results)
	return results
}

// FindGroups returns the group list that matches filter and has a specified tag
func (m *mongoCtx) FindGroups(filter bson.M, tag string) []PosixGroup {
	var results []PosixGroup

	filters := []bson.M{filter, bson.M{"is_active": true}}

	if tag != "" {
		filters = append(filters, bson.M{"tags": tag})
	}

	m.PosixGroupColl().Find(bson.M{"$and": filters}).All(&results)
	return results
}

// ldapQueryToBson convers an LDAP query to BSON filter
// the keymap maps ldap attribute to mongo doc key, e.g. userldap2bson
func ldapQueryToBson(filter ldapMsg.Filter, keymap map[string]string) bson.M {
	res := bson.M{}

	switch f := filter.(type) {
	case ldapMsg.FilterAnd:
		cfilters := []bson.M{}
		for _, child := range f {
			cf := ldapQueryToBson(child, keymap)
			if len(cf) > 0 {
				cfilters = append(cfilters, cf)
			}
		}
		if len(cfilters) > 1 {
			res["$and"] = cfilters
		} else if len(cfilters) == 1 {
			res = cfilters[0]
		}
	case ldapMsg.FilterOr:
		cfilters := []bson.M{}
		for _, child := range f {
			cf := ldapQueryToBson(child, keymap)
			if len(cf) > 0 {
				cfilters = append(cfilters, cf)
			}
		}
		if len(cfilters) > 1 {
			res["$or"] = cfilters
		} else if len(cfilters) == 1 {
			res = cfilters[0]
		}
	case ldapMsg.FilterNot:
		cf := ldapQueryToBson(f.Filter, keymap)
		if len(cf) > 0 {
			res["$not"] = cf
		}
	case ldapMsg.FilterEqualityMatch:
		lkey := string(f.AttributeDesc())
		// attributes not listed in the keymap is ignored
		if key, ok := keymap[lkey]; ok {
			res[key] = f.AssertionValue()
		}
	case ldapMsg.FilterGreaterOrEqual:
		lkey := string(f.AttributeDesc())
		if key, ok := keymap[lkey]; ok {
			res[key] = bson.M{"$gte": f.AssertionValue()}
		}
	case ldapMsg.FilterLessOrEqual:
		lkey := string(f.AttributeDesc())
		if key, ok := keymap[lkey]; ok {
			res[key] = bson.M{"$lte": f.AssertionValue()}
		}
	}
	return res
}

func (m *mongoCtx) getNextSeq(ID string) int {
	counter := mongoCounter{}

	change := mgo.Change{
		Update: bson.M{
			"$inc": bson.M{"seq": 1},
		},
		ReturnNew: true,
	}
	_, err := m.CounterColl().Find(bson.M{"_id": ID}).Apply(change, &counter)
	if err != nil {
		panic(err)
	}
	return counter.Seq
}

func (m *mongoCtx) ensureCounterMin(ID string, val int) {

	err := m.CounterColl().
		Update(bson.M{
			"$and": []bson.M{
				bson.M{"_id": ID},
				bson.M{"seq": bson.M{"$lt": val}},
			},
		}, bson.M{
			"$set": bson.M{"seq": val},
		})

	if err != nil {
		if err != mgo.ErrNotFound {
			panic(err)
		}
	}
}

func initMongo() error {

	c := dcfg.DB
	url := ""
	if c.User != "" && c.Password != "" {
		url = url + fmt.Sprintf("%s:%s@", c.User, c.Password)
	}
	url = url + fmt.Sprintf("%s:%d/%s", c.Addr, c.Port, c.Name)

	session, err := mgo.Dial(url)
	if err != nil {
		return err
	}

	_mongo = &mongoCtx{
		session: session,
		dbname:  c.Name,
	}

	// init indexes
	db := session.DB(c.Name)
	for cname, indexes := range mgoIndexes {
		for _, index := range indexes {
			err = db.C(cname).EnsureIndex(index)
			if err != nil {
				return err
			}
		}
	}

	// seqStart
	seqStart := map[string]int{
		"uid": dcfg.TUNA.MinimumGID,
		"gid": dcfg.TUNA.MinimumGID,
	}
	for k, v := range seqStart {
		if cnt, _ := _mongo.CounterColl().FindId(k).Count(); cnt == 0 {
			_mongo.CounterColl().Insert(mongoCounter{ID: k, Seq: v})
		}
	}

	return nil

}
