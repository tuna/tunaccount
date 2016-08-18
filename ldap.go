package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2/bson"

	ldapMsg "github.com/vjeantet/goldap/message"
	ldap "github.com/vjeantet/ldapserver"
)

var (
	tagRegex, userRegex, groupRegex *regexp.Regexp
)

func makeLDAPServer(listenAddr string) *ldap.Server {
	//Create a new LDAP Server
	server := ldap.NewServer()
	ldap.Logger = ldap.DiscardingLogger

	//Create routes bindings
	routes := ldap.NewRouteMux()
	routes.Abandon(handleAbandon)
	routes.Bind(handleBind)

	// Regexes
	tagRegex = regexp.MustCompile(`tag=([\w-]+)`)
	userRegex = regexp.MustCompile(`(uid)=([\w-]+)`)
	groupRegex = regexp.MustCompile(`(cn)=([\w-]+)`)

	routes.Search(handleSearch)

	//Attach routes to server
	server.Handle(routes)

	// listen on 10389 and serve
	go func() {
		if err := server.ListenAndServe(listenAddr); err != nil {
			log.Printf("LDAP Listen Error: %s", err.Error())
		}
	}()

	return server
}

func handleAbandon(w ldap.ResponseWriter, m *ldap.Message) {
	var req = m.GetAbandonRequest()
	// retreive the request to abandon, and send a abort signal to it
	if requestToAbandon, ok := m.Client.GetMessageByID(int(req)); ok {
		requestToAbandon.Abandon()
		log.Printf("Abandon signal sent to request processor [messageID=%d]", int(req))
	}
}

// handleBind handles simple authentication
func handleBind(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetBindRequest()
	logger.Debugf("Request bind: %s", string(r.Name()))

	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
	if r.AuthenticationChoice() == "simple" {
		dn := string(r.Name()) // uid=xxxx,ou=xxx
		if dn == "" {
			// Allow anonymous bind
			w.Write(res)
			return
		}
		cnPair := strings.Split(dn, ",")[0]
		cn := strings.Split(cnPair, "=")[1]

		mg := getMongo()
		defer mg.Close()

		logger.Debugf("Filter user: %s", cn)
		users := mg.FindUsers(bson.M{"username": cn}, "")
		if len(users) > 0 {
			user := users[0]
			logger.Debugf("User: %#v", user)
			pass := string(r.AuthenticationSimple())
			if user.Authenticate(pass) {
				logger.Debugf("Successfully authenticated user: %s", user.Username)
				w.Write(res)
				return
			}
			res.SetResultCode(ldap.LDAPResultInvalidCredentials)
			res.SetDiagnosticMessage("invalid credentials")
		} else {
			res.SetResultCode(ldap.LDAPResultNoSuchObject)
			res.SetDiagnosticMessage("User not found")
		}
	} else {
		res.SetResultCode(ldap.LDAPResultUnwillingToPerform)
		res.SetDiagnosticMessage("Authentication choice not supported")
	}
	w.Write(res)
}

// handle search function
func handleSearch(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	logger.Debugf("Request BaseDN: %s", r.BaseObject())
	logger.Debugf("Request Filter: %s", r.FilterString())
	logger.Debugf("Request Attributes: %s", r.Attributes())

	// filter out invalid suffixes
	if !strings.HasSuffix(string(r.BaseObject()), dcfg.LDAP.Suffix) {
		return
	}

	// Handle Stop Signal (server stop / client disconnected / Abandoned request....)
	select {
	case <-m.Done:
		log.Print("Leaving handleSearch...")
		return
	default:
	}

	mg := getMongo()
	defer mg.Close()

	// determine people/group and tag
	segs := strings.Split(string(r.BaseObject()), ",")
	baseFilter := bson.M{}
	var tag, ou, baseKey, baseVal string
	var keymap map[string]string
	for _, seg := range segs {
		switch seg {
		case "ou=people", "ou=People", "ou=users", "ou=Users":
			ou = "people"
			keymap = userldap2bson
		case "ou=groups", "ou=Group", "ou=group", "ou=Groups":
			ou = "groups"
			keymap = groupldap2bson
		}
		if tagRegex.MatchString(seg) {
			tag = tagRegex.FindStringSubmatch(seg)[1]
		} else if userRegex.MatchString(seg) {
			fields := userRegex.FindStringSubmatch(seg)
			baseKey, baseVal = fields[0], fields[1]
		} else if groupRegex.MatchString(seg) {
			fields := groupRegex.FindStringSubmatch(seg)
			baseKey, baseVal = fields[0], fields[1]
		}
	}

	if ou == "" {
		logger.Warningf("Invalid search DN")
		return
	}

	if baseKey != "" {
		baseFilter[keymap[baseKey]] = baseVal
	}

	filter := ldapQueryToBson(r.Filter(), keymap)
	if len(filter) == 0 {
		filter = baseFilter
	} else {
		filter = bson.M{"$and": []bson.M{
			filter, baseFilter,
		}}
	}

	logger.Debugf("Mongo Filter: %#v", filter)

	if ou == "people" {
		users := mg.FindUsers(filter, tag)
		for _, u := range users {
			e := ldap.NewSearchResultEntry(fmt.Sprintf("uid=%s,ou=people,%s", u.Username, dcfg.LDAP.Suffix))
			e.AddAttribute("uid", ldapMsg.AttributeValue(u.Username))
			e.AddAttribute("cn", ldapMsg.AttributeValue(u.Username))
			e.AddAttribute("uidNumber", ldapMsg.AttributeValue(strconv.Itoa(u.UID)))
			e.AddAttribute("gidNumber", ldapMsg.AttributeValue(strconv.Itoa(u.GID)))
			e.AddAttribute("loginShell", ldapMsg.AttributeValue(u.LoginShell))
			e.AddAttribute("homeDirectory", ldapMsg.AttributeValue(fmt.Sprintf("/home/%s", u.Username)))
			e.AddAttribute("userPassword", ldapMsg.AttributeValue(u.Password))
			e.AddAttribute("objectClass", "top")
			e.AddAttribute("objectClass", "posixAccount")
			e.AddAttribute("objectClass", "shadowAccount")
			e.AddAttribute("gecos", ldapMsg.AttributeValue(u.Name))
			e.AddAttribute("shadowMax", "99999")
			w.Write(e)
		}
	} else if ou == "groups" {
		groups := mg.FindGroups(filter, tag)
		for _, g := range groups {
			e := ldap.NewSearchResultEntry(fmt.Sprintf("cn=%s,ou=groups,%s", g.Name, dcfg.LDAP.Suffix))
			e.AddAttribute("cn", ldapMsg.AttributeValue(g.Name))
			e.AddAttribute("gidNumber", ldapMsg.AttributeValue(strconv.Itoa(g.GID)))
			for _, username := range g.Members {
				e.AddAttribute("memberUid", ldapMsg.AttributeValue(username))
			}
			e.AddAttribute("objectClass", "top")
			e.AddAttribute("objectClass", "posixGroup")
		}
	}

	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}
