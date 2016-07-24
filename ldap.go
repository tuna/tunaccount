package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2/bson"

	ldapMsg "github.com/vjeantet/goldap/message"
	ldap "github.com/vjeantet/ldapserver"
)

func makeLDAPServer(listenAddr string) *ldap.Server {
	//Create a new LDAP Server
	server := ldap.NewServer()
	ldap.Logger = ldap.DiscardingLogger

	//Create routes bindings
	routes := ldap.NewRouteMux()
	routes.Abandon(handleAbandon)
	routes.Bind(handleBind)

	routes.Search(handleSearchTUNA).
		// BaseDn("o=tuna").
		// Scope(ldap.SearchRequestHomeSubtree).
		Label("Search - TUNA")

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
func handleSearchTUNA(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	logger.Debugf("Request BaseDN: %s", r.BaseObject())
	logger.Debugf("Request Filter: %s", r.FilterString())
	logger.Debugf("Request Attributes: %s", r.Attributes())

	// Handle Stop Signal (server stop / client disconnected / Abandoned request....)
	select {
	case <-m.Done:
		log.Print("Leaving handleSearch...")
		return
	default:
	}

	mg := getMongo()
	defer mg.Close()
	filter := ldapQueryToBson(r.Filter(), userldap2bson)
	logger.Debugf("Mongo Filter: %#v", filter)
	users := mg.FindUsers(filter, "")
	for _, u := range users {
		e := ldap.NewSearchResultEntry(fmt.Sprintf("uid=%s,ou=people,o=tuna", u.Username))
		e.AddAttribute("uid", ldapMsg.AttributeValue(u.Username))
		e.AddAttribute("cn", ldapMsg.AttributeValue(u.Username))
		e.AddAttribute("uidNumber", ldapMsg.AttributeValue(strconv.Itoa(u.UID)))
		e.AddAttribute("gidNumber", ldapMsg.AttributeValue(strconv.Itoa(u.GID)))
		e.AddAttribute("loginShell", ldapMsg.AttributeValue(u.LoginShell))
		e.AddAttribute("homeDirectory", ldapMsg.AttributeValue(fmt.Sprintf("/home/%s", u.Username)))
		e.AddAttribute("userPassword", "{SSHA}/Sq+jS8CZM8e+DAFJkTnVtnIuEU/FwYz")
		e.AddAttribute("objectClass", "top")
		e.AddAttribute("objectClass", "posixAccount")
		e.AddAttribute("objectClass", "shadowAccount")
		e.AddAttribute("gecos", ldapMsg.AttributeValue(u.Name))
		e.AddAttribute("shadowMax", "99999")
		w.Write(e)
	}
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}
