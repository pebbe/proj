package proj

/*
#cgo darwin pkg-config: proj
#cgo !darwin LDFLAGS: -lproj
#include "proj_go.h"
*/
import "C"

import (
	"errors"
	"math"
	"runtime"
	"unsafe"
)

type Context struct {
	pj_context  *C.PJ_CONTEXT
	opened      bool
	counter     uint64
	projections map[uint64]*PJ
}

// A projection object
type PJ struct {
	pj      *C.PJ
	context *Context
	index   uint64
	opened  bool
}

type LibInfo struct {
	Major      int    // Major version number.
	Minor      int    // Minor version number.
	Patch      int    // Patch level of release.
	Release    string // Release info. Version number and release date, e.g. “Rel. 4.9.3, 15 August 2016”.
	Version    string // Text representation of the full version number, e.g. “4.9.3”.
	Searchpath string // Search path for PROJ. List of directories separated by semicolons (Windows) or colons (non-Windows).
}

type ProjInfo struct {
	ID          string  // Short ID of the operation the PJ object is based on, that is, what comes afther the +proj= in a proj-string, e.g. “merc”.
	Description string  // Long describes of the operation the PJ object is based on, e.g. “Mercator Cyl, Sph&Ell lat_ts=”.
	Definition  string  // The proj-string that was used to create the PJ object with, e.g. “+proj=merc +lat_0=24 +lon_0=53 +ellps=WGS84”.
	HasInverse  bool    // True if an inverse mapping of the defined operation exists,
	Accuracy    float64 // Expected accuracy of the transformation. -1 if unknown.
}

// The direction of a transformation
type Direction C.PJ_DIRECTION

const (
	Fwd   = Direction(C.PJ_FWD)   // Forward transformation
	Ident = Direction(C.PJ_IDENT) // Do nothing
	Inv   = Direction(C.PJ_INV)   // Inverse transformation
)

var (
	errContextClosed    = errors.New("Context is closed")
	errDataSizeMismatch = errors.New("Data size mismatch")
	errMissingData      = errors.New("Missing data")
	errProjectionClosed = errors.New("Projection is closed")
)

// Create a context
func NewContext() *Context {
	ctx := Context{
		pj_context:  C.proj_context_create(),
		counter:     0,
		projections: make(map[uint64]*PJ),
		opened:      true,
	}
	runtime.SetFinalizer(&ctx, (*Context).Close)
	return &ctx
}

// Close a context
func (ctx *Context) Close() {
	if ctx.opened {
		indexen := make([]uint64, 0, len(ctx.projections))
		for i := range ctx.projections {
			indexen = append(indexen, i)
		}
		for _, i := range indexen {
			p := ctx.projections[i]
			if p.opened {
				C.proj_destroy(p.pj)
				p.context = nil
				p.opened = false
			}
			delete(ctx.projections, i)
		}

		C.proj_context_destroy(ctx.pj_context)
		ctx.pj_context = nil
		ctx.opened = false
	}
}

// Create a transformation object
func (ctx *Context) Create(definition string) (*PJ, error) {
	if !ctx.opened {
		return &PJ{}, errContextClosed
	}

	cs := C.CString(definition)
	defer C.free(unsafe.Pointer(cs))
	pj := C.proj_create(ctx.pj_context, cs)
	if C.pjnull(pj) == 0 {
		errno := C.proj_context_errno(ctx.pj_context)
		err := C.GoString(C.proj_errno_string(errno))
		return &PJ{}, errors.New(err)
	}

	p := PJ{
		opened:  true,
		context: ctx,
		index:   ctx.counter,
		pj:      pj,
	}
	ctx.projections[ctx.counter] = &p
	ctx.counter++

	runtime.SetFinalizer(&p, (*PJ).Close)
	return &p, nil
}

// Close a transformation object
func (p *PJ) Close() {
	if p.opened {
		C.proj_destroy(p.pj)
		if p.context.opened {
			delete(p.context.projections, p.index)
		}
		p.context = nil
		p.opened = false
	}
}

// Get information about the transformation object
func (p *PJ) Info() (ProjInfo, error) {
	if !p.opened {
		return ProjInfo{}, errProjectionClosed
	}
	info := C.proj_pj_info(p.pj)
	hasInv := false
	if info.has_inverse != 0 {
		hasInv = true
	}
	return ProjInfo{
		ID:          C.GoString(info.id),
		Description: C.GoString(info.description),
		Definition:  C.GoString(info.definition),
		HasInverse:  hasInv,
		Accuracy:    float64(info.accuracy),
	}, nil
}

// Transform a single transformation
func (p *PJ) Trans(direction Direction, u1, v1, w1, t1 float64) (u2, v2, w2, t2 float64, err error) {
	if !p.opened {
		return 0, 0, 0, 0, errProjectionClosed
	}

	var u, v, w, t C.double
	C.trans(p.pj, C.PJ_DIRECTION(direction), C.double(u1), C.double(v1), C.double(w1), C.double(t1), &u, &v, &w, &t)

	e := C.proj_errno(p.pj)
	if e != 0 {
		return 0, 0, 0, 0, errors.New(C.GoString(C.proj_errno_string(e)))
	}

	return float64(u), float64(v), float64(w), float64(t), nil
}

/*
Transform a series of coordinates, where the individual coordinate dimension may be represented by a slice that is either

1. fully populated

2. nil and/or a length of zero, which will be treated as a fully populated slice of zeroes

3. of length one, i.e. a constant, which will be treated as a fully populated slice of that constant value

TODO: what if input is constant, but output is not?
*/
func (p *PJ) TransSlice(direction Direction, u1, v1, w1, t1 []float64) (u2, v2, w2, t2 []float64, err error) {
	if !p.opened {
		return nil, nil, nil, nil, errProjectionClosed
	}
	if u1 == nil || v1 == nil {
		return nil, nil, nil, nil, errMissingData
	}

	var un, vn, wn, tn int
	var u, v, w, t []C.double
	var up, vp, wp, tp *C.double
	var unc, vnc, wnc, tnc C.size_t

	un = len(u1)
	unc = C.size_t(un)
	vn = len(v1)
	vnc = C.size_t(vn)
	if w1 != nil {
		wn = len(w1)
		wnc = C.size_t(wn)
		if t1 != nil {
			tn = len(t1)
			tnc = C.size_t(tn)
		}
	}
	r := []int{un, vn, tn, wn}
	var n int
	for _, i := range r {
		if i > n {
			n = i
		}
	}
	for _, i := range r {
		if i > 1 && i < n {
			return nil, nil, nil, nil, errDataSizeMismatch
		}
	}

	u = make([]C.double, un)
	up = &u[0]
	for i := 0; i < un; i++ {
		u[i] = C.double(u1[i])
	}
	v = make([]C.double, vn)
	vp = &v[0]
	for i := 0; i < vn; i++ {
		v[i] = C.double(v1[i])
	}
	if w1 != nil {
		w = make([]C.double, wn)
		wp = &w[0]
		for i := 0; i < wn; i++ {
			w[i] = C.double(w1[i])
		}
		if t1 != nil {
			t = make([]C.double, tn)
			tp = &t[0]
			for i := 0; i < tn; i++ {
				t[i] = C.double(t1[i])
			}
		}
	}

	st := C.size_t(C.sizeof_double)

	C.proj_trans_generic(
		p.pj,
		C.PJ_DIRECTION(direction),
		up, st, unc,
		vp, st, vnc,
		wp, st, wnc,
		tp, st, tnc)

	e := C.proj_errno(p.pj)
	if e != 0 {
		return nil, nil, nil, nil, errors.New(C.GoString(C.proj_errno_string(e)))
	}

	u2 = make([]float64, un)
	for i := 0; i < un; i++ {
		u2[i] = float64(u[i])
	}
	v2 = make([]float64, vn)
	for i := 0; i < vn; i++ {
		v2[i] = float64(v[i])
	}
	if w1 != nil {
		w2 = make([]float64, wn)
		for i := 0; i < wn; i++ {
			w2[i] = float64(w[i])
		}
		if t1 != nil {
			t2 = make([]float64, tn)
			for i := 0; i < tn; i++ {
				t2[i] = float64(t[i])
			}
		}
	}

	return
}

// Calculate geodesic distance between two points in geodetic coordinates
//
// The calculated distance is between the two points located on the ellipsoid
func (p *PJ) Dist(u1, v1, u2, v2 float64) (float64, error) {
	if !p.opened {
		return 0, errProjectionClosed
	}
	a := C.uvwt(C.double(u1), C.double(v1), 0, 0)
	b := C.uvwt(C.double(u2), C.double(v2), 0, 0)
	d := C.proj_lp_dist(p.pj, a, b)
	e := C.proj_errno(p.pj)
	if e != 0 {
		return 0, errors.New(C.GoString(C.proj_errno_string(e)))
	}
	return float64(d), nil
}

// Calculate geodesic distance between two points in geodetic coordinates
//
// Similar to Dist() but also takes the height above the ellipsoid into account
func (p *PJ) Dist3(u1, v1, w1, u2, v2, w2 float64) (float64, error) {
	if !p.opened {
		return 0, errProjectionClosed
	}
	a := C.uvwt(C.double(u1), C.double(v1), C.double(w1), 0)
	b := C.uvwt(C.double(u2), C.double(v2), C.double(w2), 0)
	d := C.proj_lpz_dist(p.pj, a, b)
	e := C.proj_errno(p.pj)
	if e != 0 {
		return 0, errors.New(C.GoString(C.proj_errno_string(e)))
	}
	return float64(d), nil
}

// Get information about the current instance of the PROJ library
func Info() LibInfo {
	info := C.proj_info()
	return LibInfo{
		Major:      int(info.major),
		Minor:      int(info.minor),
		Patch:      int(info.patch),
		Release:    C.GoString(info.release),
		Version:    C.GoString(info.version),
		Searchpath: C.GoString(info.searchpath),
	}
}

// Convert degrees to radians
func DegToRad(deg float64) float64 {
	return deg / 180 * math.Pi
}

// Convert radians to degrees
func RadToDeg(rad float64) float64 {
	return rad / math.Pi * 180
}
