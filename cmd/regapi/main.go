package main

import (
	"flag"
	"fmt"
	"github.com/lamg/ldaputil"
	"github.com/lamg/regapi"
	h "net/http"
	"os"
	"time"
)

func main() {
	var addr, user, pass, srv, tmpl string
	flag.StringVar(&addr, "a", "", "Dirección del servidor")
	flag.StringVar(&user, "u", "", "Usuario")
	flag.StringVar(&pass, "p", "", "Contraseña")
	flag.StringVar(&srv, "s", ":8080",
		"Dirección para servir la API")
	flag.StringVar(&tmpl, "l", "",
		"Camino de la plantilla de la documentación")
	var adAddr, suff, bdn, adUser, adPass string
	flag.StringVar(&adAddr, "ad", "", "LDAP server address")
	flag.StringVar(&suff, "sf", "", "LDAP server account suffix")
	flag.StringVar(&bdn, "bdn", "", "LDAP server base DN")
	flag.StringVar(&adUser, "adu", "", "Usuario del AD")
	flag.StringVar(&adPass, "adp", "", "Contraseña del AD")
	flag.Parse()
	ld := ldaputil.NewLdapWithAcc(adAddr, suff, bdn, adUser, adPass)
	r, e := regapi.NewRegAPI(addr, user, pass, tmpl, ld)
	if e == nil {
		s := h.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Addr:         srv,
			Handler:      r.Handler,
		}
		e = s.ListenAndServe()
	}
	if e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e.Error())
		os.Exit(1)
	}
}
