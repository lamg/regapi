package regapi

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	ld "github.com/lamg/ldaputil"
	"github.com/rs/cors"
	"html/template"
	h "net/http"
	"sort"
	"strings"
)

type RegAPI struct {
	db       *sql.DB
	cr       *JWTCrypt
	rt       *mux.Router
	ld       *ld.Ldap
	tp       *template.Template
	evalPath string
	authPath string

	Handler h.Handler
}

func NewRegAPI(pgAddr, user, pass, rootHTML string,
	ld *ld.Ldap) (p *RegAPI, e error) {
	var db *sql.DB
	db, e = sql.Open("postgres",
		fmt.Sprintf("postgres://%s:%s@%s", user, pass, pgAddr))
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
			tp: tp,
		}
		p.authPath, p.evalPath = "/auth", "/eval"
		p.rt.HandleFunc(p.authPath, p.authHn).Methods(h.MethodPost)
		p.rt.HandleFunc(p.evalPath, p.evaluationsHn).Methods(h.MethodGet)
		p.rt.HandleFunc("/", p.docHn)
		p.Handler = cors.AllowAll().Handler(p.rt)
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
	var gs []StudentEvl
	if e == nil {
		gs, e = p.queryEvl(ci)
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

type StudentEvl struct {
	SubjectName string `json:"subjectName"`
	EvalValue   string `json:"evalValue"`
	Period      string `json:"period"`
	Year        string `json:"year"`
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

func (p *RegAPI) queryEvl(idStudent string) (es []StudentEvl, e error) {
	query := fmt.Sprintf("SELECT id_student FROM student WHERE "+
		" identification = '%s'", idStudent)
	// println("query: " + query)
	// print("idStudent: ")
	// println(idStudent)
	var r *sql.Rows
	r, e = p.db.Query(query)
	var studDBId string
	// print("error: ")
	// println(e != nil)
	ok := r.Next()
	// print("ok: ")
	// println(ok)
	// rerr := r.Err()
	// if rerr != nil {
	// 	print("rerr: ")
	// 	println(rerr.Error())
	// }
	if e == nil && ok {
		e = r.Scan(&studDBId)
	}
	// print("studDBId: ")
	// println(studDBId)
	if e == nil {
		r.Close()
		query = fmt.Sprintf(
			"SELECT evaluation_value_fk,matriculated_subject_fk "+
				" FROM evaluation WHERE student_fk = '%s'", studDBId)
		r, e = p.db.Query(query)
	}
	evalValId, matSubjId := make([]string, 0), make([]string, 0)
	for i := 0; e == nil && r.Next(); i++ {
		var ev, ms sql.NullString
		e = r.Scan(&ev, &ms)
		if e == nil && ev.Valid && ms.Valid {
			evalValId, matSubjId = append(evalValId, ev.String),
				append(matSubjId, ms.String)
		}
	}
	// print("matSubjId: ")
	// println(len(matSubjId))
	// print("evalValId: ")
	// println(len(evalValId))
	// print("error: ")
	// println(e != nil)
	evalVal := make([]string, 0)
	for i := 0; e == nil && i != len(evalValId); i++ {
		r.Close()
		query = fmt.Sprintf("SELECT value FROM evaluation_value WHERE "+
			"id_evaluation_value = '%s'", evalValId[i])
		// print("query evalValId: ")
		// println(query)
		r, e = p.db.Query(query)
		var ev string
		if e == nil && r.Next() {
			e = r.Scan(&ev)
		}
		if e == nil {
			evalVal = append(evalVal, ev)
		}
	}
	// print("evalVal: ")
	// println(len(evalVal))
	subjId := make([]string, 0)
	for i := 0; e == nil && i != len(matSubjId); i++ {
		r.Close()
		query = fmt.Sprintf(
			"SELECT subject_fk FROM matriculated_subject WHERE "+
				"matriculated_subject_id = '%s'", matSubjId[i])
		r, e = p.db.Query(query)
		var si string
		if e == nil && r.Next() {
			e = r.Scan(&si)
		}
		if e == nil {
			subjId = append(subjId, si)
		}
	}
	// print("subjId: ")
	// println(len(subjId))
	subjNameId, subjPeriod, subjYear := make([]string, 0),
		make([]string, 0), make([]string, 0)
	for i := 0; e == nil && i != len(subjId); i++ {
		r.Close()
		query = fmt.Sprintf(
			"SELECT subject_name_fk, period, year FROM subject WHERE "+
				"subject_id = '%s'", subjId[i])
		r, e = p.db.Query(query)
		var sni, period, year string
		if e == nil && r.Next() {
			e = r.Scan(&sni, &period, &year)
		}
		if e == nil {
			subjNameId, subjPeriod, subjYear =
				append(subjNameId, sni),
				append(subjPeriod, period),
				append(subjYear, year)
		}
	}
	// print("subjNameId: ")
	// println(len(subjNameId))
	subjName := make([]string, 0)
	for i := 0; e == nil && i != len(subjNameId); i++ {
		r.Close()
		query = fmt.Sprintf("SELECT name FROM subject_name WHERE "+
			"subject_name_id = '%s'", subjNameId[i])
		r, e = p.db.Query(query)
		var sn string
		if e == nil && r.Next() {
			e = r.Scan(&sn)
		}
		if e == nil {
			subjName = append(subjName, sn)
		}
	}
	// print("subjName: ")
	// println(len(subjName))
	es = make([]StudentEvl, len(subjName))
	for i := 0; e == nil && i != len(subjName); i++ {
		es[i] = StudentEvl{
			SubjectName: subjName[i],
			Period:      subjPeriod[i],
			Year:        subjYear[i],
			EvalValue:   evalVal[i],
		}
	}
	return
}
