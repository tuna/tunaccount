package main

import (
	"log"
	"strings"

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
	go server.ListenAndServe(listenAddr)

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
	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
	if r.AuthenticationChoice() == "simple" {
		if string(r.Name()) == "" {
			log.Printf("Bind succeed User=%s, Pass=%#v", string(r.Name()), r.Authentication())
			w.Write(res)
			return
		} else if string(r.Name()) == "uid=bigeagle,ou=people,o=tuna" {
			if string(r.AuthenticationSimple()) == "123" {
				log.Printf("Bind succeed User=%s, Pass=%#v", string(r.Name()), r.Authentication())
				w.Write(res)
				return
			}
		}
		log.Printf("Bind failed User=%s, Pass=%#v", string(r.Name()), r.Authentication())
		res.SetResultCode(ldap.LDAPResultInvalidCredentials)
		res.SetDiagnosticMessage("invalid credentials")
	} else {
		res.SetResultCode(ldap.LDAPResultUnwillingToPerform)
		res.SetDiagnosticMessage("Authentication choice not supported")
	}

	w.Write(res)
}

// handle search function
func handleSearchTUNA(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	log.Printf("Request BaseDn=%s", r.BaseObject())
	log.Printf("Request Filter=%s", r.Filter())
	log.Printf("Request FilterString=%s", r.FilterString())
	log.Printf("Request Attributes=%s", r.Attributes())
	log.Printf("Request TimeLimit=%d", r.TimeLimit().Int())

	// Handle Stop Signal (server stop / client disconnected / Abandoned request....)
	select {
	case <-m.Done:
		log.Print("Leaving handleSearch...")
		return
	default:
	}

	var makeFilter func(ldapMsg.Filter, []ldapMsg.FilterEqualityMatch) []ldapMsg.FilterEqualityMatch
	makeFilter = func(packet ldapMsg.Filter, acc []ldapMsg.FilterEqualityMatch) []ldapMsg.FilterEqualityMatch {
		switch f := packet.(type) {
		case ldapMsg.FilterAnd:
			for _, child := range f {
				cFilters := makeFilter(child, []ldapMsg.FilterEqualityMatch{})
				acc = append(acc, cFilters...)
			}
			return acc
		case ldapMsg.FilterEqualityMatch:
			return append(acc, f)
		}
		return []ldapMsg.FilterEqualityMatch{}
	}

	filters := makeFilter(r.Filter(), []ldapMsg.FilterEqualityMatch{})

	e := ldap.NewSearchResultEntry("uid=bigeagle,ou=people,o=tuna")
	e.AddAttribute("uid", "bigeagle")
	e.AddAttribute("cn", "bigeagle")
	e.AddAttribute("uidNumber", "1100")
	e.AddAttribute("gidNumber", "1001")
	e.AddAttribute("loginShell", "/bin/bash")
	e.AddAttribute("homeDirectory", "/home/bigeagle")
	e.AddAttribute("userPassword", "{SSHA}/Sq+jS8CZM8e+DAFJkTnVtnIuEU/FwYz")
	e.AddAttribute("objectClass", "top")
	e.AddAttribute("objectClass", "posixAccount")
	e.AddAttribute("objectClass", "shadowAccount")
	e.AddAttribute("gecos", "")
	e.AddAttribute("shadowMax", "99999")

	var isBigEagle = true
	for _, f := range filters {
		log.Printf("%v=%v", f.AttributeDesc(), f.AssertionValue())
		switch strings.ToLower(string(f.AttributeDesc())) {
		case "objectclass":
			continue
		case "uid":
			if string(f.AssertionValue()) != "bigeagle" {
				isBigEagle = false
			}
		}
	}

	log.Printf("isBigEagle: %v", isBigEagle)
	if isBigEagle {
		w.Write(e)
	}

	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}
