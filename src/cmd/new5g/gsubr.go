// Derived from Inferno utils/5c/txt.c
// http://code.google.com/p/inferno-os/source/browse/utils/5c/txt.c
//
//	Copyright © 1994-1999 Lucent Technologies Inc.  All rights reserved.
//	Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
//	Portions Copyright © 1997-1999 Vita Nuova Limited
//	Portions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com)
//	Portions Copyright © 2004,2006 Bruce Ellis
//	Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
//	Revisions Copyright © 2000-2007 Lucent Technologies Inc. and others
//	Portions Copyright © 2009 The Go Authors.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"cmd/internal/obj"
	"cmd/internal/obj/arm"
	"fmt"
)
import "cmd/internal/gc"

// TODO(rsc): Can make this bigger if we move
// the text segment up higher in 5l for all GOOS.
// At the same time, can raise StackBig in ../../runtime/stack.h.
var unmappedzero int = 4096

var resvd = []int{
	9,         // reserved for m
	10,        // reserved for g
	arm.REGSP, // reserved for SP
}

func ginit() {
	var i int

	for i = 0; i < len(reg); i++ {
		reg[i] = 0
	}
	for i = 0; i < len(resvd); i++ {
		reg[resvd[i]]++
	}
}

func gclean() {
	var i int

	for i = 0; i < len(resvd); i++ {
		reg[resvd[i]]--
	}

	for i = 0; i < len(reg); i++ {
		if reg[i] != 0 {
			gc.Yyerror("reg %v left allocated\n", gc.Ctxt.Rconv(i))
		}
	}
}

func anyregalloc() bool {
	var i int
	var j int

	for i = 0; i < len(reg); i++ {
		if reg[i] == 0 {
			goto ok
		}
		for j = 0; j < len(resvd); j++ {
			if resvd[j] == i {
				goto ok
			}
		}
		return true
	ok:
	}

	return false
}

var regpc [REGALLOC_FMAX + 1]uint32

/*
 * allocate register of type t, leave in n.
 * if o != N, o is desired fixed register.
 * caller must regfree(n).
 */
func regalloc(n *gc.Node, t *gc.Type, o *gc.Node) {
	var i int
	var et int
	var fixfree int
	var floatfree int

	if false && gc.Debug['r'] != 0 {
		fixfree = 0
		for i = REGALLOC_R0; i <= REGALLOC_RMAX; i++ {
			if reg[i] == 0 {
				fixfree++
			}
		}
		floatfree = 0
		for i = REGALLOC_F0; i <= REGALLOC_FMAX; i++ {
			if reg[i] == 0 {
				floatfree++
			}
		}
		fmt.Printf("regalloc fix %d float %d\n", fixfree, floatfree)
	}

	if t == nil {
		gc.Fatal("regalloc: t nil")
	}
	et = int(gc.Simtype[t.Etype])
	if gc.Is64(t) {
		gc.Fatal("regalloc: 64 bit type %v")
	}

	switch et {
	case gc.TINT8,
		gc.TUINT8,
		gc.TINT16,
		gc.TUINT16,
		gc.TINT32,
		gc.TUINT32,
		gc.TPTR32,
		gc.TBOOL:
		if o != nil && o.Op == gc.OREGISTER {
			i = int(o.Val.U.Reg)
			if i >= REGALLOC_R0 && i <= REGALLOC_RMAX {
				goto out
			}
		}

		for i = REGALLOC_R0; i <= REGALLOC_RMAX; i++ {
			if reg[i] == 0 {
				regpc[i] = uint32(obj.Getcallerpc(&n))
				goto out
			}
		}

		fmt.Printf("registers allocated at\n")
		for i = REGALLOC_R0; i <= REGALLOC_RMAX; i++ {
			fmt.Printf("%d %p\n", i, regpc[i])
		}
		gc.Fatal("out of fixed registers")
		goto err

	case gc.TFLOAT32,
		gc.TFLOAT64:
		if o != nil && o.Op == gc.OREGISTER {
			i = int(o.Val.U.Reg)
			if i >= REGALLOC_F0 && i <= REGALLOC_FMAX {
				goto out
			}
		}

		for i = REGALLOC_F0; i <= REGALLOC_FMAX; i++ {
			if reg[i] == 0 {
				goto out
			}
		}
		gc.Fatal("out of floating point registers")
		goto err

	case gc.TCOMPLEX64,
		gc.TCOMPLEX128:
		gc.Tempname(n, t)
		return
	}

	gc.Yyerror("regalloc: unknown type %v", gc.Tconv(t, 0))

err:
	gc.Nodreg(n, t, arm.REG_R0)
	return

out:
	reg[i]++
	gc.Nodreg(n, t, i)
}

func regfree(n *gc.Node) {
	var i int
	var fixfree int
	var floatfree int

	if false && gc.Debug['r'] != 0 {
		fixfree = 0
		for i = REGALLOC_R0; i <= REGALLOC_RMAX; i++ {
			if reg[i] == 0 {
				fixfree++
			}
		}
		floatfree = 0
		for i = REGALLOC_F0; i <= REGALLOC_FMAX; i++ {
			if reg[i] == 0 {
				floatfree++
			}
		}
		fmt.Printf("regalloc fix %d float %d\n", fixfree, floatfree)
	}

	if n.Op == gc.ONAME {
		return
	}
	if n.Op != gc.OREGISTER && n.Op != gc.OINDREG {
		gc.Fatal("regfree: not a register")
	}
	i = int(n.Val.U.Reg)
	if i == arm.REGSP {
		return
	}
	if i < 0 || i >= len(reg) || i >= len(regpc) {
		gc.Fatal("regfree: reg out of range")
	}
	if reg[i] <= 0 {
		gc.Fatal("regfree: reg %v not allocated", gc.Ctxt.Rconv(i))
	}
	reg[i]--
	if reg[i] == 0 {
		regpc[i] = 0
	}
}

/*
 * return constant i node.
 * overwritten by next call, but useful in calls to gins.
 */

var ncon_n gc.Node

func ncon(i uint32) *gc.Node {
	if ncon_n.Type == nil {
		gc.Nodconst(&ncon_n, gc.Types[gc.TUINT32], 0)
	}
	gc.Mpmovecfix(ncon_n.Val.U.Xval, int64(i))
	return &ncon_n
}

var sclean [10]gc.Node

var nsclean int

/*
 * n is a 64-bit value.  fill in lo and hi to refer to its 32-bit halves.
 */
func split64(n *gc.Node, lo *gc.Node, hi *gc.Node) {
	var n1 gc.Node
	var i int64

	if !gc.Is64(n.Type) {
		gc.Fatal("split64 %v", gc.Tconv(n.Type, 0))
	}

	if nsclean >= len(sclean) {
		gc.Fatal("split64 clean")
	}
	sclean[nsclean].Op = gc.OEMPTY
	nsclean++
	switch n.Op {
	default:
		switch n.Op {
		default:
			if !dotaddable(n, &n1) {
				igen(n, &n1, nil)
				sclean[nsclean-1] = n1
			}

			n = &n1

		case gc.ONAME:
			if n.Class == gc.PPARAMREF {
				cgen(n.Heapaddr, &n1)
				sclean[nsclean-1] = n1
				n = &n1
			}

			// nothing
		case gc.OINDREG:
			break
		}

		*lo = *n
		*hi = *n
		lo.Type = gc.Types[gc.TUINT32]
		if n.Type.Etype == gc.TINT64 {
			hi.Type = gc.Types[gc.TINT32]
		} else {
			hi.Type = gc.Types[gc.TUINT32]
		}
		hi.Xoffset += 4

	case gc.OLITERAL:
		gc.Convconst(&n1, n.Type, &n.Val)
		i = gc.Mpgetfix(n1.Val.U.Xval)
		gc.Nodconst(lo, gc.Types[gc.TUINT32], int64(uint32(i)))
		i >>= 32
		if n.Type.Etype == gc.TINT64 {
			gc.Nodconst(hi, gc.Types[gc.TINT32], int64(int32(i)))
		} else {
			gc.Nodconst(hi, gc.Types[gc.TUINT32], int64(uint32(i)))
		}
	}
}

func splitclean() {
	if nsclean <= 0 {
		gc.Fatal("splitclean")
	}
	nsclean--
	if sclean[nsclean].Op != gc.OEMPTY {
		regfree(&sclean[nsclean])
	}
}

func gmove(f *gc.Node, t *gc.Node) {
	var a int
	var ft int
	var tt int
	var fa int
	var ta int
	var cvt *gc.Type
	var r1 gc.Node
	var r2 gc.Node
	var flo gc.Node
	var fhi gc.Node
	var tlo gc.Node
	var thi gc.Node
	var con gc.Node
	var p1 *obj.Prog

	if gc.Debug['M'] != 0 {
		fmt.Printf("gmove %v -> %v\n", gc.Nconv(f, 0), gc.Nconv(t, 0))
	}

	ft = gc.Simsimtype(f.Type)
	tt = gc.Simsimtype(t.Type)
	cvt = t.Type

	if gc.Iscomplex[ft] != 0 || gc.Iscomplex[tt] != 0 {
		gc.Complexmove(f, t)
		return
	}

	// cannot have two memory operands;
	// except 64-bit, which always copies via registers anyway.
	if !gc.Is64(f.Type) && !gc.Is64(t.Type) && gc.Ismem(f) && gc.Ismem(t) {
		goto hard
	}

	// convert constant to desired type
	if f.Op == gc.OLITERAL {
		switch tt {
		default:
			gc.Convconst(&con, t.Type, &f.Val)

		case gc.TINT16,
			gc.TINT8:
			gc.Convconst(&con, gc.Types[gc.TINT32], &f.Val)
			regalloc(&r1, con.Type, t)
			gins(arm.AMOVW, &con, &r1)
			gmove(&r1, t)
			regfree(&r1)
			return

		case gc.TUINT16,
			gc.TUINT8:
			gc.Convconst(&con, gc.Types[gc.TUINT32], &f.Val)
			regalloc(&r1, con.Type, t)
			gins(arm.AMOVW, &con, &r1)
			gmove(&r1, t)
			regfree(&r1)
			return
		}

		f = &con
		ft = gc.Simsimtype(con.Type)

		// constants can't move directly to memory
		if gc.Ismem(t) && !gc.Is64(t.Type) {
			goto hard
		}
	}

	// value -> value copy, only one memory operand.
	// figure out the instruction to use.
	// break out of switch for one-instruction gins.
	// goto rdst for "destination must be register".
	// goto hard for "convert to cvt type first".
	// otherwise handle and return.

	switch uint32(ft)<<16 | uint32(tt) {
	default:
		goto fatal

		/*
		 * integer copy and truncate
		 */
	case gc.TINT8<<16 | gc.TINT8: // same size
		if !gc.Ismem(f) {
			a = arm.AMOVB
			break
		}
		fallthrough

	case gc.TUINT8<<16 | gc.TINT8,
		gc.TINT16<<16 | gc.TINT8, // truncate
		gc.TUINT16<<16 | gc.TINT8,
		gc.TINT32<<16 | gc.TINT8,
		gc.TUINT32<<16 | gc.TINT8:
		a = arm.AMOVBS

	case gc.TUINT8<<16 | gc.TUINT8:
		if !gc.Ismem(f) {
			a = arm.AMOVB
			break
		}
		fallthrough

	case gc.TINT8<<16 | gc.TUINT8,
		gc.TINT16<<16 | gc.TUINT8,
		gc.TUINT16<<16 | gc.TUINT8,
		gc.TINT32<<16 | gc.TUINT8,
		gc.TUINT32<<16 | gc.TUINT8:
		a = arm.AMOVBU

	case gc.TINT64<<16 | gc.TINT8, // truncate low word
		gc.TUINT64<<16 | gc.TINT8:
		a = arm.AMOVBS

		goto trunc64

	case gc.TINT64<<16 | gc.TUINT8,
		gc.TUINT64<<16 | gc.TUINT8:
		a = arm.AMOVBU
		goto trunc64

	case gc.TINT16<<16 | gc.TINT16: // same size
		if !gc.Ismem(f) {
			a = arm.AMOVH
			break
		}
		fallthrough

	case gc.TUINT16<<16 | gc.TINT16,
		gc.TINT32<<16 | gc.TINT16, // truncate
		gc.TUINT32<<16 | gc.TINT16:
		a = arm.AMOVHS

	case gc.TUINT16<<16 | gc.TUINT16:
		if !gc.Ismem(f) {
			a = arm.AMOVH
			break
		}
		fallthrough

	case gc.TINT16<<16 | gc.TUINT16,
		gc.TINT32<<16 | gc.TUINT16,
		gc.TUINT32<<16 | gc.TUINT16:
		a = arm.AMOVHU

	case gc.TINT64<<16 | gc.TINT16, // truncate low word
		gc.TUINT64<<16 | gc.TINT16:
		a = arm.AMOVHS

		goto trunc64

	case gc.TINT64<<16 | gc.TUINT16,
		gc.TUINT64<<16 | gc.TUINT16:
		a = arm.AMOVHU
		goto trunc64

	case gc.TINT32<<16 | gc.TINT32, // same size
		gc.TINT32<<16 | gc.TUINT32,
		gc.TUINT32<<16 | gc.TINT32,
		gc.TUINT32<<16 | gc.TUINT32:
		a = arm.AMOVW

	case gc.TINT64<<16 | gc.TINT32, // truncate
		gc.TUINT64<<16 | gc.TINT32,
		gc.TINT64<<16 | gc.TUINT32,
		gc.TUINT64<<16 | gc.TUINT32:
		split64(f, &flo, &fhi)

		regalloc(&r1, t.Type, nil)
		gins(arm.AMOVW, &flo, &r1)
		gins(arm.AMOVW, &r1, t)
		regfree(&r1)
		splitclean()
		return

	case gc.TINT64<<16 | gc.TINT64, // same size
		gc.TINT64<<16 | gc.TUINT64,
		gc.TUINT64<<16 | gc.TINT64,
		gc.TUINT64<<16 | gc.TUINT64:
		split64(f, &flo, &fhi)

		split64(t, &tlo, &thi)
		regalloc(&r1, flo.Type, nil)
		regalloc(&r2, fhi.Type, nil)
		gins(arm.AMOVW, &flo, &r1)
		gins(arm.AMOVW, &fhi, &r2)
		gins(arm.AMOVW, &r1, &tlo)
		gins(arm.AMOVW, &r2, &thi)
		regfree(&r1)
		regfree(&r2)
		splitclean()
		splitclean()
		return

		/*
		 * integer up-conversions
		 */
	case gc.TINT8<<16 | gc.TINT16, // sign extend int8
		gc.TINT8<<16 | gc.TUINT16,
		gc.TINT8<<16 | gc.TINT32,
		gc.TINT8<<16 | gc.TUINT32:
		a = arm.AMOVBS

		goto rdst

	case gc.TINT8<<16 | gc.TINT64, // convert via int32
		gc.TINT8<<16 | gc.TUINT64:
		cvt = gc.Types[gc.TINT32]

		goto hard

	case gc.TUINT8<<16 | gc.TINT16, // zero extend uint8
		gc.TUINT8<<16 | gc.TUINT16,
		gc.TUINT8<<16 | gc.TINT32,
		gc.TUINT8<<16 | gc.TUINT32:
		a = arm.AMOVBU

		goto rdst

	case gc.TUINT8<<16 | gc.TINT64, // convert via uint32
		gc.TUINT8<<16 | gc.TUINT64:
		cvt = gc.Types[gc.TUINT32]

		goto hard

	case gc.TINT16<<16 | gc.TINT32, // sign extend int16
		gc.TINT16<<16 | gc.TUINT32:
		a = arm.AMOVHS

		goto rdst

	case gc.TINT16<<16 | gc.TINT64, // convert via int32
		gc.TINT16<<16 | gc.TUINT64:
		cvt = gc.Types[gc.TINT32]

		goto hard

	case gc.TUINT16<<16 | gc.TINT32, // zero extend uint16
		gc.TUINT16<<16 | gc.TUINT32:
		a = arm.AMOVHU

		goto rdst

	case gc.TUINT16<<16 | gc.TINT64, // convert via uint32
		gc.TUINT16<<16 | gc.TUINT64:
		cvt = gc.Types[gc.TUINT32]

		goto hard

	case gc.TINT32<<16 | gc.TINT64, // sign extend int32
		gc.TINT32<<16 | gc.TUINT64:
		split64(t, &tlo, &thi)

		regalloc(&r1, tlo.Type, nil)
		regalloc(&r2, thi.Type, nil)
		gmove(f, &r1)
		p1 = gins(arm.AMOVW, &r1, &r2)
		p1.From.Type = obj.TYPE_SHIFT
		p1.From.Offset = 2<<5 | 31<<7 | int64(r1.Val.U.Reg)&15 // r1->31
		p1.From.Reg = 0

		//print("gmove: %P\n", p1);
		gins(arm.AMOVW, &r1, &tlo)

		gins(arm.AMOVW, &r2, &thi)
		regfree(&r1)
		regfree(&r2)
		splitclean()
		return

	case gc.TUINT32<<16 | gc.TINT64, // zero extend uint32
		gc.TUINT32<<16 | gc.TUINT64:
		split64(t, &tlo, &thi)

		gmove(f, &tlo)
		regalloc(&r1, thi.Type, nil)
		gins(arm.AMOVW, ncon(0), &r1)
		gins(arm.AMOVW, &r1, &thi)
		regfree(&r1)
		splitclean()
		return

		//	case CASE(TFLOAT64, TUINT64):
	/*
	* float to integer
	 */
	case gc.TFLOAT32<<16 | gc.TINT8,
		gc.TFLOAT32<<16 | gc.TUINT8,
		gc.TFLOAT32<<16 | gc.TINT16,
		gc.TFLOAT32<<16 | gc.TUINT16,
		gc.TFLOAT32<<16 | gc.TINT32,
		gc.TFLOAT32<<16 | gc.TUINT32,

		//	case CASE(TFLOAT32, TUINT64):

		gc.TFLOAT64<<16 | gc.TINT8,
		gc.TFLOAT64<<16 | gc.TUINT8,
		gc.TFLOAT64<<16 | gc.TINT16,
		gc.TFLOAT64<<16 | gc.TUINT16,
		gc.TFLOAT64<<16 | gc.TINT32,
		gc.TFLOAT64<<16 | gc.TUINT32:
		fa = arm.AMOVF

		a = arm.AMOVFW
		if ft == gc.TFLOAT64 {
			fa = arm.AMOVD
			a = arm.AMOVDW
		}

		ta = arm.AMOVW
		switch tt {
		case gc.TINT8:
			ta = arm.AMOVBS

		case gc.TUINT8:
			ta = arm.AMOVBU

		case gc.TINT16:
			ta = arm.AMOVHS

		case gc.TUINT16:
			ta = arm.AMOVHU
		}

		regalloc(&r1, gc.Types[ft], f)
		regalloc(&r2, gc.Types[tt], t)
		gins(fa, f, &r1)       // load to fpu
		p1 = gins(a, &r1, &r1) // convert to w
		switch tt {
		case gc.TUINT8,
			gc.TUINT16,
			gc.TUINT32:
			p1.Scond |= arm.C_UBIT
		}

		gins(arm.AMOVW, &r1, &r2) // copy to cpu
		gins(ta, &r2, t)          // store
		regfree(&r1)
		regfree(&r2)
		return

		/*
		 * integer to float
		 */
	case gc.TINT8<<16 | gc.TFLOAT32,
		gc.TUINT8<<16 | gc.TFLOAT32,
		gc.TINT16<<16 | gc.TFLOAT32,
		gc.TUINT16<<16 | gc.TFLOAT32,
		gc.TINT32<<16 | gc.TFLOAT32,
		gc.TUINT32<<16 | gc.TFLOAT32,
		gc.TINT8<<16 | gc.TFLOAT64,
		gc.TUINT8<<16 | gc.TFLOAT64,
		gc.TINT16<<16 | gc.TFLOAT64,
		gc.TUINT16<<16 | gc.TFLOAT64,
		gc.TINT32<<16 | gc.TFLOAT64,
		gc.TUINT32<<16 | gc.TFLOAT64:
		fa = arm.AMOVW

		switch ft {
		case gc.TINT8:
			fa = arm.AMOVBS

		case gc.TUINT8:
			fa = arm.AMOVBU

		case gc.TINT16:
			fa = arm.AMOVHS

		case gc.TUINT16:
			fa = arm.AMOVHU
		}

		a = arm.AMOVWF
		ta = arm.AMOVF
		if tt == gc.TFLOAT64 {
			a = arm.AMOVWD
			ta = arm.AMOVD
		}

		regalloc(&r1, gc.Types[ft], f)
		regalloc(&r2, gc.Types[tt], t)
		gins(fa, f, &r1)          // load to cpu
		gins(arm.AMOVW, &r1, &r2) // copy to fpu
		p1 = gins(a, &r2, &r2)    // convert
		switch ft {
		case gc.TUINT8,
			gc.TUINT16,
			gc.TUINT32:
			p1.Scond |= arm.C_UBIT
		}

		gins(ta, &r2, t) // store
		regfree(&r1)
		regfree(&r2)
		return

	case gc.TUINT64<<16 | gc.TFLOAT32,
		gc.TUINT64<<16 | gc.TFLOAT64:
		gc.Fatal("gmove UINT64, TFLOAT not implemented")
		return

		/*
		 * float to float
		 */
	case gc.TFLOAT32<<16 | gc.TFLOAT32:
		a = arm.AMOVF

	case gc.TFLOAT64<<16 | gc.TFLOAT64:
		a = arm.AMOVD

	case gc.TFLOAT32<<16 | gc.TFLOAT64:
		regalloc(&r1, gc.Types[gc.TFLOAT64], t)
		gins(arm.AMOVF, f, &r1)
		gins(arm.AMOVFD, &r1, &r1)
		gins(arm.AMOVD, &r1, t)
		regfree(&r1)
		return

	case gc.TFLOAT64<<16 | gc.TFLOAT32:
		regalloc(&r1, gc.Types[gc.TFLOAT64], t)
		gins(arm.AMOVD, f, &r1)
		gins(arm.AMOVDF, &r1, &r1)
		gins(arm.AMOVF, &r1, t)
		regfree(&r1)
		return
	}

	gins(a, f, t)
	return

	// TODO(kaib): we almost always require a register dest anyway, this can probably be
	// removed.
	// requires register destination
rdst:
	regalloc(&r1, t.Type, t)

	gins(a, f, &r1)
	gmove(&r1, t)
	regfree(&r1)
	return

	// requires register intermediate
hard:
	regalloc(&r1, cvt, t)

	gmove(f, &r1)
	gmove(&r1, t)
	regfree(&r1)
	return

	// truncate 64 bit integer
trunc64:
	split64(f, &flo, &fhi)

	regalloc(&r1, t.Type, nil)
	gins(a, &flo, &r1)
	gins(a, &r1, t)
	regfree(&r1)
	splitclean()
	return

	// should not happen
fatal:
	gc.Fatal("gmove %v -> %v", gc.Nconv(f, 0), gc.Nconv(t, 0))
}

func samaddr(f *gc.Node, t *gc.Node) bool {
	if f.Op != t.Op {
		return false
	}

	switch f.Op {
	case gc.OREGISTER:
		if f.Val.U.Reg != t.Val.U.Reg {
			break
		}
		return true
	}

	return false
}

/*
 * generate one instruction:
 *	as f, t
 */
func gins(as int, f *gc.Node, t *gc.Node) *obj.Prog {
	var p *obj.Prog
	var af obj.Addr
	//	Node nod;
	//	int32 v;

	var at obj.Addr

	if f != nil && f.Op == gc.OINDEX {
		gc.Fatal("gins OINDEX not implemented")
	}

	//		regalloc(&nod, &regnode, Z);
	//		v = constnode.vconst;
	//		cgen(f->right, &nod);
	//		constnode.vconst = v;
	//		idx.reg = nod.reg;
	//		regfree(&nod);
	if t != nil && t.Op == gc.OINDEX {
		gc.Fatal("gins OINDEX not implemented")
	}

	//		regalloc(&nod, &regnode, Z);
	//		v = constnode.vconst;
	//		cgen(t->right, &nod);
	//		constnode.vconst = v;
	//		idx.reg = nod.reg;
	//		regfree(&nod);
	af = obj.Addr{}

	at = obj.Addr{}
	if f != nil {
		gc.Naddr(f, &af, 1)
	}
	if t != nil {
		gc.Naddr(t, &at, 1)
	}
	p = gc.Prog(as)
	if f != nil {
		p.From = af
	}
	if t != nil {
		p.To = at
	}
	if gc.Debug['g'] != 0 {
		fmt.Printf("%v\n", p)
	}
	return p
}

/*
 * insert n into reg slot of p
 */
func raddr(n *gc.Node, p *obj.Prog) {
	var a obj.Addr

	gc.Naddr(n, &a, 1)
	if a.Type != obj.TYPE_REG {
		if n != nil {
			gc.Fatal("bad in raddr: %v", gc.Oconv(int(n.Op), 0))
		} else {
			gc.Fatal("bad in raddr: <null>")
		}
		p.Reg = 0
	} else {
		p.Reg = a.Reg
	}
}

/* generate a comparison
TODO(kaib): one of the args can actually be a small constant. relax the constraint and fix call sites.
*/
func gcmp(as int, lhs *gc.Node, rhs *gc.Node) *obj.Prog {
	var p *obj.Prog

	if lhs.Op != gc.OREGISTER {
		gc.Fatal("bad operands to gcmp: %v %v", gc.Oconv(int(lhs.Op), 0), gc.Oconv(int(rhs.Op), 0))
	}

	p = gins(as, rhs, nil)
	raddr(lhs, p)
	return p
}

/* generate a constant shift
 * arm encodes a shift by 32 as 0, thus asking for 0 shift is illegal.
 */
func gshift(as int, lhs *gc.Node, stype int32, sval int32, rhs *gc.Node) *obj.Prog {
	var p *obj.Prog

	if sval <= 0 || sval > 32 {
		gc.Fatal("bad shift value: %d", sval)
	}

	sval = sval & 0x1f

	p = gins(as, nil, rhs)
	p.From.Type = obj.TYPE_SHIFT
	p.From.Offset = int64(stype) | int64(sval)<<7 | int64(lhs.Val.U.Reg)&15
	return p
}

/* generate a register shift
 */
func gregshift(as int, lhs *gc.Node, stype int32, reg *gc.Node, rhs *gc.Node) *obj.Prog {
	var p *obj.Prog
	p = gins(as, nil, rhs)
	p.From.Type = obj.TYPE_SHIFT
	p.From.Offset = int64(stype) | (int64(reg.Val.U.Reg)&15)<<8 | 1<<4 | int64(lhs.Val.U.Reg)&15
	return p
}

/*
 * return Axxx for Oxxx on type t.
 */
func optoas(op int, t *gc.Type) int {
	var a int

	if t == nil {
		gc.Fatal("optoas: t is nil")
	}

	a = obj.AXXX
	switch uint32(op)<<16 | uint32(gc.Simtype[t.Etype]) {
	default:
		gc.Fatal("optoas: no entry %v-%v etype %v simtype %v", gc.Oconv(int(op), 0), gc.Tconv(t, 0), gc.Tconv(gc.Types[t.Etype], 0), gc.Tconv(gc.Types[gc.Simtype[t.Etype]], 0))

		/*	case CASE(OADDR, TPTR32):
				a = ALEAL;
				break;

			case CASE(OADDR, TPTR64):
				a = ALEAQ;
				break;
		*/
	// TODO(kaib): make sure the conditional branches work on all edge cases
	case gc.OEQ<<16 | gc.TBOOL,
		gc.OEQ<<16 | gc.TINT8,
		gc.OEQ<<16 | gc.TUINT8,
		gc.OEQ<<16 | gc.TINT16,
		gc.OEQ<<16 | gc.TUINT16,
		gc.OEQ<<16 | gc.TINT32,
		gc.OEQ<<16 | gc.TUINT32,
		gc.OEQ<<16 | gc.TINT64,
		gc.OEQ<<16 | gc.TUINT64,
		gc.OEQ<<16 | gc.TPTR32,
		gc.OEQ<<16 | gc.TPTR64,
		gc.OEQ<<16 | gc.TFLOAT32,
		gc.OEQ<<16 | gc.TFLOAT64:
		a = arm.ABEQ

	case gc.ONE<<16 | gc.TBOOL,
		gc.ONE<<16 | gc.TINT8,
		gc.ONE<<16 | gc.TUINT8,
		gc.ONE<<16 | gc.TINT16,
		gc.ONE<<16 | gc.TUINT16,
		gc.ONE<<16 | gc.TINT32,
		gc.ONE<<16 | gc.TUINT32,
		gc.ONE<<16 | gc.TINT64,
		gc.ONE<<16 | gc.TUINT64,
		gc.ONE<<16 | gc.TPTR32,
		gc.ONE<<16 | gc.TPTR64,
		gc.ONE<<16 | gc.TFLOAT32,
		gc.ONE<<16 | gc.TFLOAT64:
		a = arm.ABNE

	case gc.OLT<<16 | gc.TINT8,
		gc.OLT<<16 | gc.TINT16,
		gc.OLT<<16 | gc.TINT32,
		gc.OLT<<16 | gc.TINT64,
		gc.OLT<<16 | gc.TFLOAT32,
		gc.OLT<<16 | gc.TFLOAT64:
		a = arm.ABLT

	case gc.OLT<<16 | gc.TUINT8,
		gc.OLT<<16 | gc.TUINT16,
		gc.OLT<<16 | gc.TUINT32,
		gc.OLT<<16 | gc.TUINT64:
		a = arm.ABLO

	case gc.OLE<<16 | gc.TINT8,
		gc.OLE<<16 | gc.TINT16,
		gc.OLE<<16 | gc.TINT32,
		gc.OLE<<16 | gc.TINT64,
		gc.OLE<<16 | gc.TFLOAT32,
		gc.OLE<<16 | gc.TFLOAT64:
		a = arm.ABLE

	case gc.OLE<<16 | gc.TUINT8,
		gc.OLE<<16 | gc.TUINT16,
		gc.OLE<<16 | gc.TUINT32,
		gc.OLE<<16 | gc.TUINT64:
		a = arm.ABLS

	case gc.OGT<<16 | gc.TINT8,
		gc.OGT<<16 | gc.TINT16,
		gc.OGT<<16 | gc.TINT32,
		gc.OGT<<16 | gc.TINT64,
		gc.OGT<<16 | gc.TFLOAT32,
		gc.OGT<<16 | gc.TFLOAT64:
		a = arm.ABGT

	case gc.OGT<<16 | gc.TUINT8,
		gc.OGT<<16 | gc.TUINT16,
		gc.OGT<<16 | gc.TUINT32,
		gc.OGT<<16 | gc.TUINT64:
		a = arm.ABHI

	case gc.OGE<<16 | gc.TINT8,
		gc.OGE<<16 | gc.TINT16,
		gc.OGE<<16 | gc.TINT32,
		gc.OGE<<16 | gc.TINT64,
		gc.OGE<<16 | gc.TFLOAT32,
		gc.OGE<<16 | gc.TFLOAT64:
		a = arm.ABGE

	case gc.OGE<<16 | gc.TUINT8,
		gc.OGE<<16 | gc.TUINT16,
		gc.OGE<<16 | gc.TUINT32,
		gc.OGE<<16 | gc.TUINT64:
		a = arm.ABHS

	case gc.OCMP<<16 | gc.TBOOL,
		gc.OCMP<<16 | gc.TINT8,
		gc.OCMP<<16 | gc.TUINT8,
		gc.OCMP<<16 | gc.TINT16,
		gc.OCMP<<16 | gc.TUINT16,
		gc.OCMP<<16 | gc.TINT32,
		gc.OCMP<<16 | gc.TUINT32,
		gc.OCMP<<16 | gc.TPTR32:
		a = arm.ACMP

	case gc.OCMP<<16 | gc.TFLOAT32:
		a = arm.ACMPF

	case gc.OCMP<<16 | gc.TFLOAT64:
		a = arm.ACMPD

	case gc.OAS<<16 | gc.TBOOL:
		a = arm.AMOVB

	case gc.OAS<<16 | gc.TINT8:
		a = arm.AMOVBS

	case gc.OAS<<16 | gc.TUINT8:
		a = arm.AMOVBU

	case gc.OAS<<16 | gc.TINT16:
		a = arm.AMOVHS

	case gc.OAS<<16 | gc.TUINT16:
		a = arm.AMOVHU

	case gc.OAS<<16 | gc.TINT32,
		gc.OAS<<16 | gc.TUINT32,
		gc.OAS<<16 | gc.TPTR32:
		a = arm.AMOVW

	case gc.OAS<<16 | gc.TFLOAT32:
		a = arm.AMOVF

	case gc.OAS<<16 | gc.TFLOAT64:
		a = arm.AMOVD

	case gc.OADD<<16 | gc.TINT8,
		gc.OADD<<16 | gc.TUINT8,
		gc.OADD<<16 | gc.TINT16,
		gc.OADD<<16 | gc.TUINT16,
		gc.OADD<<16 | gc.TINT32,
		gc.OADD<<16 | gc.TUINT32,
		gc.OADD<<16 | gc.TPTR32:
		a = arm.AADD

	case gc.OADD<<16 | gc.TFLOAT32:
		a = arm.AADDF

	case gc.OADD<<16 | gc.TFLOAT64:
		a = arm.AADDD

	case gc.OSUB<<16 | gc.TINT8,
		gc.OSUB<<16 | gc.TUINT8,
		gc.OSUB<<16 | gc.TINT16,
		gc.OSUB<<16 | gc.TUINT16,
		gc.OSUB<<16 | gc.TINT32,
		gc.OSUB<<16 | gc.TUINT32,
		gc.OSUB<<16 | gc.TPTR32:
		a = arm.ASUB

	case gc.OSUB<<16 | gc.TFLOAT32:
		a = arm.ASUBF

	case gc.OSUB<<16 | gc.TFLOAT64:
		a = arm.ASUBD

	case gc.OMINUS<<16 | gc.TINT8,
		gc.OMINUS<<16 | gc.TUINT8,
		gc.OMINUS<<16 | gc.TINT16,
		gc.OMINUS<<16 | gc.TUINT16,
		gc.OMINUS<<16 | gc.TINT32,
		gc.OMINUS<<16 | gc.TUINT32,
		gc.OMINUS<<16 | gc.TPTR32:
		a = arm.ARSB

	case gc.OAND<<16 | gc.TINT8,
		gc.OAND<<16 | gc.TUINT8,
		gc.OAND<<16 | gc.TINT16,
		gc.OAND<<16 | gc.TUINT16,
		gc.OAND<<16 | gc.TINT32,
		gc.OAND<<16 | gc.TUINT32,
		gc.OAND<<16 | gc.TPTR32:
		a = arm.AAND

	case gc.OOR<<16 | gc.TINT8,
		gc.OOR<<16 | gc.TUINT8,
		gc.OOR<<16 | gc.TINT16,
		gc.OOR<<16 | gc.TUINT16,
		gc.OOR<<16 | gc.TINT32,
		gc.OOR<<16 | gc.TUINT32,
		gc.OOR<<16 | gc.TPTR32:
		a = arm.AORR

	case gc.OXOR<<16 | gc.TINT8,
		gc.OXOR<<16 | gc.TUINT8,
		gc.OXOR<<16 | gc.TINT16,
		gc.OXOR<<16 | gc.TUINT16,
		gc.OXOR<<16 | gc.TINT32,
		gc.OXOR<<16 | gc.TUINT32,
		gc.OXOR<<16 | gc.TPTR32:
		a = arm.AEOR

	case gc.OLSH<<16 | gc.TINT8,
		gc.OLSH<<16 | gc.TUINT8,
		gc.OLSH<<16 | gc.TINT16,
		gc.OLSH<<16 | gc.TUINT16,
		gc.OLSH<<16 | gc.TINT32,
		gc.OLSH<<16 | gc.TUINT32,
		gc.OLSH<<16 | gc.TPTR32:
		a = arm.ASLL

	case gc.ORSH<<16 | gc.TUINT8,
		gc.ORSH<<16 | gc.TUINT16,
		gc.ORSH<<16 | gc.TUINT32,
		gc.ORSH<<16 | gc.TPTR32:
		a = arm.ASRL

	case gc.ORSH<<16 | gc.TINT8,
		gc.ORSH<<16 | gc.TINT16,
		gc.ORSH<<16 | gc.TINT32:
		a = arm.ASRA

	case gc.OMUL<<16 | gc.TUINT8,
		gc.OMUL<<16 | gc.TUINT16,
		gc.OMUL<<16 | gc.TUINT32,
		gc.OMUL<<16 | gc.TPTR32:
		a = arm.AMULU

	case gc.OMUL<<16 | gc.TINT8,
		gc.OMUL<<16 | gc.TINT16,
		gc.OMUL<<16 | gc.TINT32:
		a = arm.AMUL

	case gc.OMUL<<16 | gc.TFLOAT32:
		a = arm.AMULF

	case gc.OMUL<<16 | gc.TFLOAT64:
		a = arm.AMULD

	case gc.ODIV<<16 | gc.TUINT8,
		gc.ODIV<<16 | gc.TUINT16,
		gc.ODIV<<16 | gc.TUINT32,
		gc.ODIV<<16 | gc.TPTR32:
		a = arm.ADIVU

	case gc.ODIV<<16 | gc.TINT8,
		gc.ODIV<<16 | gc.TINT16,
		gc.ODIV<<16 | gc.TINT32:
		a = arm.ADIV

	case gc.OMOD<<16 | gc.TUINT8,
		gc.OMOD<<16 | gc.TUINT16,
		gc.OMOD<<16 | gc.TUINT32,
		gc.OMOD<<16 | gc.TPTR32:
		a = arm.AMODU

	case gc.OMOD<<16 | gc.TINT8,
		gc.OMOD<<16 | gc.TINT16,
		gc.OMOD<<16 | gc.TINT32:
		a = arm.AMOD

		//	case CASE(OEXTEND, TINT16):
	//		a = ACWD;
	//		break;

	//	case CASE(OEXTEND, TINT32):
	//		a = ACDQ;
	//		break;

	//	case CASE(OEXTEND, TINT64):
	//		a = ACQO;
	//		break;

	case gc.ODIV<<16 | gc.TFLOAT32:
		a = arm.ADIVF

	case gc.ODIV<<16 | gc.TFLOAT64:
		a = arm.ADIVD
	}

	return a
}

const (
	ODynam = 1 << 0
	OPtrto = 1 << 1
)

var clean [20]gc.Node

var cleani int = 0

func sudoclean() {
	if clean[cleani-1].Op != gc.OEMPTY {
		regfree(&clean[cleani-1])
	}
	if clean[cleani-2].Op != gc.OEMPTY {
		regfree(&clean[cleani-2])
	}
	cleani -= 2
}

func dotaddable(n *gc.Node, n1 *gc.Node) bool {
	var o int
	var oary [10]int64
	var nn *gc.Node

	if n.Op != gc.ODOT {
		return false
	}

	o = gc.Dotoffset(n, oary[:], &nn)
	if nn != nil && nn.Addable != 0 && o == 1 && oary[0] >= 0 {
		*n1 = *nn
		n1.Type = n.Type
		n1.Xoffset += oary[0]
		return true
	}

	return false
}

/*
 * generate code to compute address of n,
 * a reference to a (perhaps nested) field inside
 * an array or struct.
 * return 0 on failure, 1 on success.
 * on success, leaves usable address in a.
 *
 * caller is responsible for calling sudoclean
 * after successful sudoaddable,
 * to release the register used for a.
 */
func sudoaddable(as int, n *gc.Node, a *obj.Addr, w *int) bool {
	var o int
	var i int
	var oary [10]int64
	var v int64
	var n1 gc.Node
	var n2 gc.Node
	var n3 gc.Node
	var n4 gc.Node
	var nn *gc.Node
	var l *gc.Node
	var r *gc.Node
	var reg *gc.Node
	var reg1 *gc.Node
	var p1 *obj.Prog
	var p2 *obj.Prog
	var t *gc.Type

	if n.Type == nil {
		return false
	}

	*a = obj.Addr{}

	switch n.Op {
	case gc.OLITERAL:
		if !gc.Isconst(n, gc.CTINT) {
			break
		}
		v = gc.Mpgetfix(n.Val.U.Xval)
		if v >= 32000 || v <= -32000 {
			break
		}
		goto lit

	case gc.ODOT,
		gc.ODOTPTR:
		cleani += 2
		reg = &clean[cleani-1]
		reg1 = &clean[cleani-2]
		reg.Op = gc.OEMPTY
		reg1.Op = gc.OEMPTY
		goto odot

	case gc.OINDEX:
		return false

		// disabled: OINDEX case is now covered by agenr
		// for a more suitable register allocation pattern.
		if n.Left.Type.Etype == gc.TSTRING {
			return false
		}
		cleani += 2
		reg = &clean[cleani-1]
		reg1 = &clean[cleani-2]
		reg.Op = gc.OEMPTY
		reg1.Op = gc.OEMPTY
		goto oindex
	}

	return false

lit:
	switch as {
	default:
		return false

	case arm.AADD,
		arm.ASUB,
		arm.AAND,
		arm.AORR,
		arm.AEOR,
		arm.AMOVB,
		arm.AMOVBS,
		arm.AMOVBU,
		arm.AMOVH,
		arm.AMOVHS,
		arm.AMOVHU,
		arm.AMOVW:
		break
	}

	cleani += 2
	reg = &clean[cleani-1]
	reg1 = &clean[cleani-2]
	reg.Op = gc.OEMPTY
	reg1.Op = gc.OEMPTY
	gc.Naddr(n, a, 1)
	goto yes

odot:
	o = gc.Dotoffset(n, oary[:], &nn)
	if nn == nil {
		goto no
	}

	if nn.Addable != 0 && o == 1 && oary[0] >= 0 {
		// directly addressable set of DOTs
		n1 = *nn

		n1.Type = n.Type
		n1.Xoffset += oary[0]
		gc.Naddr(&n1, a, 1)
		goto yes
	}

	regalloc(reg, gc.Types[gc.Tptr], nil)
	n1 = *reg
	n1.Op = gc.OINDREG
	if oary[0] >= 0 {
		agen(nn, reg)
		n1.Xoffset = oary[0]
	} else {
		cgen(nn, reg)
		gc.Cgen_checknil(reg)
		n1.Xoffset = -(oary[0] + 1)
	}

	for i = 1; i < o; i++ {
		if oary[i] >= 0 {
			gc.Fatal("can't happen")
		}
		gins(arm.AMOVW, &n1, reg)
		gc.Cgen_checknil(reg)
		n1.Xoffset = -(oary[i] + 1)
	}

	a.Type = obj.TYPE_NONE
	a.Name = obj.NAME_NONE
	n1.Type = n.Type
	gc.Naddr(&n1, a, 1)
	goto yes

oindex:
	l = n.Left
	r = n.Right
	if l.Ullman >= gc.UINF && r.Ullman >= gc.UINF {
		goto no
	}

	// set o to type of array
	o = 0

	if gc.Isptr[l.Type.Etype] != 0 {
		o += OPtrto
		if l.Type.Type.Etype != gc.TARRAY {
			gc.Fatal("not ptr ary")
		}
		if l.Type.Type.Bound < 0 {
			o += ODynam
		}
	} else {
		if l.Type.Etype != gc.TARRAY {
			gc.Fatal("not ary")
		}
		if l.Type.Bound < 0 {
			o += ODynam
		}
	}

	*w = int(n.Type.Width)
	if gc.Isconst(r, gc.CTINT) {
		goto oindex_const
	}

	switch *w {
	default:
		goto no

	case 1,
		2,
		4,
		8:
		break
	}

	// load the array (reg)
	if l.Ullman > r.Ullman {
		regalloc(reg, gc.Types[gc.Tptr], nil)
		if o&OPtrto != 0 {
			cgen(l, reg)
			gc.Cgen_checknil(reg)
		} else {
			agen(l, reg)
		}
	}

	// load the index (reg1)
	t = gc.Types[gc.TUINT32]

	if gc.Issigned[r.Type.Etype] != 0 {
		t = gc.Types[gc.TINT32]
	}
	regalloc(reg1, t, nil)
	regalloc(&n3, gc.Types[gc.TINT32], reg1)
	p2 = cgenindex(r, &n3, gc.Debug['B'] != 0 || n.Bounded)
	gmove(&n3, reg1)
	regfree(&n3)

	// load the array (reg)
	if l.Ullman <= r.Ullman {
		regalloc(reg, gc.Types[gc.Tptr], nil)
		if o&OPtrto != 0 {
			cgen(l, reg)
			gc.Cgen_checknil(reg)
		} else {
			agen(l, reg)
		}
	}

	// check bounds
	if gc.Debug['B'] == 0 {
		if o&ODynam != 0 {
			n2 = *reg
			n2.Op = gc.OINDREG
			n2.Type = gc.Types[gc.Tptr]
			n2.Xoffset = int64(gc.Array_nel)
		} else {
			if o&OPtrto != 0 {
				gc.Nodconst(&n2, gc.Types[gc.TUINT32], l.Type.Type.Bound)
			} else {
				gc.Nodconst(&n2, gc.Types[gc.TUINT32], l.Type.Bound)
			}
		}

		regalloc(&n3, n2.Type, nil)
		cgen(&n2, &n3)
		gcmp(optoas(gc.OCMP, gc.Types[gc.TUINT32]), reg1, &n3)
		regfree(&n3)
		p1 = gc.Gbranch(optoas(gc.OLT, gc.Types[gc.TUINT32]), nil, +1)
		if p2 != nil {
			gc.Patch(p2, gc.Pc)
		}
		ginscall(gc.Panicindex, 0)
		gc.Patch(p1, gc.Pc)
	}

	if o&ODynam != 0 {
		n2 = *reg
		n2.Op = gc.OINDREG
		n2.Type = gc.Types[gc.Tptr]
		n2.Xoffset = int64(gc.Array_array)
		gmove(&n2, reg)
	}

	switch *w {
	case 1:
		gins(arm.AADD, reg1, reg)

	case 2:
		gshift(arm.AADD, reg1, arm.SHIFT_LL, 1, reg)

	case 4:
		gshift(arm.AADD, reg1, arm.SHIFT_LL, 2, reg)

	case 8:
		gshift(arm.AADD, reg1, arm.SHIFT_LL, 3, reg)
	}

	gc.Naddr(reg1, a, 1)
	a.Type = obj.TYPE_MEM
	a.Reg = reg.Val.U.Reg
	a.Offset = 0
	goto yes

	// index is constant
	// can check statically and
	// can multiply by width statically

oindex_const:
	regalloc(reg, gc.Types[gc.Tptr], nil)

	if o&OPtrto != 0 {
		cgen(l, reg)
		gc.Cgen_checknil(reg)
	} else {
		agen(l, reg)
	}

	v = gc.Mpgetfix(r.Val.U.Xval)
	if o&ODynam != 0 {
		if gc.Debug['B'] == 0 && !n.Bounded {
			n1 = *reg
			n1.Op = gc.OINDREG
			n1.Type = gc.Types[gc.Tptr]
			n1.Xoffset = int64(gc.Array_nel)
			gc.Nodconst(&n2, gc.Types[gc.TUINT32], v)
			regalloc(&n3, gc.Types[gc.TUINT32], nil)
			cgen(&n2, &n3)
			regalloc(&n4, n1.Type, nil)
			cgen(&n1, &n4)
			gcmp(optoas(gc.OCMP, gc.Types[gc.TUINT32]), &n4, &n3)
			regfree(&n4)
			regfree(&n3)
			p1 = gc.Gbranch(optoas(gc.OGT, gc.Types[gc.TUINT32]), nil, +1)
			ginscall(gc.Panicindex, 0)
			gc.Patch(p1, gc.Pc)
		}

		n1 = *reg
		n1.Op = gc.OINDREG
		n1.Type = gc.Types[gc.Tptr]
		n1.Xoffset = int64(gc.Array_array)
		gmove(&n1, reg)
	}

	n2 = *reg
	n2.Op = gc.OINDREG
	n2.Xoffset = v * int64(*w)
	a.Type = obj.TYPE_NONE
	a.Name = obj.NAME_NONE
	gc.Naddr(&n2, a, 1)
	goto yes

yes:
	return true

no:
	sudoclean()
	return false
}