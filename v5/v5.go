package proj

/*
#cgo darwin pkg-config: proj
#cgo !darwin LDFLAGS: -lproj
#include "proj_go.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

type Context struct {
	pj_context  *C.PJ_CONTEXT
	opened      bool
	counter     uint64
	projections map[uint64]*Proj
}

type Proj struct {
	pj      *C.PJ
	context *Context
	index   uint64
	opened  bool
}

type Coord struct {
	U, V, W, T float64
}

var (
	errContextClosed    = errors.New("Context is closed")
	errProjectionClosed = errors.New("Projection is closed")
)

func NewContext() *Context {
	ctx := Context{
		pj_context:  C.proj_context_create(),
		counter:     0,
		projections: make(map[uint64]*Proj),
		opened:      true,
	}
	runtime.SetFinalizer(&ctx, (*Context).Close)
	return &ctx
}

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

func (ctx *Context) Create(definition string) (*Proj, error) {
	if !ctx.opened {
		return nil, errContextClosed
	}

	cs := C.CString(definition)
	defer C.free(unsafe.Pointer(cs))
	pj := C.proj_create(ctx.pj_context, cs)
	if C.pjnull(pj) != 0 {
		errno := C.proj_context_errno(ctx.pj_context)
		err := C.GoString(C.proj_errno_string(errno))
		return nil, errors.New(err)
	}

	p := Proj{
		opened:  true,
		context: ctx,
		index:   ctx.counter,
		pj:      pj,
	}
	ctx.projections[ctx.counter] = &p
	ctx.counter++

	runtime.SetFinalizer(&p, (*Proj).Close)
	return &p, nil
}

func (p *Proj) Close() {
	if p.opened {
		C.proj_destroy(p.pj)
		if p.context.opened {
			delete(p.context.projections, p.index)
		}
		p.context = nil
		p.opened = false
	}
}

func (p *Proj) Fwd(coord Coord) (Coord, error) {
	return p.trans(coord, false)
}

func (p *Proj) Inv(coord Coord) (Coord, error) {
	return p.trans(coord, true)
}

func (p *Proj) trans(coord Coord, inverse bool) (Coord, error) {
	if !p.opened {
		return Coord{}, errProjectionClosed
	}

	var direction C.PJ_DIRECTION
	if inverse {
		direction = C.PJ_INV
	} else {
		direction = C.PJ_FWD
	}

	var u, v, w, t C.double
	C.trans(p.pj, direction, C.double(coord.U), C.double(coord.V), C.double(coord.W), C.double(coord.T), &u, &v, &w, &t)

	coord2 := Coord{
		U: float64(u),
		V: float64(v),
		W: float64(w),
		T: float64(t),
	}
	return coord2, nil
}
