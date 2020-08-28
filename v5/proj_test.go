package proj_test

import (
	"github.com/jackielii/proj/v5"

	"fmt"
	"testing"
)

func TestLatlongToMerc(t *testing.T) {
	ctx := proj.NewContext()
	defer ctx.Close()

	ll, err := ctx.Create("+proj=latlong +datum=WGS84")
	if err != nil {
		t.Fatal(err)
	}
	defer ll.Close()

	merc, err := ctx.Create("+proj=merc +ellps=clrk66 +lat_ts=33")
	defer merc.Close()
	if err != nil {
		t.Fatal(err)
	}

	u, v, _, _, err := ll.Trans(proj.Inv, proj.DegToRad(-16), proj.DegToRad(20.25), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	u, v, _, _, err = merc.Trans(proj.Fwd, u, v, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	s := fmt.Sprintf("%.2f %.2f", u, v)
	s1 := "-1495284.21 1920596.79"
	if s != s1 {
		t.Fatalf("LatlongToMerc = %v, want %v", s, s1)
	}

	pj, err := ctx.Create(`
+proj=pipeline
+step +proj=unitconvert +xy_in=deg +xy_out=rad
+step +inv +proj=latlong +datum=WGS84
+step +proj=merc +ellps=clrk66 +lat_ts=33
`)
	defer pj.Close()
	if err != nil {
		t.Fatal(err)
	}
	x1 := []float64{-16, -10, 0, 30.4}
	y1 := []float64{20.25, 25, 0, 40.8}
	x2, y2, _, _, err := pj.TransSlice(proj.Fwd, x1, y1, nil, nil)
	if err != nil {
		t.Error(err)
	} else {
		s := fmt.Sprintf("[%.2f %.2f] [%.2f %.2f] [%.2f %.2f] [%.2f %.2f]", x2[0], y2[0], x2[1], y2[1], x2[2], y2[2], x2[3], y2[3])
		s1 := "[-1495284.21 1920596.79] [-934552.63 2398930.20] [0.00 0.00] [2841040.00 4159542.20]"
		if s != s1 {
			t.Errorf("LatlongToMerc = %v, want %v", s, s1)
		}
	}
}

func TestInvalidErrorProblem(t *testing.T) {
	ctx := proj.NewContext()
	defer ctx.Close()

	ll, err := ctx.Create("+proj=latlong +datum=WGS84")
	if err != nil {
		t.Fatal(err)
	}
	defer ll.Close()

	merc, err := ctx.Create("+proj=merc +ellps=clrk66 +lat_ts=33")
	defer merc.Close()
	if err != nil {
		t.Fatal(err)
	}

	u, v, _, _, err := ll.Trans(proj.Inv, proj.DegToRad(3000), proj.DegToRad(500), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err = merc.Trans(proj.Fwd, u, v, 0, 0)
	if err == nil {
		t.Error("err should not be nil")
	}

	// Try create a new projection after an error
	merc2, err := ctx.Create("+proj=merc +ellps=clrk66 +lat_ts=33")
	defer merc2.Close()
	if err != nil {
		t.Error(err)
	}
}
