package main

func main() {
	var addr, user, pass string
	flag.StringVar(&addr, "a", "", "regapi server address")
	flag.StringVar(&user, "u", "", "user name")
	flag.StringVar(&pass, "p", "", "password")

	flag.Parse()
	tr := &h.Transport{
		Proxy: nil,
	}
	h.DefaultClient.Transport = tr
	var e error
	c := &regapi.Credentials{
		User: user,
		Pass: pass,
	}
	var bs []byte
	bs, e = json.Marshal(c)
	var rq *h.Request
	if e == nil {
		bf := bytes.NewReader(bs)
		rq, e = h.NewRequest(h.MethodPost, addr+"/auth", bf)
	}
	var r *h.Response
	if e == nil {
		r, e = h.DefaultClient.Do(rq)
	}
	var q *h.Request
	if e == nil {
		q, e = h.NewRequest(h.MethodGet, addr+"/eval", nil)
	}
	if e == nil {
		q.Header.Set(regapi.AuthHd, evals)
		r, e = h.DefaultClient.Do(q)
		if e == nil && r.StatusCode != h.StatusOK {
			e = fmt.Errorf("Error: %d", r.StatusCode)
		}
	}
	if e != nil {
		if r != nil {
			log.Fatalf("error: %s code: %d", e.Error(), r.StatusCode)
		} else {
			log.Fatalf("error: %s", e.Error())
		}
	} else {
		printBody(r)
	}
}

func printBody(r *h.Response) {
	body, e := ioutil.ReadAll(r.Body)
	if e == nil {
		fmt.Println(string(body))
	}
}
