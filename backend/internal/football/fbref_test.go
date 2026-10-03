package football

import (
	"os"
	"path/filepath"
	"testing"
)

// Invented two-club export with FBref's column names; never real data.
var fbrefFixture = map[string]string{
	"times_padrao":                  "Squad,Poss,Playing Time_MP,Performance_Gls,Performance_CrdY,Performance_CrdR\nAlpha FC,55.0,10,20,15,1\nAtlético Mineiro,45.0,10,10,25,3\n",
	"times_finalizacao":             "Squad,Standard_Sh,Standard_SoT,Standard_SoT%,Standard_G/Sh\nAlpha FC,150,50,33.3,0.13\nAtlético Mineiro,100,40,40.0,0.10\n",
	"times_finalizacao_adversarios": "Squad,Standard_Sh,Standard_SoT\nAlpha FC,90,30\nAtlético Mineiro,120,45\n",
	"times_goleiros":                "Squad,Performance_GA,Performance_Save%,Performance_CS\nAlpha FC,8,75.0,5\nAtlético Mineiro,8,70.0,3\n",
	"times_diversos":                "Squad,Performance_Fls\nAlpha FC,120\nAtlético Mineiro,150\n",
	"classificacao_casa_fora":       "Rk,Squad,Home_MP,Home_W,Home_D,Home_L,Home_GF,Home_GA,Home_GD,Home_Pts,Away_MP,Away_W,Away_D,Away_L,Away_GF,Away_GA,Away_GD,Away_Pts\n1,Alpha FC,5,4,1,0,12,3,9,13,5,2,1,2,8,5,3,7\n2,Atlético Mineiro,5,2,2,1,6,4,2,8,5,1,1,3,4,4,0,4\n",
	"classificacao":                 "Rk,Squad,Attendance\n1,Alpha FC,30000\n2,Atlético Mineiro,\n",
	"jogos":                         "Wk,Date,Home,Score,Away,Venue\n1,2026-05-01,Alpha FC,2–0,Atlético Mineiro,Alpha Arena\n2,2026-05-08,Alpha FC,1–1,Atlético Mineiro,Other Ground\n3,2026-05-15,Alpha FC,3–1,Atlético Mineiro,Alpha Arena\n4,2026-06-01,Atlético Mineiro,,Alpha FC,Arena MRV\n",
	"jogadores_padrao":              "Player,Squad,Performance_Gls,Performance_Ast\nAna,Alpha FC,9,2\nBia,Alpha FC,4,6\nCaio,Atlético Mineiro,0,0\n",
	"goleiros":                      "Player,Squad,Playing Time_MP,Playing Time_Min,Performance_Save%,Performance_CS\nGil,Alpha FC,2,180,50.0,0\nHugo,Alpha FC,8,720,80.0,5\n",
}

func writeFBref(t *testing.T, skip string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range fbrefFixture {
		if name == skip {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name+".csv"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestFBrefSeasonMetricsRanksAndAliases(t *testing.T) {
	s, err := LoadFBref(writeFBref(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	// The latest played match sets as-of; an unscored fixture does not.
	if s.AsOf() != "2026-05-15" {
		t.Fatalf("as of %q", s.AsOf())
	}
	alpha, ok := s.Season(Team{Name: "Alpha FC"})
	if !ok || alpha.Matches != 10 || alpha.Stadium != "Alpha Arena" || alpha.AverageAttendance != 30000 {
		t.Fatalf("alpha %+v", alpha)
	}
	metric := func(st *SeasonStats, key string) SeasonMetric {
		for _, m := range st.Metrics {
			if m.Key == key {
				return m
			}
		}
		t.Fatalf("missing %s", key)
		return SeasonMetric{}
	}
	if g := metric(alpha, "goals"); g.Value != 2 || g.League != 1.5 || g.Rank != 1 || g.Clubs != 2 || g.Group != "attack" {
		t.Fatalf("goals %+v", g)
	}
	// Lower is better: fewer fouls ranks first; equal goals against share first place.
	if f := metric(alpha, "fouls"); f.Value != 12 || f.Rank != 1 || !f.LowerIsBetter {
		t.Fatalf("fouls %+v", f)
	}
	if ga := metric(alpha, "goals_against"); ga.Value != 0.8 || ga.Rank != 1 {
		t.Fatalf("goals against %+v", ga)
	}
	if alpha.Home == nil || alpha.Home.Points != 13 || alpha.Home.PointsPerMatch != 2.6 || alpha.Away.Losses != 2 {
		t.Fatalf("split %+v %+v", alpha.Home, alpha.Away)
	}
	if alpha.TopScorer.Player != "Ana" || alpha.TopAssists.Player != "Bia" || alpha.Goalkeeper.Player != "Hugo" || alpha.Goalkeeper.CleanSheets != 5 {
		t.Fatalf("leaders %+v %+v %+v", alpha.TopScorer, alpha.TopAssists, alpha.Goalkeeper)
	}
	// OpenFootball spelling reaches FBref's through the shared aliases.
	galo, ok := s.Season(Team{Name: "CA Mineiro"})
	if !ok || galo.Team != "Atlético Mineiro" || galo.TopScorer != nil || galo.AverageAttendance != 0 {
		t.Fatalf("alias %+v", galo)
	}
	if _, ok := s.Season(Team{Name: "Unknown United"}); ok {
		t.Fatal("unknown club matched")
	}
	// Returned copies must not change the loaded data.
	alpha.Metrics[0].Value, alpha.Home.Points = 99, 99
	again, _ := s.Season(Team{Name: "Alpha FC"})
	if again.Metrics[0].Value == 99 || again.Home.Points == 99 {
		t.Fatal("season data is externally mutable")
	}
}

func TestFBrefMissingFileOrColumnFails(t *testing.T) {
	if _, err := LoadFBref(writeFBref(t, "times_goleiros")); err == nil {
		t.Fatal("missing file accepted")
	}
	dir := writeFBref(t, "")
	if err := os.WriteFile(filepath.Join(dir, "times_padrao.csv"), []byte("Squad,Poss\nAlpha FC,55\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFBref(dir); err == nil {
		t.Fatal("missing column accepted")
	}
}
