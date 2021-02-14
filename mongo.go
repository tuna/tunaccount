// mongodb related functions
package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	ldapMsg "github.com/lor00x/goldap/message"
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

func (m *mongoCtx) FilterTagColl() *mgo.Collection {
	return m.session.DB(m.dbname).C(mgoFilterTagColl)
}

func (m *mongoCtx) CounterColl() *mgo.Collection {
	return m.session.DB(m.dbname).C(mgoCounterColl)
}

// FindUsers returns the user list that matches filter and has a specified tag
func (m *mongoCtx) FindUsers(filter bson.M, tag string) []User {
	var results []User
	isAdmin := bson.M{"is_admin": true}
	isActive := bson.M{"is_active": true}

	filters := []bson.M{filter, isActive}

	if tag != "" {
		filters = append(filters, bson.M{"tags": tag})
	}

	m.UserColl().
		Find(bson.M{
			"$or": []bson.M{
				bson.M{"$and": []bson.M{filter, isActive, isAdmin}},
				bson.M{"$and": filters},
			},
		}).
		All(&results)
	return results
}

// FindGroups returns the group list that matches filter and has a specified tag
func (m *mongoCtx) FindGroups(filter bson.M, tag string) []PosixGroup {
	var results []PosixGroup

	filters := []bson.M{
		filter,
		bson.M{"is_active": true},
		bson.M{"tag": bson.M{"$in": []string{tag, ""}}},
	}

	err := m.PosixGroupColl().Find(bson.M{"$and": filters}).All(&results)
	if err != nil {
		logger.Error(err.Error())
	}
	return results
}

func (m *mongoCtx) EnsureTag(tagName string) error {
	coll := m.FilterTagColl()
	cnt, _ := coll.Find(bson.M{"_id": tagName}).Count()
	if cnt == 0 {
		tagRegex := regexp.MustCompile(`[\w-]+`)
		if !tagRegex.MatchString(tagName) {
			return errors.New("Tag must only contains '0-9', 'a-z', 'A-z' and '-'")
		}

		tag := FilterTag{
			Name: tagName,
		}
		return coll.Insert(tag)
	}
	return nil
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
		lval := string(f.AssertionValue())
		if key, ok := keymap[lkey]; ok {
			if ldapIntegerFields[lkey] {
				val, _ := strconv.Atoi(lval)
				res[key] = val
			} else {
				res[key] = lval
			}
		}
	case ldapMsg.FilterGreaterOrEqual:
		lkey := string(f.AttributeDesc())
		lval := string(f.AssertionValue())
		if key, ok := keymap[lkey]; ok {
			if ldapIntegerFields[lkey] {
				val, _ := strconv.Atoi(lval)
				res[key] = bson.M{"$gte": val}
			} else {
				res[key] = bson.M{"$gte": lval}
			}
		}
	case ldapMsg.FilterLessOrEqual:
		lkey := string(f.AttributeDesc())
		lval := string(f.AssertionValue())
		if key, ok := keymap[lkey]; ok {
			if ldapIntegerFields[lkey] {
				val, _ := strconv.Atoi(lval)
				res[key] = bson.M{"$lte": val}
			} else {
				res[key] = bson.M{"$lte": lval}
			}
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
	var addrs []string

	if len(c.Addrs) > 0 {
		addrs = c.Addrs
	} else {
		addrs = []string{fmt.Sprintf("%s:%d", c.Addr, c.Port)}
	}

	dialInfo := &mgo.DialInfo{
		Addrs:          addrs,
		Direct:         false,
		Database:       c.Name,
		ReplicaSetName: c.Options["replica_set"],
		FailFast:       true,
		PoolLimit:      128,
	}
	if c.User != "" && c.Password != "" {
		dialInfo.Username = c.User
		dialInfo.Password = c.Password
	}

	logger.Debugf("dbconfig: %#v", c)
	logger.Debugf("dialInfo: %#v", dialInfo)

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return err
	}

	_mongo = &mongoCtx{
		session: session,
		dbname:  c.Name,
	}

	// don't need to init data in read-only mode
	if dcfg.ReadOnly {
		session.SetMode(mgo.Monotonic, false)
		return nil
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
