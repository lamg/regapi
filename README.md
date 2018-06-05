# RegAPI

RegAPI es una API para autenticar a usuarios y obtener sus notas, usando la información presente en un servidor LDAP.

```
conf
[Unit]
Description=Sigapi Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/dir
ExecStart=/usr/local/bin/regapi -a sigapi_url -ad ldap_address -sf ldap_account_suffix -bnd ldap_base_dn -adu usuario -adp contraseña -s :8081 -l doc.html
Restart=on-abort

[Install]
WantedBy=multi-user.target
```