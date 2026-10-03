package football

// Curated local artwork, separate from sporting data. Attribution and licenses:
// frontend/public/crests/manifest.json. Unknown clubs retain an empty logo URL.
var brazilCrests = map[string]string{
	"botafogo fr":             "/crests/botafogo.svg",
	"sao paulo fc":            "/crests/sao-paulo.svg",
	"ca mineiro":              "/crests/atletico-mineiro.svg",
	"cr flamengo":             "/crests/flamengo.svg",
	"clube do remo":           "/crests/remo.svg",
	"cruzeiro ec":             "/crests/cruzeiro.svg",
	"ec bahia":                "/crests/bahia.svg",
	"gremio fbpa":             "/crests/gremio.svg",
	"sc corinthians paulista": "/crests/corinthians.png",
	"se palmeiras":            "/crests/palmeiras.svg",
	"santos fc":               "/crests/santos.svg",
	"ca paranaense":           "/crests/athletico-paranaense.svg",
	"coritiba fbc":            "/crests/coritiba.svg",
	"sc internacional":        "/crests/internacional.svg",
	"ec vitoria":              "/crests/vitoria.svg",
	"fluminense fc":           "/crests/fluminense.svg",
	"chapecoense af":          "/crests/chapecoense.svg",
	"mirassol fc":             "/crests/mirassol.svg",
	"rb bragantino":           "/crests/bragantino.png",
	"cr vasco da gama":        "/crests/vasco.svg",
}

func clubCrest(league, name string) string {
	if league != "br.1" {
		return ""
	}
	return brazilCrests[normalizeTeam(name)]
}
