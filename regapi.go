package regapi

import (
	"fmt"
	"github.com/gorilla/mux"
	ld "github.com/lamg/ldaputil"
	"github.com/rs/cors"
	"html/template"
	"io/ioutil"
	h "net/http"
	"strings"
)

type RegAPI struct {
	cr        *JWTCrypt
	tp        *template.Template
	ld        *ld.Ldap
	evalPath  string
	authPath  string
	sigapiURL string
	Handler   h.Handler
}

func NewRegAPI(sigapiURL, rootHTML string,
	ld *ld.Ldap) (p *RegAPI, e error) {
	var tp *template.Template
	tp, e = template.New("doc").ParseFiles(rootHTML)
	if e == nil {
		p = &RegAPI{
			cr:        NewJWTCrypt(),
			ld:        ld,
			tp:        tp,
			sigapiURL: sigapiURL,
		}
		p.authPath, p.evalPath = "/auth", "/eval"
		rt := mux.NewRouter()
		rt.HandleFunc(p.authPath, p.authHn).Methods(h.MethodPost)
		rt.HandleFunc(p.evalPath, p.evaluationsHn).Methods(h.MethodGet)
		rt.HandleFunc("/", p.docHn)
		p.Handler = cors.AllowAll().Handler(rt)
	}
	return
}

func (p *RegAPI) docHn(w h.ResponseWriter, r *h.Request) {
	e := p.tp.ExecuteTemplate(w, "doc", struct {
		AuthPath string
		EvalPath string
	}{
		AuthPath: p.authPath,
		EvalPath: p.evalPath,
	})
	writeErr(w, e)
}

func (p *RegAPI) authHn(w h.ResponseWriter, r *h.Request) {
	c, e := credentials(r)
	if e == nil {
		e = p.ld.Authenticate(c.User, c.Pass)
	}
	var s string
	if e == nil {
		s, e = p.cr.encrypt(c)
	}
	if e == nil {
		w.Write([]byte(s))
	}
	writeErr(w, e)
}

const (
	EmployeeID = "employeeID"
)

func (p *RegAPI) evaluationsHn(w h.ResponseWriter, r *h.Request) {
	usr, e := p.cr.decrypt(r)
	var mp map[string][]string
	if e == nil {
		mp, e = p.ld.FullRecordAcc(usr)
	}
	var ci string
	if e == nil {
		cia, ok := mp[EmployeeID]
		if ok && len(cia) != 0 {
			ci = strings.TrimSpace(cia[0])
		}
		if ci == "" {
			e = NoEmployeeIDField(usr)
		}
	}
	var n *h.Response
	if e == nil {
		url := fmt.Sprintf("%s/eval/%s", p.sigapiURL, ci)
		n, e = h.DefaultClient.Get(url)
	}
	var bs []byte
	if e == nil {
		bs, e = ioutil.ReadAll(n.Body)
	}
	if e == nil {
		n.Body.Close()
		w.Write(bs)
	}
	writeErr(w, e)
}

func NoEmployeeIDField(user string) (e error) {
	e = fmt.Errorf("No %s field found for %s", EmployeeID, user)
	return
}
