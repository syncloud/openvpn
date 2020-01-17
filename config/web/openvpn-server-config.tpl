management {{ '{{' }} .Management {{ '}}' }}

port {{ '{{' }} .Port {{ '}}' }}
proto {{ '{{' }} .Proto {{ '}}' }}

dev tun

ca {{ '{{' }} .Ca {{ '}}' }}
cert {{ '{{' }} .Cert {{ '}}' }}
key {{ '{{' }} .Key {{ '}}' }}

cipher {{ '{{' }} .Cipher {{ '}}' }}
auth {{ '{{' }} .Auth {{ '}}' }}
dh {{ '{{' }} .Dh {{ '}}' }}

server 10.8.0.0 255.255.255.0
{{ ipv6_config }}
ifconfig-pool-persist {{ '{{' }} .IfconfigPoolPersist {{ '}}' }}
push "route 10.8.0.0 255.255.255.0"
push "dhcp-option DNS 8.8.8.8"
push "dhcp-option DNS 8.8.4.4"
push "dhcp-option DNS 2001:4860:4860::8888"
push "dhcp-option DNS 2001:4860:4860::8844"
push "route-ipv6 2000::/3"

keepalive {{ '{{' }} .Keepalive {{ '}}' }}

comp-lzo
max-clients {{ '{{' }} .MaxClients {{ '}}' }}

persist-key
persist-tun

verb 3

mute 10

