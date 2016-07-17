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
	// seq start value
	seqStart map[string]int

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

func (m *mongoCtx) ConterColl() *mgo.Collection {
	return m.session.DB(m.dbname).C(mgoCounterColl)
}

func ldapQueryToBson(filter ldapMsg.Filter, keymap map[string]string) bson.M {
	res := bson.M{}
	switch f := filter.(type) {
	case ldapMsg.FilterAnd:
		cfilters := []bson.M{}
		for _, child := range f {
			cfilters = append(cfilters, ldapQueryToBson(child, keymap))
		}
		res["$and"] = cfilters
	case ldapMsg.FilterOr:
		cfilters := []bson.M{}
		for _, child := range f {
			cfilters = append(cfilters, ldapQueryToBson(child, keymap))
		}
		res["$or"] = cfilters
	case ldapMsg.FilterNot:
		res["$not"] = ldapQueryToBson(f.Filter, keymap)
	case ldapMsg.FilterEqualityMatch:
		lkey := string(f.AttributeDesc())
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

func (m *mongoCtx) getNextSeq(ID string, start int) int {
	counter := mongoCounter{}

	start, ok := seqStart[ID]
	if !ok {
		start = 0
	}

	change := mgo.Change{
		Update: bson.M{
			"$inc":         bson.M{"seq": 1},
			"$setOnInsert": bson.M{"_id": ID, "seq": start},
		},
		Upsert:    true,
		ReturnNew: true,
	}
	m.ConterColl().Find(bson.M{"_id": ID}).Apply(change, &counter)
	return counter.Seq
}

func initMongo() error {

	c := dcfg.DB
	url := ""
	if c.User != "" && c.Password != "" {
		url = url + fmt.Sprintf("%s:%s@", c.User, c.Password)
	}
	url = url + fmt.Sprintf("%s:%s/%s", c.Addr, c.Port, c.Name)

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
	seqStart = map[string]int{
		"uid": dcfg.TUNA.MinimumGID,
		"gid": dcfg.TUNA.MinimumGID,
	}

	return nil

}
