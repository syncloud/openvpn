package server

type Resolver struct {
	DNS    []string
	DNS6   []string
	Domain string
}

var PublicResolver = Resolver{
	DNS:  []string{"8.8.8.8", "8.8.4.4"},
	DNS6: []string{"2001:4860:4860::8888", "2001:4860:4860::8844"},
}
