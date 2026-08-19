management {{ .ManagementSocket }} unix

port {{ .Port }}
proto {{ .Proto }}

dev tun
topology subnet
server 10.8.0.0 255.255.255.0
#server-ipv6

ca {{ .CaCert }}
cert {{ .ServerCert }}
key {{ .ServerKey }}
{{ if .Crl }}crl-verify {{ .Crl }}
{{ end }}

data-ciphers {{ .DataCiphers }}
auth {{ .Auth }}

compress migrate

ifconfig-pool-persist {{ .IfconfigPoolPersist }}

push "dhcp-option DNS 8.8.8.8"
push "dhcp-option DNS 8.8.4.4"
push "dhcp-option DNS6 2001:4860:4860::8888"
push "dhcp-option DNS6 2001:4860:4860::8844"
push "route-ipv6 ::/0"
push "redirect-gateway def1 bypass-dhcp"
push "redirect-gateway ipv6 def1 bypass-dhcp"

keepalive {{ .Keepalive }}
max-clients {{ .MaxClients }}
explicit-exit-notify 1

persist-tun

verb 3
mute 10
