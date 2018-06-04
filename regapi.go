package regapi

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	ld "github.com/lamg/ldaputil"
	"github.com/rs/cors"
	"html/template"
)

type RegAPI struct {
	db       *sql.DB
	cr       *JWTCrypt
	rt       *mux.Router
	ld       *ld.Ldap
	evalPath string
	authPath string

	Handler h.Handler
}

func NewRegAPI(pgAddr, user, pass, rootHTML string,
	ld *ld.Ldap) (p *RegAPI, e error) {
	var db *sql.DB
	db, e = sql.Open("postgres",
		fmt.Sprintf("postgres://%s:%s@%s", user, pass, addr))
	var tp *template.Template
	if e == nil {
		tp, e = template.New("doc").ParseFiles(rootHTML)
	}
	if e == nil {
		p = &RegAPI{
			db: db,
			rt: mux.NewRouter(),
			cr: NewJWTCrypt(),
			ld: ld,
		}
		p.rt.HandleFunc("/auth", r.authHn).Methods(h.MethodPost)
		p.rt.HandleFunc("/eval", r.evaluationsHn).Methods(h.MethodGet)
		p.Handler = cors.AllowAll().Handler(d.rt)
	}
	return
}

func (p *RegAPI) authHn(w h.ResponseWriter, r *h.Request) {
	c, e := credentials(r)
	if e == nil {
		e = d.Ld.Authenticate(c.User, c.Pass)
	}
	var s string
	if e == nil {
		s, e = d.cr.encrypt(c)
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
	usr, e := d.cr.decrypt(r)
	var mp map[string][]string
	if e == nil {
		mp, e = d.Ld.FullRecordAcc(usr)
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
	var gs []StudentEvl
	if e == nil {
		gs, e = d.queryEvl(ci)
	}
	if e == nil {
		ev := sortAndRemDup(gs)
		e = Encode(w, ev)
	}
	writeErr(w, e)
}

type SubjEval struct {
	Subject string `json:"subject"`
	Eval    string `json:"eval"`
}

func NoEmployeeIDField(user string) (e error) {
	e = fmt.Errorf("No %s field found for %s", EmployeeID, user)
	return
}

type EvYear struct {
	Year    string     `json:"year"`
	Periods []EvPeriod `json:"periods"`
}

type EvPeriod struct {
	Period string     `json:"period"`
	Evs    []SubjEval `json:"evs"`
}

type ByYearPeriod []StudentEvl

func (b ByYearPeriod) Len() (n int) {
	n = len(b)
	return
}

func (b ByYearPeriod) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b ByYearPeriod) Less(i, j int) (r bool) {
	r = (b[i].Year == b[j].Year && b[i].Period < b[j].Period) ||
		b[i].Year < b[j].Year
	return
}

func sortAndRemDup(ev []StudentEvl) (ys []EvYear) {
	sort.Sort(ByYearPeriod(ev))
	cy, cp, cyi, cpi := "", "", -1, -1 //current year, current period,
	// current year index, current period index
	ys = make([]EvYear, 0)
	for _, j := range ev {
		if j.Year != cy {
			ny := EvYear{
				Year:    j.Year,
				Periods: make([]EvPeriod, 0),
			}
			ys = append(ys, ny)
			cy, cyi = j.Year, cyi+1
			cp, cpi = "", -1
		}
		if j.Period != cp {
			np := EvPeriod{
				Period: j.Period,
				Evs:    make([]SubjEval, 0),
			}
			ys[len(ys)-1].Periods = append(ys[len(ys)-1].Periods, np)
			cp, cpi = j.Period, cpi+1
		}
		nv := SubjEval{
			Eval:    j.EvalValue,
			Subject: j.SubjectName,
		}
		ok := canUpdate(ys[cyi].Periods[cpi].Evs, nv)
		if !ok {
			ys[cyi].Periods[cpi].Evs = append(ys[cyi].Periods[cpi].Evs, nv)
		}
	}

	return
}

func canUpdate(a []SubjEval, v SubjEval) (ok bool) {
	ok = false
	f, i := false, 0 //f: found
	for !f && i != len(a) {
		f = a[i].Subject == v.Subject
		if !f {
			i = i + 1
		}
	}
	if f && v.Eval > a[i].Eval {
		a[i] = v
	}
	ok = f
	return
}
