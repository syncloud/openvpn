client
dev tun
proto {{ .Proto }}
remote {{ .ServerAddress }} {{ .Port }}
resolv-retry infinite
nobind
persist-tun
remote-cert-tls server
data-ciphers {{ .DataCiphers }}
auth {{ .Auth }}
verb 3
<ca>
{{ .Ca }}</ca>
<cert>
{{ .Cert }}</cert>
<key>
{{ .Key }}</key>
